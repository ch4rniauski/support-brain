package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode"

	"github.com/ch4rniauski/support-brain/internal/generator/domain"
	"github.com/ch4rniauski/support-brain/internal/generator/dto"
	"github.com/ch4rniauski/support-brain/pkg/kafka"
	"github.com/ch4rniauski/support-brain/pkg/outbox"
	"github.com/google/uuid"
)

const systemPrompt = `You are helping a support team. Given several operator answers to similar customer questions, write ONE clear draft reply in the same language as the examples.
Do not invent policies, prices, personal data, or ticket numbers.
Do not include greetings/sign-offs unless most examples have them.
Return only the reply text.`

type GeneratorConfig struct {
	ModelName      string
	PromptVersion  string
	MaxAttempts    int
	MaxOutputChars int
	MinOutputChars int
}

type llmClient interface {
	Model() string
	GenerateReply(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

type intakeClient interface {
	ListQAPairs(ctx context.Context, ids []uuid.UUID) (*dto.ListQAPairsResponse, error)
}

type jobStore interface {
	FindActiveByClusterID(ctx context.Context, clusterID uuid.UUID) (*domain.GenerationJob, error)
	HasSucceededByClusterID(ctx context.Context, clusterID uuid.UUID) (bool, error)
	Create(ctx context.Context, job *domain.GenerationJob) error
	SaveExamples(ctx context.Context, examples []domain.GenerationJobExample) error
	MarkRunning(ctx context.Context, jobID uuid.UUID, startedAt time.Time) error
	MarkPendingWithError(ctx context.Context, jobID uuid.UUID, errorMessage string) error
	MarkFailed(ctx context.Context, jobID uuid.UUID, errorMessage string, finishedAt time.Time) error
	MarkSucceededWithOutbox(
		ctx context.Context,
		jobID uuid.UUID,
		outputText string,
		finishedAt time.Time,
		event *outbox.Event,
	) error
}

type GeneratorService struct {
	jobs   jobStore
	intake intakeClient
	llm    llmClient
	cfg    GeneratorConfig
}

func NewGeneratorService(
	jobs jobStore,
	intake intakeClient,
	llm llmClient,
	cfg GeneratorConfig,
) *GeneratorService {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.MaxOutputChars <= 0 {
		cfg.MaxOutputChars = 4000
	}
	if cfg.MinOutputChars <= 0 {
		cfg.MinOutputChars = 20
	}
	if cfg.PromptVersion == "" {
		cfg.PromptVersion = "v1"
	}
	if cfg.ModelName == "" {
		cfg.ModelName = llm.Model()
	}

	return &GeneratorService{
		jobs:   jobs,
		intake: intake,
		llm:    llm,
		cfg:    cfg,
	}
}

func permanent(err error) error {
	return fmt.Errorf("%w: %w", kafka.ErrPermanent, err)
}

func (s *GeneratorService) Handle(ctx context.Context, eventType string, payload []byte) error {
	if eventType != "" && eventType != dto.EventTypeClusterReadyForGeneration {
		log.Printf("generator: skip unsupported event type %q", eventType)
		return nil
	}

	var event dto.ClusterReadyForGenerationEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return permanent(fmt.Errorf("unmarshal cluster ready for generation event: %w", err))
	}

	if event.ClusterID == uuid.Nil {
		return permanent(fmt.Errorf("cluster_id is required"))
	}
	if len(event.QAPairIDs) == 0 {
		return permanent(fmt.Errorf("qa_pair_ids are required"))
	}

	active, err := s.jobs.FindActiveByClusterID(ctx, event.ClusterID)
	if err != nil {
		return err
	}
	if active != nil {
		log.Printf("generator: skip cluster %s — active job %s (%s)", event.ClusterID, active.ID, active.Status)
		return nil
	}

	succeeded, err := s.jobs.HasSucceededByClusterID(ctx, event.ClusterID)
	if err != nil {
		return err
	}
	if succeeded {
		log.Printf("generator: skip cluster %s — succeeded job already exists", event.ClusterID)
		return nil
	}

	job := &domain.GenerationJob{
		ID:            uuid.New(),
		ClusterID:     event.ClusterID,
		Status:        domain.GenerationJobStatusPending,
		ModelName:     s.cfg.ModelName,
		PromptVersion: s.cfg.PromptVersion,
		Attempts:      0,
	}
	if err := s.jobs.Create(ctx, job); err != nil {
		return err
	}

	examples, err := s.loadAndPersistExamples(ctx, job.ID, event.QAPairIDs)
	if err != nil {
		_ = s.jobs.MarkFailed(ctx, job.ID, err.Error(), time.Now().UTC())
		if isRetryableLoadError(err) {
			return err
		}
		return permanent(err)
	}

	userPrompt := buildUserPrompt(examples)

	for {
		if err := s.jobs.MarkRunning(ctx, job.ID, time.Now().UTC()); err != nil {
			return err
		}
		job.Attempts++

		output, err := s.llm.GenerateReply(ctx, systemPrompt, userPrompt)
		if err != nil {
			retry, markErr := s.failOrRetry(ctx, job, fmt.Errorf("generate reply: %w", err))
			if markErr != nil {
				return markErr
			}
			if !retry {
				return nil
			}
			continue
		}

		validated, err := validateOutput(output, s.cfg.MinOutputChars, s.cfg.MaxOutputChars)
		if err != nil {
			retry, markErr := s.failOrRetry(ctx, job, err)
			if markErr != nil {
				return markErr
			}
			if !retry {
				return nil
			}
			continue
		}

		outboxPayload, err := json.Marshal(dto.TemplateDraftCreatedEvent{
			ClusterID:       event.ClusterID,
			Text:            validated,
			ModelName:       s.cfg.ModelName,
			PromptVersion:   s.cfg.PromptVersion,
			GenerationJobID: job.ID,
		})
		if err != nil {
			return fmt.Errorf("marshal template draft created event: %w", err)
		}

		now := time.Now().UTC()
		outboxEvent := &outbox.Event{
			ID:          uuid.New(),
			EventType:   domain.EventTypeTemplateDraftCreated,
			AggregateID: event.ClusterID,
			Payload:     outboxPayload,
			CreatedAt:   now,
		}

		if err := s.jobs.MarkSucceededWithOutbox(ctx, job.ID, validated, now, outboxEvent); err != nil {
			return err
		}

		return nil
	}
}

func (s *GeneratorService) failOrRetry(
	ctx context.Context,
	job *domain.GenerationJob,
	cause error,
) (bool, error) {
	if job.Attempts >= s.cfg.MaxAttempts {
		msg := fmt.Sprintf("generation failed after %d attempts: %s", job.Attempts, cause.Error())
		if err := s.jobs.MarkFailed(ctx, job.ID, msg, time.Now().UTC()); err != nil {
			return false, err
		}
		log.Printf("generator: job %s exhausted attempts: %v", job.ID, cause)
		return false, nil
	}

	if err := s.jobs.MarkPendingWithError(ctx, job.ID, cause.Error()); err != nil {
		return false, err
	}

	return true, nil
}

func (s *GeneratorService) loadAndPersistExamples(
	ctx context.Context,
	jobID uuid.UUID,
	qaPairIDs []uuid.UUID,
) ([]domain.GenerationJobExample, error) {
	resp, err := s.intake.ListQAPairs(ctx, qaPairIDs)
	if err != nil {
		return nil, fmt.Errorf("fetch qa pairs from intake: %w", err)
	}

	byID := make(map[uuid.UUID]dto.IntakeQAPairItem, len(resp.Items))
	for _, item := range resp.Items {
		byID[item.ID] = item
	}

	examples := make([]domain.GenerationJobExample, 0, len(qaPairIDs))
	for _, id := range qaPairIDs {
		item, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("qa pair %s not found in intake", id)
		}
		question := strings.TrimSpace(item.Question)
		answer := strings.TrimSpace(item.Answer)
		if question == "" || answer == "" {
			return nil, fmt.Errorf("qa pair %s has empty question or answer", id)
		}

		examples = append(examples, domain.GenerationJobExample{
			ID:       uuid.New(),
			JobID:    jobID,
			QAPairID: id,
			Question: question,
			Answer:   answer,
		})
	}

	if err := s.jobs.SaveExamples(ctx, examples); err != nil {
		return nil, err
	}

	return examples, nil
}

func buildUserPrompt(examples []domain.GenerationJobExample) string {
	var b strings.Builder
	b.WriteString("Examples:\n")
	for i, example := range examples {
		fmt.Fprintf(&b, "%d.\nQ: %s\nA: %s\n", i+1, example.Question, example.Answer)
	}
	return b.String()
}

func validateOutput(text string, minChars, maxChars int) (string, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", fmt.Errorf("llm output is empty")
	}
	if len(trimmed) < minChars {
		return "", fmt.Errorf("llm output too short: %d < %d", len(trimmed), minChars)
	}
	if len(trimmed) > maxChars {
		return "", fmt.Errorf("llm output too long: %d > %d", len(trimmed), maxChars)
	}
	if isObviousGarbage(trimmed) {
		return "", fmt.Errorf("llm output looks like garbage")
	}
	return trimmed, nil
}

func isObviousGarbage(text string) bool {
	runes := []rune(text)
	if len(runes) == 0 {
		return true
	}

	first := runes[0]
	same := true
	letterOrDigit := false
	for _, r := range runes {
		if r != first {
			same = false
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			letterOrDigit = true
		}
	}
	return same || !letterOrDigit
}

func isRetryableLoadError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "call intake") ||
		strings.Contains(msg, "status 5") ||
		strings.Contains(msg, "timeout")
}
