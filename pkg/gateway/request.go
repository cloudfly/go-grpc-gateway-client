package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/alevinval/sse/pkg/decoder"
	rpcstatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/status"
)

type streamingResponse map[string]json.RawMessage

const (
	streamingResponseResultKey = "result"
	streamingResponseErrorKey  = "error"
)

func DoStreamingRequest[T any](ctx context.Context, c *Client, req *http.Request) (<-chan *T, <-chan error, error) {
	req = req.WithContext(ctx)
	if c.rewriteRequest != nil {
		var err error
		req, err = c.rewriteRequest(ctx, req)
		if err != nil {
			return nil, nil, fmt.Errorf("rewrite request error: %w", err)
		}
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("do request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 399 {
		body := resp.Body
		defer func() { _ = body.Close() }()
		data, err := io.ReadAll(body)
		if err != nil {
			return nil, nil, fmt.Errorf("read error response body: %w", err)
		}

		var res streamingResponse
		if err := json.Unmarshal(data, &res); err != nil {
			return nil, nil, fmt.Errorf("unmarshal raw response: %w", err)
		}
		rawErrRes, ok := res[streamingResponseErrorKey]
		if !ok {
			return nil, nil, errors.New(string(data))
		}
		var errRes rpcstatus.Status
		if err := c.unmarshaler(ctx, resp.Header.Get("Content-Type"), resp.StatusCode, bytes.NewBuffer([]byte(rawErrRes)), &errRes); err != nil {
			return nil, nil, fmt.Errorf("unmarshal error response: %w", err)
		}
		if err := status.ErrorProto(&errRes); err != nil {
			return nil, nil, err
		}
		return nil, nil, status.Error(HTTPStatusToCode(resp.StatusCode), errRes.String())
	}

	resCh := make(chan *T)
	errCh := make(chan error)

	go func() {
		body := resp.Body
		defer func() { _ = body.Close() }()
		eventDecoder := decoder.New(body)
		for {
			if ctx.Err() != nil {
				return
			}
			event, err := eventDecoder.Decode()
			if err != nil {
				if errors.Is(err, io.EOF) {
					close(resCh)
					return
				}
				errCh <- err
				return
			}

			var res streamingResponse
			if err := json.Unmarshal([]byte(event.GetData()), &res); err != nil {
				errCh <- fmt.Errorf("unmarshal streaming response: %w", err)
				return
			}
			rawResult, ok := res[streamingResponseResultKey]
			if !ok {
				continue
			}

			var data T
			if err := c.unmarshaler(ctx, resp.Header.Get("Content-Type"), resp.StatusCode, bytes.NewBuffer([]byte(rawResult)), &data); err != nil {
				errCh <- err
				return
			}
			resCh <- &data
		}
	}()
	return resCh, errCh, nil
}

func DoHTTPRequest[T any](ctx context.Context, c *Client, req *http.Request) (*T, error) {
	req = req.WithContext(ctx)
	if c.rewriteRequest != nil {
		var err error
		req, err = c.rewriteRequest(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("rewrite request error: %w", err)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 399 {
		var d rpcstatus.Status
		if err := c.unmarshaler(ctx, resp.Header.Get("Content-Type"), resp.StatusCode, resp.Body, &d); err != nil {
			return nil, fmt.Errorf("unmarshal fail response error: %w", err)
		}
		if err := status.ErrorProto(&d); err != nil {
			return nil, err
		}
		return nil, status.Error(HTTPStatusToCode(resp.StatusCode), d.String())
	}

	var bodyDst T

	if _, ok := any(bodyDst).([]byte); ok {
		content, err := io.ReadAll(resp.Body)
		return any(content).(*T), err
	}

	if err := c.unmarshaler(ctx, resp.Header.Get("Content-Type"), resp.StatusCode, resp.Body, &bodyDst); err != nil {
		return nil, fmt.Errorf("unmarshal response error: %w", err)
	}
	return &bodyDst, nil
}
