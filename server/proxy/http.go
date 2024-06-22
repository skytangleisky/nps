package proxy

import (
	"bufio"
	"ehang.io/nps/lib/rate"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"ehang.io/nps/bridge"
	"ehang.io/nps/lib/cache"
	"ehang.io/nps/lib/common"
	"ehang.io/nps/lib/conn"
	"ehang.io/nps/lib/file"
	"ehang.io/nps/server/connection"
	"github.com/astaxie/beego/logs"
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
		//tcpAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(s.httpPort))
		//tcpListener, err := net.ListenTCP("tcp", tcpAddr)
		//if err != nil {
		// logs.Error(err)
		//  os.Exit(0)
		//} else {
		// go func() {
		//    for {
		//       tcpConn, err := tcpListener.AcceptTCP()
		//       if err != nil {
		//          logs.Error(err)
		//          tcpConn.Close()
		//       } else {
		//          go func() {
		//             s.Process(tcpConn)
		//          }()
		//       }
		//
		//    }
		// }()
		//}

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

func (s *httpServer) handleTunneling(w http.ResponseWriter, r *http.Request) {
	var (
		host       *file.Host
		target     net.Conn
		err        error
		connClient net.Conn
		//scheme     = r.URL.Scheme
		lk         *conn.Link
		targetAddr string
	)
	if host, err = file.GetDb().GetInfoByHost(r.Host, r); err != nil {
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
		http.Error(w, fmt.Sprintf("connect to target %s error, the client is not connect.", lk.Host), http.StatusOK)
		return
	}
	//change the host and header and set proxy setting
	common.ChangeHostAndHeader(r, host.HostChange, host.HeaderChange, r.RemoteAddr, s.addOrigin)
	logs.Trace("%s request, method %s, host %s, url %s, remote address %s, target %s", r.URL.Scheme, r.Method, r.Host, r.URL.Path, r.RemoteAddr, lk.Host)

	hijacker, ok := w.(http.Hijacker)
	if ok {
		c, _, err := hijacker.Hijack()
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		defer func() {
			if connClient != nil {
				connClient.Close()
			} else {
				s.writeConnFail(c)
			}
			c.Close()
		}()
		if host.Target.LocalProxy {
			err = r.Write(target)
			if err != nil {
				logs.Error(err)
				return
			}
			conn.CopyWaitGroup2(target, c, host.Flow)
		} else {
			connClient = conn.GetConn(target, lk.Crypt, lk.Compress, host.Client.Rate, true)
			err = r.Write(connClient)
			if err != nil {
				logs.Error(err)
				return
			}
			conn.CopyWaitGroup(target, c, lk.Crypt, lk.Compress, host.Client.Rate, host.Flow, true, nil)
		}
	} else {
		ctx := r.Context()
		go func() {
			<-ctx.Done()
			target.Close()
		}()
		if host.Target.LocalProxy {
			targetConn := rate.NewRateConn(target, host.Client.Rate)
			defer targetConn.Close()
			err = r.Write(targetConn)
			if err != nil {
				logs.Error(err)
				return
			}
			targetReader := bufio.NewReader(targetConn)
			resp, err := http.ReadResponse(targetReader, nil)
			if err != nil {
				logs.Error(err)
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
				logs.Error(err)
				w.Header().Set("Access-Control-Allow-Origin", "*")
				http.Error(w, fmt.Sprintf("Failed to read response from target %s.", lk.Host), http.StatusOK)
				return
			}
			targetConn2 := conn.GetConn(target, lk.Crypt, lk.Compress, host.Client.Rate, true)
			defer targetConn2.Close()
			targetReader := bufio.NewReader(targetConn2)
			resp, err := http.ReadResponse(targetReader, nil)
			if err != nil {
				logs.Error(err)
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
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.URL.Scheme = scheme
			s.handleTunneling(w, r)
		}),
		// Disable HTTP/2.
		//TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
}
