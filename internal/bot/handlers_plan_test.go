package bot

import (
	"testing"
)

func TestIsPlanResponse(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		output   string
		expected bool
	}{
		{
			name:     "Plan mode always true",
			mode:     "plan",
			output:   "Here is what I will do.",
			expected: true,
		},
		{
			name:     "Auto mode with markdown plan header",
			mode:     "auto",
			output:   "I evaluated the code.\n\n### Plan\n1. Step one\n2. Step two",
			expected: true,
		},
		{
			name:     "Auto mode with Indonesian rencana header",
			mode:     "auto",
			output:   "Berikut analisisnya.\n\n## Rencana Kerja\n- Langkah 1",
			expected: true,
		},
		{
			name:     "Auto mode with action plan mention",
			mode:     "auto",
			output:   "We need an action plan before touching DB.",
			expected: true,
		},
		{
			name:     "Regular response without plan",
			mode:     "auto",
			output:   "Tests passed successfully with 0 errors.",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := isPlanResponse(tc.mode, tc.output)
			if actual != tc.expected {
				t.Errorf("isPlanResponse(%q, %q) = %v; want %v", tc.mode, tc.output, actual, tc.expected)
			}
		})
	}
}

func TestAttachPlanApprovalButtons(t *testing.T) {
	h := &Handler{}
	menu := h.attachPlanApprovalButtons(nil, "id")

	if menu == nil {
		t.Fatalf("expected non-nil menu")
	}
	if len(menu.InlineKeyboard) != 1 {
		t.Fatalf("expected 1 row of buttons, got %d", len(menu.InlineKeyboard))
	}
	row := menu.InlineKeyboard[0]
	if len(row) != 2 {
		t.Fatalf("expected 2 buttons in row, got %d", len(row))
	}
	if row[0].Text != "🚀 Eksekusi Plan Ini" {
		t.Errorf("expected approve button, got %s", row[0].Text)
	}
	if row[1].Text != "✏️ Minta Revisi Plan" {
		t.Errorf("expected revise button, got %s", row[1].Text)
	}
}
