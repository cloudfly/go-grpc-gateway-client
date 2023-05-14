package gateway

import (
	"context"
	"net/http"
)

type ClientOption func(*Client)

func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = hc
	}
}

func WithMarshaller(m Marshaler) ClientOption {
	return func(c *Client) {
		c.marshaler = m
	}
}

func WithUnmarshaller(m Unmarshaler) ClientOption {
	return func(c *Client) {
		c.unmarshaler = m
	}
}

func SkipTLSVerify(skip bool) ClientOption {
	return func(c *Client) {
		c.skipTLSVerify = skip
	}
}

func WithRewriteRequest(f func(context.Context, *http.Request) (*http.Request, error)) ClientOption {
	return func(c *Client) {
		c.rewriteRequest = f
	}
}
