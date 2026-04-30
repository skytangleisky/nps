package main

import (
	"encoding/json"
	"flag"
	"github.com/astaxie/beego/logs"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

type Config struct {
	Addr        string   `json:"addr"`
	RootDir     string   `json:"rootDir"`
	Suffix      string   `json:"suffix"`
	ContentType string   `json:"contentType"`
	Proxy       *string  `json:"proxy,omitempty"`
	TileUrls    []string `json:"tileUrls"`
}

var count int64 = 0

// 读取配置
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// 构建 HTTP client（支持代理）
func buildClient(proxy *string) *http.Client {
	if proxy == nil || *proxy == "" {
		return &http.Client{}
	}
	uProxy, err := url.Parse(*proxy)
	if err != nil {
		log.Println("proxy parse error:", err)
		return &http.Client{}
	}
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(uProxy),
		},
	}
}
func expandPath(p string) string {
	if strings.HasPrefix(p, "~") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[1:])
	}
	path, err := filepath.Abs(p)
	if err != nil {
		log.Fatal(err)
	}
	return path
}

var process = func(w http.ResponseWriter, r *http.Request, rootDir string, tileUrl string, suffix string, proxy *string) {
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
		client := buildClient(proxy)
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

func main() {
	configPath := flag.String("config", "./config.json", "配置文件路径")
	flag.Parse()
	log.SetFlags(log.Llongfile | log.Lmicroseconds | log.Ldate)

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	rootDir := expandPath(cfg.RootDir)
	log.Println("rootDir:", rootDir)

	log.SetFlags(log.Llongfile | log.Lmicroseconds | log.Ldate)
	var handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		//fmt.Println(r.Host)
		if len(cfg.TileUrls) == 0 {
			http.Error(w, "no tile url configured", 500)
			return
		}
		rd := rand.New(rand.NewSource(time.Now().UnixNano()))
		w.Header().Set("Content-Type", "image/jpeg")
		process(w, r, rootDir, cfg.TileUrls[rd.Intn(len(cfg.TileUrls))], cfg.Suffix, cfg.Proxy)
	})
	log.Println("Server running at", cfg.Addr)
	err = http.ListenAndServe(cfg.Addr, h2c.NewHandler(handler, &http2.Server{}))
	if err != nil {
		log.Fatal(err)
	}
}
