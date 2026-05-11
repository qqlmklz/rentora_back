package ai

import (
	"context"
	"encoding/json"
	"log"
	"strings"
)

// MockPriorityAnalyzer - локальный анализатор для dev/testing.
type MockPriorityAnalyzer struct{}

func NewMockPriorityAnalyzer() *MockPriorityAnalyzer {
	return &MockPriorityAnalyzer{}
}

func (a *MockPriorityAnalyzer) Analyze(ctx context.Context, title, description, category string) (PriorityResult, string, error) {
	_ = ctx
	_ = category
	normalizedTitle := strings.TrimSpace(title)
	normalizedDescription := strings.TrimSpace(description)
	fullText := strings.ToLower(strings.TrimSpace(normalizedTitle + " " + normalizedDescription))
	log.Printf("[AI priority] mock input title=%q description=%q fullText=%q", normalizedTitle, normalizedDescription, fullText)
	if fullText == "" {
		return PriorityResult{}, "", ErrAIUnavailable
	}

	highKeywords := []string{"затопили", "затопление", "вода", "прорвало", "протечка", "течет", "потоп", "соседи затопили"}
	for _, kw := range highKeywords {
		if strings.Contains(fullText, kw) {
			out := PriorityResult{
				Priority:       priorityHigh,
				PriorityScore:  0.95,
				PriorityReason: "Обнаружены признаки аварийной проблемы, связанной с водой",
			}
			raw, _ := json.Marshal(out)
			log.Printf("[AI priority] mock rule=high_water keyword=%q priority=%s", kw, out.Priority)
			return out, string(raw), nil
		}
	}

	mediumKeywords := []string{"не работает", "сломалось", "не включается", "поломка"}
	for _, kw := range mediumKeywords {
		if strings.Contains(fullText, kw) {
			out := PriorityResult{
				Priority:       priorityMedium,
				PriorityScore:  0.7,
				PriorityReason: "Обнаружена неисправность, требующая решения, но не аварийная",
			}
			raw, _ := json.Marshal(out)
			log.Printf("[AI priority] mock rule=medium_breakdown keyword=%q priority=%s", kw, out.Priority)
			return out, string(raw), nil
		}
	}

	out := PriorityResult{
		Priority:       priorityLow,
		PriorityScore:  0.3,
		PriorityReason: "Проблема не выглядит срочной",
	}
	raw, _ := json.Marshal(out)
	log.Printf("[AI priority] mock rule=low_default priority=%s", out.Priority)
	return out, string(raw), nil
}
