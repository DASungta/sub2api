//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetChatCompletionsTraceIDHeader_Configured(t *testing.T) {
	a := &Account{
		Type: AccountTypeAPIKeyChatCompletions,
		Credentials: map[string]any{
			"chat_completions_url": "https://api.example.com/v1/chat/completions",
			"api_key":              "sk-test",
			"trace_id_header":      "st-gateway-request-id",
		},
	}
	assert.Equal(t, "st-gateway-request-id", a.GetChatCompletionsTraceIDHeader())
}

func TestGetChatCompletionsTraceIDHeader_TrimsWhitespace(t *testing.T) {
	a := &Account{
		Type: AccountTypeAPIKeyChatCompletions,
		Credentials: map[string]any{
			"api_key":         "sk-test",
			"trace_id_header": "  x-my-trace  ",
		},
	}
	assert.Equal(t, "x-my-trace", a.GetChatCompletionsTraceIDHeader())
}

func TestGetChatCompletionsTraceIDHeader_NotConfigured(t *testing.T) {
	a := &Account{
		Type: AccountTypeAPIKeyChatCompletions,
		Credentials: map[string]any{
			"chat_completions_url": "https://api.example.com/v1/chat/completions",
			"api_key":              "sk-test",
			// trace_id_header absent
		},
	}
	assert.Equal(t, "", a.GetChatCompletionsTraceIDHeader())
}

func TestGetChatCompletionsTraceIDHeader_EmptyString(t *testing.T) {
	a := &Account{
		Type: AccountTypeAPIKeyChatCompletions,
		Credentials: map[string]any{
			"api_key":         "sk-test",
			"trace_id_header": "",
		},
	}
	assert.Equal(t, "", a.GetChatCompletionsTraceIDHeader())
}

func TestGetChatCompletionsTraceIDHeader_WrongType(t *testing.T) {
	a := &Account{
		Type: AccountTypeAPIKeyChatCompletions,
		Credentials: map[string]any{
			"api_key":         "sk-test",
			"trace_id_header": 12345, // wrong type, not a string
		},
	}
	assert.Equal(t, "", a.GetChatCompletionsTraceIDHeader())
}

func TestGetChatCompletionsTraceIDHeader_NonCCAccountType(t *testing.T) {
	a := &Account{
		Type: AccountTypeOAuth,
		Credentials: map[string]any{
			"trace_id_header": "x-my-trace",
		},
	}
	assert.Equal(t, "", a.GetChatCompletionsTraceIDHeader(), "non-CC account types should return empty")
}

func TestGetChatCompletionsTraceIDHeader_NilAccount(t *testing.T) {
	var a *Account
	assert.Equal(t, "", a.GetChatCompletionsTraceIDHeader())
}

func TestGetChatCompletionsTraceIDHeader_NilCredentials(t *testing.T) {
	a := &Account{
		Type:        AccountTypeAPIKeyChatCompletions,
		Credentials: nil,
	}
	assert.Equal(t, "", a.GetChatCompletionsTraceIDHeader())
}
