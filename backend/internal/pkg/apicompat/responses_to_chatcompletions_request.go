package apicompat

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

// ConvertResponsesOptions controls optional behaviour during Responses→ChatCompletions conversion.
type ConvertResponsesOptions struct {
	// StripReasoningEffort omits reasoning_effort from the Chat Completions output.
	// Enable for upstreams that reject reasoning_effort + tools together (e.g. b.ai /v1/chat/completions).
	StripReasoningEffort bool
	// PreserveInstructionsField keeps the non-standard instructions field in the
	// Chat Completions request. By default instructions are represented only as a
	// leading system message because strict Chat-Completions-only upstreams often
	// reject the Responses-only instructions field.
	PreserveInstructionsField bool
}

// responsesToChatCompletionsRequestBase performs the fork-specific scalar and
// message conversion. The public entry points layer effective-tool handling on
// top in chatcompletions_responses_bridge.go.
func responsesToChatCompletionsRequestBase(req *ResponsesRequest, opts ConvertResponsesOptions) (*ChatCompletionsRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("nil ResponsesRequest")
	}

	messages, err := convertResponsesInputToChatMessages(req.Input)
	if err != nil {
		return nil, fmt.Errorf("convert input: %w", err)
	}

	// instructions 在 Chat Completions 中没有等价独立字段。为保持语义完整，
	// 默认在最前置插入一条 system 消息；非标准 instructions 字段仅在显式
	// PreserveInstructionsField 时透传，避免严格 CC-only upstream 拒绝请求。
	if req.Instructions != "" {
		raw, err := json.Marshal(req.Instructions)
		if err != nil {
			return nil, fmt.Errorf("marshal instructions: %w", err)
		}
		sysMsg := ChatMessage{Role: "system", Content: raw}
		messages = append([]ChatMessage{sysMsg}, messages...)
	}

	out := &ChatCompletionsRequest{
		Model:             req.Model,
		Messages:          messages,
		Temperature:       req.Temperature,
		TopP:              req.TopP,
		Stop:              req.Stop,
		User:              req.User,
		Metadata:          req.Metadata,
		Seed:              req.Seed,
		PresencePenalty:   req.PresencePenalty,
		FrequencyPenalty:  req.FrequencyPenalty,
		Logprobs:          req.Logprobs,
		TopLogprobs:       req.TopLogprobs,
		Stream:            req.Stream,
		StreamOptions:     req.StreamOptions,
		ServiceTier:       req.ServiceTier,
		ParallelToolCalls: req.ParallelToolCalls,
	}
	if opts.PreserveInstructionsField {
		out.Instructions = req.Instructions
	}

	if req.MaxOutputTokens != nil && *req.MaxOutputTokens > 0 {
		v := *req.MaxOutputTokens
		out.MaxTokens = &v
	}

	if req.Reasoning != nil && req.Reasoning.Effort != "" {
		if opts.StripReasoningEffort {
			slog.Debug("apicompat: stripping reasoning_effort per account config")
		} else {
			out.ReasoningEffort = req.Reasoning.Effort
		}
	}

	if len(req.Tools) > 0 {
		out.Tools = convertResponsesToolsToChat(req.Tools)
	}

	if len(req.ToolChoice) > 0 {
		toolChoice, err := convertResponsesToolChoiceToChat(req.ToolChoice)
		if err != nil {
			return nil, fmt.Errorf("convert tool_choice: %w", err)
		}
		out.ToolChoice = toolChoice
	}

	logIgnoredResponsesFields(req)

	return out, nil
}

// convertResponsesInputToChatMessages 把 Responses API 的 input 还原为 Chat
// Completions 的 messages 数组。input 既可能是字符串（等价于一条 user 消息），
// 也可能是 ResponsesInputItem 数组。
func convertResponsesInputToChatMessages(raw json.RawMessage) ([]ChatMessage, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	// Plain string input → single user message.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		content, err := json.Marshal(s)
		if err != nil {
			return nil, err
		}
		return []ChatMessage{{Role: "user", Content: content}}, nil
	}

	var rawItems []json.RawMessage
	if err := json.Unmarshal(raw, &rawItems); err != nil {
		return nil, fmt.Errorf("input is neither string nor array: %w", err)
	}

	built, mediaByCallID, err := buildChatMessagesFromItems(nil, rawItems)
	if err != nil {
		return nil, err
	}
	return normalizeChatMessagesForToolCallPairs(built, mediaByCallID), nil
}

// normalizeChatMessagesForToolCallPairs enforces the Chat Completions invariant
// required by strict upstreams: an assistant message containing tool_calls must
// be immediately followed by tool messages for those tool_call IDs. Responses
// histories can contain pending function_call items without corresponding
// function_call_output items; forwarding those verbatim causes upstream 400s
// such as "An assistant message with 'tool_calls' must be followed by tool
// messages". Unanswered tool calls are removed from the assistant message, and
// orphan tool messages are preserved as user-visible text instead of illegal
// role=tool messages.
func normalizeChatMessagesForToolCallPairs(messages []ChatMessage, mediaByCallID toolOutputMediaByCallID) []ChatMessage {
	replies := make(map[string]ChatMessage)
	for _, msg := range messages {
		if msg.Role == "tool" && msg.ToolCallID != "" {
			replies[msg.ToolCallID] = msg
		}
	}
	pairedIDs := make(map[string]bool, len(replies))
	for _, msg := range messages {
		if msg.Role != "assistant" {
			continue
		}
		for _, tc := range msg.ToolCalls {
			if tc.ID != "" {
				if _, ok := replies[tc.ID]; ok {
					pairedIDs[tc.ID] = true
				}
			}
		}
	}

	out := make([]ChatMessage, 0, len(messages))

	for _, msg := range messages {
		if msg.Role == "tool" {
			if msg.ToolCallID != "" && pairedIDs[msg.ToolCallID] {
				continue
			}
			if msg.ToolCallID == "" || !pairedIDs[msg.ToolCallID] {
				out = append(out, orphanToolMessageAsUser(msg))
				if media := mediaByCallID[msg.ToolCallID]; len(media) > 0 {
					parts := append([]ChatContentPart{{
						Type: "text",
						Text: fmt.Sprintf(toolOutputMediaAttribution, msg.ToolCallID),
					}}, media...)
					content, _ := json.Marshal(parts)
					out = append(out, ChatMessage{Role: "user", Content: content})
				}
			}
			continue
		}
		if msg.Role != "assistant" || len(msg.ToolCalls) == 0 {
			out = append(out, msg)
			continue
		}

		validIDs := make(map[string]bool, len(msg.ToolCalls))
		validCalls := make([]ChatToolCall, 0, len(msg.ToolCalls))
		for _, tc := range msg.ToolCalls {
			if tc.ID == "" {
				slog.Debug("apicompat: dropping assistant tool_call without id")
				continue
			}
			if _, ok := replies[tc.ID]; !ok {
				slog.Debug("apicompat: dropping unanswered assistant tool_call",
					slog.String("tool_call_id", tc.ID),
					slog.String("name", tc.Function.Name))
				continue
			}
			validIDs[tc.ID] = true
			validCalls = append(validCalls, tc)
		}

		if len(validCalls) > 0 {
			msg.ToolCalls = validCalls
			out = append(out, msg)
		} else if chatMessageHasNonEmptyContent(msg) {
			msg.ToolCalls = nil
			out = append(out, msg)
		}

		for _, tc := range validCalls {
			if validIDs[tc.ID] {
				out = append(out, replies[tc.ID])
			}
		}

		var mediaParts []ChatContentPart
		for _, tc := range validCalls {
			media := mediaByCallID[tc.ID]
			if len(media) == 0 {
				continue
			}
			mediaParts = append(mediaParts, ChatContentPart{
				Type: "text",
				Text: fmt.Sprintf(toolOutputMediaAttribution, tc.ID),
			})
			mediaParts = append(mediaParts, media...)
		}
		if len(mediaParts) > 0 {
			content, _ := json.Marshal(mediaParts)
			out = append(out, ChatMessage{Role: "user", Content: content})
		}
	}

	return out
}

func chatMessageHasNonEmptyContent(msg ChatMessage) bool {
	if len(msg.Content) == 0 || string(msg.Content) == "null" {
		return false
	}
	var s string
	if err := json.Unmarshal(msg.Content, &s); err == nil {
		return s != ""
	}
	return true
}

func orphanToolMessageAsUser(msg ChatMessage) ChatMessage {
	toolOutput := ""
	if len(msg.Content) > 0 {
		if err := json.Unmarshal(msg.Content, &toolOutput); err != nil {
			toolOutput = string(msg.Content)
		}
	}
	text := "Tool result"
	if msg.ToolCallID != "" {
		text += " for " + msg.ToolCallID
	}
	if toolOutput != "" {
		text += ": " + toolOutput
	}
	content, _ := json.Marshal(text)
	return ChatMessage{Role: "user", Content: content}
}

// convertResponsesToolsToChat 把 Responses 的 tool 定义还原为 Chat Completions
// 形态（type=function, function={name,description,parameters,strict}）。
func convertResponsesToolsToChat(tools []ResponsesTool) []ChatTool {
	out := make([]ChatTool, 0, len(tools))
	for _, t := range tools {
		if t.Type != "" && t.Type != "function" {
			// 内置 server-side 工具（web_search / local_shell 等）在 Chat
			// Completions 中没有对应表达，跳过并记录。
			slog.Debug("apicompat: drop non-function Responses tool",
				slog.String("type", t.Type),
				slog.String("name", t.Name))
			continue
		}
		params := t.Parameters
		if len(params) == 0 || string(params) == "null" {
			params = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		out = append(out, ChatTool{
			Type: "function",
			Function: &ChatFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
				Strict:      t.Strict,
			},
		})
	}
	return out
}

// convertResponsesToolChoiceToChat maps Responses tool_choice variants to Chat
// Completions variants. String choices are compatible. Responses object form
// {"type":"function","name":"foo"} becomes Chat's
// {"type":"function","function":{"name":"foo"}}. Already-Chat-shaped
// objects are preserved.
func convertResponsesToolChoiceToChat(raw json.RawMessage) (json.RawMessage, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return json.Marshal(s)
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}

	var typ string
	if v := obj["type"]; len(v) > 0 {
		_ = json.Unmarshal(v, &typ)
	}
	if typ != "function" {
		// Built-in/server-side tool choices have no Chat Completions equivalent;
		// pass through so permissive upstreams can still decide what to do.
		return raw, nil
	}
	if _, ok := obj["function"]; ok {
		return raw, nil
	}
	var name string
	if v := obj["name"]; len(v) > 0 {
		_ = json.Unmarshal(v, &name)
	}
	if name == "" {
		return raw, nil
	}
	return json.Marshal(map[string]any{
		"type": "function",
		"function": map[string]any{
			"name": name,
		},
	})
}

// logIgnoredResponsesFields 输出对未承载到 Chat Completions 的字段的 debug 日志，
// 便于上游侧排障，但不影响请求的实际效果。
func logIgnoredResponsesFields(req *ResponsesRequest) {
	if req.PreviousResponseID != "" {
		slog.Debug("apicompat: ignoring previous_response_id (no Chat Completions equivalent)",
			slog.String("previous_response_id", req.PreviousResponseID))
	}
	if req.PromptCacheKey != "" {
		slog.Debug("apicompat: ignoring prompt_cache_key",
			slog.String("prompt_cache_key", req.PromptCacheKey))
	}
	if len(req.Include) > 0 {
		slog.Debug("apicompat: ignoring Responses include[]",
			slog.Any("include", req.Include))
	}
	if req.Store != nil {
		slog.Debug("apicompat: ignoring store flag",
			slog.Bool("store", *req.Store))
	}
	if req.ParallelToolCalls != nil {
		slog.Debug("apicompat: mapping parallel_tool_calls to Chat Completions",
			slog.Bool("parallel_tool_calls", *req.ParallelToolCalls))
	}
	if req.Text != nil {
		slog.Debug("apicompat: ignoring text verbosity",
			slog.String("verbosity", req.Text.Verbosity))
	}
	if req.Reasoning != nil && req.Reasoning.Summary != "" {
		slog.Debug("apicompat: ignoring reasoning.summary",
			slog.String("summary", req.Reasoning.Summary))
	}
}
