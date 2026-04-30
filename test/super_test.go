package test

import (
	"fmt"
	"github.com/astaxie/beego/logs"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func Test_tiles0(t *testing.T) {
	uProxy, _ := url.Parse("http://172.18.7.38:4444")
	transport := &http.Transport{
		Proxy: http.ProxyURL(uProxy),
		//Proxy: nil,
	}
	log.SetFlags(log.Llongfile | log.Lmicroseconds | log.Ldate)
	var homeDir, _ = os.UserHomeDir()
	var count int64 = 0
	var process = func(w http.ResponseWriter, r *http.Request, rootDir string, tileUrl string, suffix string) {
		ctx := r.Context()
		atomic.AddInt64(&count, 1)
		value := atomic.LoadInt64(&count)
		defer func() {
			atomic.AddInt64(&count, -1)
			value = atomic.LoadInt64(&count)
			log.Print(value)
		}()
		query := r.URL.Query()
		z := query.Get("z")
		y := query.Get("y")
		x := query.Get("x")
		var file = rootDir + "/" + z + "/" + y + "/" + x + suffix
		log.Print(value)
		_, err := os.Stat(file)
		if err != nil {
			client := &http.Client{
				Transport: transport,
			}
			var tmpUrl = tileUrl
			tmpUrl = strings.Replace(tmpUrl, "{x}", x, -1)
			tmpUrl = strings.Replace(tmpUrl, "{y}", y, -1)
			tmpUrl = strings.Replace(tmpUrl, "{z}", z, -1)
			req, err := http.NewRequestWithContext(ctx, "GET", tmpUrl, nil)
			if err != nil {
				logs.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
			resp, err := client.Do(req)
			if err != nil {
				logs.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				imgBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					logs.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				_, err = os.Stat(rootDir + "/" + z + "/" + y)
				if err != nil {
					os.MkdirAll(rootDir+"/"+z+"/"+y, os.ModePerm)
				}
				err = os.WriteFile(file, imgBytes, 0644)
				if err != nil {
					logs.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				log.Println(file)
				http.ServeFile(w, r, file)
			} else {
				w.WriteHeader(resp.StatusCode)
			}
		} else {
			http.ServeFile(w, r, file)
		}
	}
	var handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		//fmt.Println(r.Host)
		var rootDir string
		var tileUrl string
		var suffix string
		rootDir = homeDir + "/" + "maps/" + "道路/"
		urls := []string{
			"http://10.225.6.188:3141/xbry/maps/DataServer?T=cia_w&tk=b5c6d22f3ea7a78d2526bcc2552882ef&x={x}&y={y}&l={z}",
		}
		rd := rand.New(rand.NewSource(time.Now().UnixNano()))
		tileUrl = urls[rd.Intn(len(urls))]
		suffix = ".jpg"
		w.Header().Set("Content-Type", "image/jpeg")
		process(w, r, rootDir, tileUrl, suffix)
	})
	http.ListenAndServe(":3140", h2c.NewHandler(handler, &http2.Server{}))
}
func Test_tiles1(t *testing.T) {
	//uProxy, _ := url.Parse("http://127.0.0.1:7890")
	//transport := &http.Transport{
	//	Proxy: http.ProxyURL(uProxy),
	//	//Proxy: nil,
	//}
	log.SetFlags(log.Llongfile | log.Lmicroseconds | log.Ldate)
	var homeDir, _ = os.UserHomeDir()
	var count int64 = 0
	var process = func(w http.ResponseWriter, r *http.Request, rootDir string, tileUrl string, suffix string) {
		ctx := r.Context()
		atomic.AddInt64(&count, 1)
		value := atomic.LoadInt64(&count)
		defer func() {
			atomic.AddInt64(&count, -1)
			value = atomic.LoadInt64(&count)
			log.Print(value)
		}()
		query := r.URL.Query()
		z := query.Get("z")
		y := query.Get("y")
		x := query.Get("x")
		var file = rootDir + "/" + z + "/" + y + "/" + x + suffix
		log.Print(value)
		_, err := os.Stat(file)
		if err != nil {
			client := &http.Client{
				//Transport: transport,
			}
			var tmpUrl = tileUrl
			tmpUrl = strings.Replace(tmpUrl, "{x}", x, -1)
			tmpUrl = strings.Replace(tmpUrl, "{y}", y, -1)
			tmpUrl = strings.Replace(tmpUrl, "{z}", z, -1)
			req, err := http.NewRequestWithContext(ctx, "GET", tmpUrl, nil)
			if err != nil {
				logs.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
			resp, err := client.Do(req)
			if err != nil {
				logs.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				imgBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					logs.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				_, err = os.Stat(rootDir + "/" + z + "/" + y)
				if err != nil {
					os.MkdirAll(rootDir+"/"+z+"/"+y, os.ModePerm)
				}
				err = os.WriteFile(file, imgBytes, 0644)
				if err != nil {
					logs.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				log.Println(file)
				http.ServeFile(w, r, file)
			} else {
				w.WriteHeader(resp.StatusCode)
			}
		} else {
			http.ServeFile(w, r, file)
		}
	}
	var handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		//fmt.Println(r.Host)
		var rootDir string
		var tileUrl string
		var suffix string
		rootDir = homeDir + "/" + "maps/" + "矢量/"
		urls := []string{
			"https://services.arcgisonline.com/ArcGIS/rest/services/Canvas/World_Dark_Gray_Base/MapServer/WMTS/tile/1.0.0/Canvas_World_Dark_Gray_Base/default/default028mm/{z}/{y}/{x}/",
		}
		rd := rand.New(rand.NewSource(time.Now().UnixNano()))
		tileUrl = urls[rd.Intn(len(urls))]
		suffix = ".jpg"
		w.Header().Set("Content-Type", "image/jpeg")
		process(w, r, rootDir, tileUrl, suffix)
	})
	http.ListenAndServe(":3141", h2c.NewHandler(handler, &http2.Server{}))
}
func Test_tiles2(t *testing.T) {
	//uProxy, _ := url.Parse("http://127.0.0.1:7890")
	//transport := &http.Transport{
	//	Proxy: http.ProxyURL(uProxy),
	//	//Proxy: nil,
	//}
	log.SetFlags(log.Llongfile | log.Lmicroseconds | log.Ldate)
	var homeDir, _ = os.UserHomeDir()
	var count int64 = 0
	var process = func(w http.ResponseWriter, r *http.Request, rootDir string, tileUrl string, suffix string) {
		ctx := r.Context()
		atomic.AddInt64(&count, 1)
		value := atomic.LoadInt64(&count)
		defer func() {
			atomic.AddInt64(&count, -1)
			value = atomic.LoadInt64(&count)
			log.Print(value)
		}()
		query := r.URL.Query()
		z := query.Get("z")
		y := query.Get("y")
		x := query.Get("x")
		var file = rootDir + "/" + z + "/" + y + "/" + x + suffix
		log.Print(value)
		_, err := os.Stat(file)
		if err != nil {
			client := &http.Client{
				//Transport: transport,
			}
			var tmpUrl = tileUrl
			tmpUrl = strings.Replace(tmpUrl, "{x}", x, -1)
			tmpUrl = strings.Replace(tmpUrl, "{y}", y, -1)
			tmpUrl = strings.Replace(tmpUrl, "{z}", z, -1)
			req, err := http.NewRequestWithContext(ctx, "GET", tmpUrl, nil)
			if err != nil {
				logs.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
			req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36")
			req.Header.Set("Sec-Fetch-Mode", "navigate")
			req.Header.Set("Referer", "https://www.arcgis.com/")
			resp, err := client.Do(req)
			if err != nil {
				logs.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				imgBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					logs.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				_, err = os.Stat(rootDir + "/" + z + "/" + y)
				if err != nil {
					os.MkdirAll(rootDir+"/"+z+"/"+y, os.ModePerm)
				}
				err = os.WriteFile(file, imgBytes, 0644)
				if err != nil {
					logs.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				log.Println(file)
				http.ServeFile(w, r, file)
			} else {
				w.WriteHeader(resp.StatusCode)
			}
		} else {
			http.ServeFile(w, r, file)
		}
	}
	var handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		//fmt.Println(r.Host)
		var rootDir string
		var tileUrl string
		var suffix string
		rootDir = homeDir + "/" + "maps/" + "影像/"
		urls := []string{
			"https://services.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}/",
		}
		rd := rand.New(rand.NewSource(time.Now().UnixNano()))
		tileUrl = urls[rd.Intn(len(urls))]
		suffix = ".jpg"
		w.Header().Set("Content-Type", "image/jpeg")
		process(w, r, rootDir, tileUrl, suffix)
	})
	http.ListenAndServe(":3142", h2c.NewHandler(handler, &http2.Server{}))
}
func Test_tiles3(t *testing.T) {
	//uProxy, _ := url.Parse("http://127.0.0.1:7890")
	//transport := &http.Transport{
	//	Proxy: http.ProxyURL(uProxy),
	//	//Proxy: nil,
	//}
	log.SetFlags(log.Llongfile | log.Lmicroseconds | log.Ldate)
	var homeDir, _ = os.UserHomeDir()
	var count int64 = 0
	var process = func(w http.ResponseWriter, r *http.Request, rootDir string, tileUrl string, suffix string) {
		ctx := r.Context()
		atomic.AddInt64(&count, 1)
		value := atomic.LoadInt64(&count)
		defer func() {
			atomic.AddInt64(&count, -1)
			value = atomic.LoadInt64(&count)
			log.Print(value)
		}()
		query := r.URL.Query()
		z := query.Get("z")
		y := query.Get("y")
		x := query.Get("x")
		var file = rootDir + "/" + z + "/" + y + "/" + x + suffix
		log.Print(value)
		_, err := os.Stat(file)
		if err != nil {
			client := &http.Client{
				//Transport: transport,
			}
			var tmpUrl = tileUrl
			tmpUrl = strings.Replace(tmpUrl, "{x}", x, -1)
			tmpUrl = strings.Replace(tmpUrl, "{y}", y, -1)
			tmpUrl = strings.Replace(tmpUrl, "{z}", z, -1)
			req, err := http.NewRequestWithContext(ctx, "GET", tmpUrl, nil)
			if err != nil {
				logs.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
			resp, err := client.Do(req)
			if err != nil {
				logs.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				imgBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					logs.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				_, err = os.Stat(rootDir + "/" + z + "/" + y)
				if err != nil {
					os.MkdirAll(rootDir+"/"+z+"/"+y, os.ModePerm)
				}
				err = os.WriteFile(file, imgBytes, 0644)
				if err != nil {
					logs.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				log.Println(file)
				http.ServeFile(w, r, file)
			} else {
				w.WriteHeader(resp.StatusCode)
			}
		} else {
			http.ServeFile(w, r, file)
		}
	}
	var handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		//fmt.Println(r.Host)
		var rootDir string
		var tileUrl string
		var suffix string
		rootDir = homeDir + "/" + "maps/" + "地形/"
		urls := []string{
			"https://maps-for-free.com/layer/relief/z{z}/row{y}/{z}_{x}-{y}.jpg",
		}
		rd := rand.New(rand.NewSource(time.Now().UnixNano()))
		tileUrl = urls[rd.Intn(len(urls))]
		suffix = ".jpg"
		w.Header().Set("Content-Type", "image/jpeg")
		process(w, r, rootDir, tileUrl, suffix)
	})
	http.ListenAndServe(":3143", h2c.NewHandler(handler, &http2.Server{}))
}
func Test_tiles4(t *testing.T) {
	//uProxy, _ := url.Parse("http://127.0.0.1:7890")
	//transport := &http.Transport{
	//	Proxy: http.ProxyURL(uProxy),
	//	//Proxy: nil,
	//}
	log.SetFlags(log.Llongfile | log.Lmicroseconds | log.Ldate)
	var homeDir, _ = os.UserHomeDir()
	var count int64 = 0
	var process = func(w http.ResponseWriter, r *http.Request, rootDir string, tileUrl string, suffix string) {
		ctx := r.Context()
		atomic.AddInt64(&count, 1)
		value := atomic.LoadInt64(&count)
		defer func() {
			atomic.AddInt64(&count, -1)
			value = atomic.LoadInt64(&count)
			log.Print(value)
		}()
		query := r.URL.Query()
		z := query.Get("z")
		y := query.Get("y")
		x := query.Get("x")
		var file = rootDir + "/" + z + "/" + y + "/" + x + suffix
		log.Print(value)
		_, err := os.Stat(file)
		if err != nil {
			client := &http.Client{
				//Transport: transport,
			}
			var tmpUrl = tileUrl
			tmpUrl = strings.Replace(tmpUrl, "{x}", x, -1)
			tmpUrl = strings.Replace(tmpUrl, "{y}", y, -1)
			tmpUrl = strings.Replace(tmpUrl, "{z}", z, -1)
			req, err := http.NewRequestWithContext(ctx, "GET", tmpUrl, nil)
			if err != nil {
				logs.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
			resp, err := client.Do(req)
			if err != nil {
				logs.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				imgBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					logs.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				_, err = os.Stat(rootDir + "/" + z + "/" + y)
				if err != nil {
					os.MkdirAll(rootDir+"/"+z+"/"+y, os.ModePerm)
				}
				err = os.WriteFile(file, imgBytes, 0644)
				if err != nil {
					logs.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				log.Println(file)
				http.ServeFile(w, r, file)
			} else {
				w.WriteHeader(resp.StatusCode)
			}
		} else {
			http.ServeFile(w, r, file)
		}
	}
	var handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		//fmt.Println(r.Host)
		var rootDir string
		var tileUrl string
		var suffix string
		rootDir = homeDir + "/" + "maps/" + "航线/"
		urls := []string{
			"http://127.0.0.1:3000/backend/image?x={x}&y={y}&z={z}&strokeColor=rgba(0,255,255,1)&fillColor=rgba(0,255,255,0.2)&textColor=rgba(0,255,255,1)",
		}
		rd := rand.New(rand.NewSource(time.Now().UnixNano()))
		tileUrl = urls[rd.Intn(len(urls))]
		suffix = ".jpg"
		w.Header().Set("Content-Type", "image/jpeg")
		process(w, r, rootDir, tileUrl, suffix)
	})
	http.ListenAndServe(":3144", h2c.NewHandler(handler, &http2.Server{}))
}

func Test_server(t *testing.T) {
	addr := ":5001"
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	fmt.Println("UDP server listening on", addr)

	buf := make([]byte, 65535)
	var total int64
	start := time.Now()

	for {
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			fmt.Println("Read error:", err)
			continue
		}
		total += int64(n)
		if time.Since(start) >= time.Second {
			mbps := float64(total) / 1e6
			fmt.Printf("Received %.2f MB/s\n", mbps)
			total = 0
			start = time.Now()
		}
	}
}

func Test_client(t *testing.T) {
	serverAddr := "127.0.0.1:5001"
	conn, err := net.Dial("udp", serverAddr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	fmt.Println("UDP client sending to", serverAddr)

	payload := make([]byte, 1470) // 常用MTU大小
	//payload := make([]byte, 1024*9) // 常用MTU大小
	var total int64
	start := time.Now()

	for {
		n, err := conn.Write(payload)
		if err != nil {
			fmt.Println("Write error:", err)
			continue
		}
		total += int64(n)
		if time.Since(start) >= time.Second {
			mbps := float64(total) / 1e6
			fmt.Printf("Sent %.2f MB/s\n", mbps)
			total = 0
			start = time.Now()
		}
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 64*1024) // 64KB缓冲
	var total int64
	start := time.Now()

	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				fmt.Println("Read error:", err)
			}
			break
		}
		total += int64(n)
		if time.Since(start) >= time.Second {
			mb := float64(total) / 1e6
			fmt.Printf("Received %.2f MB/s\n", mb)
			total = 0
			start = time.Now()
		}
	}
}
func Test_S(t *testing.T) {
	listener, err := net.Listen("tcp", ":5002")
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	fmt.Println("TCP server listening on :5002")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept error:", err)
			continue
		}
		go handleConn(conn)
	}
}

func Test_C(t *testing.T) {
	conn, err := net.Dial("tcp", "127.0.0.1:5002")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	fmt.Println("TCP client connected to 127.0.0.1:5002")

	payload := make([]byte, 64*1024) // 64KB数据
	var total int64
	start := time.Now()

	for {
		n, err := conn.Write(payload)
		if err != nil {
			fmt.Println("Write error:", err)
			break
		}
		total += int64(n)
		if time.Since(start) >= time.Second {
			mb := float64(total) / 1e6
			fmt.Printf("Sent %.2f MB/s\n", mb)
			total = 0
			start = time.Now()
		}
	}
}
