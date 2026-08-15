package elevenlabs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// George from the shared voice library, warm and low, the radio host for now
const DefaultVoiceID = "JBFqnCBsd6RMkjVDRZzb"

type VoiceSettings struct {
	Stability       float64 `json:"stability"`
	SimilarityBoost float64 `json:"similarity_boost"`
}

func (s VoiceSettings) String() string {
	return fmt.Sprintf("stability=%.2f similarity=%.2f", s.Stability, s.SimilarityBoost)
}

type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

func New(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: "https://api.elevenlabs.io",
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

// Synthesize turns text into mp3 bytes with one text to speech call.
func (c *Client) Synthesize(ctx context.Context, text, voiceID string, settings VoiceSettings) ([]byte, error) {
	body, err := json.Marshal(map[string]any{
		"text":           text,
		"model_id":       "eleven_multilingual_v2",
		"voice_settings": settings,
	})
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/v1/text-to-speech/%s?output_format=mp3_44100_128", c.baseURL, voiceID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", c.apiKey)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 20<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("elevenlabs status %d: %.200s", res.StatusCode, data)
	}
	return data, nil
}

// Subscription is the slice of the user subscription endpoint the
// budget report needs: characters used and the plan's cap.
type Subscription struct {
	Tier           string `json:"tier"`
	CharacterCount int    `json:"character_count"`
	CharacterLimit int    `json:"character_limit"`
	NextResetUnix  int64  `json:"next_character_count_reset_unix"`
}

// Remaining is the credits left in the current period.
func (s Subscription) Remaining() int { return s.CharacterLimit - s.CharacterCount }

// GetSubscription reads the account's character usage and cap.
func (c *Client) GetSubscription(ctx context.Context) (*Subscription, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/v1/user/subscription", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("xi-api-key", c.apiKey)
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
		return nil, fmt.Errorf("subscription status %d: %.200s", res.StatusCode, raw)
	}
	var sub Subscription
	if err := json.Unmarshal(raw, &sub); err != nil {
		return nil, fmt.Errorf("subscription response not json: %w", err)
	}
	return &sub, nil
}
