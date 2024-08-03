package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

// 定义Socks5协议的常量
const (
	socks5Version        = 0x05
	socks5AuthNone       = 0x00
	socks5AuthPassword   = 0x02
	socks5CmdConnect     = 0x01
	socks5CmdBind        = 0x02
	socks5CmdUDP         = 0x03
	socks5AddrTypeIPv4   = 0x01
	socks5AddrTypeDomain = 0x03
	socks5AddrTypeIPv6   = 0x04
	socks5AuthSuccess    = 0x00
	socks5AuthFailure    = 0x01
	socks5RepSuccess     = 0x00
	socks5RepFailure     = 0x01
)

func handleConnection(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 1024)
	// 读取版本和认证方法
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Failed to read from client:", err)
		return
	}
	if buf[0] != socks5Version {
		fmt.Println("Unsupported SOCKS version:", buf[0])
		return
	}

	// 支持无认证和用户名/密码认证
	authMethod := buf[2]
	if authMethod != socks5AuthNone && authMethod != socks5AuthPassword {
		fmt.Println("Unsupported auth method:", authMethod)
		conn.Write([]byte{socks5Version, socks5AuthFailure})
		return
	}

	// 返回选择的认证方法
	conn.Write([]byte{socks5Version, socks5AuthNone})

	// 读取客户端的请求
	_, err = conn.Read(buf)
	if err != nil {
		fmt.Println("Failed to read from client:", err)
		return
	}
	if buf[0] != socks5Version {
		fmt.Println("Unsupported SOCKS version:", buf[0])
		return
	}
	cmd := buf[1]
	addrType := buf[3]
	var addr string
	var port uint16

	switch addrType {
	case socks5AddrTypeIPv4:
		addr = net.IP(buf[4:8]).String()
		port = binary.BigEndian.Uint16(buf[8:10])
	case socks5AddrTypeDomain:
		addrLen := buf[4]
		addr = string(buf[5 : 5+addrLen])
		port = binary.BigEndian.Uint16(buf[5+addrLen : 5+addrLen+2])
	case socks5AddrTypeIPv6:
		addr = net.IP(buf[4:20]).String()
		port = binary.BigEndian.Uint16(buf[20:22])
	default:
		fmt.Println("Unsupported address type:", addrType)
		conn.Write([]byte{socks5Version, socks5RepFailure})
		return
	}

	switch cmd {
	case socks5CmdConnect:
		handleConnect(conn, addr, port)
	case socks5CmdBind:
		handleBind(conn, addr, port)
	case socks5CmdUDP:
		handleUDP(conn, addr, port)
	default:
		fmt.Println("Unsupported command:", cmd)
		conn.Write([]byte{socks5Version, socks5RepFailure})
	}
}

func handleConnect(conn net.Conn, addr string, port uint16) {
	destAddr := fmt.Sprintf("%s:%d", addr, port)
	destConn, err := net.Dial("tcp", destAddr)
	if err != nil {
		fmt.Println("Failed to connect to destination:", err)
		conn.Write([]byte{socks5Version, socks5RepFailure})
		return
	}
	defer destConn.Close()

	// 返回成功响应
	conn.Write([]byte{socks5Version, socks5RepSuccess, 0x00, socks5AddrTypeIPv4, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	go io.Copy(destConn, conn)
	io.Copy(conn, destConn)
}

func handleBind(conn net.Conn, addr string, port uint16) {
	// TODO: 实现BIND命令的处理
	fmt.Println("BIND command not implemented")
	conn.Write([]byte{socks5Version, socks5RepFailure})
}

func handleUDP(conn net.Conn, addr string, port uint16) {
	// TODO: 实现UDP命令的处理
	fmt.Println("UDP command not implemented")
	conn.Write([]byte{socks5Version, socks5RepFailure})
}

func main() {
	listener, err := net.Listen("tcp", ":1080")
	if err != nil {
		fmt.Println("Failed to start server:", err)
		return
	}
	defer listener.Close()
	fmt.Println("SOCKS5 server is listening on port 1080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Failed to accept connection:", err)
			continue
		}
		go handleConnection(conn)
	}
}
