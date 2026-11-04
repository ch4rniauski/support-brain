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

	"github.com/ch4rniauski/support-brain/internal/generator/dto"
	"github.com/google/uuid"
)

const qaPairsPath = "/qa-pairs"

type IntakeClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewIntakeClient(baseURL string) *IntakeClient {
	return &IntakeClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *IntakeClient) ListQAPairs(ctx context.Context, ids []uuid.UUID) (*dto.ListQAPairsResponse, error) {
	if len(ids) == 0 {
		return &dto.ListQAPairsResponse{Items: []dto.IntakeQAPairItem{}}, nil
	}

	payload, err := json.Marshal(dto.ListQAPairsRequest{IDs: ids})
	if err != nil {
		return nil, fmt.Errorf("marshal list qa pairs request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+qaPairsPath,
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("create list qa pairs request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call intake list qa pairs: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read intake response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("intake list qa pairs status %d: %s", resp.StatusCode, string(body))
	}

	var result dto.ListQAPairsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal intake qa pairs response: %w", err)
	}

	return &result, nil
}
