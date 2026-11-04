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
	"github.com/google/uuid"
)

const templatesByQAPairsPath = "/clusters/templates-by-qa-pairs"

type KnowledgeClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewKnowledgeClient(baseURL string) *KnowledgeClient {
	return &KnowledgeClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *KnowledgeClient) FindTemplatesByQAPairs(
	ctx context.Context,
	qaPairIDs []uuid.UUID,
) (*dto.FindTemplatesByQAPairsResponse, error) {
	payload, err := json.Marshal(dto.FindTemplatesByQAPairsRequest{
		QAPairIDs: qaPairIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal find templates by qa pairs request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+templatesByQAPairsPath,
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("create find templates by qa pairs request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call knowledge find templates by qa pairs: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read knowledge response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"knowledge find templates by qa pairs status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var result dto.FindTemplatesByQAPairsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal find templates by qa pairs response: %w", err)
	}

	return &result, nil
}
