package client

import (
	"bytes"
	"ehang.io/nps/nps-mux"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/xtaci/kcp-go"

	"ehang.io/nps/lib/common"
	"ehang.io/nps/lib/config"
	"ehang.io/nps/lib/conn"
	"ehang.io/nps/lib/crypt"
)

type TRPClient struct {
	svrAddr        string
	bridgeConnType string
	proxyUrl       string
	vKey           string
	p2pAddr        map[string]string
	tunnel         *nps_mux.Mux
	signal         *conn.Conn
	ticker         *time.Ticker
	cnf            *config.Config
	disconnectTime int
	once           sync.Once
	logout         bool
}

// new client
func NewRPClient(svraddr string, vKey string, bridgeConnType string, proxyUrl string, cnf *config.Config, disconnectTime int) *TRPClient {
	return &TRPClient{
		svrAddr:        svraddr,
		p2pAddr:        make(map[string]string, 0),
		vKey:           vKey,
		bridgeConnType: bridgeConnType,
		proxyUrl:       proxyUrl,
		cnf:            cnf,
		disconnectTime: disconnectTime,
		once:           sync.Once{},
		logout:         false,
	}
}

var NowStatus int

// start
func (s *TRPClient) Start() {
	s.logout = false
retry:
	if s.logout {
		return
	}
	NowStatus = 0
	c, err := NewConn(s.bridgeConnType, s.vKey, s.svrAddr, common.WORK_MAIN, s.proxyUrl)
	if err != nil {
		logs.Error("The connection server failed and will be reconnected in five seconds, error", err.Error())
		time.Sleep(time.Second * 5)
		goto retry
	}
	if c == nil {
		logs.Error("Error data from server, and will be reconnected in five seconds")
		time.Sleep(time.Second * 5)
		goto retry
	}
	logs.Info("%s -> %s", c.LocalAddr(), s.svrAddr)
	//monitor the connection
	go s.ping2()
	s.signal = c
	//start a channel connection
	go s.newChan() //tanglei
	//start health check if it's open
	if s.cnf != nil && len(s.cnf.Healths) > 0 {
		go healthCheck(s.cnf.Healths, s.signal)
	}
	NowStatus = 1
	//msg connection, eg udp
	s.handleMain()
}

// handle main connection
func (s *TRPClient) handleMain() {
	for {
		flags, err := s.signal.ReadFlag()
		if err != nil {
			logs.Error("Accept server data error %s, end this service,%s", err.Error(), s.signal.LocalAddr())
			break
		}
		logs.Warn("flag", "=>", flags)
		switch flags {
		case common.NEW_UDP_CONN:
			//read server udp addr and password
			if lAddr, err := s.signal.GetShortLenContent(); err != nil {
				logs.Warn(err)
				return
			} else if pwd, err := s.signal.GetShortLenContent(); err == nil {
				var localAddr string
				//The local port remains unchanged for a certain period of time
				if v, ok := s.p2pAddr[crypt.Md5(string(pwd)+strconv.Itoa(int(time.Now().Unix()/100)))]; !ok {
					tmpConn, err := common.GetLocalUdpAddr()
					if err != nil {
						logs.Error(err)
						return
					}
					localAddr = tmpConn.LocalAddr().String()
				} else {
					localAddr = v
				}
				go s.newUdpConn(localAddr, string(lAddr), string(pwd))
			}
		case common.RES_CLOSE:
			s.logout = true
			os.Exit(0)
			goto exit
		}
	}
exit:
	s.Close()
}

//func logout(s *TRPClient)  {
//	key, _, err := registry.CreateKey(registry.LOCAL_MACHINE, "software\\lollipop", registry.ALL_ACCESS)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer key.Close()
//
//	//subKey, _, err := registry.CreateKey(key, "test", registry.ALL_ACCESS)
//	//if err != nil {
//	//	log.Fatal(err)
//	//}
//	//defer subKey.Close()
//
//	key1, err := registry.OpenKey(registry.LOCAL_MACHINE, "software\\lollipop", registry.ALL_ACCESS)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer key1.Close()
//
//	//err = key.SetStringValue("K", "V")
//	//if err != nil {
//	//	log.Fatal(err)
//	//}
//
//	v, _, err := key.GetStringValue("config")
//	if err != nil {
//		log.Fatal(err)
//	}
//	//fmt.Println(v, vt)
//
//	data:=make([]map[string]interface{},0)
//	e := json.Unmarshal([]byte(v), &data)
//	if e != nil {
//		panic(e)
//	}
//	for _,person :=range data{
//		if person["unionid"] == s.vKey{
//			person["isLogined"]=false
//		}
//	}
//
//
//	out,er:=json.MarshalIndent(data,"","\t")
//	if er != nil{
//		panic(er)
//	}
//	fmt.Print(string(out))
//
//	err = key.SetStringValue("config", string(out))
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	/*
//		kns, err := key.ReadSubKeyNames(0)
//		if err != nil {
//			log.Fatal(err)
//		}
//		fmt.Println(kns)
//	*/
//	/*
//		vns, err := key.ReadValueNames(0)
//		if err != nil {
//			log.Fatal(err)
//		}
//		fmt.Println(vns)
//	*/
//	/*
//		err = key.DeleteValue("A")
//		if err != nil {
//			log.Fatal(err)
//		}
//	*/
//
//	/*
//		err = registry.DeleteKey(registry.LOCAL_MACHINE, "test")
//		if err != nil {
//			log.Fatal(err)
//		}
//	*/
//	/*
//		err = registry.DeleteKey(key, "test_sub")
//		if err != nil {
//			log.Fatal(err)
//		}
//	*/
//}

func (s *TRPClient) newUdpConn(localAddr, rAddr string, md5Password string) {
	var localConn net.PacketConn
	var err error
	var remoteAddress string
	if remoteAddress, localConn, err = handleP2PUdp2(localAddr, rAddr, md5Password, common.WORK_P2P_PROVIDER); err != nil {
		logs.Error(err)
		return
	}
	l, err := kcp.ServeConn(nil, 150, 3, localConn)
	if err != nil {
		logs.Error(err)
		return
	}
	logs.Trace("start local p2p udp listen, local address", localConn.LocalAddr().String())
	for {
		udpTunnel, err := l.AcceptKCP()
		if err != nil {
			logs.Error(err)
			l.Close()
			return
		}
		if udpTunnel.RemoteAddr().String() == string(remoteAddress) {
			conn.SetUdpSession(udpTunnel)
			logs.Trace("successful connection with client ,address %s", udpTunnel.RemoteAddr().String())
			//read link info from remote
			conn.Accept(nps_mux.NewMux(udpTunnel, s.bridgeConnType, s.disconnectTime), func(c net.Conn) {
				go s.handleChan(c)
			})
			break
		}
	}
}

// pmux tunnel
func (s *TRPClient) newChan() {
	tunnel, err := NewConn(s.bridgeConnType, s.vKey, s.svrAddr, common.WORK_CHAN, s.proxyUrl)
	if err != nil {
		logs.Error("connect to ", s.svrAddr, "error:", err.Error())
		return
	}
	logs.Info("%s -> %s", tunnel.LocalAddr(), s.svrAddr)
	s.tunnel = nps_mux.NewMux(tunnel.Conn, s.bridgeConnType, s.disconnectTime)
	for {
		src, err := s.tunnel.Accept()
		if err != nil {
			logs.Error(err.Error(), tunnel.Conn.LocalAddr())
			s.Close()
			break
		}
		go s.handleChan(src)
	}
}

func (s *TRPClient) handleChan(src net.Conn) {
	lk, err := conn.NewConn(src).GetLinkInfo()
	if err != nil || lk == nil {
		src.Close()
		logs.Error("get connection info from server error ", err)
		return
	}
	//host for target processing
	lk.Host = common.FormatAddress(lk.Host)
	//if Conn type is http, read the request and log
	if lk.ConnType == "http" {
		if targetConn, err := net.DialTimeout(common.CONN_TCP, lk.Host, lk.Option.Timeout); err != nil {
			logs.Warn("connect to %s error %s", lk.Host, err.Error())
			src.Close()
		} else {
			logs.Trace("new %s connection with the goal of %s, remote address:%s", lk.ConnType, lk.Host, lk.RemoteAddr)
			conn.CopyWaitGroup(src, targetConn, lk.Crypt, lk.Compress, nil, nil, false, nil)
			/*srcConn := conn.GetConn(src, lk.Crypt, lk.Compress, nil, false)
			go func() {
				common.CopyBuffer(srcConn, targetConn)
				srcConn.Close()
				targetConn.Close()
			}()
			if r, err := http.ReadRequest(bufio.NewReader(srcConn)); err != nil {
				srcConn.Close()
				targetConn.Close()
				return
			} else {
				logs.Trace("http request, method %s, host %s, url %s, remote address %s", r.Method, r.Host, r.URL.Path, r.RemoteAddr)
				r.Write(targetConn)
			}
			common.CopyBuffer(targetConn, srcConn)
			srcConn.Close()
			targetConn.Close()*/
		}
		return
	}
	if lk.ConnType == "udp5" {
		logs.Trace("new %s connection with the goal of %s, remote address:%s", lk.ConnType, lk.Host, lk.RemoteAddr)
		s.handleUdp(src)
	}
	//connect to target if conn type is tcp or udp
	if targetConn, err := net.DialTimeout(lk.ConnType, lk.Host, lk.Option.Timeout); err != nil {
		logs.Warn("connect to %s error %s", lk.Host, err.Error())
		src.Close()
	} else {
		logs.Trace("new %s connection with the goal of %s, remote address:%s", lk.ConnType, lk.Host, lk.RemoteAddr)
		conn.CopyWaitGroup(src, targetConn, lk.Crypt, lk.Compress, nil, nil, false, nil)
	}
}

func (s *TRPClient) handleUdp(serverConn net.Conn) {
	// bind a local udp port
	local, err := net.ListenUDP("udp", nil)
	defer serverConn.Close()
	if err != nil {
		logs.Error("bind local udp port error ", err.Error())
		return
	}
	defer local.Close()
	go func() {
		defer serverConn.Close()
		b := common.BufPoolUdp.Get().([]byte)
		defer common.BufPoolUdp.Put(b)
		for {
			n, raddr, err := local.ReadFrom(b)
			if err != nil {
				logs.Error("read data from remote server error", err.Error())
			}
			buf := bytes.Buffer{}
			dgram := common.NewUDPDatagram(common.NewUDPHeader(0, 0, common.ToSocksAddr(raddr)), b[:n])
			dgram.Write(&buf)
			b, err := conn.GetLenBytes(buf.Bytes())
			if err != nil {
				logs.Warn("get len bytes error", err.Error())
				continue
			}
			if _, err := serverConn.Write(b); err != nil {
				logs.Error("write data to remote  error", err.Error())
				return
			}
		}
	}()
	b := common.BufPoolUdp.Get().([]byte)
	defer common.BufPoolUdp.Put(b)
	for {
		n, err := serverConn.Read(b)
		if err != nil {
			logs.Error("read udp data from server error ", err.Error())
			return
		}

		udpData, err := common.ReadUDPDatagram(bytes.NewReader(b[:n]))
		if err != nil {
			logs.Error("unpack data error", err.Error())
			return
		}
		raddr, err := net.ResolveUDPAddr("udp", udpData.Header.Addr.String())
		if err != nil {
			logs.Error("build remote addr err", err.Error())
			continue // drop silently
		}
		_, err = local.WriteTo(udpData.Data, raddr)
		if err != nil {
			logs.Error("write data to remote ", raddr.String(), "error", err.Error())
			return
		}
	}
}

// Whether the monitor channel is closed
func (s *TRPClient) ping() {
	s.ticker = time.NewTicker(time.Second * 5)
loop:
	for {
		select {
		case <-s.ticker.C:
			if s.tunnel != nil && s.tunnel.IsClose {
				s.Close()
				break loop
			}
		}
	}
}
func (s *TRPClient) ping2() {
	s.ticker = time.NewTicker(time.Second * 5)
loop:
	for {
		select {
		case <-s.ticker.C:
			var buffer bytes.Buffer
			buffer.WriteString("1234")
			s.signal.Write(buffer.Bytes()) //对一次连接中 三个TCP连接 的第二个连接 保活 (和bridge.go中的ping()方法是否有关？)
			//logs.Info(buffer.String())
			if s.tunnel != nil && s.tunnel.IsClose {
				s.Close()
				break loop
			}
		}
	}
}
func (s *TRPClient) Close() {
	s.once.Do(s.closing)
}

func (s *TRPClient) closing() {
	//s.logout = true
	NowStatus = 0
	if s.tunnel != nil {
		_ = s.tunnel.Close()
	}
	if s.signal != nil {
		_ = s.signal.Close()
	}
	if s.ticker != nil {
		s.ticker.Stop()
	}
}
