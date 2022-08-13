package tcpClient

import (
	"ehang.io/nps/lib/conn"
	"github.com/astaxie/beego/logs"
	"log"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"
)

func Test_TC(t *testing.T) {
	logs.Reset()
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)

	var mRevlen int64
	var mSenlen int64
	var mRev int64
	var mSen int64
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		t := time.NewTicker(time.Second)
		for {
			select {
			case <-t.C:
				logs.Warn(conn.Changeunit(mSen)+"/s", conn.Changeunit(mSenlen), conn.Changeunit(mRev)+"/s", conn.Changeunit(mRevlen))
				mRev = 0
				mSen = 0
				if mRevlen == mSenlen {
					//wg.Done()
				}
			}
		}
	}()
	tcpAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:8024")

	tcpConn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		logs.Error("Client connect error ! " + err.Error())
		return
	}
	defer tcpConn.Close()
	logs.Info(tcpConn.LocalAddr().String() + " : Client connected!")

	buf1 := make([]byte, 1024*1024*1024)
	rand.Read(buf1)
	go func() {
		buf2 := make([]byte, 4096)
		for {
			if n, err := tcpConn.Read(buf2); err == nil {
				mRevlen += int64(n)
				mRev += int64(n)
			} else {
				log.Fatal(err)
			}
		}
	}()
	for {
		mSenlen += int64(len(buf1))
		mSen += int64(len(buf1))
		if _, err := tcpConn.Write(buf1); err != nil {
			log.Fatal(err)
		}
		time.Sleep(time.Second)
		break
	}
	wg.Wait()
}
