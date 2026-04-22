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

// PriorityInput - данные заявки, которые отправляем на анализ приоритета.
type PriorityInput struct {
	Title       string
	Description string
	Category    string
}

// PriorityResult - что вернул AI по приоритету.
type PriorityResult struct {
	Priority       string  `json:"priority"`
	PriorityScore  float64 `json:"priority_score"`
	PriorityReason string  `json:"priority_reason"`
}

// PriorityAnalyzer - общий интерфейс для любого анализатора приоритета.
type PriorityAnalyzer interface {
	Analyze(ctx context.Context, title, description, category string) (PriorityResult, string, error)
}

var ErrAIUnavailable = errors.New("ai unavailable")

// DefaultPriorityResult - fallback, когда AI недоступен.
func DefaultPriorityResult() PriorityResult {
	return PriorityResult{
		Priority:       priorityMedium,
		PriorityScore:  0,
		PriorityReason: "Приоритет определён по умолчанию",
	}
}
