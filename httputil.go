package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const userAgent = "Mozilla/5.0 (compatible; RobloxTracker/2.0)"

func newGetRequest(target string) func(ctx context.Context) (*http.Request, error) {
	return func(ctx context.Context) (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", userAgent)
		return req, nil
	}
}

func newJSONPost(target string, payload any) func(ctx context.Context) (*http.Request, error) {
	return func(ctx context.Context) (*http.Request, error) {
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type httpResult struct {
	Status int
	Body   []byte
}

func fetchWithRetry(ctx context.Context, client *http.Client, newReq func(ctx context.Context) (*http.Request, error), maxRetries int) (*httpResult, error) {
	backoff := 2 * time.Second
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		req, err := newReq(ctx)
		if err != nil {
			return nil, fmt.Errorf("building request: %w", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if sleepErr := sleepCtx(ctx, backoff); sleepErr != nil {
				return nil, sleepErr
			}
			backoff *= 2
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := 10 * time.Second
			if v := resp.Header.Get("Retry-After"); v != "" {
				if secs, perr := time.ParseDuration(v + "s"); perr == nil {
					retryAfter = secs
				}
			}
			resp.Body.Close()
			lastErr = fmt.Errorf("rate limited (429)")
			if sleepErr := sleepCtx(ctx, retryAfter); sleepErr != nil {
				return nil, sleepErr
			}
			continue
		}

		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error %d", resp.StatusCode)
			if sleepErr := sleepCtx(ctx, backoff); sleepErr != nil {
				return nil, sleepErr
			}
			backoff *= 2
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			if sleepErr := sleepCtx(ctx, backoff); sleepErr != nil {
				return nil, sleepErr
			}
			backoff *= 2
			continue
		}

		return &httpResult{Status: resp.StatusCode, Body: body}, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("exhausted retries")
	}
	return nil, lastErr
}
