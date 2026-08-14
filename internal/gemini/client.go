package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const DefaultModel = "gemini-2.5-flash"

type Client struct {
	apiKey  string
	model   string
	baseURL string
	http    *http.Client
}

func New(apiKey, model string) *Client {
	if model == "" {
		model = DefaultModel
	}
	return &Client{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://generativelanguage.googleapis.com",
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

type Options struct {
	Temperature     float64
	MaxOutputTokens int
	ResponseSchema  map[string]any
	// thinking tokens count against maxOutputTokens on 2.5 models,
	// extraction turns thinking off so the budget is all answer
	DisableThinking bool
}

// GenerateJSON runs one structured output call and returns the raw JSON text.
// Callers validate against their own type and decide about retries.
func (c *Client) GenerateJSON(ctx context.Context, prompt string, opts Options) ([]byte, error) {
	generationConfig := map[string]any{
		"temperature":      opts.Temperature,
		"maxOutputTokens":  opts.MaxOutputTokens,
		"responseMimeType": "application/json",
		"responseSchema":   opts.ResponseSchema,
	}
	if opts.DisableThinking {
		generationConfig["thinkingConfig"] = map[string]any{"thinkingBudget": 0}
	}
	body := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": prompt}}},
		},
		"generationConfig": generationConfig,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", c.baseURL, c.model)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	// key travels in a header, never in the url
	req.Header.Set("x-goog-api-key", c.apiKey)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini status %d: %.200s", res.StatusCode, raw)
	}
	return candidateText(raw)
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
