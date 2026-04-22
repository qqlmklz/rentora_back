package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"
)

const defaultOpenAIModel = "gpt-4o-mini"

// OpenAIPriorityAnalyzer - реальный анализатор через OpenAI Responses API.
type OpenAIPriorityAnalyzer struct {
	client openai.Client
	model  string
}

func NewOpenAIPriorityAnalyzerFromEnv() (*OpenAIPriorityAnalyzer, error) {
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		return nil, ErrAIUnavailable
	}
	model := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	if model == "" {
		model = defaultOpenAIModel
	}
	httpClient := &http.Client{Timeout: 25 * time.Second}
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithHTTPClient(httpClient),
	)
	return &OpenAIPriorityAnalyzer{
		client: client,
		model:  model,
	}, nil
}

func (a *OpenAIPriorityAnalyzer) Analyze(ctx context.Context, title, description, category string) (PriorityResult, string, error) {
	if strings.TrimSpace(title) == "" && strings.TrimSpace(description) == "" && strings.TrimSpace(category) == "" {
		return PriorityResult{}, "", ErrAIUnavailable
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	userPrompt := "Проанализируй заявку на неисправность:\n" +
		"title: " + strings.TrimSpace(title) + "\n" +
		"description: " + strings.TrimSpace(description) + "\n" +
		"category: " + strings.TrimSpace(category)

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"priority": map[string]any{
				"type": "string",
				"enum": []string{"low", "medium", "high"},
			},
			"priority_score": map[string]any{
				"type":    "number",
				"minimum": 0,
				"maximum": 1,
			},
			"priority_reason": map[string]any{
				"type": "string",
			},
		},
		"required":             []string{"priority", "priority_score", "priority_reason"},
		"additionalProperties": false,
	}

	resp, err := a.client.Responses.New(ctx, responses.ResponseNewParams{
		Model:       openai.ResponsesModel(a.model),
		Temperature: openai.Float(0.1),
		Instructions: openai.String(
			"Ты анализируешь заявки на неисправности в квартире. " +
				"Верни только JSON по схеме: priority (low|medium|high), priority_score (0..1), priority_reason (короткая причина).",
		),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam{
				responses.ResponseInputItemParamOfMessage(userPrompt, responses.EasyInputMessageRoleUser),
			},
		},
		Text: responses.ResponseTextConfigParam{
			Format: responses.ResponseFormatTextConfigParamOfJSONSchema("priority_analysis", schema),
		},
	})
	if err != nil {
		return PriorityResult{}, "", fmt.Errorf("openai responses create: %w", err)
	}

	rawResponse := strings.TrimSpace(resp.OutputText())
	if rawResponse == "" {
		rawResponse = strings.TrimSpace(resp.RawJSON())
	}
	log.Printf("[AI priority] openai raw_response=%q", rawResponse)

	if resp.Status != responses.ResponseStatusCompleted && strings.TrimSpace(resp.OutputText()) == "" {
		return PriorityResult{}, rawResponse, fmt.Errorf("openai response status=%s", resp.Status)
	}

	var out PriorityResult
	if err := json.Unmarshal([]byte(resp.OutputText()), &out); err != nil {
		return PriorityResult{}, rawResponse, fmt.Errorf("parse structured ai json: %w", err)
	}
	if !isValidPriority(out.Priority) {
		return PriorityResult{}, rawResponse, errors.New("ai returned invalid priority")
	}
	out.Priority = strings.ToLower(strings.TrimSpace(out.Priority))
	if out.PriorityScore < 0 || out.PriorityScore > 1 {
		out.PriorityScore = 0
	}
	out.PriorityReason = strings.TrimSpace(out.PriorityReason)
	if out.PriorityReason == "" {
		out.PriorityReason = "Причина не указана AI"
	}
	return out, rawResponse, nil
}

func isValidPriority(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case priorityLow, priorityMedium, priorityHigh:
		return true
	default:
		return false
	}
}
