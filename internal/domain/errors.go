package domain

import "errors"

var (
	// ErrAgentBusy is returned when a prompt execution is requested while another is in progress.
	ErrAgentBusy = errors.New("agent is currently busy processing another request")

	// ErrNoActiveProcess is returned when cancellation is requested but no process is running.
	ErrNoActiveProcess = errors.New("no active agent process to cancel")

	// ErrExecutionCancelled is returned when an execution was cancelled by the user.
	ErrExecutionCancelled = errors.New("agent execution was cancelled")

	// ErrEmptyPrompt is returned when an empty prompt is submitted.
	ErrEmptyPrompt = errors.New("prompt cannot be empty")
)
