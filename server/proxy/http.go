package proxy

import (
	"bufio"
	"ehang.io/nps/bridge"
	"ehang.io/nps/lib/cache"
	"ehang.io/nps/lib/common"
	"ehang.io/nps/lib/conn"
	"ehang.io/nps/lib/file"
	"ehang.io/nps/lib/rate"
	"ehang.io/nps/server/connection"
	"ehang.io/nps/web/Debug"
	"fmt"
	"github.com/astaxie/beego/logs"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

type httpServer struct {
	BaseServer
	httpPort      int
	httpsPort     int
	httpServer    *http.Server
	httpsServer   *http.Server
	httpsListener net.Listener
	useCache      bool
	addOrigin     bool
	cache         *cache.Cache
	cacheLen      int
}

var TcpCount int64

func NewHttp(bridge *bridge.Bridge, c *file.Tunnel, httpPort, httpsPort int, useCache bool, cacheLen int, addOrigin bool) *httpServer {
	httpServer := &httpServer{
		BaseServer: BaseServer{
			task:   c,
			bridge: bridge,
			Mutex:  sync.Mutex{},
		},
		httpPort:  httpPort,
		httpsPort: httpsPort,
		useCache:  useCache,
		cacheLen:  cacheLen,
		addOrigin: addOrigin,
	}
	if useCache {
		httpServer.cache = cache.New(cacheLen)
	}
	Debug.SetCallback(func() {
		Debug.Send(map[string]interface{}{"tcp": atomic.LoadInt64(&TcpCount)})
	})
	return httpServer
}

func (s *httpServer) Start() error {
	var err error
	if s.errorContent, err = common.ReadAllFromFile(filepath.Join(common.GetRunPath(), "web", "static", "page", "error.html")); err != nil {
		s.errorContent = []byte("nps 404")
	}
	if s.httpPort > 0 {
		s.httpServer = s.NewServer(s.httpPort, "http")
		go func() {
			l, err := connection.GetHttpListener()
			if err != nil {
				logs.Error(err)
				os.Exit(0)
			}
			err = s.httpServer.Serve(l)
			if err != nil {
				logs.Error(err)
				os.Exit(0)
			}
		}()
	}
	if s.httpsPort > 0 {
		s.httpsServer = s.NewServer(s.httpsPort, "https")
		go func() {
			s.httpsListener, err = connection.GetHttpsListener()
			if err != nil {
				logs.Error(err)
				os.Exit(0)
			}
			logs.Error(NewHttpsServer(s.httpsListener, s.bridge, s.useCache, s.cacheLen, s.addOrigin).Start())
		}()
	}
	return nil
}

func (s *httpServer) Close() error {
	if s.httpsListener != nil {
		s.httpsListener.Close()
	}
	if s.httpsServer != nil {
		s.httpsServer.Close()
	}
	if s.httpServer != nil {
		s.httpServer.Close()
	}
	return nil
}

func (s *httpServer) handleTunneling(w http.ResponseWriter, r *http.Request, scheme string) {
	var (
		host       *file.Host
		target     net.Conn
		err        error
		lk         *conn.Link
		targetAddr string
	)
	if host, err = file.GetDb().GetInfoByHost(r.Host, r, scheme); err != nil {
		logs.Notice("the url %s %s %s can't be parsed!", r.URL.Scheme, r.Host, r.RequestURI)
		return
	}
	defer host.Client.AddConn()
	if err := s.CheckFlowAndConnNum(host.Client); err != nil {
		logs.Warn("client id %d, host id %d, error %s, when https connection", host.Client.Id, host.Id, err.Error())
		return
	}
	if err = s.auth(r, host.Client.Cnf.U, host.Client.Cnf.P); err != nil {
		w.Write([]byte(common.UnauthorizedBytes))
		logs.Warn("auth error", err, r.RemoteAddr)
		return
	}
	if targetAddr, err = host.Target.GetRandomTarget(); err != nil {
		logs.Warn(err.Error())
		return
	}
	lk = conn.NewLink("http", targetAddr, host.Client.Cnf.Crypt, host.Client.Cnf.Compress, r.RemoteAddr, host.Target.LocalProxy)
	if target, err = s.bridge.SendLinkInfo(host.Client.Id, lk, nil); err != nil {
		logs.Notice("connect to target %s error %s", lk.Host, err)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		http.Error(w, fmt.Sprintf("connect to target %s error, the client is not connected.", lk.Host), http.StatusOK)
		return
	}
	defer target.Close()
	//change the host and header and set proxy setting
	common.ChangeHostAndHeader(r, host.HostChange, host.HeaderChange, r.RemoteAddr, s.addOrigin)

	ctx := r.Context()
	go func() {
		<-ctx.Done()
		target.Close()
	}()
	if r.Header.Get("Upgrade") == "websocket" && r.Header.Get("Connection") == "Upgrade" {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
			return
		}
		cc, _, err := hijacker.Hijack()
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		c := &conn.LenConn{Conn: cc}
		//https://go-review.googlesource.com/c/go/+/133416/1/src/net/http/server.go#1909
		defer c.Close()
		atomic.AddInt64(&TcpCount, 1)
		Debug.Send(map[string]interface{}{"tcp": atomic.LoadInt64(&TcpCount)})
		defer func() {
			atomic.AddInt64(&TcpCount, -1)
			Debug.Send(map[string]interface{}{"tcp": atomic.LoadInt64(&TcpCount)})
		}()
		w = NewConnResponseWriter(c)
		bytes, _ := httputil.DumpRequest(r, true)
		if host.Target.LocalProxy {
			_, err = target.Write(bytes)
			if err != nil {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				http.Error(w, fmt.Sprintf("Failed to write request to target %s.", lk.Host), http.StatusOK)
				return
			}
			conn.CopyWaitGroup2(target, c, host.Flow)
		} else {
			targetConn := conn.GetConn(target, lk.Crypt, lk.Compress, host.Client.Rate, true)
			_, err = targetConn.Write(bytes)
			if err != nil {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				http.Error(w, fmt.Sprintf("Failed to write request to target %s.", lk.Host), http.StatusOK)
				return
			}
			conn.CopyWaitGroup(target, c, lk.Crypt, lk.Compress, host.Client.Rate, host.Flow, true, nil)
		}
		parsedURL, _ := url.QueryUnescape(r.URL.String())
		logs.Trace(strings.Join([]string{
			fmt.Sprintf("\u001B[41m%*s\u001B[0m", 10, common.Changeunit(int64(len(bytes))+c.ReadLen)) + fmt.Sprintf("\u001B[42m%*s\u001B[0m", 10, common.Changeunit(c.WriteLen)) +
				fmt.Sprintf("%s", conn.FormatMethod(r.Method)) +
				fmt.Sprintf("\u001B[1;36m%s\u001B[0m", parsedURL),
			fmt.Sprintf("host %s", r.Host),
			fmt.Sprintf("%s->%s", r.RemoteAddr, lk.Host),
		}, ", "))
	} else {
		var dim *conn.LenConn
		r.ProtoMajor = 1
		r.ProtoMinor = 1
		bytes, _ := httputil.DumpRequest(r, true)
		if host.Target.LocalProxy {
			targetConn := rate.NewRateConn(target, host.Client.Rate)
			defer targetConn.Close()
			_, err = targetConn.Write(bytes)
			if err != nil {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				http.Error(w, fmt.Sprintf("Failed to write request to target %s.", lk.Host), http.StatusOK)
				return
			}
			dim = &conn.LenConn{Conn: targetConn}
			resp, err := http.ReadResponse(bufio.NewReader(dim), nil)
			if err != nil {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				http.Error(w, fmt.Sprintf("Failed to read response from target %s.", lk.Host), http.StatusOK)
				return
			}
			defer resp.Body.Close()
			for k, v := range resp.Header {
				for _, vv := range v {
					w.Header().Add(k, vv)
				}
			}
			w.WriteHeader(resp.StatusCode)
			common.CopyBuffer(w, resp.Body)
		} else {
			targetConn := conn.GetConn(target, lk.Crypt, lk.Compress, host.Client.Rate, true)
			defer targetConn.Close()
			_, err = targetConn.Write(bytes)
			if err != nil {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				http.Error(w, fmt.Sprintf("Failed to write request to target %s.", lk.Host), http.StatusOK)
				return
			}
			targetConn2 := conn.GetConn(target, lk.Crypt, lk.Compress, host.Client.Rate, true)
			defer targetConn2.Close()
			dim = &conn.LenConn{Conn: targetConn2}
			resp, err := http.ReadResponse(bufio.NewReader(dim), nil)
			if err != nil {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				http.Error(w, fmt.Sprintf("Failed to read response from target %s.", lk.Host), http.StatusOK)
				return
			}
			defer resp.Body.Close()
			for k, v := range resp.Header {
				for _, vv := range v {
					w.Header().Add(k, vv)
				}
			}
			w.WriteHeader(resp.StatusCode)
			common.CopyBuffer(w, resp.Body)
		}
		parsedURL, _ := url.QueryUnescape(r.URL.String())
		logs.Trace(strings.Join([]string{
			fmt.Sprintf("\u001B[41m%*s\u001B[0m", 10, common.Changeunit(int64(len(bytes))+dim.WriteLen)) + fmt.Sprintf("\u001B[42m%*s\u001B[0m", 10, common.Changeunit(int64(len(bytes))+dim.ReadLen)) +
				fmt.Sprintf("%s", conn.FormatMethod(r.Method)) +
				fmt.Sprintf("\u001B[1;36m%s\u001B[0m", parsedURL),
			fmt.Sprintf("host %s", r.Host),
			fmt.Sprintf("%s->%s", r.RemoteAddr, lk.Host),
		}, ", "))
	}
}

func (s *httpServer) handleTunneling1(w http.ResponseWriter, r *http.Request, scheme string) {
	var (
		host       *file.Host
		target     net.Conn
		err        error
		lk         *conn.Link
		targetAddr string
	)
	if host, err = file.GetDb().GetInfoByHost(r.Host, r, scheme); err != nil {
		logs.Notice("the url %s %s %s can't be parsed!", r.URL.Scheme, r.Host, r.RequestURI)
		return
	}
	defer host.Client.AddConn()
	if err := s.CheckFlowAndConnNum(host.Client); err != nil {
		logs.Warn("client id %d, host id %d, error %s, when https connection", host.Client.Id, host.Id, err.Error())
		return
	}
	if err = s.auth(r, host.Client.Cnf.U, host.Client.Cnf.P); err != nil {
		w.Write([]byte(common.UnauthorizedBytes))
		logs.Warn("auth error", err, r.RemoteAddr)
		return
	}
	if targetAddr, err = host.Target.GetRandomTarget(); err != nil {
		logs.Warn(err.Error())
		return
	}
	lk = conn.NewLink("http", targetAddr, host.Client.Cnf.Crypt, host.Client.Cnf.Compress, r.RemoteAddr, host.Target.LocalProxy)
	if target, err = s.bridge.SendLinkInfo(host.Client.Id, lk, nil); err != nil {
		logs.Notice("connect to target %s error %s", lk.Host, err)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		http.Error(w, fmt.Sprintf("connect to target %s error, the client is not connected.", lk.Host), http.StatusOK)
		return
	}
	//change the host and header and set proxy setting
	common.ChangeHostAndHeader(r, host.HostChange, host.HeaderChange, r.RemoteAddr, s.addOrigin)
	logs.Trace("%s request, method %s, host %s, url %s, remote address %s, target %s", r.URL.Scheme, r.Method, r.Host, r.URL.Path, r.RemoteAddr, lk.Host)

	ctx := r.Context()
	go func() {
		<-ctx.Done()
		target.Close()
	}()
	hijacker, ok := w.(http.Hijacker)
	if ok {
		c, _, err := hijacker.Hijack()
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		defer c.Close()
		w = NewConnResponseWriter(c)
		atomic.AddInt64(&TcpCount, 1)
		Debug.Send(map[string]interface{}{
			"tcp": atomic.LoadInt64(&TcpCount),
		})
		defer func() {
			atomic.AddInt64(&TcpCount, -1)
			Debug.Send(map[string]interface{}{
				"tcp": atomic.LoadInt64(&TcpCount),
			})
		}()
		if host.Target.LocalProxy {
			err = r.Write(target)
			if err != nil {
				w.Header().Set("Connection", "close")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				http.Error(w, fmt.Sprintf("Failed to write request to target %s.", lk.Host), http.StatusOK)
				return
			}
			conn.CopyWaitGroup2(target, c, host.Flow)
		} else {
			//https://go-review.googlesource.com/c/go/+/133416/1/src/net/http/server.go#1909
			targetConn := conn.GetConn(target, lk.Crypt, lk.Compress, host.Client.Rate, true)
			err = r.Write(targetConn)
			if err != nil {
				w.Header().Set("Connection", "close")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				http.Error(w, fmt.Sprintf("Failed to write request to target %s.", lk.Host), http.StatusOK)
				return
			}
			conn.CopyWaitGroup(target, c, lk.Crypt, lk.Compress, host.Client.Rate, host.Flow, true, nil)
		}
	} else {
		if host.Target.LocalProxy {
			targetConn := rate.NewRateConn(target, host.Client.Rate)
			defer targetConn.Close()
			err = r.Write(targetConn)
			if err != nil {
				w.Header().Set("Connection", "close")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				http.Error(w, fmt.Sprintf("Failed to write request to target %s.", lk.Host), http.StatusOK)
				return
			}
			targetReader := bufio.NewReader(targetConn)
			resp, err := http.ReadResponse(targetReader, nil)
			if err != nil {
				w.Header().Set("Connection", "close")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				http.Error(w, fmt.Sprintf("Failed to read response from target %s.", lk.Host), http.StatusOK)
				return
			}
			defer resp.Body.Close()
			for k, v := range resp.Header {
				for _, vv := range v {
					w.Header().Add(k, vv)
				}
			}
			w.WriteHeader(resp.StatusCode)
			//io.Copy(w, resp.Body)
			common.CopyBuffer(w, resp.Body)
		} else {
			defer target.Close()
			targetConn := conn.GetConn(target, lk.Crypt, lk.Compress, host.Client.Rate, true)
			defer targetConn.Close()
			err = r.Write(targetConn)
			if err != nil {
				w.Header().Set("Connection", "close")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				http.Error(w, fmt.Sprintf("Failed to write request to target %s.", lk.Host), http.StatusOK)
				return
			}
			targetConn2 := conn.GetConn(target, lk.Crypt, lk.Compress, host.Client.Rate, true)
			defer targetConn2.Close()
			targetReader := bufio.NewReader(targetConn2)
			resp, err := http.ReadResponse(targetReader, nil)
			if err != nil {
				w.Header().Set("Connection", "close")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				http.Error(w, fmt.Sprintf("Failed to read response from target %s.", lk.Host), http.StatusOK)
				return
			}
			defer resp.Body.Close()
			for k, v := range resp.Header {
				for _, vv := range v {
					w.Header().Add(k, vv)
				}
			}
			w.WriteHeader(resp.StatusCode)
			common.CopyBuffer(w, resp.Body)
			//io.Copy(w, resp.Body)
		}
	}

}

func (s *httpServer) NewServer(port int, scheme string) *http.Server {
	return &http.Server{
		Addr: ":" + strconv.Itoa(port),
		ConnState: func(conn net.Conn, state http.ConnState) {
			switch state {
			case http.StateNew:
				atomic.AddInt64(&TcpCount, 1)
				Debug.Send(map[string]interface{}{"tcp": atomic.LoadInt64(&TcpCount)})
			case http.StateClosed, http.StateHijacked:
				atomic.AddInt64(&TcpCount, -1)
				Debug.Send(map[string]interface{}{"tcp": atomic.LoadInt64(&TcpCount)})
			}
		},
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//r.URL.Scheme = scheme
			s.handleTunneling(w, r, scheme)
		}),
		// Disable HTTP/2.
		//TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
}

type ConnResponseWriter struct {
	Conn   net.Conn
	header http.Header
	status int
}

func NewConnResponseWriter(conn net.Conn) *ConnResponseWriter {
	return &ConnResponseWriter{
		Conn:   conn,
		header: make(http.Header),
	}
}

func (w *ConnResponseWriter) Header() http.Header {
	return w.header
}

func (w *ConnResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.Conn.Write(data)
}

func (w *ConnResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	statusLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, http.StatusText(statusCode))
	w.Conn.Write([]byte(statusLine))
	for key, values := range w.header {
		for _, value := range values {
			headerLine := fmt.Sprintf("%s: %s\r\n", key, value)
			w.Conn.Write([]byte(headerLine))
		}
	}
	w.Conn.Write([]byte("\r\n"))
}
