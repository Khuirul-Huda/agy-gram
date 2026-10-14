package agent

import (
	"testing"
)

func TestSessionLifecycle(t *testing.T) {
	s := NewSession("auto", "")

	if s.ShouldContinue() {
		t.Errorf("expected initial continueSession to be false")
	}
	if s.GetMode() != "auto" {
		t.Errorf("expected default mode 'auto', got '%s'", s.GetMode())
	}

	s.SetContinue(true)
	if !s.ShouldContinue() {
		t.Errorf("expected continueSession to be true")
	}

	s.SetMode("plan")
	if s.GetMode() != "plan" {
		t.Errorf("expected mode 'plan', got '%s'", s.GetMode())
	}

	s.SetModel("gemini-2.5-pro")
	if s.GetModel() != "gemini-2.5-pro" {
		t.Errorf("expected model 'gemini-2.5-pro', got '%s'", s.GetModel())
	}

	s.Reset()
	if s.ShouldContinue() {
		t.Errorf("expected continueSession to be reset to false")
	}
}
