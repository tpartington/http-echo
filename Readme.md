# HTTP Echo

A small utility for echoing http requests and returning a customised response

## Usage

```plain
Usage of http-echo:
  -address string
    	the TCP address to listen on (default "127.0.0.1")
  -code string
    	the (int) response code to send, or if set to 'r' or 'random' use a random one
  -codes string
    	A list of comma response codes to use when randomising responses
  -colour
    	show coloured output (default true)
  -debug
    	show debug ouput
  -delay int
    	the time to wait (in milliseconds) before sending a response
  -enableProxyParam
    	enable the upstream proxy url to be set as a query parameter
  -headers string
    	A list of comma separated key,values to add to the response headers
  -help
    	show this help menu
  -idleTimeout int
    	the idle timeout value (in milliseconds) (default 15000)
  -jitter int
    	the maximum amount of jitter (in milliseconds) to add to the response
  -port int
    	the TCP port to listen on (default 8000)
  -printBody
    	print the HTTP request body (default true)
  -printProxy
    	print the proxy request and response
  -printRequest
    	print the request (default true)
  -printResponse
    	print the response (default true)
  -proxy string
    	A remote address to proxy the connection to
  -quiet
    	hide all log output
  -readTimeout int
    	the read timeout value (in milliseconds) (default 5000)
  -shortBody int
    	the number of bytes to print of the request body start and end, 0 will print the whole body
  -timestamp
    	show the request/response timestamp (default true)
  -writeTimeout int
    	the write timeout value (in milliseconds) (default 10000)
```

## Query Parameters

The following query parameters are accepted, query parameters will overwrite any command line parameters:

```plain
  -headers
     comma separated of keys and value pairs
  -location
     URL to set the location header to (useful when setting the status code to 3xx)
  -code
    The response code to set, or the string random for a random response code
  -codes
    comma separter list of response codes to use when used with code=random
  -delay
   the time to wait (in milliseconds) before sending a response, overrides the command line value if given
  -jitter
  the maximum amount of jitter (in milliseconds) to add to the response, overrides the command line value if given
  -close
  if 'true' close the connection immediately before sending a response
  -replace
  if 'true' replay the request body as the response, use '\n' for a newline
  -proxy
  proxy the request to the provided url, this will only work if the server is started with -enableProxyParam 
```

## Examples

Start the http-echo server, responding to request in between 100ms and 300ms (100ms of delay +200ms of jitter):

```shell
http-echo -delay=100 -jitter=200
```

Use curl to connect to http-echo, http-echo will return a 302, with a redirect location of itself
on the second request, http-echo will respond with either a 200 or a 502

```shell
curl --location 'http://localhost:8000/?code=302&location=/?code=random&codes=200,502'
```

Use curl to connect to http-echo, 3/4 of the time http-echo will return a 200, 1/4 of the time it will return a 502 

```shell
curl 'http://localhost:8000/?code=random&codes=200,200,200,502'
```

Use curl to connect to http-echo, http-echo will return a 200 with several custom headers set

```shell
curl 'http://localhost:8000/?headers=cache-control,private,content-length,500'
```

Use curl to connect to http-echo, http-echo will return the value of the post body, in this case

```plain
hello
world
```

```shell
curl -v -d "hello\nworld" 'localhost:8000/?hijack=true'
```

Run the http-echo server but proxy requests to http://checkip.amazonaws.com, printing the proxied requests and setting the response code to 500, and the server header to Apache

```shell
http-echo -proxy=https://checkip.amazonaws.com -printProxy=true -code=500 -headers=Server,Apache
```

Use curl to connect, replacing the response body with the example body

```shell
curl -v 'http://localhost:8000?replace=true' -d @example.body
```

Run the http-echo server, setting the response body to the contents of the example.body file

```shell
http-echo -body=example.body
```
