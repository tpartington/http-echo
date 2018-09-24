package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	delay      int64
	jitter     int64
	httpPort   int
	httpAddr   string
	listenAddr string
	showHelp   bool
	verbose    bool
	printBody  bool
)

func readFlags() {
	flag.BoolVar(&showHelp, "help", false, "show this help menu")
	flag.StringVar(&httpAddr, "address", "127.0.0.1", "the TCP address to listen on")
	flag.IntVar(&httpPort, "port", 8000, "the TCP port to listen on")
	flag.Int64Var(&delay, "delay", 0, "the time to wait (in milliseconds) before sending a response")
	flag.Int64Var(&jitter, "jitter", 0, "the amount of jitter (in milliseconds) to add to the response")
	flag.BoolVar(&verbose, "v", false, "show more ouput")
	flag.BoolVar(&printBody, "printBody", true, "print the HTTP request body")
	flag.Parse()

	listenAddr = httpAddr + ":" + strconv.Itoa(httpPort)

	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

}

func main() {
	readFlags()
	router := http.NewServeMux()
	router.Handle("/", index())

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	fmt.Printf("Listening on: %s\n\n", listenAddr)
	log.Fatal(server.ListenAndServe())
}

func index() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestLogger(r)
		addDelay()
		addJitter()
		fmt.Printf("\n")
		fmt.Fprintf(w, "OK\n")
	})
}

func addDelay() {
	if delay != 0 {
		time.Sleep(time.Duration((delay)) * time.Millisecond)
		if verbose {
			fmt.Printf("Adding %dms of delay", delay)
		}
	}
}

func addJitter() {
	if jitter != 0 {
		s1 := rand.NewSource(time.Now().UnixNano())
		r1 := rand.New(s1).Float64()
		j1 := int64(r1 * float64(jitter))
		d1 := time.Duration(j1) * time.Millisecond
		if verbose {
			fmt.Printf("Adding %v of jitter", d1)
		}
		time.Sleep(d1)
	}
}

func requestLogger(r *http.Request) {

	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	body := buf.String()

	fmt.Printf("Method: %s, URI: %s, Proto: %s, Content-Length: %d\n", r.Method, r.RequestURI, r.Proto, r.ContentLength)
	fmt.Printf("Headers: %s\n", r.Header)
	if printBody {
		fmt.Printf("Body: %s\n", body)
	}
}
