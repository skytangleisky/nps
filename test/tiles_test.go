package test

import (
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func Test_tiles(t *testing.T) {
	log.SetFlags(log.Llongfile | log.Lmicroseconds | log.Ldate)
	var homeDir, _ = os.UserHomeDir()
	var count int64 = 0
	var process = func(w http.ResponseWriter, r *http.Request, rootDir string, tileUrl string, suffix string) {
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
		lyrs := query.Get("lyrs")
		var file = rootDir + lyrs + "/" + z + "/" + y + "/" + x + suffix
		log.Print(value)
		_, err := os.Stat(file)
		if err != nil {
			uProxy, _ := url.Parse("http://127.0.0.1:7890")
			transport := &http.Transport{
				Proxy: http.ProxyURL(uProxy),
				//Proxy: nil,
			}
			client := &http.Client{
				Transport: transport,
			}
			var tmpUrl = tileUrl
			tmpUrl = strings.Replace(tmpUrl, "{x}", x, 1)
			tmpUrl = strings.Replace(tmpUrl, "{y}", y, 1)
			tmpUrl = strings.Replace(tmpUrl, "{z}", z, 1)
			tmpUrl = strings.Replace(tmpUrl, "{lyrs}", lyrs, 1)
			req, err := http.NewRequest("GET", tmpUrl, nil)
			if err != nil {
				panic(err)
			}
			req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
			resp, err := client.Do(req)
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				imgBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					panic(err)
					return
				}
				_, err = os.Stat(rootDir + lyrs + "/" + z + "/" + y)
				if err != nil {
					os.MkdirAll(rootDir+lyrs+"/"+z+"/"+y, os.ModePerm)
				}
				err = os.WriteFile(file, imgBytes, 0644)
				if err != nil {
					panic(err)
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

		var rootDir string
		var tileUrl string
		var suffix string
		switch r.Host {
		case "tile.tanglei.site":
			rootDir = homeDir + "/" + "maps/google/CN/"
			urls := []string{
				"https://mt0.google.com/vt?gl=CN&lyrs={lyrs}&x={x}&y={y}&z={z}",
				"https://mt1.google.com/vt?gl=CN&lyrs={lyrs}&x={x}&y={y}&z={z}",
				"https://mt2.google.com/vt?gl=CN&lyrs={lyrs}&x={x}&y={y}&z={z}",
				"https://mt3.google.com/vt?gl=CN&lyrs={lyrs}&x={x}&y={y}&z={z}",
			}
			rand.Seed(time.Now().UnixNano())
			tileUrl = urls[rand.Intn(4)]
			suffix = ".jpg"
			w.Header().Set("Content-Type", "image/jpeg")
			process(w, r, rootDir, tileUrl, suffix)
		case "terrain.tanglei.site":
			rootDir = homeDir + "/" + "maps/mapbox/"
			tileUrl = "https://api.mapbox.com/raster/v1/mapbox.mapbox-terrain-dem-v1/{z}/{x}/{y}.webp?sku=101tGqRwUCYc3&access_token=pk.eyJ1IjoidGFuZ2xlaTIwMTMxNCIsImEiOiJjbGtmOTdyNWoxY2F1M3Jqczk4cGllYXp3In0.9N-H_79ehy4dJeuykZa0xA"
			suffix = ".webp"
			w.Header().Set("Content-Type", "image/webp")
			process(w, r, rootDir, tileUrl, suffix)
		case "vector.tanglei.site":
			rootDir = homeDir + "/" + "maps/mapbox/"
			tileUrl = "https://api.mapbox.com/v4/mapbox.mapbox-streets-v8,mapbox.mapbox-terrain-v2/{z}/{x}/{y}.vector.pbf?sku=101F9W9FEMRxs&access_token=pk.eyJ1Ijoic2hldmF3ZW4iLCJhIjoiY2lwZXN2OGlvMDAwMXR1bmh0aG5vbDFteiJ9.2fsD37adZ1hC2MUU-2xByA"
			suffix = ".pbf"
			w.Header().Set("Content-Type", "application/x-protobuf")
			process(w, r, rootDir, tileUrl, suffix)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	go func() {
		http.ListenAndServeTLS(":3240", "../conf/tanglei.site.pem", "../conf/tanglei.site.key", handler)
	}()
	http.ListenAndServe(":3241", h2c.NewHandler(handler, &http2.Server{}))
	//http.ListenAndServe(":3211", handler)
}
