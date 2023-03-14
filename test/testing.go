package test

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/lensesio/lenses-go/v5/pkg/api"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	auth = api.BasicAuthentication{
		Username: "user",
		Password: "pass",
	}
	//ClientConfig mocked for testing
	ClientConfig = api.ClientConfig{
		Authentication: auth,
		Debug:          true,
		Host:           "domain.com",
		Timeout:        "15s",
		Token:          "secret",
	}
)

// CheckStringContains check if string contains the expected value
func CheckStringContains(t *testing.T, got, expected string) {
	if !strings.Contains(got, expected) {
		t.Errorf("Expected to contain: \n %v\nGot:\n %v\n", expected, got)
	}
}

// CheckStringOmits check if string doesn't contain the expected value
func CheckStringOmits(t *testing.T, got, expected string) {
	if strings.Contains(got, expected) {
		t.Errorf("Expected to not contain: \n %v\nGot: %v", expected, got)
	}
}

// EmptyRun an empty run
func EmptyRun(*cobra.Command, []string) {}

// ExecuteCommand execute a command
func ExecuteCommand(root *cobra.Command, args ...string) (output string, err error) {
	_, output, err = executeCommandC(root, args...)
	return output, err
}

// ResetCommandLineFlagSet resets the flagset
func ResetCommandLineFlagSet() {
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
}

// TestingHTTPClient tests an http client
func TestingHTTPClient(handler http.Handler) (*http.Client, func()) {
	s := httptest.NewServer(handler)

	cli := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, s.Listener.Addr().String())
			},
		},
	}

	return cli, s.Close
}

func executeCommandC(root *cobra.Command, args ...string) (c *cobra.Command, output string, err error) {
	buf := new(bytes.Buffer)
	root.SetOutput(buf)
	root.SetArgs(args)

	c, err = root.ExecuteC()

	return c, buf.String(), err
}
