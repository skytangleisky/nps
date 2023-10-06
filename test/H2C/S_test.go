package H2C

import (
	"fmt"
	"golang.org/x/net/http2/h2c"
	"log"
	"net/http"
	"testing"

	"golang.org/x/net/http2"
)

func Test_S(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello h2c")
	})
	s := &http.Server{
		Addr:    ":8972",
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}
	log.Fatal(s.ListenAndServe())

}
