package kcp

import (
	"ehang.io/nps/lib/common"
	"ehang.io/nps/smux"
	"github.com/astaxie/beego/logs"
	"github.com/xtaci/kcp-go/v5"
	"log"
	"math/rand"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

func Test_KS(t *testing.T) {
	logs.Reset()
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)
	//key := pbkdf2.Key([]byte("demo pass"), []byte("demo salt"), 1024, 32, sha1.New)
	//block, _ := kcp.NewAESBlockCrypt(key)
	if listener, err := kcp.Listen("0.0.0.0:4444"); err == nil {
		//if listener, err := net.Listen("tcp","0.0.0.0:6666"); err == nil {
		logs.Info(listener.Addr().String())
		for {
			kcpConn, err := listener.Accept()
			if err != nil {
				logs.Error(err)
				break
			}
			logs.Info("A client connected :" + kcpConn.RemoteAddr().String())
			go func(conn net.Conn) {
				ipStr := conn.RemoteAddr().String()
				defer func() {
					conn.Close()
					logs.Error(" Disconnected : " + ipStr)
				}()
				buf := make([]byte, 4096)
				for {
					n, err := conn.Read(buf)
					if err != nil {
						log.Println(err)
						return
					}
					n, err = conn.Write(buf[:n])
					if err != nil {
						log.Println(err)
						return
					}
				}
			}(kcpConn)
		}
	} else {
		logs.Error(err)
		os.Exit(0)
	}
}
func Test_KC(t *testing.T) {
	logs.Reset()
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)

	var mRevlen int64
	var mSenlen int64
	var mRev int64
	var mSen int64
	var wg sync.WaitGroup
	wg.Add(1)
	buf1 := make([]byte, 1024*1024*100)
	rand.Read(buf1)
	mSenlen += int64(len(buf1))
	mSen += int64(len(buf1))
	go func() {
		t := time.NewTicker(time.Second)
		for {
			select {
			case <-t.C:
				logs.Warn(common.Changeunit(mSen)+"/s", common.Changeunit(mSenlen), common.Changeunit(mRev)+"/s", common.Changeunit(mRevlen))
				mRev = 0
				mSen = 0
				if mRevlen == mSenlen {
					wg.Done()
				}
			}
		}
	}()

	// dial to the echo server
	if sess, err := kcp.Dial("127.0.0.1:4444"); err == nil {
		//if sess, err := net.Dial("tcp","127.0.0.1:6666"); err == nil {
		go func() {
			buf2 := make([]byte, 4096)
			for {
				if n, err := sess.Read(buf2); err == nil {
					mRevlen += int64(n)
					mRev += int64(n)
				} else {
					log.Fatal(err)
				}
			}
		}()

		if _, err := sess.Write(buf1); err != nil {
			log.Fatal(err)
		}

		wg.Wait()
	} else {
		log.Print(err)
	}

}

func Test_KMS(t *testing.T) {
	logs.Reset()
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)
	//key := pbkdf2.Key([]byte("demo pass"), []byte("demo salt"), 1024, 32, sha1.New)
	//block, _ := kcp.NewAESBlockCrypt(key)
	//if listener, err := kcp.Listen("0.0.0.0:4444"); err == nil {
	if listener, err := net.Listen("tcp", ":4444"); err == nil {
		logs.Info(listener.Addr().String())
		for {
			conn, err := listener.Accept()
			if err != nil {
				logs.Error(err)
				break
			}
			logs.Info("A client connected :" + conn.RemoteAddr().String())
			//conn.SetUdpSession(kcpConn)
			//muxSession := nps_mux.NewMux(conn, "tcp", 0)
			muxSession, _ := smux.Client(conn, nil)
			peer, _ := muxSession.Accept()
			go func(conn net.Conn) {
				ipStr := conn.RemoteAddr().String()
				defer func() {
					conn.Close()
					logs.Error(" Disconnected : " + ipStr)
				}()
				buf := make([]byte, 32768)
				var len = int64(0)
				for {
					m, err := conn.Read(buf)
					if err != nil {
						log.Println(err)
						return
					}
					_, err = conn.Write(buf[:m])
					if err != nil {
						log.Println(err)
						return
					}
					len += int64(m)
				}
			}(peer)
		}
	} else {
		logs.Error(err)
		os.Exit(0)
	}
}
func Test_KMC(t *testing.T) {
	logs.Reset()
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)

	var mRevlen int64
	var mSenlen int64
	var mRev int64
	var mSen int64
	var wg sync.WaitGroup
	wg.Add(1)
	buf1 := make([]byte, 1024*1024*100)
	//rand.Read(buf1)
	mSenlen = int64(len(buf1))
	mSen += int64(len(buf1))
	go func() {
		t := time.NewTicker(time.Second)
		for {
			select {
			case <-t.C:
				logs.Warn(common.Changeunit(mSen)+"/s", common.Changeunit(mSenlen), common.Changeunit(mRev)+"/s", common.Changeunit(mRevlen))
				mRev = 0
				mSen = 0
				if mRevlen == mSenlen {
					logs.Alert(mSenlen, mRevlen)
					wg.Done()
				}
			}
		}
	}()

	// dial to the echo server
	//if sess, err := kcp.Dial("127.0.0.1:4444"); err == nil {
	if sess, err := net.Dial("tcp", "127.0.0.1:4444"); err == nil {
		//conn.SetUdpSession(sess)
		//sess.SetDeadline(time.Now().Add(time.Second * 5))

		//muxSession := nps_mux.NewMux(sess, "tcp", 0)
		muxSession, _ := smux.Client(sess, nil)
		client, _ := muxSession.OpenStream()
		go func(client net.Conn) {
			buf2 := make([]byte, 32768)
			for {
				if n, err := client.Read(buf2); err == nil {
					mRevlen += int64(n)
					mRev += int64(n)
				} else {
					log.Fatal(err)
				}
			}
		}(client)

		if _, err := client.Write(buf1); err != nil {
			log.Fatal(err)
		}

		wg.Wait()
	} else {
		log.Print(err)
	}

}
