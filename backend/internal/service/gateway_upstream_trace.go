package service

import (
	"context"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

// logUpstreamTraceID 从上游响应头中读取账号配置的 trace_id_header 字段值，
// 并以结构化日志记录，自动携带上下文中的本地 request_id。
//
// 使用场景：apikey-chat-completions 账号的运维排查，将上游网关的请求追踪 ID
// 与本地请求 ID 关联，便于在日志中 grep "upstream trace id captured" 定位链路。
//
// 无副作用：账号未配置 trace_id_header、或响应头中该字段缺失/为空时，函数为 no-op。
func logUpstreamTraceID(ctx context.Context, account *Account, header http.Header, model string) {
	if account == nil || header == nil {
		return
	}
	name := account.GetChatCompletionsTraceIDHeader()
	if name == "" {
		return
	}
	value := strings.TrimSpace(header.Get(name))
	if value == "" {
		return
	}
	logger.FromContext(ctx).Info("upstream trace id captured",
		zap.Int64("account_id", account.ID),
		zap.String("account_name", account.Name),
		zap.String("model", model),
		zap.String("trace_header", name),
		zap.String("trace_id", value),
	)
}
