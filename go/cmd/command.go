package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"ericc-ch/shittycannon/cannon"
	"ericc-ch/shittycannon/report"
	"ericc-ch/shittycannon/runoptions"

	"github.com/spf13/cobra"
)

const (
	defaultConnections = 10
	defaultDuration    = 10
	defaultTimeout     = 10
	defaultMethod      = "GET"
)

func checkPositive(name string, value int) error {
	if value <= 0 {
		return fmt.Errorf("%s: must be greater than 0, got %d", name, value)
	}
	return nil
}

func parseMethod(input string) (runoptions.Method, error) {
	upper := strings.ToUpper(input)
	switch runoptions.Method(upper) {
	case runoptions.MethodGet, runoptions.MethodPost, runoptions.MethodPut, runoptions.MethodDelete, runoptions.MethodPatch, runoptions.MethodHead, runoptions.MethodOptions, runoptions.MethodTrace:
		return runoptions.Method(upper), nil
	default:
		return "", fmt.Errorf("method: unsupported HTTP method %q", input)
	}
}

func parseTargetURL(input string) (*url.URL, error) {
	parsed, err := url.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("url: %v", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("url: must use http or https")
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("url: must include a host")
	}
	return parsed, nil
}

func parseHeaders(pairs []string) (map[string]string, error) {
	headers := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		key, value, ok := strings.Cut(pair, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("headers: %q: expected key=value", pair)
		}
		headers[key] = value
	}
	return headers, nil
}

func newCommand() *cobra.Command {
	var (
		connections int
		duration    int
		amount      int
		method      string
		timeout     int
		body        string
		input       string
		headerPairs []string
		asJSON      bool
		latency     bool
	)

	cmd := &cobra.Command{
		Use:   "shittycannon [url]",
		Short: "HTTP/1 load tester (autocannon subset)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			useAmount := cmd.Flags().Changed("amount")
			if err := checkPositive("connections", connections); err != nil {
				return err
			}
			if err := checkPositive("timeout", timeout); err != nil {
				return err
			}
			if useAmount {
				if err := checkPositive("amount", amount); err != nil {
					return err
				}
			} else if err := checkPositive("duration", duration); err != nil {
				return err
			}
			parsedURL, err := parseTargetURL(args[0])
			if err != nil {
				return err
			}
			parsedMethod, err := parseMethod(method)
			if err != nil {
				return err
			}
			headers, err := parseHeaders(headerPairs)
			if err != nil {
				return err
			}
			bodySet := cmd.Flags().Changed("body")
			inputSet := cmd.Flags().Changed("input")
			if bodySet && inputSet {
				return fmt.Errorf("use either -b/--body or -i/--input, not both")
			}
			var requestBody string
			if bodySet {
				requestBody = body
			}
			if inputSet {
				raw, readErr := os.ReadFile(input)
				if readErr != nil {
					return readErr
				}
				requestBody = string(raw)
			}
			var stop runoptions.Stop = runoptions.Duration{Seconds: duration}
			if useAmount {
				stop = runoptions.Amount{Requests: amount}
			}
			result := cannon.Run(runoptions.RunOptions{
				URL:            parsedURL,
				Connections:    connections,
				Stop:           stop,
				Method:         parsedMethod,
				Headers:        headers,
				Body:           requestBody,
				TimeoutSeconds: timeout,
			})
			if asJSON {
				encoded, encErr := json.Marshal(result)
				if encErr != nil {
					return encErr
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(encoded))
				return nil
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), report.FormatText(result, latency))
			return nil
		},
	}

	cmd.Flags().IntVarP(&connections, "connections", "c", defaultConnections, "The number of concurrent connections to use. default: 10.")
	cmd.Flags().IntVarP(&duration, "duration", "d", defaultDuration, "The number of seconds to run the autocannon. default: 10.")
	cmd.Flags().IntVarP(&amount, "amount", "a", 0, "The number of requests to make before exiting the benchmark. If set, duration is ignored.")
	cmd.Flags().StringVarP(&method, "method", "m", defaultMethod, "The HTTP method to use. default: 'GET'.")
	cmd.Flags().IntVarP(&timeout, "timeout", "t", defaultTimeout, "The number of seconds before timing out and resetting a connection. default: 10")
	cmd.Flags().StringVarP(&body, "body", "b", "", "The body of the request.")
	cmd.Flags().StringVarP(&input, "input", "i", "", "The body of the request. See '-b/body' for more details.")
	cmd.Flags().StringArrayVarP(&headerPairs, "headers", "H", nil, "The request headers.")
	cmd.Flags().BoolVarP(&asJSON, "json", "j", false, "Print the output as newline delimited JSON. This will cause the progress bar and results not to be rendered. default: false.")
	cmd.Flags().BoolVarP(&latency, "latency", "l", false, "Print all the latency data. default: false.")
	return cmd
}
