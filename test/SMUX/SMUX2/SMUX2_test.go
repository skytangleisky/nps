package SMUX2

import (
	"ehang.io/nps/lib/common"
	"github.com/astaxie/beego/logs"
	"github.com/xtaci/kcp-go/v5"
	"github.com/xtaci/smux"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"
)

func TestEcho(t *testing.T) {
	_, stop, cli, err := setupServer(t)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	handleClient(cli)
}

// setupServer starts new server listening on a random localhost port and
// returns address of the server, function to stop the server, new client
// connection to this server or an error.
func setupServer(tb testing.TB) (addr string, stopfunc func(), client net.Conn, err error) {
	ln, err := kcp.Listen("localhost:0")
	//ln, err := net.Listen("tcp", ":6666")
	if err != nil {
		return "", nil, nil, err
	}
	go func() {
		cc, err := ln.Accept()
		if err != nil {
			return
		}
		go handleConnection(cc)
	}()
	addr = ln.Addr().String()
	cc, err := kcp.Dial(addr)
	//cc, err := net.Dial("tcp","127.0.0.1:6666")
	if err != nil {
		ln.Close()
		return "", nil, nil, err
	}
	return ln.Addr().String(), func() { ln.Close() }, cc, nil
}

func handleConnection(conn net.Conn) {
	//go func(s io.ReadWriteCloser) {
	//	buf := make([]byte, 65536)
	//	for {
	//		n, err := s.Read(buf)
	//		if err != nil {
	//			return
	//		}
	//		s.Write(buf[:n])
	//	}
	//}(conn)
	go func(conn io.ReadWriteCloser) {
		session, _ := smux.Server(conn, nil)
		for {
			if s, err := session.AcceptStream(); err == nil {
				buf := make([]byte, 32768)
				for {
					n, err := s.Read(buf)
					if err != nil {
						return
					}
					s.Write(buf[:n])
				}
			} else {
				return
			}
		}
	}(conn)
}

func handleClient(conn net.Conn) {
	var mSen int64
	var mSenlen int64
	var mRev int64
	var mRevlen int64
	buf1 := make([]byte, 1024*1024*1024)
	rand.Read(buf1)
	mSenlen = int64(len(buf1))
	mSen = mSenlen
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
				if mSenlen == mRevlen {
					wg.Done()
				}
			}
		}
	}()

	//go func() {
	//	buf:=make([]byte,65536)
	//	for{
	//		if n,err:=conn.Read(buf);err==nil{
	//			mRevlen+=int64(n)
	//			mRev+=int64(n)
	//		}else{
	//			log.Fatal(err)
	//		}
	//	}
	//}()
	//conn.Write(buf1)

	session, _ := smux.Client(conn, nil)
	stream, _ := session.OpenStream()
	go func() {
		buf := make([]byte, 32768)
		for {
			if n, err := stream.Read(buf); err == nil {
				mRevlen += int64(n)
				mRev += int64(n)
			} else {
				log.Fatal(err)
			}
		}
	}()
	stream.Write(buf1)

	wg.Wait()
}
