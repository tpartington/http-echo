package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	delay      int
	maxJitter  int
	httpPort   int
	httpAddr   string
	listenAddr string
	showHelp   bool
	verbose    bool
	printBody  bool
	showColour bool
)

func readFlags() {
	flag.BoolVar(&showHelp, "help", false, "show this help menu")
	flag.StringVar(&httpAddr, "address", "127.0.0.1", "the TCP address to listen on")
	flag.IntVar(&httpPort, "port", 8000, "the TCP port to listen on")
	flag.IntVar(&delay, "delay", 0, "the time to wait (in milliseconds) before sending a response")
	flag.IntVar(&maxJitter, "jitter", 0, "the maximum amount of jitter (in milliseconds) to add to the response")
	flag.BoolVar(&verbose, "v", false, "show more ouput")
	flag.BoolVar(&showColour, "colour", true, "show coloured output")
	flag.BoolVar(&printBody, "printBody", true, "print the HTTP request body")
	flag.Parse()

	listenAddr = httpAddr + ":" + strconv.Itoa(httpPort)

	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

}

type response struct {
	code    int
	headers map[string]string
	body    string
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

func addDelay(delay int) {
	if delay != 0 {
		if verbose {
			fmt.Printf("Adding %dms of delay", delay)
		}
		time.Sleep(time.Duration((delay)) * time.Millisecond)
	}
}

func addJitter(maxJitter int) {
	if maxJitter != 0 {
		seed := rand.NewSource(time.Now().UnixNano())
		random := rand.New(seed).Float64() * float64(maxJitter)
		jitter := time.Duration(random) * time.Millisecond
		if verbose {
			fmt.Printf("max-jitter=%vms, jitter=%vms\n", maxJitter, jitter)
		}
		time.Sleep(jitter)
	}
}

func requestLogger(r *http.Request) {

	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	body := buf.String()

	fmt.Printf("\n---------- %s ----------\n", time.Now().Local())
	fmt.Printf("> %s %s %s\n", r.Method, r.RequestURI, r.Proto)
	for k, v := range r.Header {
		fmt.Printf("> %s: %s ", k, v)
	}
	fmt.Printf("\n")
	if printBody {
		fmt.Printf("> %s\n", body)
	}
}

func responseLogger(resp response) {

	colour := colorCodes(resp.code)
	fmt.Printf("\n---------- %s ----------\n", time.Now().Local())
	fmt.Printf("%s< %d\n", colour, resp.code)
	for k, v := range resp.headers {
		fmt.Printf("%s> %s: %s ", colour, k, v)
	}
	fmt.Printf("\n")
	if printBody {
		fmt.Printf("%s< %s\n", colour, resp.body)
	}

	// clear any colour codes that were set
	print("\033[0m")
}

func colorCodes(code int) string {

	colourCode := ""
	friendly := "none"

	if showColour {
		if code == 418 {
			colourCode = "\033[35m"
			friendly = "purple"
		}
		if code >= 100 && code <= 199 {
			colourCode = "\033[36m"
			friendly = "cyan"
		}
		if code >= 200 && code <= 299 {
			colourCode = "\033[32m"
			friendly = "green"
		}
		if code >= 300 && code <= 399 {
			colourCode = "\033[33m"
			friendly = "yellow"
		}
		if code >= 400 && code <= 417 {
			colourCode = "\033[34m"
			friendly = "blue"
		}
		if code >= 419 && code <= 499 {
			colourCode = "\033[34m"
			friendly = "blue"
		}
		if code >= 500 && code <= 599 {
			colourCode = "\033[31m"
			friendly = "red"
		}
	}

	if verbose {
		fmt.Printf("showColour=%t, code=%d, colour=%s\n", showColour, code, friendly)
	}

	return colourCode
}

func index() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		// default respond code
		resp := response{
			code: 200,
		}

		// default headers
		headers := make(map[string]string)
		headers["http"] = "echo"
		resp.headers = headers

		parseParams(req, &resp)

		// set the response body
		if resp.code >= 100 && resp.code < 199 {
			resp.body = "Continue"
		}
		if resp.code >= 200 && resp.code < 299 {
			resp.body = "OK"
		}
		if resp.code >= 300 && resp.code < 399 {
			resp.body = "Redirect"
		}
		if resp.code >= 400 && resp.code < 499 {
			resp.body = "Client Failure"
		}
		if resp.code >= 500 && resp.code < 599 {
			resp.body = "Server Failure"
		}

		addDelay(delay)
		addJitter(maxJitter)
		requestLogger(req)
		responseLogger(resp)

		w.WriteHeader(resp.code)
		if rec := recover(); rec != nil {
			fmt.Println("Recovered in f", rec)
		}
		w.Write([]byte(resp.body))

	})
}

func parseParams(req *http.Request, resp *response) {

	// default response codes to use if random is set
	codes := []int{200, 500}

	// parse the params
	u, err := url.Parse(req.URL.String())
	if err != nil {
		fmt.Printf("error parsing url: %v\n", err)
	}
	q := u.Query()

	v := q.Get("delay")
	if v != "" {
		delay, err = strconv.Atoi(v)
		if err != nil {
			fmt.Printf("delay error: %v", err)
		}
	}

	v = q.Get("jitter")
	if v != "" {
		maxJitter, err = strconv.Atoi(v)
		if err != nil {
			fmt.Printf("jitter error: %v", err)
		}
	}

	v = q.Get("codes")
	if v != "" {
		codes = []int{}
		cs := strings.Split(v, ",")
		for _, x := range cs {
			y, err := strconv.Atoi(x)
			if err != nil {
				panic(err)
			}
			codes = append(codes, y)
		}
	}

	v = q.Get("code")
	if v != "" {
		if v == "random" {
			*resp = randomiseResponseCode(resp, len(codes), codes)
		}
		v, err := strconv.Atoi(v)
		if err == nil {
			resp.code = v
		}
	}

}

//
func randomiseResponseCode(resp *response, chance int, codes []int) response {
	rn := rand.Intn(chance)
	i := rn % len(codes)

	resp.code = codes[i]

	return *resp
}
