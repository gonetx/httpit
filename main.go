package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gonetx/httpit/pit"
	"github.com/spf13/cobra"
)

const version = "0.0.1"

func main() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
}

var config pit.Config

func init() {
	rootCmd.PersistentFlags().IntVarP(&config.Connections, "connections", "c", 128, "Maximum number of concurrent connections")
	rootCmd.PersistentFlags().IntVarP(&config.Count, "requests", "n", 0, "Number of requests")
	rootCmd.PersistentFlags().DurationVarP(&config.Duration, "duration", "d", time.Second*10, "Duration of test")
	rootCmd.PersistentFlags().DurationVarP(&config.Timeout, "timeout", "t", time.Second*3, "Socket/request timeout")
	rootCmd.PersistentFlags().StringVarP(&config.Method, "method", "X", http.MethodGet, "Http request method")
	rootCmd.PersistentFlags().StringSliceVarP(&config.Headers, "header", "H", nil, "HTTP request header with format \"K: V\", can be repeated")
	rootCmd.PersistentFlags().StringVar(&config.Host, "host", "", "Http request host")
	rootCmd.PersistentFlags().StringVarP(&config.Body, "body", "b", "", "Http request body")
	rootCmd.PersistentFlags().StringVarP(&config.File, "file", "f", "", "Read http request body from file path")
	rootCmd.PersistentFlags().BoolVarP(&config.Stream, "stream", "s", false, "Use stream body")
	rootCmd.PersistentFlags().StringVar(&config.Cert, "cert", "", "Path to the client's TLS Certificate")
	rootCmd.PersistentFlags().StringVar(&config.Key, "key", "", "Path to the client's TLS Certificate Private Key")
	rootCmd.PersistentFlags().BoolVarP(&config.DisableKeepAlives, "disableKeepAlives", "a", false, "Disable HTTP keep-alive, if true, will set header Connection: close")
	rootCmd.PersistentFlags().BoolVarP(&config.Insecure, "insecure", "k", false, "Controls whether a client verifies the server's certificate chain and host name")
	rootCmd.PersistentFlags().BoolVar(&config.Http1, "http1", false, "Use net/http client with HTTP/1.x")
	rootCmd.PersistentFlags().BoolVar(&config.Http2, "http2", false, "Use net/http client with HTTP/2.0")
}

var rootCmd = &cobra.Command{
	Use:     "httpit",
	Short:   "httpit is a rapid http benchmark tool",
	Version: version,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.PrintErrln("missing url")
			return
		}
		if err := pit.New(config).Run(args[0]); err != nil {
			cmd.PrintErrln(err)
		}
	},
}
