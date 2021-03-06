package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
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
	// vars for command line options
	showHelp         bool
	debug            bool
	quiet            bool
	printBody        bool
	printRequest     bool
	printResponse    bool
	printProxy       bool
	colour           bool
	timestamp        bool
	enableProxyParam bool

	delay        int
	httpPort     int
	maxJitter    int
	bodySize     int
	idleTimeout  int
	readTimeout  int
	writeTimeout int

	body     string
	code     string
	codes    string
	httpAddr string
	proxyURL string
	headers  string

	cert string
	key  string

	// vars used internally
	listenAddr string
	replace    bool
	empty      bool
	headerMap  map[string]string
	codesArray []int
)

func readFlags() {
	flag.BoolVar(&showHelp, "help", false, "show this help menu")
	flag.BoolVar(&debug, "debug", false, "show debug ouput")
	flag.BoolVar(&quiet, "quiet", false, "hide all log output")
	flag.BoolVar(&printBody, "printBody", true, "print the HTTP request body")
	flag.BoolVar(&printRequest, "printRequest", true, "print the request")
	flag.BoolVar(&printResponse, "printResponse", true, "print the response")
	flag.BoolVar(&printProxy, "printProxy", false, "print the proxy request and response")
	flag.BoolVar(&colour, "colour", true, "show coloured output")
	flag.BoolVar(&timestamp, "timestamp", true, "show the request/response timestamp")
	flag.BoolVar(&enableProxyParam, "enableProxyParam", false, "enable the upstream proxy url to be set as a query parameter")

	flag.IntVar(&delay, "delay", 0, "the time to wait (in milliseconds) before sending a response")
	flag.IntVar(&httpPort, "port", 8000, "the TCP port to listen on")
	flag.IntVar(&maxJitter, "jitter", 0, "the maximum amount of jitter (in milliseconds) to add to the response")
	flag.IntVar(&bodySize, "bodySize", 0, "the number of bytes to print of the request body start and end, 0 will print the whole body")
	flag.IntVar(&idleTimeout, "idleTimeout", 15000, "the idle timeout value (in milliseconds)")
	flag.IntVar(&readTimeout, "readTimeout", 5000, "the read timeout value (in milliseconds)")
	flag.IntVar(&writeTimeout, "writeTimeout", 10000, "the write timeout value (in milliseconds)")

	flag.StringVar(&body, "body", "", "use the contents of this file as the request body")
	flag.StringVar(&code, "code", "", "the (int) response code to send, or if set to 'r' or 'random' use a random one")
	flag.StringVar(&codes, "codes", "", "A list of comma response codes to use when randomising responses")
	flag.StringVar(&headers, "headers", "", "A list of comma separated key,values to add to the response headers")
	flag.StringVar(&httpAddr, "address", "127.0.0.1", "the TCP address to listen on")
	flag.StringVar(&proxyURL, "proxy", "", "A remote address to proxy the connection to")
	flag.StringVar(&cert, "cert", "", "The TLS server certificate to use")
	flag.StringVar(&key, "key", "", "The TLS server key to use")

	flag.Parse()

	listenAddr = httpAddr + ":" + strconv.Itoa(httpPort)

	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

}

func main() {
	readFlags()
	if quiet {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}
	router := http.NewServeMux()
	router.Handle("/", index())

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      router,
		ReadTimeout:  time.Duration(readTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(writeTimeout) * time.Millisecond,
		IdleTimeout:  time.Duration(idleTimeout) * time.Millisecond,
	}

	fmt.Printf("Listening on: %s\n\n", listenAddr)
	if cert == "" || key == "" {
		log.Fatal(server.ListenAndServe())
	} else {
		log.Fatal(server.ListenAndServeTLS(cert, key))
	}
}

func index() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		headerMap = make(map[string]string)

		// set the defaults
		resp := http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
		}
		resp.Header.Add("Server", "http-echo")
		codesArray = []int{200, 500}
		// end of defaults

		if codes != "" {
			parseCodes(codes)
		}

		if code != "" {
			parseResponseCode(code, &resp)

		}

		if headers != "" {
			parseHeaders(headers)
		}

		// parse the query parameters, this is done after parsing the command line options so that
		// the query params will overwrite whatever was set on the command line
		parseQueryParams(req, &resp)

		// set the response body to the status text
		resp.Body = ioutil.NopCloser(bytes.NewBufferString(fmt.Sprintf("%s\n", http.StatusText(resp.StatusCode))))

		if body != "" {
			resp.Body = ioutil.NopCloser(bytes.NewReader(readBodyFromFile(body)))
		}

		if printRequest {
			requestLogger(req, false)
		}
		addDelay(delay)
		addJitter(maxJitter)

		// if the server was started with a proxy
		if proxyURL != "" {
			proxyResp, err := proxy(proxyURL, req)
			if err != nil {
				fmt.Printf("%v\n", err)
			}
			if err == nil {
				// set our response headers with the headers from the upstream
				resp.Header = proxyResp.Header

				// set our response body with the body from the upstream
				resp.Body = proxyResp.Body
			}
		}

		// set the response headers
		for k, v := range resp.Header {
			if debug {
				fmt.Printf("! Setting Response Header: %s=%s\n", k, v)
			}
			w.Header().Set(k, v[0])
		}
		// set the custom response headers
		for k, v := range headerMap {
			if debug {
				fmt.Printf("! Setting Custom Header: %s=%s\n", k, v)
			}
			w.Header().Set(k, v)
			resp.Header.Set(k, v)
		}

		if empty == true {
			closeConnection(w)
		}

		if replace == true {
			replaceBody(req, w)
			resp.Body = req.Body
		}

		// read the response body
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
		// restore the response body
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))

		// send the response
		w.WriteHeader(resp.StatusCode)
		w.Write(body)

		if printResponse {
			responseLogger(&resp, false)
		}
	})
}

func parseQueryParams(req *http.Request, resp *http.Response) {
	// parse the params
	u, err := url.Parse(req.URL.String())
	if err != nil {
		fmt.Printf("error parsing url: %v\n", err)
	}
	q := u.Query()

	// the delay to add the request
	v := q.Get("delay")
	if v != "" {
		delay, err = strconv.Atoi(v)
		if err != nil {
			fmt.Printf("delay error: %v", err)
		}
	}

	// the maximum amount of jitter to add to a request
	v = q.Get("jitter")
	if v != "" {
		maxJitter, err = strconv.Atoi(v)
		if err != nil {
			fmt.Printf("jitter error: %v", err)
		}
	}

	// the response codes to use for a random response
	v = q.Get("codes")
	if v != "" {
		parseCodes(v)
	}

	// set the response code to the given int, or if random to a random value of codes
	v = q.Get("code")
	if v != "" {
		parseResponseCode(v, resp)
	}

	// set the proxy url
	v = q.Get("proxy")
	if v != "" && enableProxyParam {
		proxyURL = v
	}

	// set the location header
	v = q.Get("location")
	if v != "" {
		headerMap["Location"] = v
	}

	// the key,value pairs of custom headers to set
	v = q.Get("headers")
	if v != "" {
		parseHeaders(v)
	}

	// if close = true shutdown the server
	v = q.Get("empty")
	if v == "true" {
		empty = true
	}

	// if true replace the response body with
	// whatever was provided in the request body
	v = q.Get("replace")
	if v == "true" {
		replace = true
	}
}

func parseCodes(codes string) {
	codesArray = []int{}
	cs := strings.Split(codes, ",")
	for _, x := range cs {
		y, err := strconv.Atoi(x)
		if err != nil {
			panic(err)
		}
		codesArray = append(codesArray, y)
	}
}

func parseResponseCode(code string, resp *http.Response) {
	if code == "random" || code == "r" {
		resp = randomiseResponseCode(resp, codesArray)
	}
	v, err := strconv.Atoi(code)
	if err == nil {
		resp.StatusCode = v
	}
}

func parseHeaders(headers string) {
	hs := strings.Split(headers, ",")
	if debug {
		fmt.Printf("! headers=%v\n", hs)
	}
	size := len(hs)
	i := 0
	for i <= (size - 1) {
		headerMap[hs[i]] = hs[i+1]
		i = i + 2
	}
}

func readBodyFromFile(file string) []byte {
	body, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	return body
}

func addDelay(delay int) {
	if delay != 0 {
		if debug {
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
		if debug {
			fmt.Printf("! max-jitter=%vms, jitter=%vms\n", maxJitter, jitter)
		}
		time.Sleep(jitter)
	}
}

func proxy(proxyURL string, req *http.Request) (http.Response, error) {
	// check the proxyURL is valid
	_, err := url.Parse(proxyURL)
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	nr, err := http.NewRequest(req.Method, proxyURL, req.Body)
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	tr := &http.Transport{
		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
	}
	client := &http.Client{Transport: tr}

	nr.Header = req.Header

	if printProxy {
		requestLogger(nr, true)
	}
	resp, err := client.Do(nr)
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	if printProxy {
		responseLogger(resp, true)
	}
	return *resp, err

}

func requestLogger(req *http.Request, proxy bool) {

	// if proxy is true offset the output
	o := ""
	p := ""
	if proxy {
		o = "   "
		p = "Proxy "
	}

	// read the request body
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	// restore the request body
	req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	if timestamp {
		fmt.Printf("\n%s---------- %sRequest: %s ----------\n", o, p, time.Now().Local())
	}
	fmt.Printf("%s> %s %s %s\n", o, req.Method, req.RequestURI, req.Proto)
	for k, v := range req.Header {
		fmt.Printf("%s> %s: %s\n", o, k, v)
	}
	if string(body) != "" {
		if printBody {
			if bodySize < 1 || len(body) <= (bodySize*2) {
				s := strings.Replace(string(body), "\\n", "\n", -1)
				fmt.Printf("%s%s\n", o, s)
			} else {
				bodyStart := body[0:bodySize]
				bodyEnd := body[len(body)-bodySize:]
				fmt.Printf("%s%s\n", o, bodyStart)
				fmt.Printf("%s...\n", o)
				fmt.Printf("%s%s\n", o, bodyEnd)
			}
		}
	}
}

func responseLogger(resp *http.Response, proxy bool) {

	// if proxy is true offset the output
	o := ""
	p := ""
	if proxy {
		o = "   "
		p = "Proxy "
	}

	// read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	// restore the response body
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	if timestamp {
		fmt.Printf("\n%s---------- %sResponse: %s ----------\n", o, p, time.Now().Local())
	}
	print(colorCodes(resp.StatusCode))
	fmt.Printf("%s< %d\n", o, resp.StatusCode)
	for k, v := range resp.Header {
		fmt.Printf("%s< %s: %s\n", o, k, v)
	}
	if string(body) != "" {
		if printBody {
			s := strings.Replace(string(body), "\\n", "\n", -1)
			fmt.Printf("%s%s\n", o, s)
		}
	}

	// clear any colour codes that were set
	print("\033[0m")
}

func colorCodes(code int) string {

	colourCode := ""
	friendly := "none"

	if colour {
		if code == 418 {
			colourCode = "\033[35m"
			friendly = "purple"
		}
		if code >= 100 && code <= 199 {
			colourCode = "\033[36m"
			friendly = "cyan"
		}
		if code >= 200 && code <= 299 {
			colourCode = "\033[34m"
			friendly = "blue"
		}
		if code >= 300 && code <= 399 {
			colourCode = "\033[32m"
			friendly = "green"
		}
		if code >= 400 && code <= 417 {
			colourCode = "\033[33m"
			friendly = "yellow"
		}
		if code >= 419 && code <= 499 {
			colourCode = "\033[33m"
			friendly = "yellow"
		}
		if code >= 500 && code <= 599 {
			colourCode = "\033[31m"
			friendly = "red"
		}
	}

	if debug {
		fmt.Printf("! colour=%t, code=%d, friendly=%s\n", colour, code, friendly)
	}

	return colourCode
}

func closeConnection(w http.ResponseWriter) {
	hj, _ := w.(http.Hijacker)
	conn, _, err := hj.Hijack()
	if err != nil {
		fmt.Printf("error hijacking connection: %v\n", err)
	}
	conn.Close()
}

func replaceBody(req *http.Request, w http.ResponseWriter) {

	// read the request body
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	// restore the request body
	req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	hj, _ := w.(http.Hijacker)
	conn, buf, err := hj.Hijack()
	if err != nil {
		fmt.Printf("error hijacking connection: %v\n", err)
	}

	bs := strings.Split(string(body), "\\n")
	if len(bs) > 0 {
		for _, b := range bs {
			if debug {
				fmt.Printf("! %s\n", b)
			}
			buf.WriteString(b)
			buf.WriteString("\n")
		}
	}
	buf.Flush()
	conn.Close()
}

func randomiseResponseCode(resp *http.Response, codes []int) *http.Response {
	rn := rand.Intn(len(codes))
	i := rn % len(codes)
	resp.StatusCode = codes[i]

	if debug {
		fmt.Printf("! len=%d, rn=%d, i=%d, code=%d", len(codes), rn, i, resp.StatusCode)
	}

	return resp
}
