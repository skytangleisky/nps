package websocketClient

import (
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"testing"
	"time"
)

//var websocketAddr = flag.String("websocketAddr", "ws://127.0.0.1:9999/debug", "websocket address")
var websocketAddr = flag.String("websocketAddr", "wss://union.tanglei.top/debug", "websocket address")
var proxy = flag.String("proxy", "http://192.168.0.112:1122", "http proxy address")

func Test_wsC(t *testing.T) {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	//Initialize the Proxy URL and the Path to follow
	uProxy, _ := url.Parse(*proxy)
	//Set the Dialer (especially the proxy)
	dialer := websocket.Dialer{
		Proxy: http.ProxyURL(uProxy),
	}
	//dialer := websocket.DefaultDialer ==> with this default dialer, it works !

	c, _, err := dialer.Dial(*websocketAddr, nil) // ==> With the proxy config, it fails here !
	if err != nil {
		fmt.Print(err)
		return
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer c.Close()
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Println("write:", err)
				return
			}
		case <-interrupt:
			log.Println("interrupt")
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			c.Close()
			return
		}
	}
}
