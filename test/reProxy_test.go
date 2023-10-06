package test

import (
	"crypto/tls"
	"ehang.io/nps/lib/common"
	"fmt"
	"github.com/astaxie/beego/logs"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"testing"
)

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
