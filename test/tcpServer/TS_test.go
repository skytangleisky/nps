package tcpServer

import (
	"fmt"
	"github.com/astaxie/beego/logs"
	"net"
	"os"
	"testing"
)

func Test_TS(t *testing.T) {
	logs.Reset()
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)

	tcpAddr, er := net.ResolveTCPAddr("tcp", "0.0.0.0:8024")
	if er != nil {
		logs.Error(er)
		os.Exit(0)
	}

	tcpListener, err := net.ListenTCP("tcp", tcpAddr)
	defer tcpListener.Close()
	if err != nil {
		logs.Error(err)
		os.Exit(0)
	}
	logs.Info(tcpListener.Addr())
	for {
		tcpConn, err := tcpListener.AcceptTCP()
		if err != nil {
			fmt.Println(err)
			break
		}
		logs.Info(tcpConn.RemoteAddr().String())
		go func(conn *net.TCPConn) {
			ipStr := conn.RemoteAddr().String()
			defer func() {
				conn.Close()
				logs.Error(ipStr)
			}()

			buf := make([]byte, 4096)
			for {
				n, err := conn.Read(buf)
				if err != nil {
					return
				}
				go func() {
					n, err = conn.Write(buf[:n])
					if err != nil {
						return
					}
				}()
			}
		}(tcpConn)
	}
}
