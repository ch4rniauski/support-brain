package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ch4rniauski/support-brain/internal/generator/dto"
	"github.com/google/uuid"
)

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

	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, id.String())
	}

	q := url.Values{}
	q.Set("ids", strings.Join(parts, ","))
	endpoint := fmt.Sprintf("%s/qa-pairs?%s", c.baseURL, q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create list qa pairs request: %w", err)
	}

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
