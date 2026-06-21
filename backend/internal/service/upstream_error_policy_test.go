package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUpstreamErrorPolicyDefaultsListAllCategories(t *testing.T) {
	policies := DefaultUpstreamErrorPolicies()
	require.NotEmpty(t, policies)
	require.Len(t, policies, 13)

	byCategory := make(map[string]UpstreamErrorPolicy)
	for _, policy := range policies {
		byCategory[policy.Category] = policy
	}

	require.Equal(t, http.StatusTooManyRequests, byCategory["rate_limited"].DefaultStatusCode)
	require.True(t, byCategory["rate_limited"].DefaultRetryable)
	require.False(t, byCategory["authentication_failed"].DefaultRetryable)
	require.Equal(t, "Upstream authentication failed, please contact administrator", byCategory["authentication_failed"].EffectiveMessage)
}

func TestSettingServiceUpstreamErrorPoliciesOverrideDefaults(t *testing.T) {
	repo := newMemorySettingRepo()
	svc := NewSettingService(repo, nil)

	err := svc.UpdateUpstreamErrorPolicy(context.Background(), "rate_limited", UpstreamErrorPolicyUpdate{
		CustomEnabled: policyBoolPtr(true),
		StatusCode:    policyIntPtr(http.StatusServiceUnavailable),
		ErrorType:     stringPtr("custom_rate_limit"),
		Message:       stringPtr("请稍后再试"),
		RetryEnabled:  policyBoolPtr(true),
		MaxRetries:    policyIntPtr(2),
		Note:          stringPtr("高峰期友好提示"),
	})
	require.NoError(t, err)

	policies, err := svc.ListUpstreamErrorPolicies(context.Background())
	require.NoError(t, err)
	policy := findPolicyForTest(t, policies, "rate_limited")
	require.True(t, policy.CustomEnabled)
	require.Equal(t, http.StatusServiceUnavailable, policy.EffectiveStatusCode)
	require.Equal(t, "custom_rate_limit", policy.EffectiveErrorType)
	require.Equal(t, "请稍后再试", policy.EffectiveMessage)
	require.True(t, policy.RetryEnabled)
	require.Equal(t, 2, policy.MaxRetries)
	require.Equal(t, "高峰期友好提示", policy.Note)

	resolved, ok := svc.ResolveUpstreamErrorPolicy(context.Background(), "rate_limited")
	require.True(t, ok)
	require.Equal(t, policy.EffectiveMessage, resolved.EffectiveMessage)
}

func TestSettingServiceUpstreamErrorPolicyRejectsUnsafeRetry(t *testing.T) {
	svc := NewSettingService(newMemorySettingRepo(), nil)

	err := svc.UpdateUpstreamErrorPolicy(context.Background(), "authentication_failed", UpstreamErrorPolicyUpdate{
		RetryEnabled: policyBoolPtr(true),
		MaxRetries:   policyIntPtr(1),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not retryable")
}

func TestClassifyOpenAIUpstreamHTTPErrorAppliesPolicyOverride(t *testing.T) {
	repo := newMemorySettingRepo()
	settingSvc := NewSettingService(repo, nil)
	require.NoError(t, settingSvc.UpdateUpstreamErrorPolicy(context.Background(), "rate_limited", UpstreamErrorPolicyUpdate{
		CustomEnabled: policyBoolPtr(true),
		StatusCode:    policyIntPtr(http.StatusServiceUnavailable),
		ErrorType:     stringPtr("custom_rate_limit"),
		Message:       stringPtr("自定义限流提示"),
		RetryEnabled:  policyBoolPtr(true),
		MaxRetries:    policyIntPtr(3),
	}))

	got := classifyOpenAIUpstreamHTTPErrorWithPolicy(
		context.Background(),
		settingSvc,
		http.StatusTooManyRequests,
		http.Header{},
		[]byte(`{"error":{"message":"rate limit"}}`),
		"",
	)
	require.Equal(t, openAIUpstreamErrorRateLimited, got.Category)
	require.Equal(t, http.StatusServiceUnavailable, got.ClientStatus)
	require.Equal(t, "custom_rate_limit", got.ClientType)
	require.Equal(t, "自定义限流提示", got.ClientMessage)
	require.True(t, got.PolicyRetryEnabled)
	require.Equal(t, 3, got.PolicyMaxRetries)
}

func findPolicyForTest(t *testing.T, policies []UpstreamErrorPolicy, category string) UpstreamErrorPolicy {
	t.Helper()
	for _, policy := range policies {
		if policy.Category == category {
			return policy
		}
	}
	t.Fatalf("policy %s not found", category)
	return UpstreamErrorPolicy{}
}

func policyIntPtr(v int) *int { return &v }

func stringPtr(v string) *string { return &v }

func policyBoolPtr(v bool) *bool { return &v }

type memorySettingRepo struct {
	values map[string]string
}

func newMemorySettingRepo() *memorySettingRepo {
	return &memorySettingRepo{values: map[string]string{}}
}

func (r *memorySettingRepo) Get(_ context.Context, key string) (*Setting, error) {
	value, ok := r.values[key]
	if !ok {
		return nil, ErrSettingNotFound
	}
	return &Setting{Key: key, Value: value, UpdatedAt: time.Now()}, nil
}

func (r *memorySettingRepo) GetValue(ctx context.Context, key string) (string, error) {
	setting, err := r.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

func (r *memorySettingRepo) Set(_ context.Context, key, value string) error {
	r.values[key] = value
	return nil
}

func (r *memorySettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (r *memorySettingRepo) SetMultiple(_ context.Context, settings map[string]string) error {
	for key, value := range settings {
		r.values[key] = value
	}
	return nil
}

func (r *memorySettingRepo) GetAll(_ context.Context) (map[string]string, error) {
	out := make(map[string]string, len(r.values))
	for key, value := range r.values {
		out[key] = value
	}
	return out, nil
}

func (r *memorySettingRepo) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}
