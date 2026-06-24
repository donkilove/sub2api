package handler

import (
	"context"
	"fmt"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestShouldRetryOpenAISelectionExhausted(t *testing.T) {
	tests := []struct {
		name                 string
		err                  error
		excludedAccountCount int
		retryCount           int
		want                 bool
	}{
		{
			name: "no available accounts",
			err:  service.ErrNoAvailableAccounts,
			want: true,
		},
		{
			name: "wrapped no available accounts",
			err:  fmt.Errorf("select account: %w", service.ErrNoAvailableAccounts),
			want: true,
		},
		{
			name:       "retry limit reached",
			err:        service.ErrNoAvailableAccounts,
			retryCount: openAISelectionExhaustedRetryLimit,
			want:       false,
		},
		{
			name:                 "failover excluded account exists",
			err:                  service.ErrNoAvailableAccounts,
			excludedAccountCount: 1,
			want:                 false,
		},
		{
			name: "compact account exhaustion is not retried",
			err:  service.ErrNoAvailableCompactAccounts,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRetryOpenAISelectionExhausted(tt.err, tt.excludedAccountCount, tt.retryCount)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestWaitBeforeRetryOpenAISelectionCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ok := waitBeforeRetryOpenAISelection(ctx, "test.openai.selection_retry", nil, service.ErrNoAvailableAccounts, 1)
	require.False(t, ok)
}
