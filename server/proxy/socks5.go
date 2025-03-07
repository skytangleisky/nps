package proxy

import (
	"ehang.io/nps/lib/common"
	"ehang.io/nps/lib/conn"
	"ehang.io/nps/lib/file"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/astaxie/beego/logs"
	"io"
	"net"
	"strconv"
)

const (
	ipV4            = 1
	domainName      = 3
	ipV6            = 4
	connectMethod   = 1
	bindMethod      = 2
	associateMethod = 3
	// The maximum packet size of any udp Associate packet, based on ethernet's max size,
	// minus the IP and UDP headers. IPv4 has a 20 byte header, UDP adds an
	// additional 4 bytes.  This is a total overhead of 24 bytes.  Ethernet's
	// max packet size is 1500 bytes,  1500 - 24 = 1476.
	maxUDPPacketSize = 1476
)

const (
	succeeded uint8 = iota
	serverFailure
	notAllowed
	networkUnreachable
	hostUnreachable
	connectionRefused
	ttlExpired
	commandNotSupported
	addrTypeNotSupported
)

const (
	UserPassAuth    = uint8(2)
	userAuthVersion = uint8(1)
	authSuccess     = uint8(0)
	authFailure     = uint8(1)
)

type Sock5ModeServer struct {
	BaseServer
	listener net.Listener
}

// req
func (s *Sock5ModeServer) handleRequest(c net.Conn) {
	/*
		The SOCKS request is formed as follows:
		+----+-----+-------+------+----------+----------+
		|VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
		+----+-----+-------+------+----------+----------+
		| 1  |  1  | X'00' |  1   | Variable |    2     |
		+----+-----+-------+------+----------+----------+
	*/
	header := make([]byte, 3)

	n, err := c.Read(header)
	if err != nil {
		logs.Warn("read header err", err)
		_ = c.Close()
		return
	}
	if n < 3 {
		logs.Error("header is less than 3 bytes", hex.EncodeToString(header[:n]))
		_ = c.Close()
		return
	}

	switch header[1] {
	case connectMethod:
		s.handleConnect(c)
	case bindMethod:
		s.handleBind(c)
	case associateMethod:
		s.handleUDP(c)
	default:
		s.sendReply(c, "", commandNotSupported)
		_ = c.Close()
	}
}

// reply
func (s *Sock5ModeServer) sendReply(c net.Conn, addr string, rep uint8) {
	reply := []byte{5, rep, 0}
	if addr != "" {
		logs.Alert("====>", addr)
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			logs.Error("failed to split host and port", err)
			_ = c.Close()
			return
		}
		ipBytes := net.ParseIP(host).To4()
		if ipBytes != nil {
			reply = append(reply, 1)
		} else {
			ipBytes = net.ParseIP(host).To16()
			reply = append(reply, 4)
		}
		nPort, err := strconv.Atoi(port)
		if err != nil {
			logs.Error("failed to convert port to int", err)
			_ = c.Close()
			return
		}
		reply = append(reply, ipBytes...)
		portBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(portBytes, uint16(nPort))
		reply = append(reply, portBytes...)
	} else {
		reply[1] = networkUnreachable
	}
	_, err := c.Write(reply)
	if err != nil {
		logs.Error("failed to send reply", hex.EncodeToString(reply))
		_ = c.Close()
		return
	}
}

// do conn
func (s *Sock5ModeServer) doConnect(c net.Conn, command uint8) {
	addrType := make([]byte, 1)
	_, err := c.Read(addrType)
	if err != nil {
		logs.Error("failed to read addrType", err)
		_ = c.Close()
		return
	}
	var host string
	switch addrType[0] {
	case ipV4:
		ipv4 := make(net.IP, net.IPv4len)
		_, err = c.Read(ipv4)
		if err != nil {
			logs.Error("failed to read ipv4", err)
			_ = c.Close()
			return
		}
		host = ipv4.String()
	case ipV6:
		ipv6 := make(net.IP, net.IPv6len)
		_, err = c.Read(ipv6)
		if err != nil {
			logs.Error("failed to read ipv6", err)
			_ = c.Close()
			return
		}
		host = ipv6.String()
	case domainName:
		var domainLen uint8
		err = binary.Read(c, binary.BigEndian, &domainLen)
		if err != nil {
			logs.Error("failed to read domainLen", err)
			_ = c.Close()
			return
		}
		domain := make([]byte, domainLen)
		_, err = c.Read(domain)
		if err != nil {
			logs.Error("failed to read domain", err)
			_ = c.Close()
			return
		}
		host = string(domain)
	default:
		s.sendReply(c, "", addrTypeNotSupported)
		_ = c.Close()
		return
	}

	var port uint16
	err = binary.Read(c, binary.BigEndian, &port)
	if err != nil {
		logs.Error("failed to read port", err)
		_ = c.Close()
		return
	}
	// connect to host
	addr := net.JoinHostPort(host, strconv.Itoa(int(port)))
	var ltype string
	if command == associateMethod {
		ltype = common.CONN_UDP
	} else {
		ltype = common.CONN_TCP
	}
	err = s.DealClient(conn.NewConn(c), s.task.Client, addr, nil, ltype, func(addr string) {
		s.sendReply(c, addr, succeeded)
	}, s.task.Flow, s.task.Target.LocalProxy)
	if err != nil {
		logs.Error("DealClient failed", err)
		_ = c.Close()
		return
	}
	return
}

// conn
func (s *Sock5ModeServer) handleConnect(c net.Conn) {
	s.doConnect(c, connectMethod)
}

// 对bind支持的alpha版本，市面上使用极少（像新版ftp客户端,filezilla,curl等均不支持bind指令,包括下面的命令也不支持）
// curl -u 2413081441@qq.com:Tanglei201314 ftp://192.168.101.104:21/ --socks5 socks5://tanglei.site:5555 --ftp-port :1234 (实际并没有发送bind指令)
func (s *Sock5ModeServer) handleBind(c net.Conn) {
	reply := []byte{5, 7, 0}
	buffer := make([]byte, 1)
	_, err := c.Read(buffer)
	if err != nil {
		reply[1] = serverFailure
		_, err = c.Write(reply)
		if err != nil {
			logs.Error("failed to write reply", hex.EncodeToString(reply))
			_ = c.Close()
			return
		}
		return
	}
	addrType := buffer[0]
	var host string
	switch addrType {
	case ipV4:
		ipv4 := make(net.IP, net.IPv4len)
		_, err = c.Read(ipv4)
		if err != nil {
			logs.Error("failed to read ipv4", err)
			_ = c.Close()
			return
		}
		host = ipv4.String()
	case ipV6:
		ipv6 := make(net.IP, net.IPv6len)
		_, err = c.Read(ipv6)
		if err != nil {
			logs.Error("failed to read ipv6", err)
			_ = c.Close()
			return
		}
		host = ipv6.String()
	case domainName:
		var domainLen uint8
		err = binary.Read(c, binary.BigEndian, &domainLen)
		if err != nil {
			logs.Error("failed to read domainLen", err)
			_ = c.Close()
			return
		}
		domain := make([]byte, domainLen)
		_, err = c.Read(domain)
		if err != nil {
			logs.Error("failed to read domain", err)
			_ = c.Close()
			return
		}
		host = string(domain)
	default:
		reply[1] = addrTypeNotSupported
		_, err = c.Write(reply)
		if err != nil {
			logs.Error("failed to write reply", hex.EncodeToString(reply))
			_ = c.Close()
			return
		}
		return
	}
	var port uint16
	err = binary.Read(c, binary.BigEndian, &port)
	if err != nil {
		logs.Error("failed to read port", err)
		_ = c.Close()
		return
	}
	addr := net.JoinHostPort(host, strconv.Itoa(int(port)))
	err = s.DealClient(conn.NewConn(c), s.task.Client, addr, nil, common.CONN_BIND, func(addr string) {
		s.sendReply(c, addr, succeeded)
	}, s.task.Flow, s.task.Target.LocalProxy)
	if err != nil {
		logs.Error("DealClient failed", err)
		_ = c.Close()
		return
	}
}
func (s *Sock5ModeServer) sendUdpReply(writeConn net.Conn, c net.Conn, rep uint8, serverIp string) {
	reply := []byte{5, rep, 0}
	localHost, localPort, err := net.SplitHostPort(c.LocalAddr().String())
	if err != nil {
		logs.Error("failed to string local address", err)
		_ = c.Close()
		return
	}
	localHost = serverIp
	ipBytes := net.ParseIP(localHost).To4()
	if ipBytes != nil {
		reply = append(reply, 1)
	} else {
		ipBytes = net.ParseIP(localHost).To16()
		reply = append(reply, 4)
	}
	nPort, _ := strconv.Atoi(localPort)
	reply = append(reply, ipBytes...)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, uint16(nPort))
	reply = append(reply, portBytes...)
	_, err = writeConn.Write(reply)
	if err != nil {
		logs.Error("failed to write reply", hex.EncodeToString(reply))
		_ = c.Close()
		return
	}
}

func (s *Sock5ModeServer) handleUDP(c net.Conn) {
	addrType := make([]byte, 1)
	_, err := c.Read(addrType)
	if err != nil {
		logs.Error("failed to read addrType", hex.EncodeToString(addrType))
		_ = c.Close()
		return
	}
	var host string
	switch addrType[0] {
	case ipV4:
		ipv4 := make(net.IP, net.IPv4len)
		_, err = c.Read(ipv4)
		if err != nil {
			logs.Error("failed to read ipv4", err)
			_ = c.Close()
			return
		}
		host = ipv4.String()
	case ipV6:
		ipv6 := make(net.IP, net.IPv6len)
		_, err = c.Read(ipv6)
		if err != nil {
			logs.Error("failed to read ipv6", err)
			_ = c.Close()
			return
		}
		host = ipv6.String()
	case domainName:
		var domainLen uint8
		err = binary.Read(c, binary.BigEndian, &domainLen)
		if err != nil {
			logs.Error("failed to read domainLen", err)
			_ = c.Close()
			return
		}
		domain := make([]byte, domainLen)
		_, err = c.Read(domain)
		if err != nil {
			logs.Error("failed to read domain", err)
			_ = c.Close()
			return
		}
		host = string(domain)
	default:
		s.sendReply(c, "", addrTypeNotSupported)
		return
	}
	//读取端口
	var port uint16
	err = binary.Read(c, binary.BigEndian, &port)
	if err != nil {
		logs.Error("failed to read port", err)
		_ = c.Close()
		return
	}
	logs.Warn(host, strconv.Itoa(int(port)))
	replyAddr, err := net.ResolveUDPAddr("udp", s.task.ServerIp+":0")
	if err != nil {
		logs.Error("build local reply addr error", err)
		return
	}
	reply, err := net.ListenUDP("udp", replyAddr)
	if err != nil {
		s.sendReply(c, "", addrTypeNotSupported)
		logs.Error("listen local reply udp port error")
		return
	}
	// reply the local addr
	s.sendUdpReply(c, reply, succeeded, common.GetServerIpByClientIp(c.RemoteAddr().(*net.TCPAddr).IP))
	defer func() {
		_ = reply.Close()
	}()
	// new a tunnel to client
	link := conn.NewLink("udp5", "", s.task.Client.Cnf.Crypt, s.task.Client.Cnf.Compress, c.RemoteAddr().String(), false)
	target, err := s.bridge.SendLinkInfo(s.task.Client.Id, link, s.task)
	if err != nil {
		logs.Warn("get connection from client id %d  error %s", s.task.Client.Id, err.Error())
		return
	}

	var clientAddr net.Addr
	// copy buffer
	go func() {
		b := common.BufPoolUdp.Get().([]byte)
		defer common.BufPoolUdp.Put(b)
		defer func() {
			_ = c.Close()
		}()

		for {
			n, laddr, err := reply.ReadFrom(b)
			if err != nil {
				logs.Error("read data from %s err %s", reply.LocalAddr().String(), err.Error())
				return
			}
			if clientAddr == nil {
				clientAddr = laddr
			}
			if _, err := target.Write(b[:n]); err != nil {
				logs.Error("write data to client error", err.Error())
				return
			}
		}
	}()

	go func() {
		var l int32
		b := common.BufPoolUdp.Get().([]byte)
		defer common.BufPoolUdp.Put(b)
		defer func() {
			_ = c.Close()
		}()
		for {
			if err := binary.Read(target, binary.LittleEndian, &l); err != nil || l >= common.PoolSizeUdp || l <= 0 {
				logs.Warn("read len bytes error", err.Error())
				return
			}
			err = binary.Read(target, binary.LittleEndian, b[:l])
			if err != nil {
				logs.Warn("read data form target error", err.Error())
				return
			}
			if _, err := reply.WriteTo(b[:l], clientAddr); err != nil {
				logs.Warn("write data to user ", err.Error())
				return
			}
		}
	}()

	b := common.BufPoolUdp.Get().([]byte)
	defer common.BufPoolUdp.Put(b)
	defer func() {
		_ = target.Close()
	}()
	for {
		_, err := c.Read(b)
		if err != nil {
			_ = c.Close()
			return
		}
	}
}

// new conn
func (s *Sock5ModeServer) handleConn(c net.Conn) {
	buf := make([]byte, 2)
	n, err := c.Read(buf)
	if err != nil {
		logs.Warn("negotiation err", err)
		_ = c.Close()
		return
	}
	if n < 2 {
		logs.Error("negotiation is less than 2 bytes", hex.EncodeToString(buf[:n]))
		_ = c.Close()
		return
	}

	if version := buf[0]; version != 5 {
		logs.Warn("only support socks5, request from: ", c.RemoteAddr())
		_ = c.Close()
		return
	}
	nMethods := buf[1]

	methods := make([]byte, nMethods)
	if length, err := c.Read(methods); length != int(nMethods) || err != nil {
		logs.Warn("wrong method")
		_ = c.Close()
		return
	}
	if (s.task.Client.Cnf.U != "" && s.task.Client.Cnf.P != "") || (s.task.MultiAccount != nil && len(s.task.MultiAccount.AccountMap) > 0) {
		buf[1] = UserPassAuth
		_, err := c.Write(buf)
		if err != nil {
			logs.Error("failed to write UserPassAuth", hex.EncodeToString(buf))
			_ = c.Close()
			return
		}
		if err := s.Auth(c); err != nil {
			_ = c.Close()
			logs.Warn("Validation failed:", err)
			return
		}
	} else {
		buf[1] = 0
		_, err := c.Write(buf)
		if err != nil {
			logs.Error("failed to write UserPassAuth passed", hex.EncodeToString(buf))
			_ = c.Close()
			return
		}
	}
	s.handleRequest(c)
}

// socks5 auth
func (s *Sock5ModeServer) Auth(c net.Conn) error {
	header := []byte{0, 0}
	if _, err := io.ReadAtLeast(c, header, 2); err != nil {
		return err
	}
	if header[0] != userAuthVersion {
		return errors.New("验证方式不被支持")
	}
	userLen := int(header[1])
	user := make([]byte, userLen)
	if _, err := io.ReadAtLeast(c, user, userLen); err != nil {
		return err
	}
	if _, err := c.Read(header[:1]); err != nil {
		return errors.New("密码长度获取错误")
	}
	passLen := int(header[0])
	pass := make([]byte, passLen)
	if _, err := io.ReadAtLeast(c, pass, passLen); err != nil {
		return err
	}

	var U, P string
	if s.task.MultiAccount != nil {
		// enable multi user auth
		U = string(user)
		var ok bool
		P, ok = s.task.MultiAccount.AccountMap[U]
		if !ok {
			return errors.New("验证不通过")
		}
	} else {
		U = s.task.Client.Cnf.U
		P = s.task.Client.Cnf.P
	}

	if string(user) == U && string(pass) == P {
		if _, err := c.Write([]byte{userAuthVersion, authSuccess}); err != nil {
			return err
		}
		return nil
	} else {
		if _, err := c.Write([]byte{userAuthVersion, authFailure}); err != nil {
			return err
		}
		return errors.New("验证不通过")
	}
}

// start
func (s *Sock5ModeServer) Start() error {
	return conn.NewTcpListenerAndProcess(s.task.ServerIp+":"+strconv.Itoa(s.task.Port), func(c net.Conn) {
		if err := s.CheckFlowAndConnNum(s.task.Client); err != nil {
			logs.Warn("client id %d, task id %d, error %s, when socks5 connection", s.task.Client.Id, s.task.Id, err.Error())
			_ = c.Close()
			return
		}
		logs.Trace("New socks5 connection,client %d,remote address %s", s.task.Client.Id, c.RemoteAddr())
		s.handleConn(c)
		s.task.Client.AddConn()
	}, &s.listener)
}

// new
func NewSock5ModeServer(bridge NetBridge, task *file.Tunnel) *Sock5ModeServer {
	s := new(Sock5ModeServer)
	s.bridge = bridge
	s.task = task
	return s
}

// close
func (s *Sock5ModeServer) Close() error {
	return s.listener.Close()
}
