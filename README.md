# gonetx/httpit
<p align="center">
  <a href="https://github.com/gonetx/httpit/actions?query=workflow%3ASecurity">
    <img src="https://img.shields.io/github/workflow/status/gofiber/fiber/Security?label=%F0%9F%94%91%20gosec&style=flat&color=75C46B">
  </a>
  <a href="https://github.com/gonetx/httpit/actions?query=workflow%3ATest">
    <img src="https://img.shields.io/github/workflow/status/gofiber/fiber/Test?label=%F0%9F%A7%AA%20tests&style=flat&color=75C46B">
  </a>
  <a href="https://codecov.io/gh/gonetx/httpit">
    <img src="https://codecov.io/gh/gonetx/httpit/branch/main/graph/badge.svg?token=RABB5SC45Y"/>
  </a>

</p>

`httpit` is a rapid http(s) benchmark tool which on top of [fasthttp](https://github.com/valyala/fasthttp). Also thanks to [cobra](https://github.com/spf13/cobra) and [bubbletea](https://github.com/charmbracelet/bubbletea).

## Installation
Get binaries from [releases](https://github.com/gonetx/httpit/releases) or just run `go get -u github.com/gonetx/httpit`.

## Usage
```bash
Usage:
  httpit url [flags]

Flags:
  -b, --body string         Http request body
      --cert string         Path to the client's TLS Certificate
  -c, --connections int     Maximum number of concurrent connections (default 128)
  -a, --disableKeepAlives   Disable HTTP keep-alive, if true, will set header Connection: close
  -d, --duration duration   Duration of test (default 10s)
  -f, --file string         Read http request body from file path
  -H, --header strings      HTTP request header with format "K: V", can be repeated
  -h, --help                help for httpit
      --host string         Http request host
      --httpProxy string    Http proxy address
  -k, --insecure            Controls whether a client verifies the server's certificate chain and host name
      --key string          Path to the client's TLS Certificate Private Key
  -X, --method string       Http request method (default "GET")
  -p, --pipeline            Use fasthttp pipeline client
  -n, --requests int        Number of requests
      --socksProxy string   Socks proxy address
  -s, --stream              Use stream body to reduce memory usage
  -t, --timeout duration    Socket/request timeout (default 3s)
  -v, --version             version for httpit
```

### Override host
Use `--host` to override `Host` header for the use case like `curl "http://127.0.0.1" -H "Host: www.example.com"` to bypass DNS resolving.

### Proxy
Use `--httpProxy` and `--socksProxy` to specific proxies for some rare cases.

### Pipeline
Use `-p|--pipeline` to specific fasthttp pipeline client.

## Examples
### Use duration
`httpit -X GET "http://httpbin.org/get" -H "accept: application/json" -c2 -d3s`

![duration](capture/duration.gif)

### Use count
`httpit -X GET "http://httpbin.org/get" -H "accept: application/json" -c2 -n15`

![count](capture/count.gif)
