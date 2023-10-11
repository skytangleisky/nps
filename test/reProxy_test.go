package test

import (
	"bytes"
	"crypto/tls"
	"ehang.io/nps/lib/common"
	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/golang/snappy"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"testing"
)

func TestNewBufferedWriter(t *testing.T) {
	// Test all 32 possible sub-sequences of these 5 input slices.
	//
	// Their lengths sum to 400,000, which is over 6 times the Writer ibuf
	// capacity: 6 * maxBlockSize is 393,216.
	//input := bytes.Repeat([]byte{'a'}, 40000)
	input := []byte{
		71, 69, 84, 32, 47, 32, 72, 84, 84, 80, 47, 49, 46, 49, 13, 10, 72, 111, 115, 116, 58, 32, 119, 101, 98, 115, 111, 99, 107, 101, 116, 46, 116, 97, 110, 103, 108, 101, 105, 46, 116, 111, 112, 58, 56, 48, 13, 10, 85, 115, 101, 114, 45, 65, 103, 101, 110, 116, 58, 32, 76, 111, 108, 108, 105, 112, 111, 112, 47, 49, 46, 49, 13, 10, 67, 111, 110, 110, 101, 99, 116, 105, 111, 110, 58, 32, 85, 112, 103, 114, 97, 100, 101, 13, 10, 83, 101, 99, 45, 87, 101, 98, 115, 111, 99, 107, 101, 116, 45, 75, 101, 121, 58, 32, 78, 50, 86, 68, 76, 71, 117, 74, 87, 107, 49, 113, 75, 82, 78, 103, 43, 117, 104, 69, 89, 103, 61, 61, 13, 10, 83, 101, 99, 45, 87, 101, 98, 115, 111, 99, 107, 101, 116, 45, 86, 101, 114, 115, 105, 111, 110, 58, 32, 49, 51, 13, 10, 85, 112, 103, 114, 97, 100, 101, 58, 32, 119, 101, 98, 115, 111, 99, 107, 101, 116, 13, 10, 88, 45, 70, 111, 114, 119, 97, 114, 100, 101, 100, 45, 70, 111, 114, 58, 32, 47, 91, 50, 52, 48, 101, 58, 51, 57, 97, 58, 51, 48, 51, 58, 55, 50, 50, 48, 58, 53, 56, 53, 102, 58, 53, 97, 99, 56, 58, 101, 102, 101, 97, 58, 57, 56, 53, 53, 93, 58, 54, 49, 50, 51, 56, 13, 10, 88, 45, 82, 101, 97, 108, 45, 73, 112, 58, 32, 47, 91, 50, 52, 48, 101, 58, 51, 57, 97, 58, 51, 48, 51, 58, 55, 50, 50, 48, 58, 53, 56, 53, 102, 58, 53, 97, 99, 56, 58, 101, 102, 101, 97, 58, 57, 56, 53, 53, 93, 58, 54, 49, 50, 51, 56, 13, 10, 13, 10,
	}
	fmt.Println(len(input), input)
	buf := new(bytes.Buffer)
	w := snappy.NewBufferedWriter(buf)
	if _, err := w.Write(input); err != nil {
		t.Errorf("Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	fmt.Println(len(buf.Bytes()), buf.Bytes(), string(buf.Bytes()))
	bytes := make([]byte, 1024)
	reader := snappy.NewReader(buf)
	n, err := reader.Read(bytes)
	if n > 0 {
		got := bytes[:n]
		if err != nil {
			t.Errorf("ReadAll: %v", err)
		}
		fmt.Println(len(got), got)
		if err := cmp(got, input); err != nil {
			t.Errorf("%v", err)
		}
	}
}
func cmp(a, b []byte) error {
	if bytes.Equal(a, b) {
		return nil
	}
	if len(a) != len(b) {
		return fmt.Errorf("got %d bytes, want %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			return fmt.Errorf("byte #%d: got 0x%02x, want 0x%02x", i, a[i], b[i])
		}
	}
	return nil
}

func Test_reProxy(t *testing.T) {
	test3()
}
func test1() {
	url, _ := url.Parse("http://127.0.0.1:3210")
	rProxy := httputil.NewSingleHostReverseProxy(url)
	log.Fatalln(http.ListenAndServeTLS(":6677", "/Users/admin/Desktop/nps/conf/9983347_tanglei.site.pem", "/Users/admin/Desktop/nps/conf/9983347_tanglei.site.key", rProxy))
}
func test2() {
	url, _ := url.Parse("http://127.0.0.1:3210")
	rProxy := httputil.NewSingleHostReverseProxy(url)
	log.Fatalln(http.ListenAndServeTLS(":6677", "/Users/admin/Desktop/nps/conf/9983347_tanglei.site.pem", "/Users/admin/Desktop/nps/conf/9983347_tanglei.site.key", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rProxy.ServeHTTP(w, r)
	})))
}

func test3() {
	tcpAddr, er := net.ResolveTCPAddr("tcp", ":6677")
	if er != nil {
		logs.Error(er)
		os.Exit(0)
	}
	tcpListener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return
	}
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println(r.Proto)
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				fmt.Println("not OK")
				http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
				return
			}
			c, _, err := hijacker.Hijack()
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
			}

			tcpAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:3211")
			if err != nil {
				w.WriteHeader(500)
				return
			}
			tcpConn, err := net.DialTCP("tcp", nil, tcpAddr)
			if err != nil {
				w.WriteHeader(500)
				return
			}
			r.Write(tcpConn)
			go func() {
				common.CopyBuffer(c, tcpConn)
				c.Close()
				tcpConn.Close()
			}()
			common.CopyBuffer(tcpConn, c)
			c.Close()
			tcpConn.Close()
		}),
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	server.ServeTLS(tcpListener, "/Users/admin/Desktop/nps/conf/9983347_tanglei.site.pem", "/Users/admin/Desktop/nps/conf/9983347_tanglei.site.key") //http->https1.x
}
func test4() {
	tcpAddr, er := net.ResolveTCPAddr("tcp", ":6677")
	if er != nil {
		logs.Error(er)
		os.Exit(0)
	}
	tcpListener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return
	}
	for {
		c, err := tcpListener.Accept()
		if err != nil {
			log.Println(err)
		} else {
			go func(c net.Conn) {
				tcpAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:3210")
				if err != nil {
					log.Println(err)
					return
				}
				tcpConn, err := net.DialTCP("tcp", nil, tcpAddr)
				if err != nil {
					log.Println(err)
					return
				}
				go func() {
					common.CopyBuffer(c, tcpConn)
				}()
				common.CopyBuffer(tcpConn, c)
			}(c)
		}
	}
}
