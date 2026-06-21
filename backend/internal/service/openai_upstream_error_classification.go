package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
)

type openAIUpstreamErrorCategory string

const (
	openAIUpstreamErrorAuthenticationFailed  openAIUpstreamErrorCategory = "authentication_failed"
	openAIUpstreamErrorBillingRequired       openAIUpstreamErrorCategory = "billing_required"
	openAIUpstreamErrorPermissionDenied      openAIUpstreamErrorCategory = "permission_denied"
	openAIUpstreamErrorRateLimited           openAIUpstreamErrorCategory = "rate_limited"
	openAIUpstreamErrorModelUnavailable      openAIUpstreamErrorCategory = "model_unavailable"
	openAIUpstreamErrorInvalidModel          openAIUpstreamErrorCategory = "invalid_model"
	openAIUpstreamErrorContextLengthExceeded openAIUpstreamErrorCategory = "context_length_exceeded"
	openAIUpstreamErrorContentPolicy         openAIUpstreamErrorCategory = "content_policy"
	openAIUpstreamErrorTimeout               openAIUpstreamErrorCategory = "upstream_timeout"
	openAIUpstreamErrorServerError           openAIUpstreamErrorCategory = "upstream_server_error"
	openAIUpstreamErrorTransport             openAIUpstreamErrorCategory = "transport_error"
	openAIUpstreamErrorBadRequest            openAIUpstreamErrorCategory = "upstream_bad_request"
	openAIUpstreamErrorUnknown               openAIUpstreamErrorCategory = "unknown"
)

type openAIUpstreamErrorClassification struct {
	Category           openAIUpstreamErrorCategory
	ClientStatus       int
	ClientType         string
	ClientMessage      string
	Retryable          bool
	PolicyCustom       bool
	PolicyRetryEnabled bool
	PolicyMaxRetries   int
}

func classifyOpenAIUpstreamHTTPError(status int, headers http.Header, body []byte, upstreamMsg string) openAIUpstreamErrorClassification {
	text := openAIUpstreamErrorSearchText(body, upstreamMsg)

	switch {
	case status == http.StatusUnauthorized ||
		containsAny(text, "invalid_api_key", "incorrect api key", "invalid api key", "authentication"):
		return openAIUpstreamErrorMapping(openAIUpstreamErrorAuthenticationFailed)
	case status == http.StatusPaymentRequired ||
		containsAny(text, "insufficient_quota", "quota exceeded", "billing", "payment required", "insufficient balance"):
		return openAIUpstreamErrorMapping(openAIUpstreamErrorBillingRequired)
	case status == http.StatusForbidden ||
		containsAny(text, "forbidden", "permission denied", "not authorized", "access denied"):
		return openAIUpstreamErrorMapping(openAIUpstreamErrorPermissionDenied)
	case status == http.StatusTooManyRequests ||
		containsAny(text, "rate limit", "rate_limit", "too many requests"):
		return openAIUpstreamErrorMapping(openAIUpstreamErrorRateLimited)
	case containsAny(text, "context length", "maximum context", "too many tokens", "max tokens", "token limit"):
		return openAIUpstreamErrorMapping(openAIUpstreamErrorContextLengthExceeded)
	case containsAny(text, "content policy", "policy_violation", "safety", "blocked by policy", "cyber_policy"):
		return openAIUpstreamErrorMapping(openAIUpstreamErrorContentPolicy)
	case status == http.StatusBadRequest &&
		containsAny(text, "model") &&
		containsAny(text, "not found", "does not exist", "unavailable", "unsupported"):
		return openAIUpstreamErrorMapping(openAIUpstreamErrorInvalidModel)
	case status == http.StatusNotFound &&
		containsAny(text, "model") &&
		containsAny(text, "not found", "does not exist", "unavailable", "unsupported"):
		return openAIUpstreamErrorMapping(openAIUpstreamErrorModelUnavailable)
	case status == http.StatusRequestTimeout || status == http.StatusGatewayTimeout ||
		containsAny(text, "timeout", "timed out"):
		return openAIUpstreamErrorMapping(openAIUpstreamErrorTimeout)
	case status == http.StatusInternalServerError ||
		status == http.StatusBadGateway ||
		status == http.StatusServiceUnavailable ||
		status == 529:
		return openAIUpstreamErrorMapping(openAIUpstreamErrorServerError)
	case status == http.StatusBadRequest || status == http.StatusUnprocessableEntity:
		return openAIUpstreamErrorMapping(openAIUpstreamErrorBadRequest)
	default:
		return openAIUpstreamErrorMapping(openAIUpstreamErrorUnknown)
	}
}

func OpenAIUpstreamErrorMapping(status int) (int, string, string) {
	cls := classifyOpenAIUpstreamHTTPError(status, nil, nil, "")
	return cls.ClientStatus, cls.ClientType, cls.ClientMessage
}

func OpenAIUpstreamTransportErrorMapping() (int, string, string) {
	cls := openAIUpstreamErrorMapping(openAIUpstreamErrorTransport)
	return cls.ClientStatus, cls.ClientType, cls.ClientMessage
}

func openAIUpstreamErrorResponseBody(cls openAIUpstreamErrorClassification) []byte {
	body, err := json.Marshal(map[string]any{
		"error": map[string]string{
			"type":    strings.TrimSpace(cls.ClientType),
			"message": strings.TrimSpace(cls.ClientMessage),
		},
	})
	if err != nil {
		return []byte(`{"error":{"type":"upstream_error","message":"Upstream request failed"}}`)
	}
	return body
}

func classifyOpenAIUpstreamHTTPErrorWithPolicy(
	ctx context.Context,
	settingSvc *SettingService,
	status int,
	headers http.Header,
	body []byte,
	upstreamMsg string,
) openAIUpstreamErrorClassification {
	cls := classifyOpenAIUpstreamHTTPError(status, headers, body, upstreamMsg)
	return applyOpenAIUpstreamErrorPolicy(ctx, settingSvc, cls)
}

func applyOpenAIUpstreamErrorPolicy(
	ctx context.Context,
	settingSvc *SettingService,
	cls openAIUpstreamErrorClassification,
) openAIUpstreamErrorClassification {
	if settingSvc == nil {
		return cls
	}
	policy, ok := settingSvc.ResolveUpstreamErrorPolicy(ctx, string(cls.Category))
	if !ok {
		return cls
	}
	cls.ClientStatus = policy.EffectiveStatusCode
	cls.ClientType = policy.EffectiveErrorType
	cls.ClientMessage = policy.EffectiveMessage
	cls.PolicyCustom = policy.CustomEnabled
	cls.PolicyRetryEnabled = policy.RetryEnabled
	cls.PolicyMaxRetries = policy.MaxRetries
	return cls
}

func openAIUpstreamErrorMapping(category openAIUpstreamErrorCategory) openAIUpstreamErrorClassification {
	switch category {
	case openAIUpstreamErrorAuthenticationFailed:
		return openAIUpstreamErrorClassification{
			Category:      category,
			ClientStatus:  http.StatusBadGateway,
			ClientType:    "upstream_authentication_error",
			ClientMessage: "Upstream authentication failed, please contact administrator",
		}
	case openAIUpstreamErrorBillingRequired:
		return openAIUpstreamErrorClassification{
			Category:      category,
			ClientStatus:  http.StatusBadGateway,
			ClientType:    "upstream_billing_error",
			ClientMessage: "Upstream billing or quota issue, please contact administrator",
		}
	case openAIUpstreamErrorPermissionDenied:
		return openAIUpstreamErrorClassification{
			Category:      category,
			ClientStatus:  http.StatusBadGateway,
			ClientType:    "upstream_permission_error",
			ClientMessage: "Upstream access forbidden, please contact administrator",
		}
	case openAIUpstreamErrorRateLimited:
		return openAIUpstreamErrorClassification{
			Category:      category,
			ClientStatus:  http.StatusTooManyRequests,
			ClientType:    "rate_limit_error",
			ClientMessage: "Upstream rate limit exceeded, please retry later",
			Retryable:     true,
		}
	case openAIUpstreamErrorModelUnavailable:
		return openAIUpstreamErrorClassification{
			Category:      category,
			ClientStatus:  http.StatusBadGateway,
			ClientType:    "upstream_model_error",
			ClientMessage: "Upstream model is unavailable, please contact administrator",
		}
	case openAIUpstreamErrorInvalidModel:
		return openAIUpstreamErrorClassification{
			Category:      category,
			ClientStatus:  http.StatusBadRequest,
			ClientType:    "invalid_request_error",
			ClientMessage: "Requested model is not available",
		}
	case openAIUpstreamErrorContextLengthExceeded:
		return openAIUpstreamErrorClassification{
			Category:      category,
			ClientStatus:  http.StatusBadRequest,
			ClientType:    "context_length_exceeded",
			ClientMessage: "Request exceeds upstream context length limit",
		}
	case openAIUpstreamErrorContentPolicy:
		return openAIUpstreamErrorClassification{
			Category:      category,
			ClientStatus:  http.StatusBadRequest,
			ClientType:    "content_policy_error",
			ClientMessage: "Request was rejected by upstream content policy",
		}
	case openAIUpstreamErrorTimeout:
		return openAIUpstreamErrorClassification{
			Category:      category,
			ClientStatus:  http.StatusBadGateway,
			ClientType:    "upstream_timeout_error",
			ClientMessage: "Upstream request timed out, please retry later",
			Retryable:     true,
		}
	case openAIUpstreamErrorServerError:
		return openAIUpstreamErrorClassification{
			Category:      category,
			ClientStatus:  http.StatusBadGateway,
			ClientType:    "upstream_server_error",
			ClientMessage: "Upstream service temporarily unavailable, please retry later",
			Retryable:     true,
		}
	case openAIUpstreamErrorTransport:
		return openAIUpstreamErrorClassification{
			Category:      category,
			ClientStatus:  http.StatusBadGateway,
			ClientType:    "upstream_transport_error",
			ClientMessage: "Upstream connection failed, please retry later",
			Retryable:     true,
		}
	case openAIUpstreamErrorBadRequest:
		return openAIUpstreamErrorClassification{
			Category:      category,
			ClientStatus:  http.StatusBadGateway,
			ClientType:    "upstream_error",
			ClientMessage: "Upstream rejected the request",
		}
	default:
		return openAIUpstreamErrorClassification{
			Category:      openAIUpstreamErrorUnknown,
			ClientStatus:  http.StatusBadGateway,
			ClientType:    "upstream_error",
			ClientMessage: "Upstream request failed",
		}
	}
}

func openAIUpstreamErrorSearchText(body []byte, upstreamMsg string) string {
	parts := []string{upstreamMsg}
	for _, path := range []string{
		"error.code",
		"error.type",
		"error.message",
		"error.param",
		"code",
		"type",
		"message",
		"detail",
	} {
		if value := strings.TrimSpace(gjson.GetBytes(body, path).String()); value != "" {
			parts = append(parts, value)
		}
	}
	return strings.ToLower(strings.Join(parts, " "))
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}
