package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Marshaler func(ctx context.Context, method string, path string, data any) ([]byte, error)
type Unmarshaler func(ctx context.Context, contentType string, code int, body io.Reader, dst any) error

func defaultMarshaler(ctx context.Context, method string, path string, data any) ([]byte, error) {
	return json.Marshal(data)
}
func defaultUnmarshaler(ctx context.Context, contentType string, code int, body io.Reader, dst any) error {
	return json.NewDecoder(body).Decode(dst)
}

type Client struct {
	ctx            context.Context
	httpClient     *http.Client
	skipTLSVerify  bool
	baseUrl        string
	rewriteRequest func(context.Context, *http.Request) (*http.Request, error)
	marshaler      Marshaler
	unmarshaler    Unmarshaler
}

func NewClient(ctx context.Context, baseUrl string, opts ...ClientOption) *Client {
	c := &Client{
		ctx:           ctx,
		baseUrl:       baseUrl,
		marshaler:     defaultMarshaler,
		unmarshaler:   defaultUnmarshaler,
		httpClient:    &http.Client{},
		skipTLSVerify: true,
	}
	for _, opt := range opts {
		opt(c)
	}

	if c.httpClient.Transport == nil {
		c.httpClient.Transport = http.DefaultTransport.(*http.Transport).Clone()
	}
	setSkipTLSVerify(c.httpClient, c.skipTLSVerify)
	return c
}

func (c *Client) Context() context.Context {
	return c.ctx
}

func (c *Client) Marshal(ctx context.Context, method string, path string, data any) ([]byte, error) {
	return c.marshaler(ctx, method, path, data)
}

func (c *Client) SetPathParam(ctx context.Context, r *http.Request, key, value string) *http.Request {
	r.URL.Path = strings.ReplaceAll(r.URL.Path, "{"+key+"}", url.PathEscape(value))
	return r
}

func (c *Client) NewRequest(ctx context.Context, method, path string) (*http.Request, error) {
	req, err := http.NewRequest(method, c.baseUrl+path, nil)
	if err != nil {
		return nil, fmt.Errorf("new request error: %w", err)
	}
	return req, nil
}
