package ai

import (
	"context"
	"errors"
)

const (
	priorityLow    = "low"
	priorityMedium = "medium"
	priorityHigh   = "high"
)

// PriorityInput — данные заявки для анализа приоритета.
type PriorityInput struct {
	Title       string
	Description string
	Category    string
}

// PriorityResult — результат анализа приоритета от AI.
type PriorityResult struct {
	Priority       string  `json:"priority"`
	PriorityScore  float64 `json:"priority_score"`
	PriorityReason string  `json:"priority_reason"`
}

// PriorityAnalyzer — интерфейс анализатора приоритета заявок.
type PriorityAnalyzer interface {
	Analyze(ctx context.Context, title, description, category string) (PriorityResult, string, error)
}

var ErrAIUnavailable = errors.New("ai unavailable")

// DefaultPriorityResult — значения по умолчанию, когда AI недоступен.
func DefaultPriorityResult() PriorityResult {
	return PriorityResult{
		Priority:       priorityMedium,
		PriorityScore:  0,
		PriorityReason: "Приоритет определён по умолчанию",
	}
}
