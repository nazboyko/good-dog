package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// pinned on purpose: flash quota and latency, never a pro model,
// the 2.5 line is retired for new projects, see the trap table
const model = "gemini-3.6-flash"

const requestsPerMinute = 8

// Client is safe for concurrent use, construct one per process so the
// rate limit covers every generate call in the app.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
	limiter *rateLimiter
	backoff []time.Duration
}

func New(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: "https://generativelanguage.googleapis.com",
		http:    &http.Client{Timeout: 30 * time.Second},
		limiter: newRateLimiter(requestsPerMinute, time.Minute),
		backoff: []time.Duration{20 * time.Second, 40 * time.Second, 80 * time.Second},
	}
}

type Options struct {
	Temperature     float64
	MaxOutputTokens int
	ResponseSchema  map[string]any
	// thinking tokens count against maxOutputTokens,
	// extraction turns thinking off so the budget is all answer
	DisableThinking bool
}

// GenerateJSON runs one structured output call and returns the raw JSON
// text. It queues behind the rate limit and retries 429 and 5xx with
// backoff, so callers only retry semantic failures.
func (c *Client) GenerateJSON(ctx context.Context, prompt string, opts Options) ([]byte, error) {
	generationConfig := map[string]any{
		"temperature":      opts.Temperature,
		"maxOutputTokens":  opts.MaxOutputTokens,
		"responseMimeType": "application/json",
		"responseSchema":   opts.ResponseSchema,
	}
	if opts.DisableThinking {
		// the 3.x line replaced thinkingBudget with thinkingLevel,
		// minimal emits zero thought tokens on flash
		generationConfig["thinkingConfig"] = map[string]any{"thinkingLevel": "minimal"}
	}
	payload, err := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": prompt}}},
		},
		"generationConfig": generationConfig,
	})
	if err != nil {
		return nil, err
	}

	for attempt := 0; ; attempt++ {
		if err := c.limiter.wait(ctx); err != nil {
			return nil, err
		}
		raw, status, err := c.post(ctx, payload)
		if err == nil && status == http.StatusOK {
			return candidateText(raw)
		}
		if err == nil && !retryable(status) {
			return nil, fmt.Errorf("gemini status %d: %.200s", status, raw)
		}
		if attempt >= len(c.backoff) {
			if err != nil {
				return nil, fmt.Errorf("gemini gave up after %d attempts: %w", attempt+1, err)
			}
			return nil, fmt.Errorf("gemini gave up after %d attempts, status %d: %.200s", attempt+1, status, raw)
		}
		delay := c.backoff[attempt]
		if err == nil {
			if hinted, ok := parseRetryDelay(raw); ok {
				// honor the server hint, capped so a stuck caller frees up
				delay = min(hinted, 2*time.Minute)
			}
		}
		reason := fmt.Sprintf("status %d", status)
		if err != nil {
			reason = err.Error()
		}
		log.Printf("gemini attempt %d failed (%s), retrying in %s", attempt+1, reason, delay)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func (c *Client) post(ctx context.Context, payload []byte) (body []byte, status int, err error) {
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", c.baseURL, model)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	// key travels in a header, never in the url
	req.Header.Set("x-goog-api-key", c.apiKey)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, res.StatusCode, err
	}
	return raw, res.StatusCode, nil
}

// transport errors arrive as err and are retried like a 5xx
func retryable(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

// parseRetryDelay pulls RetryInfo.retryDelay like "27s" out of a google error body.
func parseRetryDelay(raw []byte) (time.Duration, bool) {
	var body struct {
		Error struct {
			Details []struct {
				Type       string `json:"@type"`
				RetryDelay string `json:"retryDelay"`
			} `json:"details"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &body) != nil {
		return 0, false
	}
	for _, d := range body.Error.Details {
		if strings.HasSuffix(d.Type, "RetryInfo") && d.RetryDelay != "" {
			if dur, err := time.ParseDuration(d.RetryDelay); err == nil && dur > 0 {
				return dur, true
			}
		}
	}
	return 0, false
}

// candidateText pulls the first candidate text out of a generateContent response.
func candidateText(raw []byte) ([]byte, error) {
	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("gemini response not json: %w", err)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini response has no candidate text")
	}
	return []byte(parsed.Candidates[0].Content.Parts[0].Text), nil
}
