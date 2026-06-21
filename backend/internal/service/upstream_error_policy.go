package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

const SettingKeyUpstreamErrorPolicies = "upstream_error_policies"

type UpstreamErrorPolicy struct {
	Category          string `json:"category"`
	Label             string `json:"label"`
	Description       string `json:"description"`
	DefaultStatusCode int    `json:"default_status_code"`
	DefaultErrorType  string `json:"default_error_type"`
	DefaultMessage    string `json:"default_message"`
	DefaultRetryable  bool   `json:"default_retryable"`

	CustomEnabled bool   `json:"custom_enabled"`
	StatusCode    *int   `json:"status_code,omitempty"`
	ErrorType     string `json:"error_type,omitempty"`
	Message       string `json:"message,omitempty"`
	RetryEnabled  bool   `json:"retry_enabled"`
	MaxRetries    int    `json:"max_retries"`
	Note          string `json:"note,omitempty"`

	EffectiveStatusCode int    `json:"effective_status_code"`
	EffectiveErrorType  string `json:"effective_error_type"`
	EffectiveMessage    string `json:"effective_message"`
}

type UpstreamErrorPolicyUpdate struct {
	CustomEnabled *bool   `json:"custom_enabled"`
	StatusCode    *int    `json:"status_code"`
	ErrorType     *string `json:"error_type"`
	Message       *string `json:"message"`
	RetryEnabled  *bool   `json:"retry_enabled"`
	MaxRetries    *int    `json:"max_retries"`
	Note          *string `json:"note"`
}

type upstreamErrorPolicyOverride struct {
	CustomEnabled bool   `json:"custom_enabled,omitempty"`
	StatusCode    *int   `json:"status_code,omitempty"`
	ErrorType     string `json:"error_type,omitempty"`
	Message       string `json:"message,omitempty"`
	RetryEnabled  bool   `json:"retry_enabled,omitempty"`
	MaxRetries    int    `json:"max_retries,omitempty"`
	Note          string `json:"note,omitempty"`
}

type upstreamErrorPolicyStorage struct {
	Overrides map[string]upstreamErrorPolicyOverride `json:"overrides"`
}

type upstreamErrorPolicyDefault struct {
	Category    openAIUpstreamErrorCategory
	Label       string
	Description string
}

var upstreamErrorPolicyDefaultOrder = []upstreamErrorPolicyDefault{
	{openAIUpstreamErrorAuthenticationFailed, "Authentication failed", "上游账号认证失败，通常需要管理员检查账号或密钥。"},
	{openAIUpstreamErrorBillingRequired, "Billing or quota required", "上游账号余额、订阅或额度不足。"},
	{openAIUpstreamErrorPermissionDenied, "Permission denied", "上游账号没有访问该资源或模型的权限。"},
	{openAIUpstreamErrorRateLimited, "Rate limited", "上游限流，通常可以稍后重试或切换账号。"},
	{openAIUpstreamErrorInvalidModel, "Invalid model", "客户端请求的模型不可用或不存在。"},
	{openAIUpstreamErrorModelUnavailable, "Model unavailable", "上游路由或模型不可用，需要管理员排查。"},
	{openAIUpstreamErrorContextLengthExceeded, "Context length exceeded", "请求超过上游上下文长度限制。"},
	{openAIUpstreamErrorContentPolicy, "Content policy", "请求被上游内容安全策略拒绝。"},
	{openAIUpstreamErrorTimeout, "Upstream timeout", "上游请求超时。"},
	{openAIUpstreamErrorServerError, "Upstream server error", "上游服务 5xx 或过载。"},
	{openAIUpstreamErrorTransport, "Transport error", "DNS、TCP、TLS、代理等连接层错误。"},
	{openAIUpstreamErrorBadRequest, "Upstream bad request", "上游拒绝请求，但未命中更具体分类。"},
	{openAIUpstreamErrorUnknown, "Unknown", "未能识别的上游错误。"},
}

func DefaultUpstreamErrorPolicies() []UpstreamErrorPolicy {
	policies := make([]UpstreamErrorPolicy, 0, len(upstreamErrorPolicyDefaultOrder))
	for _, item := range upstreamErrorPolicyDefaultOrder {
		mapping := openAIUpstreamErrorMapping(item.Category)
		policy := UpstreamErrorPolicy{
			Category:          string(item.Category),
			Label:             item.Label,
			Description:       item.Description,
			DefaultStatusCode: mapping.ClientStatus,
			DefaultErrorType:  mapping.ClientType,
			DefaultMessage:    mapping.ClientMessage,
			DefaultRetryable:  mapping.Retryable,
			RetryEnabled:      false,
		}
		policy.applyEffective()
		policies = append(policies, policy)
	}
	return policies
}

func (s *SettingService) ListUpstreamErrorPolicies(ctx context.Context) ([]UpstreamErrorPolicy, error) {
	policies := DefaultUpstreamErrorPolicies()
	overrides, err := s.loadUpstreamErrorPolicyOverrides(ctx)
	if err != nil {
		return nil, err
	}
	for i := range policies {
		if override, ok := overrides[policies[i].Category]; ok {
			policies[i].applyOverride(override)
		}
	}
	sortUpstreamErrorPolicies(policies)
	return policies, nil
}

func (s *SettingService) ResolveUpstreamErrorPolicy(ctx context.Context, category string) (UpstreamErrorPolicy, bool) {
	category = strings.TrimSpace(category)
	if category == "" {
		return UpstreamErrorPolicy{}, false
	}
	policies, err := s.ListUpstreamErrorPolicies(ctx)
	if err != nil {
		return findDefaultUpstreamErrorPolicy(category)
	}
	for _, policy := range policies {
		if policy.Category == category {
			return policy, true
		}
	}
	return UpstreamErrorPolicy{}, false
}

func (s *SettingService) UpdateUpstreamErrorPolicy(ctx context.Context, category string, update UpstreamErrorPolicyUpdate) error {
	category = strings.TrimSpace(category)
	base, ok := findDefaultUpstreamErrorPolicy(category)
	if !ok {
		return fmt.Errorf("unknown upstream error category: %s", category)
	}

	overrides, err := s.loadUpstreamErrorPolicyOverrides(ctx)
	if err != nil {
		return err
	}
	override := overrides[category]
	if update.CustomEnabled != nil {
		override.CustomEnabled = *update.CustomEnabled
	}
	if update.StatusCode != nil {
		if *update.StatusCode < 400 || *update.StatusCode > 599 {
			return fmt.Errorf("status_code must be between 400 and 599")
		}
		override.StatusCode = update.StatusCode
	}
	if update.ErrorType != nil {
		override.ErrorType = strings.TrimSpace(*update.ErrorType)
	}
	if update.Message != nil {
		override.Message = strings.TrimSpace(*update.Message)
	}
	if update.RetryEnabled != nil {
		override.RetryEnabled = *update.RetryEnabled
	}
	if update.MaxRetries != nil {
		if *update.MaxRetries < 0 || *update.MaxRetries > 5 {
			return fmt.Errorf("max_retries must be between 0 and 5")
		}
		override.MaxRetries = *update.MaxRetries
	}
	if update.Note != nil {
		override.Note = strings.TrimSpace(*update.Note)
	}
	if override.RetryEnabled {
		if !base.DefaultRetryable {
			return fmt.Errorf("category %s is not retryable", category)
		}
		if override.MaxRetries <= 0 {
			override.MaxRetries = 1
		}
	} else {
		override.MaxRetries = 0
	}
	if override.CustomEnabled {
		if override.StatusCode == nil {
			code := base.DefaultStatusCode
			override.StatusCode = &code
		}
		if strings.TrimSpace(override.ErrorType) == "" {
			override.ErrorType = base.DefaultErrorType
		}
		if strings.TrimSpace(override.Message) == "" {
			override.Message = base.DefaultMessage
		}
	} else {
		override.StatusCode = nil
		override.ErrorType = ""
		override.Message = ""
	}

	overrides[category] = override
	return s.saveUpstreamErrorPolicyOverrides(ctx, overrides)
}

func findDefaultUpstreamErrorPolicy(category string) (UpstreamErrorPolicy, bool) {
	for _, policy := range DefaultUpstreamErrorPolicies() {
		if policy.Category == category {
			return policy, true
		}
	}
	return UpstreamErrorPolicy{}, false
}

func (s *SettingService) loadUpstreamErrorPolicyOverrides(ctx context.Context) (map[string]upstreamErrorPolicyOverride, error) {
	if s == nil || s.settingRepo == nil {
		return map[string]upstreamErrorPolicyOverride{}, nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyUpstreamErrorPolicies)
	if err != nil {
		if err == ErrSettingNotFound {
			return map[string]upstreamErrorPolicyOverride{}, nil
		}
		return nil, err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]upstreamErrorPolicyOverride{}, nil
	}
	var storage upstreamErrorPolicyStorage
	if err := json.Unmarshal([]byte(raw), &storage); err != nil {
		return nil, err
	}
	if storage.Overrides == nil {
		storage.Overrides = map[string]upstreamErrorPolicyOverride{}
	}
	return storage.Overrides, nil
}

func (s *SettingService) saveUpstreamErrorPolicyOverrides(ctx context.Context, overrides map[string]upstreamErrorPolicyOverride) error {
	if s == nil || s.settingRepo == nil {
		return nil
	}
	for category, override := range overrides {
		if override.isZero() {
			delete(overrides, category)
		}
	}
	storage := upstreamErrorPolicyStorage{Overrides: overrides}
	data, err := json.Marshal(storage)
	if err != nil {
		return err
	}
	return s.settingRepo.Set(ctx, SettingKeyUpstreamErrorPolicies, string(data))
}

func (p *UpstreamErrorPolicy) applyOverride(override upstreamErrorPolicyOverride) {
	p.CustomEnabled = override.CustomEnabled
	p.StatusCode = override.StatusCode
	p.ErrorType = strings.TrimSpace(override.ErrorType)
	p.Message = strings.TrimSpace(override.Message)
	p.RetryEnabled = override.RetryEnabled
	p.MaxRetries = override.MaxRetries
	p.Note = strings.TrimSpace(override.Note)
	if p.RetryEnabled && p.MaxRetries <= 0 {
		p.MaxRetries = 1
	}
	p.applyEffective()
}

func (p *UpstreamErrorPolicy) applyEffective() {
	p.EffectiveStatusCode = p.DefaultStatusCode
	p.EffectiveErrorType = p.DefaultErrorType
	p.EffectiveMessage = p.DefaultMessage
	if p.CustomEnabled {
		if p.StatusCode != nil {
			p.EffectiveStatusCode = *p.StatusCode
		}
		if strings.TrimSpace(p.ErrorType) != "" {
			p.EffectiveErrorType = strings.TrimSpace(p.ErrorType)
		}
		if strings.TrimSpace(p.Message) != "" {
			p.EffectiveMessage = strings.TrimSpace(p.Message)
		}
	}
	if p.EffectiveStatusCode == 0 {
		p.EffectiveStatusCode = http.StatusBadGateway
	}
}

func (o upstreamErrorPolicyOverride) isZero() bool {
	return !o.CustomEnabled &&
		o.StatusCode == nil &&
		strings.TrimSpace(o.ErrorType) == "" &&
		strings.TrimSpace(o.Message) == "" &&
		!o.RetryEnabled &&
		o.MaxRetries == 0 &&
		strings.TrimSpace(o.Note) == ""
}

func sortUpstreamErrorPolicies(policies []UpstreamErrorPolicy) {
	order := make(map[string]int, len(upstreamErrorPolicyDefaultOrder))
	for i, item := range upstreamErrorPolicyDefaultOrder {
		order[string(item.Category)] = i
	}
	sort.SliceStable(policies, func(i, j int) bool {
		return order[policies[i].Category] < order[policies[j].Category]
	})
}
