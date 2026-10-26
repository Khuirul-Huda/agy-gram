package agent

import (
	"fmt"
	"strings"

	"github.com/Khuirul-Huda/agy-gram/internal/domain"
)

// handleToolStep inspects an active tool step and dispatches to tool-specific handlers.
func handleToolStep(step *streamStep, conversationID string, opts domain.PromptOptions) {
	if step == nil || step.ToolInfo == nil {
		return
	}

	params := step.ToolInfo.Parameters
	if params == nil {
		params = make(map[string]interface{})
	}

	switch step.ToolName {
	case "run_command":
		handleRunCommand(step.ToolName, params, step.StepIndex, conversationID, opts)
	case "edit_file", "replace_file_content":
		handleEditFile(step.ToolName, params, step.StepIndex, conversationID, opts)
	case "write_file", "write_to_file":
		handleWriteFile(step.ToolName, params, step.StepIndex, conversationID, opts)
	case "read_file":
		handleReadFile(step.ToolName, params, step.StepIndex, conversationID, opts)
	case "ask_question":
		handleAskQuestion(step.ToolName, params, step.StepIndex, conversationID, opts)
	}
}

func handleRunCommand(toolName string, params map[string]interface{}, stepIndex int, conversationID string, opts domain.PromptOptions) {
	cmdStr, _ := params["CommandLine"].(string)
	summary, _ := params["toolSummary"].(string)

	if cmdStr == "" && conversationID != "" {
		toolArgs := fetchToolCallDetails(conversationID, stepIndex, "run_command")
		cmdStr = getToolArgString(toolArgs, "CommandLine")
		if summary == "" {
			summary = getToolArgString(toolArgs, "toolSummary")
		}
	}

	cmdStr = strings.TrimSpace(cmdStr)
	if cmdStr == "" {
		return
	}

	if opts.OnCommand != nil {
		opts.OnCommand(cmdStr)
	}

	if opts.OnToolEvent != nil {
		opts.OnToolEvent(domain.ToolEvent{
			Kind:     domain.ToolKindCommand,
			ToolName: toolName,
			Target:   cmdStr,
			Detail:   summary,
		})
	}
}

func handleEditFile(toolName string, params map[string]interface{}, stepIndex int, conversationID string, opts domain.PromptOptions) {
	targetFile, _ := params["Path"].(string)
	if targetFile == "" {
		targetFile, _ = params["TargetFile"].(string)
	}

	target, _ := params["Target"].(string)
	if target == "" {
		target, _ = params["TargetContent"].(string)
	}

	replacement, _ := params["Replacement"].(string)
	if replacement == "" {
		replacement, _ = params["ReplacementContent"].(string)
	}

	instruction, _ := params["Instruction"].(string)

	needsFallback := targetFile == "" || (target == "" && replacement == "")
	if needsFallback && conversationID != "" {
		toolArgs := fetchToolCallDetails(conversationID, stepIndex, toolName)
		if targetFile == "" {
			targetFile = getToolArgString(toolArgs, "Path")
			if targetFile == "" {
				targetFile = getToolArgString(toolArgs, "TargetFile")
			}
		}
		if target == "" {
			target = getToolArgString(toolArgs, "Target")
			if target == "" {
				target = getToolArgString(toolArgs, "TargetContent")
			}
		}
		if replacement == "" {
			replacement = getToolArgString(toolArgs, "Replacement")
			if replacement == "" {
				replacement = getToolArgString(toolArgs, "ReplacementContent")
			}
		}
		if instruction == "" {
			instruction = getToolArgString(toolArgs, "Instruction")
		}
	}

	var diffBuilder strings.Builder
	if instruction != "" {
		diffBuilder.WriteString(fmt.Sprintf("Instruction: %s\n\n", instruction))
	}
	diffStr := generateGitDiff(targetFile, target, replacement)
	if diffStr != "" {
		diffBuilder.WriteString(diffStr)
	}
	diff := diffBuilder.String()

	if opts.OnFileEdit != nil {
		opts.OnFileEdit("edit_file", targetFile, diff)
	}

	if opts.OnToolEvent != nil {
		opts.OnToolEvent(domain.ToolEvent{
			Kind:     domain.ToolKindEdit,
			ToolName: toolName,
			Target:   targetFile,
			Detail:   diff,
		})
	}
}

func handleWriteFile(toolName string, params map[string]interface{}, stepIndex int, conversationID string, opts domain.PromptOptions) {
	targetFile, _ := params["Path"].(string)
	if targetFile == "" {
		targetFile, _ = params["TargetFile"].(string)
	}

	content, _ := params["Content"].(string)
	if content == "" {
		content, _ = params["CodeContent"].(string)
	}

	desc, _ := params["Description"].(string)

	needsFallback := targetFile == "" || content == ""
	if needsFallback && conversationID != "" {
		toolArgs := fetchToolCallDetails(conversationID, stepIndex, toolName)
		if targetFile == "" {
			targetFile = getToolArgString(toolArgs, "Path")
			if targetFile == "" {
				targetFile = getToolArgString(toolArgs, "TargetFile")
			}
		}
		if desc == "" {
			desc = getToolArgString(toolArgs, "Description")
		}
		if content == "" {
			content = getToolArgString(toolArgs, "Content")
			if content == "" {
				content = getToolArgString(toolArgs, "CodeContent")
			}
		}
	}

	var diffBuilder strings.Builder
	if desc != "" {
		diffBuilder.WriteString(fmt.Sprintf("Description: %s\n\n", desc))
	}
	diffStr := formatWriteContent(content)
	if diffStr != "" {
		diffBuilder.WriteString(diffStr)
	}
	diff := diffBuilder.String()

	if opts.OnFileEdit != nil {
		opts.OnFileEdit("write_file", targetFile, diff)
	}

	if opts.OnToolEvent != nil {
		opts.OnToolEvent(domain.ToolEvent{
			Kind:     domain.ToolKindWrite,
			ToolName: toolName,
			Target:   targetFile,
			Detail:   diff,
		})
	}
}

func handleReadFile(toolName string, params map[string]interface{}, stepIndex int, conversationID string, opts domain.PromptOptions) {
	targetFile, _ := params["Path"].(string)
	if targetFile == "" {
		targetFile, _ = params["TargetFile"].(string)
	}

	if targetFile == "" && conversationID != "" {
		toolArgs := fetchToolCallDetails(conversationID, stepIndex, "read_file")
		targetFile = getToolArgString(toolArgs, "Path")
		if targetFile == "" {
			targetFile = getToolArgString(toolArgs, "TargetFile")
		}
	}

	var detailParts []string
	if startLine, ok := params["StartLine"]; ok && startLine != nil {
		detailParts = append(detailParts, fmt.Sprintf("StartLine: %v", startLine))
	}
	if endLine, ok := params["EndLine"]; ok && endLine != nil {
		detailParts = append(detailParts, fmt.Sprintf("EndLine: %v", endLine))
	}
	detail := strings.Join(detailParts, ", ")

	if opts.OnFileRead != nil {
		opts.OnFileRead(targetFile)
	}

	if opts.OnToolEvent != nil {
		opts.OnToolEvent(domain.ToolEvent{
			Kind:     domain.ToolKindRead,
			ToolName: toolName,
			Target:   targetFile,
			Detail:   detail,
		})
	}
}

func handleAskQuestion(toolName string, params map[string]interface{}, stepIndex int, conversationID string, opts domain.PromptOptions) {
	rawQuestions, _ := params["questions"].([]interface{})
	if len(rawQuestions) == 0 && conversationID != "" {
		toolArgs := fetchToolCallDetails(conversationID, stepIndex, "ask_question")
		rawQuestions, _ = toolArgs["questions"].([]interface{})
	}

	for _, item := range rawQuestions {
		qMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		question, _ := qMap["question"].(string)
		isMulti, _ := qMap["is_multi_select"].(bool)
		var options []string
		if rawOpts, ok := qMap["options"].([]interface{}); ok {
			for _, o := range rawOpts {
				if s, ok := o.(string); ok && strings.TrimSpace(s) != "" {
					options = append(options, strings.TrimSpace(s))
				}
			}
		}

		if opts.OnQuestion != nil {
			opts.OnQuestion(question, options, isMulti)
		}

		if opts.OnToolEvent != nil {
			opts.OnToolEvent(domain.ToolEvent{
				Kind:     domain.ToolKindQuestion,
				ToolName: toolName,
				Target:   question,
				Detail:   strings.Join(options, " | "),
			})
		}
	}
}
