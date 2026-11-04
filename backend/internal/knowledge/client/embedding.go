package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ch4rniauski/support-brain/internal/knowledge/dto"
	"github.com/google/uuid"
)

const similarPath = "/similar"

type EmbeddingClient struct {
	baseURL    string
	modelName  string
	httpClient *http.Client
}

func NewEmbeddingClient(baseURL, modelName string) *EmbeddingClient {
	return &EmbeddingClient{
		baseURL:   baseURL,
		modelName: modelName,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *EmbeddingClient) FindSimilar(
	ctx context.Context,
	qaPairID uuid.UUID,
	threshold float64,
	limit int,
) (*dto.FindSimilarResponse, error) {
	reqBody := dto.FindSimilarRequest{
		QAPairID:  qaPairID,
		ModelName: c.modelName,
		Threshold: threshold,
		Limit:     limit,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal find similar request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+similarPath,
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("create find similar request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call embedding find similar: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read embedding response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"embedding find similar status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var result dto.FindSimilarResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal find similar response: %w", err)
	}

	return &result, nil
}
