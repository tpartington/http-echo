# HTTP Echo

A small utility for echoing http requests and returning a customised response

## Usage

```shell
Usage of http-echo:
  -address string
    the TCP address to listen on (default "127.0.0.1")
  -colour
    show coloured output (default true)
  -debug
    show debug ouput
  -delay int
    the time to wait (in milliseconds) before sending a response
  -help
    show this help menu
  -jitter int
    the maximum amount of jitter (in milliseconds) to add to the response
  -port int
    the TCP port to listen on (default 8000)
  -printBody
    print the HTTP request body (default true)
  -timestamp
    show the request/response timestamp (default true)
```

## Query Parameters

The following query parameters are accepted:

```shell
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
curl --location 'http://localhost:8000/?code=random&codes=200,200,200,502'
```

Use curl to connect to http-echo, http-echo will return a 200 with several custom headers set

```shell
curl 'http://localhost:8000/?headers=cache-control,private,content-length,500'
```