package agent

import (
	"testing"

	"github.com/Khuirul-Huda/agy-gram/internal/domain"
)

func TestHandleToolStep_AskQuestion(t *testing.T) {
	var capturedQuestion string
	var capturedOptions []string
	var capturedEvent domain.ToolEvent

	opts := domain.PromptOptions{
		OnQuestion: func(q string, opts []string, isMulti bool) {
			capturedQuestion = q
			capturedOptions = opts
		},
		OnToolEvent: func(ev domain.ToolEvent) {
			capturedEvent = ev
		},
	}

	step := &streamStep{
		StepType: "tool",
		State:    "ACTIVE",
		ToolName: "ask_question",
		ToolInfo: &streamToolInfo{
			Name: "ask_question",
			Parameters: map[string]interface{}{
				"questions": []interface{}{
					map[string]interface{}{
						"question":        "Apakah ingin deploy ke staging?",
						"is_multi_select": false,
						"options": []interface{}{
							"Ya, deploy sekarang",
							"Tidak, batalkan",
						},
					},
				},
			},
		},
	}

	handleToolStep(step, "", opts)

	if capturedQuestion != "Apakah ingin deploy ke staging?" {
		t.Errorf("unexpected question: %s", capturedQuestion)
	}
	if len(capturedOptions) != 2 || capturedOptions[0] != "Ya, deploy sekarang" {
		t.Errorf("unexpected options: %v", capturedOptions)
	}
	if capturedEvent.Kind != domain.ToolKindQuestion {
		t.Errorf("unexpected event kind: %s", capturedEvent.Kind)
	}
}
