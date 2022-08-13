package udpServer

import (
	"fmt"
	"github.com/astaxie/beego/logs"
	"net"
	"os"
	"testing"
)

func Test_US(t *testing.T) {

	logs.Reset()
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)

	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:6666")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer conn.Close()
	logs.Info(conn.LocalAddr())

	buf := make([]byte, 4096)
	for {
		n, rAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			logs.Error(err)
			return
		}
		go func() {
			n, err = conn.WriteToUDP(buf[:n], rAddr)
			if err != nil {
				fmt.Println(err)
				return
			}
		}()
	}
}
