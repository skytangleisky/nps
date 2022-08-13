package SOCKS5_http

import (
	"fmt"
	_ "io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"

	"golang.org/x/net/proxy"
)

func Test_socks5(t *testing.T) {
	// create a socks5 dialer
	dialer, err := proxy.SOCKS5("tcp", "tanglei.top:5555", nil, proxy.Direct)
	if err != nil {
		fmt.Fprintln(os.Stderr, "can't connect to the proxy:", err)
		os.Exit(1)
	}
	// setup a http client
	httpTransport := &http.Transport{}
	httpClient := &http.Client{Transport: httpTransport}
	// set our socks5 as the dialer
	httpTransport.Dial = dialer.Dial
	if resp, err := httpClient.Get("https://baidu.com"); err != nil {
		log.Fatalln(err)
	} else {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Printf("%s\n", body)
	}
}
