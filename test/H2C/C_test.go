package H2C

import (
	"crypto/tls"
	"fmt"
	"golang.org/x/net/http2"
	"log"
	"net"
	"net/http"
	"testing"
)

func Test_C(t *testing.T) {
	client := http.Client{
		//Skip TLS dial
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
	}

	//resp, err := client.Get("http://localhost:8972")
	resp, err := client.Get("http://tanglei.site:3211/maps/vt?lyrs=s&gl=CN&x=106&y=54&z=7")
	if err != nil {
		log.Fatal(fmt.Errorf("error making request: %v", err))
	}
	fmt.Println(resp.StatusCode)
	fmt.Println(resp.Proto)
}
