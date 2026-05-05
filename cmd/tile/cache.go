package main

import (
	"encoding/json"
	"flag"
	"fmt"
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
	Disabled    *bool    `json:"disabled,omitempty"`
}

var count int64 = 0

func loadConfig(path string) ([]Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// 先尝试数组
	var cfgs []Config
	if err := json.Unmarshal(data, &cfgs); err == nil {
		return cfgs, nil
	}

	// 再尝试单对象
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return []Config{cfg}, nil
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
		log.Println(value)
	}()
	var before = time.Now()
	query := r.URL.Query()
	z := query.Get("z")
	y := query.Get("y")
	x := query.Get("x")
	var file = rootDir + "/" + z + "/" + y + "/" + x + suffix
	log.Print(value)
	var size int64 = 0
	info, err := os.Stat(file)
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
			size = int64(len(imgBytes))
			var after = time.Now()
			log.Println(fmt.Sprintf("%s", formatDuration(after.Sub(before))), fmt.Sprintf("%.2fKB", float64(size)/1024), file)
			http.ServeFile(w, r, file)
		} else {
			w.WriteHeader(resp.StatusCode)
		}
	} else {
		size = info.Size()
		http.ServeFile(w, r, file)
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%+dms", d.Milliseconds())
	}
	return fmt.Sprintf("%+.2fs", d.Seconds())
}
func exeDir() string {
	exe, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return filepath.Dir(exe)
}

// ResolvePath 统一路径解析
func ResolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}

	base := exeDir()

	// 重点：./ 或 . 开头 => 相对 exeDir
	if strings.HasPrefix(p, "./") {
		return filepath.Join(base, p[2:])
	}

	if p == "." {
		return base
	}

	// 可选：其他相对路径也按 exeDir 处理
	return filepath.Join(base, p)
}
func main() {
	configPath := flag.String("config", "./config.json", "配置文件路径")
	flag.Parse()
	log.SetFlags(log.Llongfile | log.Lmicroseconds | log.Ldate)
	path := ResolvePath(*configPath)
	cfgs, err := loadConfig(path)
	if err != nil {
		log.Fatal(err)
	}
	for _, cfg := range cfgs {
		cfg := cfg
		if cfg.Disabled == nil || *cfg.Disabled == false {
			go func() {
				rootDir := expandPath(cfg.RootDir)
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
				log.Println("rootDir:", rootDir, "Server running at", cfg.Addr)
				err = http.ListenAndServe(cfg.Addr, h2c.NewHandler(handler, &http2.Server{}))
				if err != nil {
					log.Fatal(err)
				}
			}()
		}
	}
	select {}
}
