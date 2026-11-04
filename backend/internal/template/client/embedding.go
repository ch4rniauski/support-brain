package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ch4rniauski/support-brain/internal/template/dto"
	"github.com/ch4rniauski/support-brain/pkg/auth"
)

const similarByTextPath = "/similar/by-text"

type EmbeddingClient struct {
	baseURL    string
	httpClient *http.Client
	tokens     *auth.TokenManager
}

func NewEmbeddingClient(baseURL string, tokens *auth.TokenManager) *EmbeddingClient {
	return &EmbeddingClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		tokens:  tokens,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *EmbeddingClient) FindSimilarByText(
	ctx context.Context,
	text string,
	threshold float64,
	limit int,
) (*dto.FindSimilarByTextResponse, error) {
	payload, err := json.Marshal(dto.FindSimilarByTextRequest{
		Text:      text,
		Threshold: threshold,
		Limit:     limit,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal find similar by text request: %w", err)
	}

	token, err := c.tokens.Issue(auth.RoleOperator)
	if err != nil {
		return nil, fmt.Errorf("issue embedding auth token: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+similarByTextPath,
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("create find similar by text request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call embedding find similar by text: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read embedding response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"embedding find similar by text status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var result dto.FindSimilarByTextResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal find similar by text response: %w", err)
	}

	return &result, nil
}
