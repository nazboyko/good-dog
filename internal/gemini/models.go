package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// listModels returns every model name visible to this key.
func (c *Client) listModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/v1beta/models?pageSize=1000", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-goog-api-key", c.apiKey)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list models status %d: %.200s", res.StatusCode, raw)
	}
	var parsed struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("list models response not json: %w", err)
	}
	names := make([]string, 0, len(parsed.Models))
	for _, m := range parsed.Models {
		names = append(names, m.Name)
	}
	return names, nil
}

// PreflightModel checks once at startup that the pinned model is visible
// to this key. Detect only: it logs and returns, it never switches models.
func (c *Client) PreflightModel(ctx context.Context) {
	names, err := c.listModels(ctx)
	if err != nil {
		log.Printf("gemini preflight skipped: %v", err)
		return
	}
	if warning, missing := modelPinWarning(names, model); missing {
		log.Print(warning)
		return
	}
	log.Printf("gemini preflight ok, %s is available", model)
}

// modelPinWarning names the usable flash models when the pin is missing.
func modelPinWarning(models []string, pinned string) (string, bool) {
	var flash []string
	found := false
	for _, m := range models {
		name := strings.TrimPrefix(m, "models/")
		if name == pinned {
			found = true
		}
		if strings.Contains(name, "flash") {
			flash = append(flash, name)
		}
	}
	if found {
		return "", false
	}
	return fmt.Sprintf(
		"gemini preflight: pinned model %s is not available to this key, update the pin by hand, available flash models: %s",
		pinned, strings.Join(flash, ", "),
	), true
}
