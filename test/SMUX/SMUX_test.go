package SMUX

import (
	"ehang.io/nps/lib/common"
	"github.com/xtaci/smux"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"testing"
)

func TestEcho(t *testing.T) {
	_, stop, cli, err := setupServer(t)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	session, _ := smux.Client(cli, nil)
	stream, _ := session.OpenStream()
	var wg sync.WaitGroup
	buf := make([]byte, 1024*1024)
	buf1 := make([]byte, 1024*1024*1024)
	rand.Read(buf1)
	go func() {
		var len = int64(0)
		for {
			if n, err := stream.Read(buf); err == nil {
				len += int64(n)
			} else {
				log.Fatal(err)
			}
			log.Print(common.Changeunit(len))
		}
	}()
	stream.Write(buf1)
	wg.Add(1)
	wg.Wait()

	session.Close()
}

// setupServer starts new server listening on a random localhost port and
// returns address of the server, function to stop the server, new client
// connection to this server or an error.
func setupServer(tb testing.TB) (addr string, stopfunc func(), client net.Conn, err error) {
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", nil, nil, err
	}
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go handleConnection(conn)
	}()
	addr = ln.Addr().String()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		ln.Close()
		return "", nil, nil, err
	}
	return ln.Addr().String(), func() { ln.Close() }, conn, nil
}

func handleConnection(conn net.Conn) {
	session, _ := smux.Server(conn, nil)
	for {
		if stream, err := session.AcceptStream(); err == nil {
			go func(s io.ReadWriteCloser) {
				buf := make([]byte, 65536)
				for {
					n, err := s.Read(buf)
					if err != nil {
						return
					}
					s.Write(buf[:n])
				}
			}(stream)
		} else {
			return
		}
	}
}
