package proxy

import (
	"bufio"
	"crypto/tls"
	"io"
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
		//	logs.Error(err)
		//  os.Exit(0)
		//} else {
		//	go func() {
		//		for {
		//			tcpConn, err := tcpListener.AcceptTCP()
		//			if err != nil {
		//				logs.Error(err)
		//				tcpConn.Close()
		//			} else {
		//				go func() {
		//					s.Process(tcpConn)
		//				}()
		//			}
		//
		//		}
		//	}()
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
			logs.Error(NewHttpsServer(s.httpsListener, s.bridge, s.useCache, s.cacheLen).Start())
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
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	c, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	s.handleHttp(conn.NewConn(c), r)
}

func (s *httpServer) handleHttp(c *conn.Conn, r *http.Request) {
	var (
		host       *file.Host
		target     net.Conn
		err        error
		connClient net.Conn
		//scheme     = r.URL.Scheme
		lk         *conn.Link
		targetAddr string
		isReset    bool
	)
	defer func() {
		if connClient != nil {
			connClient.Close()
		} else {
			s.writeConnFail(c.Conn)
		}
		c.Close()
	}()
	//reset:
	if isReset {
		host.Client.AddConn()
	}
	if host, err = file.GetDb().GetInfoByHost(r.Host, r); err != nil {
		logs.Notice("the url %s %s %s can't be parsed!", r.URL.Scheme, r.Host, r.RequestURI)
		return
	}
	if err := s.CheckFlowAndConnNum(host.Client); err != nil {
		logs.Warn("client id %d, host id %d, error %s, when https connection", host.Client.Id, host.Id, err.Error())
		return
	}
	if !isReset {
		defer host.Client.AddConn()
	}
	if err = s.auth(r, c, host.Client.Cnf.U, host.Client.Cnf.P); err != nil {
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
		return
	}
	connClient = conn.GetConn(target, lk.Crypt, lk.Compress, host.Client.Rate, true)

	//change the host and header and set proxy setting
	common.ChangeHostAndHeader(r, host.HostChange, host.HeaderChange, c.Conn.RemoteAddr().String(), s.addOrigin)
	logs.Trace("%s request, method %s, host %s, url %s, remote address %s, target %s", r.URL.Scheme, r.Method, r.Host, r.URL.Path, c.RemoteAddr().String(), lk.Host)

	err = r.Write(connClient)
	if err != nil {
		logs.Error(err)
		return
	}
	conn.CopyWaitGroup(target, c.Conn, lk.Crypt, lk.Compress, host.Client.Rate, host.Flow, true, c.Rb)
	//conn.CopyWaitGroup(connClient, c.Conn, lk.Crypt, lk.Compress, host.Client.Rate, host.Flow, true, c.Rb)//错误用法
}
func (s *httpServer) handleHttp2(c *conn.Conn, r *http.Request) {
	var (
		host       *file.Host
		target     net.Conn
		err        error
		connClient io.ReadWriteCloser
		//scheme     = r.URL.Scheme
		lk         *conn.Link
		targetAddr string
		lenConn    *conn.LenConn
		isReset    bool
		wg         sync.WaitGroup
	)
	defer func() {
		if connClient != nil {
			connClient.Close()
		} else {
			s.writeConnFail(c.Conn)
		}
		c.Close()
	}()
	//reset:
	if isReset {
		host.Client.AddConn()
	}
	if host, err = file.GetDb().GetInfoByHost(r.Host, r); err != nil {
		logs.Notice("the url %s %s %s can't be parsed!", r.URL.Scheme, r.Host, r.RequestURI)
		return
	}
	if err := s.CheckFlowAndConnNum(host.Client); err != nil {
		logs.Warn("client id %d, host id %d, error %s, when https connection", host.Client.Id, host.Id, err.Error())
		return
	}
	if !isReset {
		defer host.Client.AddConn()
	}
	if err = s.auth(r, c, host.Client.Cnf.U, host.Client.Cnf.P); err != nil {
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
		return
	}
	//change the host and header and set proxy setting
	common.ChangeHostAndHeader(r, host.HostChange, host.HeaderChange, c.Conn.RemoteAddr().String(), s.addOrigin)
	logs.Trace("%s request, method %s, host %s, url %s, remote address %s, target %s", r.URL.Scheme, r.Method, r.Host, r.URL.Path, c.RemoteAddr().String(), lk.Host)
	connClient = conn.GetConn(target, lk.Crypt, lk.Compress, host.Client.Rate, true)

	//read from inc-client
	go func() {
		wg.Add(1)
		isReset = false
		defer connClient.Close()
		defer func() {
			wg.Done()
			if !isReset {
				c.Close()
			}
		}()
		for {
			/*
				if resp, err := http.ReadResponse(bufio.NewReader(connClient), r); err != nil || resp == nil || r == nil {
					// if there got broken pipe, http.ReadResponse will get a nil
					return
				} else {
					//if the cache is start and the response is in the extension,store the response to the cache list
					if s.useCache && r.URL != nil && strings.Contains(r.URL.Path, ".") {
						b, err := httputil.DumpResponse(resp, true)
						if err != nil {
							return
						}
						c.Write(b)
						host.Flow.Add(0, int64(len(b)))
						s.cache.Add(filepath.Join(host.Host, r.URL.Path), b)
					} else {
						lenConn := conn.NewLenConn(c)
						if err := resp.Write(lenConn); err != nil {
							logs.Error(err)
							return
						}
						host.Flow.Add(0, int64(lenConn.Len))
					}
				}*/

			len, err := common.CopyBuffer(c, connClient)
			if err != nil {
				c.Close()
				return
			} else {
				host.Flow.Add(len, 0)
				logs.Error("----------", len)
			}
		}
	}()

	//write
	lenConn = conn.NewLenConn(connClient)

	if err := r.Write(lenConn); err != nil {
		logs.Error(err)
	}
	_, err = common.CopyBuffer(connClient, c)
	if err != nil {
		c.Close()
		connClient.Close()
		return
	}

	//for {
	//	//if the cache start and the request is in the cache list, return the cache
	//	if s.useCache {
	//		if v, ok := s.cache.Get(filepath.Join(host.Host, r.URL.Path)); ok {
	//			n, err := c.Write(v.([]byte))
	//			if err != nil {
	//				break
	//			}
	//			logs.Trace("%s request, method %s, host %s, url %s, remote address %s, return cache", r.URL.Scheme, r.Method, r.Host, r.URL.Path, c.RemoteAddr().String())
	//			host.Flow.Add(0, int64(n))
	//			//if return cache and does not create a new conn with client and Connection is not set or close, close the connection.
	//			if strings.ToLower(r.Header.Get("Connection")) == "close" || strings.ToLower(r.Header.Get("Connection")) == "" {
	//				break
	//			}
	//			goto readReq
	//		}
	//	}
	//
	//	//change the host and header and set proxy setting
	//	common.ChangeHostAndHeader(r, host.HostChange, host.HeaderChange, c.Conn.RemoteAddr().String(), s.addOrigin)
	//	logs.Trace("%s request, method %s, host %s, url %s, remote address %s, target %s", r.URL.Scheme, r.Method, r.Host, r.URL.Path, c.RemoteAddr().String(), lk.Host)
	//	//write
	//	lenConn = conn.NewLenConn(connClient)
	//	if err := r.Write(lenConn); err != nil {
	//		logs.Error(err)
	//		break
	//	}
	//	host.Flow.Add(int64(lenConn.Len), 0)
	//
	//readReq:
	//
	//	_,err=common.CopyBuffer(target,c)
	//	if err!=nil{
	//		target.Close()
	//		break
	//	}
	//
	//	//read req from connection
	//	if r, err = http.ReadRequest(bufio.NewReader(c)); err != nil {
	//		break
	//	}
	//	r.URL.Scheme = scheme
	//	//What happened ，Why one character less???
	//	r.Method = resetReqMethod(r.Method)
	//	if hostTmp, err := file.GetDb().GetInfoByHost(r.Host, r); err != nil {
	//		logs.Notice("the url %s %s %s can't be parsed!", r.URL.Scheme, r.Host, r.RequestURI)
	//		break
	//	} else if host != hostTmp {
	//		host = hostTmp
	//		isReset = true
	//		connClient.Close()
	//		goto reset
	//	}
	//
	//}
	wg.Wait()
}

func resetReqMethod(method string) string {
	if method == "ET" {
		return "GET"
	}
	if method == "OST" {
		return "POST"
	}
	return method
}

func (s *httpServer) NewServer(port int, scheme string) *http.Server {
	return &http.Server{
		Addr: ":" + strconv.Itoa(port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.URL.Scheme = scheme
			s.handleTunneling(w, r)
		}),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
}

func (s *httpServer) Process(tcpConn *net.TCPConn) {
	if r, err := http.ReadRequest(bufio.NewReader(tcpConn)); err != nil || r == nil {
		// if there got broken pipe, http.ReadResponse will get a nil
		return
	} else {
		r.URL.Scheme = "http"
		s.handleHttp(conn.NewConn(tcpConn), r)
	}
}
