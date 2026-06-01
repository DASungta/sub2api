//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// buildObserverCtx 返回一个注入了 observer logger 的 context 及 observer 本身。
func buildObserverCtx() (context.Context, *observer.ObservedLogs) {
	core, obs := observer.New(zapcore.InfoLevel)
	l := zap.New(core)
	ctx := logger.IntoContext(context.Background(), l)
	return ctx, obs
}

func TestLogUpstreamTraceID_EmitsLogWhenConfiguredAndHeaderPresent(t *testing.T) {
	account := &Account{
		ID:   42,
		Name: "test-upstream",
		Type: AccountTypeAPIKeyChatCompletions,
		Credentials: map[string]any{
			"chat_completions_url": "https://example.com/v1/chat/completions",
			"api_key":              "sk-test",
			"trace_id_header":      "st-gateway-request-id",
		},
	}
	header := http.Header{}
	header.Set("St-Gateway-Request-Id", "cf9b946d-0335-40be-8c40-2d0c257225fb")

	ctx, obs := buildObserverCtx()
	logUpstreamTraceID(ctx, account, header, "gpt-4o")

	assert.Equal(t, 1, obs.Len(), "expected exactly one log entry")
	entry := obs.All()[0]
	assert.Equal(t, "upstream trace id captured", entry.Message)
	fields := make(map[string]any, len(entry.Context))
	for _, f := range entry.Context {
		fields[f.Key] = f.String
	}
	assert.Equal(t, "cf9b946d-0335-40be-8c40-2d0c257225fb", fields["trace_id"])
	assert.Equal(t, "st-gateway-request-id", fields["trace_header"])
	assert.Equal(t, "gpt-4o", fields["model"])
}

func TestLogUpstreamTraceID_NoLogWhenTraceHeaderNotConfigured(t *testing.T) {
	account := &Account{
		ID:   43,
		Name: "no-trace",
		Type: AccountTypeAPIKeyChatCompletions,
		Credentials: map[string]any{
			"chat_completions_url": "https://example.com/v1/chat/completions",
			"api_key":              "sk-test",
			// trace_id_header not set
		},
	}
	header := http.Header{}
	header.Set("St-Gateway-Request-Id", "some-id")

	ctx, obs := buildObserverCtx()
	logUpstreamTraceID(ctx, account, header, "gpt-4o")

	assert.Equal(t, 0, obs.Len(), "expected no log when trace_id_header not configured")
}

func TestLogUpstreamTraceID_NoLogWhenHeaderValueMissing(t *testing.T) {
	account := &Account{
		ID:   44,
		Name: "missing-header",
		Type: AccountTypeAPIKeyChatCompletions,
		Credentials: map[string]any{
			"chat_completions_url": "https://example.com/v1/chat/completions",
			"api_key":              "sk-test",
			"trace_id_header":      "x-my-trace",
		},
	}
	// Response header does not contain x-my-trace
	header := http.Header{}
	header.Set("Content-Type", "text/event-stream")

	ctx, obs := buildObserverCtx()
	logUpstreamTraceID(ctx, account, header, "gpt-4o")

	assert.Equal(t, 0, obs.Len(), "expected no log when upstream response header is absent")
}

func TestLogUpstreamTraceID_NoLogWhenHeaderValueEmpty(t *testing.T) {
	account := &Account{
		ID:   45,
		Name: "empty-value",
		Type: AccountTypeAPIKeyChatCompletions,
		Credentials: map[string]any{
			"chat_completions_url": "https://example.com/v1/chat/completions",
			"api_key":              "sk-test",
			"trace_id_header":      "x-my-trace",
		},
	}
	header := http.Header{}
	header.Set("X-My-Trace", "   ") // whitespace only

	ctx, obs := buildObserverCtx()
	logUpstreamTraceID(ctx, account, header, "gpt-4o")

	assert.Equal(t, 0, obs.Len(), "expected no log when upstream response header value is blank")
}

func TestLogUpstreamTraceID_NoLogForNonCCAccountType(t *testing.T) {
	account := &Account{
		ID:   46,
		Name: "oauth-account",
		Type: AccountTypeOAuth, // not apikey-chat-completions
		Credentials: map[string]any{
			"trace_id_header": "x-my-trace",
		},
	}
	header := http.Header{}
	header.Set("X-My-Trace", "abc123")

	ctx, obs := buildObserverCtx()
	logUpstreamTraceID(ctx, account, header, "claude-3-5-sonnet")

	assert.Equal(t, 0, obs.Len(), "expected no log for non-apikey-cc account type")
}

func TestLogUpstreamTraceID_NilInputsAreNoOp(t *testing.T) {
	ctx, obs := buildObserverCtx()
	// nil account
	logUpstreamTraceID(ctx, nil, http.Header{}, "gpt-4o")
	// nil header
	logUpstreamTraceID(ctx, &Account{Type: AccountTypeAPIKeyChatCompletions}, nil, "gpt-4o")
	assert.Equal(t, 0, obs.Len(), "expected no log for nil inputs")
}
