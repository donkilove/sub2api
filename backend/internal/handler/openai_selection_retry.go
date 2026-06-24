package handler

import (
	"context"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"go.uber.org/zap"
)

const (
	openAISelectionExhaustedRetryLimit = 2
	openAISelectionExhaustedRetryDelay = 500 * time.Millisecond
)

func shouldRetryOpenAISelectionExhausted(err error, excludedAccountCount int, retryCount int) bool {
	if err == nil || excludedAccountCount != 0 || retryCount >= openAISelectionExhaustedRetryLimit {
		return false
	}
	if errors.Is(err, service.ErrNoAvailableCompactAccounts) {
		return false
	}
	return errors.Is(err, service.ErrNoAvailableAccounts)
}

func waitBeforeRetryOpenAISelection(ctx context.Context, logName string, reqLog *zap.Logger, err error, retryCount int) bool {
	if reqLog != nil {
		reqLog.Warn(logName,
			zap.Error(err),
			zap.Int("retry_count", retryCount),
			zap.Int("retry_limit", openAISelectionExhaustedRetryLimit),
			zap.Duration("retry_delay", openAISelectionExhaustedRetryDelay),
		)
	}
	select {
	case <-ctx.Done():
		return false
	case <-time.After(openAISelectionExhaustedRetryDelay):
		return true
	}
}
