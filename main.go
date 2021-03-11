package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/gonetx/httpit/pit"
	"github.com/spf13/cobra"
)

const version = "0.1.0"

func main() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
}

var config pit.Config

func init() {
	rootCmd.Flags().SortFlags = false
	rootCmd.Flags().IntVarP(&config.Connections, "connections", "c", 128, "Maximum number of concurrent connections")
	rootCmd.Flags().IntVarP(&config.Count, "requests", "n", 0, "Number of requests(if specified, then ignore the --duration)")
	rootCmd.Flags().DurationVarP(&config.Duration, "duration", "d", time.Second*10, "Duration of test")
	rootCmd.Flags().DurationVarP(&config.Timeout, "timeout", "t", time.Second*3, "Socket/request timeout")
	rootCmd.Flags().StringVarP(&config.Method, "method", "X", "GET", "Http request method")
	rootCmd.Flags().StringSliceVarP(&config.Headers, "header", "H", nil, "HTTP request header with format \"K: V\", can be repeated")
	rootCmd.Flags().StringVar(&config.Host, "host", "", "Http request host")
	rootCmd.Flags().BoolVarP(&config.DisableKeepAlives, "disableKeepAlives", "a", false, "Disable HTTP keep-alive, if true, will set header Connection: close")
	rootCmd.Flags().StringVarP(&config.Body, "body", "b", "", "Http request body string")
	rootCmd.Flags().StringVarP(&config.File, "file", "f", "", "Read http request body from file path")
	rootCmd.Flags().BoolVarP(&config.Stream, "stream", "s", false, "Use stream body to reduce memory usage")
	rootCmd.Flags().BoolVarP(&config.JSON, "json", "J", true, "Send json request by setting the Content-Type header to application/json")
	rootCmd.Flags().BoolVarP(&config.Form, "form", "F", false, "Send form request by setting the Content-Type header to application/x-www-form-urlencoded")
	rootCmd.Flags().BoolVarP(&config.MultipartForm, "multipartForm", "M", false, "Send multipart form request by setting the Content-Type header to multipart/form-data")
	rootCmd.Flags().StringSliceVar(&config.MultipartFormFiles, "multipartFormFiles", nil, "Files paths to be attached to multipart form")
	rootCmd.Flags().StringVar(&config.Boundary, "boundary", "", "Multipart form boundary")
	rootCmd.Flags().BoolVarP(&config.Insecure, "insecure", "k", false, "Controls whether a client verifies the server's certificate chain and host name")
	rootCmd.Flags().StringVar(&config.Cert, "cert", "", "Path to the client's TLS Certificate")
	rootCmd.Flags().StringVar(&config.Key, "key", "", "Path to the client's TLS Certificate Private Key")
	rootCmd.Flags().StringVar(&config.HttpProxy, "httpProxy", "", "Http proxy address")
	rootCmd.Flags().StringVar(&config.SocksProxy, "socksProxy", "", "Socks proxy address")
	rootCmd.Flags().BoolVarP(&config.Pipeline, "pipeline", "p", false, "Use fasthttp pipeline client")
	rootCmd.Flags().BoolVar(&config.Follow, "follow", false, "Follow 30x Location redirects")
	rootCmd.Flags().IntVar(&config.MaxRedirects, "maxRedirects", 0, "Max redirect count of following 30x, default is 30 (work with --follow)")
	rootCmd.Flags().StringVarP(&config.Output, "output", "o", "", "Save output to FILE instead of stdout")
	rootCmd.Flags().BoolVarP(&config.Quite, "quiet", "q", false, "Do not print any result to stdout")
	rootCmd.Flags().BoolVarP(&config.Debug, "debug", "D", false, "Send request once and show request and response detail")
}

var rootCmd = &cobra.Command{
	Use:           "httpit url",
	Short:         "httpit is a rapid http benchmark tool",
	Version:       version,
	Args:          rootArgs,
	Run:           rootRun,
	SilenceErrors: true,
}

func rootArgs(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("missing url")
	}
	return nil
}

func rootRun(cmd *cobra.Command, args []string) {
	if err := pit.New(config).Run(args[0], args[1:]...); err != nil {
		cmd.PrintErrln(err)
	}
}
