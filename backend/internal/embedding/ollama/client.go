package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const embedPath = "/api/embed"

type Client struct {
	baseURL    string
	httpClient *http.Client
	model      string
}

func NewClient(baseURL, model string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) Model() string {
	return c.model
}

type embedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embedResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}

func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	body, err := json.Marshal(embedRequest{
		Model: c.model,
		Input: text,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal embed request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+embedPath,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("create embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call ollama embed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read ollama embed response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama embed status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed embedResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshal ollama embed response: %w", err)
	}

	if len(parsed.Embeddings) == 0 || len(parsed.Embeddings[0]) == 0 {
		return nil, fmt.Errorf("ollama embed returned empty embedding")
	}

	src := parsed.Embeddings[0]
	out := make([]float32, len(src))
	for i, value := range src {
		out[i] = float32(value)
	}

	return out, nil
}
