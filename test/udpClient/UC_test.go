package udpClient

import (
	"ehang.io/nps/lib/common"
	"fmt"
	"github.com/astaxie/beego/logs"
	"log"
	"math/rand"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

func Test_UC(t *testing.T) {

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
				logs.Warn(common.Changeunit(mSen)+"/s", common.Changeunit(mSenlen), common.Changeunit(mRev)+"/s", common.Changeunit(mRevlen))
				mRev = 0
				mSen = 0
				if mRevlen == mSenlen {
					//wg.Done()
				}
			}
		}
	}()

	conn, err := net.Dial("udp", "127.0.0.1:6666")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer conn.Close()

	buf1 := make([]byte, 1024*1024*1024)
	rand.Read(buf1)
	go func() {
		buf2 := make([]byte, 4096)
		for {
			if n, err := conn.Read(buf2); err == nil {
				mRevlen += int64(n)
				mRev += int64(n)
			} else {
				logs.Error(err)
				os.Exit(0)
			}
		}
	}()

	for {
		mi := min(len(buf1), 1024*4) //1024*64-29
		if mi == 0 {
			break
		}
		buf := buf1[:mi]
		var n int
		if n, err = conn.Write(buf); err != nil {
			log.Fatal(err)
		}
		mSenlen += int64(n)
		mSen += int64(n)
		buf1 = buf1[n:]
	}

	wg.Wait()

}

func min(a int, b int) int {
	if a > b {
		return b
	} else {
		return a
	}
}
