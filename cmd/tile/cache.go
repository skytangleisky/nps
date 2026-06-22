package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/astaxie/beego/logs"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/singleflight"
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
		return &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy:               http.ProxyURL(uProxy),
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     90 * time.Second,
			ForceAttemptHTTP2:   true,
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

var process = func(w http.ResponseWriter, r *http.Request, rootDir string, tileUrl string, suffix string, client *http.Client) {
	ctx := r.Context()
	atomic.AddInt64(&count, 1)
	//value := atomic.LoadInt64(&count)
	defer func() {
		atomic.AddInt64(&count, -1)
		//value = atomic.LoadInt64(&count)
		//log.Println(value)
	}()
	//var before = time.Now()
	query := r.URL.Query()
	z := query.Get("z")
	y := query.Get("y")
	x := query.Get("x")
	var file = rootDir + "/" + z + "/" + y + "/" + x + suffix
	//log.Print(value)
	f, err, _ := sf.Do(file, func() (interface{}, error) {
		if _, err := os.Stat(file); err == nil {
			return nil, nil
		}
		tmpUrl := strings.ReplaceAll(tileUrl, "{x}", x)
		tmpUrl = strings.ReplaceAll(tmpUrl, "{y}", y)
		tmpUrl = strings.ReplaceAll(tmpUrl, "{z}", z)

		req, err := http.NewRequestWithContext(ctx, "GET", tmpUrl, nil)
		if err != nil {
			return nil, err
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			w.WriteHeader(resp.StatusCode)
			w.Write([]byte(fmt.Sprintf("bad status: %d", resp.StatusCode)))
			return nil, nil
		}

		os.MkdirAll(filepath.Dir(file), 0755)

		tmp := file + ".tmp"
		f, err := os.Create(tmp)
		if err != nil {
			return nil, err
		}

		_, err = io.Copy(f, resp.Body)
		f.Close()
		if err != nil {
			os.Remove(tmp)
			return f, err
		}

		return nil, os.Rename(tmp, file)
	})
	if err != nil {
		logs.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if f != nil {
		var file = f.(*os.File)
		file.Seek(0, 0)
		io.Copy(w, file)
	} else {
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

var sf singleflight.Group

func main() {
	rand.Seed(time.Now().UnixNano())
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
				client := buildClient(cfg.Proxy)
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
					w.Header().Set("Content-Type", cfg.ContentType)
					process(w, r, rootDir, cfg.TileUrls[rand.Intn(len(cfg.TileUrls))], cfg.Suffix, client)
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
