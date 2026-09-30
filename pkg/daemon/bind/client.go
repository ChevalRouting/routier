package bind

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	Key          string
	Addr         string
	Port         int
	Resolver     string
	ResolverPort int

	TSIGName      string
	TSIGAlgorithm string
	TSIGSecret    string
}

func New(key string) *Client {
	if key == "" {
		key = RndcKey
	}

	return &Client{Key: key, Addr: ControlAddr, Port: ControlPort, Resolver: DefaultResolver}
}

const commandTimeout = 4 * time.Second

var runner = defaultRunner

func defaultRunner(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

func SetRunner(fn func(name string, args ...string) ([]byte, error)) func() {
	prev := runner
	runner = fn

	return func() { runner = prev }
}

func (c *Client) controlAddr() string {
	if c.Addr == "" {
		return ControlAddr
	}

	return c.Addr
}

func (c *Client) controlPort() int {
	if c.Port <= 0 {
		return ControlPort
	}

	return c.Port
}

func (c *Client) keyFile() string {
	if c.Key == "" {
		return RndcKey
	}

	return c.Key
}

func (c *Client) resolverAddr() string {
	if c.Resolver == "" {
		return DefaultResolver
	}

	return c.Resolver
}

func (c *Client) rndc(args ...string) (string, error) {
	argv := append([]string{
		"-s", c.controlAddr(),
		"-p", strconv.Itoa(c.controlPort()),
		"-k", c.keyFile(),
	}, args...)

	out, err := runner("rndc", argv...)
	text := strings.TrimSpace(string(out))

	if err != nil {
		if text != "" {
			return "", fmt.Errorf("rndc %s: %w: %s", strings.Join(args, " "), err, text)
		}

		return "", fmt.Errorf("rndc %s: %w", strings.Join(args, " "), err)
	}

	return text, nil
}

func atoi(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}

	return n
}

func atof(s string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}

	return f
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}

	return s
}
