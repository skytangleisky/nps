package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

const (
	socks5Version           = 0x05
	socks5AuthNone          = 0x00
	socks5AuthPassword      = 0x02
	socks5AuthNoAccept      = 0xFF
	socks5CmdConnect        = 0x01
	socks5CmdBind           = 0x02
	socks5CmdUDP            = 0x03
	socks5AddressTypeIPV4   = 0x01
	socks5AddressTypeDomain = 0x03
	socks5AddressTypeIPV6   = 0x04
)

var (
	Username = "username"
	Password = "password"
)

func main() {
	ln, err := net.Listen("tcp", ":1181")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer ln.Close()

	fmt.Println("SOCKS5 proxy listening on :1181")
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// 1. 协商认证方式
	if err := negotiateAuth(conn); err != nil {
		fmt.Println("Authentication negotiation failed:", err)
		return
	}

	// 2. 处理客户端请求
	if err := handleRequest(conn); err != nil {
		fmt.Println("Request handling failed:", err)
		return
	}
}

func negotiateAuth(conn net.Conn) error {
	// 读取版本和支持的认证方式
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}
	version := buf[0]
	nmethods := buf[1]

	if version != socks5Version {
		return fmt.Errorf("unsupported SOCKS version: %d", version)
	}

	// 读取认证方式列表
	methods := make([]byte, nmethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return err
	}

	// 检查是否支持无认证或用户名密码认证
	authMethod := socks5AuthNoAccept
	for _, method := range methods {
		if method == socks5AuthNone {
			authMethod = socks5AuthNone
			break
		} else if method == socks5AuthPassword {
			authMethod = socks5AuthPassword
		}
	}

	// 回复客户端选择的认证方式
	conn.Write([]byte{socks5Version, byte(authMethod)})

	if authMethod == socks5AuthPassword {
		return handlePasswordAuth(conn)
	} else if authMethod == socks5AuthNoAccept {
		return fmt.Errorf("no supported authentication methods")
	}
	return nil
}

func handlePasswordAuth(conn net.Conn) error {
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}

	version := buf[0]
	if version != 0x01 {
		return fmt.Errorf("unsupported sub-negotiation version: %d", version)
	}

	ulen := buf[1]
	uname := make([]byte, ulen)
	if _, err := io.ReadFull(conn, uname); err != nil {
		return err
	}

	if _, err := io.ReadFull(conn, buf[:1]); err != nil {
		return err
	}

	plen := buf[0]
	passwd := make([]byte, plen)
	if _, err := io.ReadFull(conn, passwd); err != nil {
		return err
	}

	if string(uname) != Username || string(passwd) != Password {
		conn.Write([]byte{0x01, 0x01}) // Authentication failure
		return fmt.Errorf("authentication failed")
	}

	conn.Write([]byte{0x01, 0x00}) // Authentication success
	return nil
}

func handleRequest(conn net.Conn) error {
	// 读取请求头
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}

	version := header[0]
	cmd := header[1]
	// reserved := header[2]
	atyp := header[3]

	if version != socks5Version {
		return fmt.Errorf("unsupported SOCKS version: %d", version)
	}

	var destAddr string
	switch atyp {
	case socks5AddressTypeIPV4:
		ip := make([]byte, 4)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return err
		}
		port := make([]byte, 2)
		if _, err := io.ReadFull(conn, port); err != nil {
			return err
		}
		destAddr = fmt.Sprintf("%s:%d", net.IP(ip).String(), binary.BigEndian.Uint16(port))
	case socks5AddressTypeDomain:
		domainLen := make([]byte, 1)
		if _, err := io.ReadFull(conn, domainLen); err != nil {
			return err
		}
		domain := make([]byte, domainLen[0])

		if _, err := io.ReadFull(conn, domain); err != nil {
			return err
		}
		port := make([]byte, 2)
		if _, err := io.ReadFull(conn, port); err != nil {
			return err
		}
		destAddr = fmt.Sprintf("%s:%d", string(domain), binary.BigEndian.Uint16(port))
	case socks5AddressTypeIPV6:
		ip := make([]byte, 16)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return err
		}
		port := make([]byte, 2)
		if _, err := io.ReadFull(conn, port); err != nil {
			return err
		}
		destAddr = fmt.Sprintf("[%s]:%d", net.IP(ip).String(), binary.BigEndian.Uint16(port))
	default:
		return fmt.Errorf("unsupported address type: %d", atyp)
	}

	switch cmd {
	case socks5CmdConnect:
		return handleConnect(conn, destAddr, atyp)
	case socks5CmdBind:
		return handleBind(conn, destAddr, atyp)
	case socks5CmdUDP:
		return handleUDP(conn, destAddr, atyp)
	default:
		return fmt.Errorf("unsupported command: %d", cmd)
	}
}

func handleConnect(conn net.Conn, destAddr string, atyp byte) error {
	// 连接目标服务器
	targetConn, err := net.Dial("tcp", destAddr)
	if err != nil {
		conn.Write([]byte{socks5Version, 0x01, 0x00, atyp})
		return err
	}
	fmt.Println(destAddr)
	defer targetConn.Close()
	bytes := []byte{socks5Version, 0x00, 0x00}
	localAddr := targetConn.LocalAddr().(*net.TCPAddr)
	localIP, localPort := localAddr.IP, localAddr.Port
	ipBytes := localIP.To4()
	if ipBytes != nil {
		bytes = append(bytes, socks5AddressTypeIPV4)
	} else {
		ipBytes = localIP.To16()
		bytes = append(bytes, socks5AddressTypeIPV6)
	}
	bytes = append(bytes, ipBytes...)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, uint16(localPort))
	bytes = append(bytes, portBytes...)
	// 响应客户端连接成功
	conn.Write(bytes)

	// 双向数据转发
	go io.Copy(targetConn, conn)
	io.Copy(conn, targetConn)

	return nil
}

func handleBind(conn net.Conn, destAddr string, atyp byte) error {
	// 创建监听端口
	ln, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		conn.Write([]byte{socks5Version, 0x01, 0x00, atyp})
		return err
	}
	defer ln.Close()

	// 获取绑定的地址
	localAddr := ln.Addr().(*net.TCPAddr)
	localHost, localPort := localAddr.IP, localAddr.Port

	// 响应客户端绑定成功
	resp := make([]byte, 4)
	resp[0] = socks5Version
	resp[1] = 0x00
	resp[2] = 0x00
	resp[3] = socks5AddressTypeIPV4

	resp = append(resp, localHost.To4()...)
	port := make([]byte, 2)
	binary.BigEndian.PutUint16(port, uint16(localPort))
	resp = append(resp, port...)

	conn.Write(resp)

	// 等待入站连接
	peerConn, err := ln.Accept()
	if err != nil {
		conn.Write([]byte{socks5Version, 0x01, 0x00, atyp})
		return err
	}
	defer peerConn.Close()

	// 获取对等连接的地址
	peerAddr := peerConn.RemoteAddr().(*net.TCPAddr)
	peerHost, peerPort := peerAddr.IP, peerAddr.Port

	// 响应客户端连接成功
	resp = make([]byte, 4)
	resp[0] = socks5Version
	resp[1] = 0x00
	resp[2] = 0x00
	resp[3] = socks5AddressTypeIPV4

	resp = append(resp, peerHost.To4()...)
	port = make([]byte, 2)
	binary.BigEndian.PutUint16(port, uint16(peerPort))
	resp = append(resp, port...)

	conn.Write(resp)

	// 双向数据转发
	go io.Copy(peerConn, conn)
	io.Copy(conn, peerConn)

	return nil
}

func handleUDP(conn net.Conn, destAddr string, atyp byte) error {
	// 创建 UDP 监听
	udpAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		conn.Write([]byte{socks5Version, 0x01, 0x00, atyp})
		return err
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		conn.Write([]byte{socks5Version, 0x01, 0x00, atyp})
		return err
	}
	defer udpConn.Close()

	// 获取 UDP 监听地址
	localAddr := udpConn.LocalAddr().(*net.UDPAddr)
	localHost, localPort := localAddr.IP, localAddr.Port

	// 响应客户端 UDP 关联成功
	resp := make([]byte, 4)
	resp[0] = socks5Version
	resp[1] = 0x00
	resp[2] = 0x00
	resp[3] = socks5AddressTypeIPV4

	resp = append(resp, localHost.To4()...)
	port := make([]byte, 2)
	binary.BigEndian.PutUint16(port, uint16(localPort))
	resp = append(resp, port...)

	conn.Write(resp)

	// 处理 UDP 数据包
	go handleUDPRelay(udpConn)

	return nil
}

func handleUDPRelay(udpConn *net.UDPConn) {
	buf := make([]byte, 4096)
	for {
		n, addr, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error reading UDP packet:", err)
			return
		}

		// 解析 UDP 包头
		if n < 3 {
			fmt.Println("Invalid UDP packet received")
			continue
		}

		// 跳过前3个字节
		packet := buf[3:n]

		// 发送数据到目标地址
		targetAddr, err := parseUDPAddress(packet)
		if err != nil {
			fmt.Println("Error parsing target address:", err)
			continue
		}

		targetConn, err := net.Dial("udp", targetAddr)
		if err != nil {
			fmt.Println("Error connecting to target address:", err)
			continue
		}
		defer targetConn.Close()

		_, err = targetConn.Write(packet)
		if err != nil {
			fmt.Println("Error sending UDP packet to target:", err)
		}

		// 从目标地址接收响应
		respBuf := make([]byte, 4096)
		n, err = targetConn.Read(respBuf)
		if err != nil {
			fmt.Println("Error reading UDP response:", err)
			continue
		}

		// 发送响应给客户端
		udpConn.WriteToUDP(respBuf[:n], addr)
	}
}

func parseUDPAddress(packet []byte) (string, error) {
	if len(packet) < 4 {
		return "", errors.New("invalid UDP packet")
	}

	atyp := packet[0]
	var addr string
	var port uint16

	switch atyp {
	case socks5AddressTypeIPV4:
		if len(packet) < 7 {
			return "", errors.New("invalid IPv4 address")
		}
		addr = net.IP(packet[1:5]).String()
		port = binary.BigEndian.Uint16(packet[5:7])
	case socks5AddressTypeDomain:
		domainLen := int(packet[1])
		if len(packet) < 2+domainLen+2 {
			return "", errors.New("invalid domain address")
		}
		addr = string(packet[2 : 2+domainLen])
		port = binary.BigEndian.Uint16(packet[2+domainLen : 2+domainLen+2])
	case socks5AddressTypeIPV6:
		if len(packet) < 19 {
			return "", errors.New("invalid IPv6 address")
		}
		addr = net.IP(packet[1:17]).String()
		port = binary.BigEndian.Uint16(packet[17:19])
	default:
		return "", fmt.Errorf("unsupported address type: %d", atyp)
	}

	return fmt.Sprintf("%s:%d", addr, port), nil
}
