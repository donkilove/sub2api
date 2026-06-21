package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvaluateUserRiskEscalatesHighRiskSignals(t *testing.T) {
	score, level, reasons := evaluateUserRisk(userRiskEvaluationInput{
		RequestCount:       30,
		PreviousRequests:   10,
		Cost:               1.2,
		PreviousCost:       0.2,
		ErrorCount:         35,
		RateLimitedCount:   12,
		AuthErrorCount:     6,
		Upstream5xxCount:   4,
		TimeoutCount:       2,
		UniqueIPs:          7,
		ActiveAPIKeys:      5,
		Balance:            0.05,
		CurrentConcurrency: 5,
		WaitingInQueue:     1,
		MaxConcurrency:     5,
		UserStatus:         StatusActive,
	})

	require.GreaterOrEqual(t, score, 75)
	require.Equal(t, UserRiskLevelCritical, level)
	require.NotEmpty(t, reasons)
	require.Contains(t, reasonCodes(reasons), "high_error_rate")
	require.Contains(t, reasonCodes(reasons), "auth_errors")
	require.Contains(t, reasonCodes(reasons), "concurrency_queue")
}

func TestEvaluateUserRiskKeepsQuietUserLowRisk(t *testing.T) {
	score, level, reasons := evaluateUserRisk(userRiskEvaluationInput{
		RequestCount:     20,
		PreviousRequests: 18,
		Cost:             0.02,
		PreviousCost:     0.02,
		ErrorCount:       0,
		UniqueIPs:        1,
		ActiveAPIKeys:    1,
		Balance:          3,
		UserStatus:       StatusActive,
	})

	require.Equal(t, 0, score)
	require.Equal(t, UserRiskLevelLow, level)
	require.Empty(t, reasons)
}

func TestEvaluateUserRiskEscalatesSharedIPSignals(t *testing.T) {
	score, level, reasons := evaluateUserRisk(userRiskEvaluationInput{
		RequestCount:  10,
		UniqueIPs:     1,
		ActiveAPIKeys: 1,
		Balance:       5,
		UserStatus:    StatusActive,
		IPRisk: UserRiskIPRisk{
			SharedIPCount:      2,
			LinkedUserCount:    12,
			MaxUsersOnSameIP:   8,
			NewUsersOnSharedIP: 3,
			SameUAUserCount:    4,
			RegisterEventCount: 3,
		},
	})

	require.GreaterOrEqual(t, score, 50)
	require.Equal(t, UserRiskLevelHigh, level)
	codes := reasonCodes(reasons)
	require.Contains(t, codes, "shared_ip_many_accounts")
	require.Contains(t, codes, "shared_ip_new_accounts_some")
	require.Contains(t, codes, "shared_ip_same_ua")
}

func TestParseUserRiskWindow(t *testing.T) {
	_, label, ok := ParseUserRiskWindow("1d")
	require.True(t, ok)
	require.Equal(t, "24h", label)

	_, _, ok = ParseUserRiskWindow("2d")
	require.False(t, ok)
}

func reasonCodes(reasons []UserRiskReason) []string {
	out := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		out = append(out, reason.Code)
	}
	return out
}
