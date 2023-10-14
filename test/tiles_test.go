package test

import (
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"io"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
)

func Test_tiles(t *testing.T) {
	var count int64 = 0
	log.SetFlags(log.Llongfile | log.Lmicroseconds | log.Ldate)
	var handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&count, 1)
		value := atomic.LoadInt64(&count)
		log.Print(value)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		query := r.URL.Query()
		z := query.Get("z")
		y := query.Get("y")
		x := query.Get("x")
		lyrs := query.Get("lyrs")
		_, err := os.Stat(lyrs + "/" + z + "/" + y + "/" + x + ".jpg")
		if err != nil {
			client := &http.Client{}
			req, err := http.NewRequest("GET", "https://gac-geo.googlecnapps.cn/maps/vt?lyrs="+lyrs+"&gl=CN&x="+x+"&y="+y+"&z="+z, nil)
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
				_, err = os.Stat(lyrs + "/" + z + "/" + y)
				if err != nil {
					os.MkdirAll(lyrs+"/"+z+"/"+y, os.ModePerm)
				}
				err = os.WriteFile(lyrs+"/"+z+"/"+y+"/"+x+".jpg", imgBytes, 0644)
				if err != nil {
					panic(err)
				}
				http.ServeFile(w, r, lyrs+"/"+z+"/"+y+"/"+x+".jpg")
			} else {
				w.WriteHeader(resp.StatusCode)
			}
		} else {
			http.ServeFile(w, r, lyrs+"/"+z+"/"+y+"/"+x+".jpg")
		}
		atomic.AddInt64(&count, -1)
	})
	go func() {
		http.ListenAndServeTLS(":3210", "../conf/9983347_tanglei.site.pem", "../conf/9983347_tanglei.site.key", handler)
	}()
	http.ListenAndServe(":3211", h2c.NewHandler(handler, &http2.Server{}))
	//http.ListenAndServe(":3211", handler)
}
