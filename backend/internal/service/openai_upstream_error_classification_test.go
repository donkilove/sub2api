package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestClassifyOpenAIUpstreamHTTPError(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		body          []byte
		wantCategory  openAIUpstreamErrorCategory
		wantStatus    int
		wantType      string
		wantMessage   string
		wantRetryable bool
	}{
		{
			name:          "authentication failed",
			status:        http.StatusUnauthorized,
			body:          []byte(`{"error":{"code":"invalid_api_key","message":"Incorrect API key provided"}}`),
			wantCategory:  openAIUpstreamErrorAuthenticationFailed,
			wantStatus:    http.StatusBadGateway,
			wantType:      "upstream_authentication_error",
			wantMessage:   "Upstream authentication failed, please contact administrator",
			wantRetryable: false,
		},
		{
			name:          "rate limited",
			status:        http.StatusTooManyRequests,
			body:          []byte(`{"error":{"message":"Rate limit reached for requests"}}`),
			wantCategory:  openAIUpstreamErrorRateLimited,
			wantStatus:    http.StatusTooManyRequests,
			wantType:      "rate_limit_error",
			wantMessage:   "Upstream rate limit exceeded, please retry later",
			wantRetryable: true,
		},
		{
			name:          "context length exceeded",
			status:        http.StatusBadRequest,
			body:          []byte(`{"error":{"message":"This model's maximum context length is 128000 tokens"}}`),
			wantCategory:  openAIUpstreamErrorContextLengthExceeded,
			wantStatus:    http.StatusBadRequest,
			wantType:      "context_length_exceeded",
			wantMessage:   "Request exceeds upstream context length limit",
			wantRetryable: false,
		},
		{
			name:          "model unavailable",
			status:        http.StatusNotFound,
			body:          []byte(`{"error":{"message":"The model gpt-x does not exist"}}`),
			wantCategory:  openAIUpstreamErrorModelUnavailable,
			wantStatus:    http.StatusBadGateway,
			wantType:      "upstream_model_error",
			wantMessage:   "Upstream model is unavailable, please contact administrator",
			wantRetryable: false,
		},
		{
			name:          "invalid model request",
			status:        http.StatusBadRequest,
			body:          []byte(`{"error":{"type":"invalid_request_error","message":"model not found"}}`),
			wantCategory:  openAIUpstreamErrorInvalidModel,
			wantStatus:    http.StatusBadRequest,
			wantType:      "invalid_request_error",
			wantMessage:   "Requested model is not available",
			wantRetryable: false,
		},
		{
			name:          "server error",
			status:        http.StatusServiceUnavailable,
			body:          []byte(`{"error":{"message":"upstream overloaded"}}`),
			wantCategory:  openAIUpstreamErrorServerError,
			wantStatus:    http.StatusBadGateway,
			wantType:      "upstream_server_error",
			wantMessage:   "Upstream service temporarily unavailable, please retry later",
			wantRetryable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyOpenAIUpstreamHTTPError(tt.status, http.Header{}, tt.body, "")
			require.Equal(t, tt.wantCategory, got.Category)
			require.Equal(t, tt.wantStatus, got.ClientStatus)
			require.Equal(t, tt.wantType, got.ClientType)
			require.Equal(t, tt.wantMessage, got.ClientMessage)
			require.Equal(t, tt.wantRetryable, got.Retryable)
		})
	}
}

func TestOpenAIHandleErrorResponseUsesFriendlyClassification(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	svc := &OpenAIGatewayService{}
	resp := &http.Response{
		StatusCode: http.StatusUnauthorized,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":{"code":"invalid_api_key","message":"sk-test should not be echoed"}}`))),
		Header:     http.Header{"X-Request-Id": []string{"req-auth"}},
	}
	account := &Account{ID: 31, Platform: PlatformOpenAI, Name: "openai-a", Type: AccountTypeAPIKey}

	_, err := svc.handleErrorResponse(context.Background(), resp, c, account, nil)
	require.Error(t, err)
	require.Equal(t, http.StatusBadGateway, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	errField := payload["error"].(map[string]any)
	require.Equal(t, "upstream_authentication_error", errField["type"])
	require.Equal(t, "Upstream authentication failed, please contact administrator", errField["message"])
	require.NotContains(t, rec.Body.String(), "sk-test")

	events, ok := c.MustGet(OpsUpstreamErrorsKey).([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	require.Len(t, events, 1)
	require.Equal(t, "authentication_failed", events[0].Classification)
}

func TestOpenAIHandleCompatErrorResponseKeepsWriterShapeWithFriendlyMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	svc := &OpenAIGatewayService{}
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":{"message":"This request has too many tokens for the maximum context length"}}`))),
		Header:     http.Header{"X-Request-Id": []string{"req-context"}},
	}
	var gotStatus int
	var gotType, gotMsg string
	writeError := func(_ *gin.Context, statusCode int, errType, message string) {
		gotStatus, gotType, gotMsg = statusCode, errType, message
	}

	_, err := svc.handleCompatErrorResponse(resp, c, &Account{ID: 32, Platform: PlatformOpenAI, Name: "openai-b", Type: AccountTypeAPIKey}, writeError)
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, gotStatus)
	require.Equal(t, "context_length_exceeded", gotType)
	require.Equal(t, "Request exceeds upstream context length limit", gotMsg)

	events, ok := c.MustGet(OpsUpstreamErrorsKey).([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	require.Len(t, events, 1)
	require.Equal(t, "context_length_exceeded", events[0].Classification)
}

func TestOpenAIHandleErrorResponsePassthroughRuleStillWins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	respCode := http.StatusTeapot
	customMessage := "自定义上游提示"
	ruleSvc := &ErrorPassthroughService{}
	ruleSvc.setLocalCache([]*model.ErrorPassthroughRule{{
		ID:              1,
		Name:            "custom-401",
		Enabled:         true,
		Priority:        1,
		ErrorCodes:      []int{http.StatusUnauthorized},
		Keywords:        []string{"invalid_api_key"},
		MatchMode:       model.MatchModeAll,
		PassthroughCode: false,
		ResponseCode:    &respCode,
		PassthroughBody: false,
		CustomMessage:   &customMessage,
	}})
	BindErrorPassthroughService(c, ruleSvc)

	svc := &OpenAIGatewayService{}
	resp := &http.Response{
		StatusCode: http.StatusUnauthorized,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":{"code":"invalid_api_key","message":"bad key"}}`))),
		Header:     http.Header{},
	}

	_, err := svc.handleErrorResponse(context.Background(), resp, c, &Account{ID: 33, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}, nil)
	require.Error(t, err)
	require.Equal(t, http.StatusTeapot, rec.Code)
	require.Contains(t, rec.Body.String(), customMessage)
}

func TestOpenAIHandleErrorResponsePolicyRetryDoesNotRequirePoolMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	settingSvc := NewSettingService(newMemorySettingRepo(), nil)
	require.NoError(t, settingSvc.UpdateUpstreamErrorPolicy(context.Background(), "upstream_server_error", UpstreamErrorPolicyUpdate{
		RetryEnabled: policyBoolPtr(true),
		MaxRetries:   policyIntPtr(2),
	}))
	svc := &OpenAIGatewayService{settingService: settingSvc}
	resp := &http.Response{
		StatusCode: http.StatusServiceUnavailable,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":{"message":"temporarily overloaded"}}`))),
		Header:     http.Header{},
	}
	account := &Account{ID: 34, Platform: PlatformOpenAI, Name: "openai-c", Type: AccountTypeAPIKey}

	_, err := svc.handleErrorResponse(context.Background(), resp, c, account, nil)
	require.Error(t, err)

	var failoverErr *UpstreamFailoverError
	require.True(t, errors.As(err, &failoverErr))
	require.True(t, failoverErr.RetryableOnSameAccount)
	require.Equal(t, 2, failoverErr.MaxSameAccountRetries)
}

func TestHandleOpenAIUpstreamTransportErrorPolicyRetryDoesNotRequirePoolMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	settingSvc := NewSettingService(newMemorySettingRepo(), nil)
	require.NoError(t, settingSvc.UpdateUpstreamErrorPolicy(context.Background(), "transport_error", UpstreamErrorPolicyUpdate{
		RetryEnabled: policyBoolPtr(true),
		MaxRetries:   policyIntPtr(2),
	}))
	svc := &OpenAIGatewayService{settingService: settingSvc}
	account := &Account{ID: 35, Platform: PlatformOpenAI, Name: "openai-transport", Type: AccountTypeAPIKey}

	err := svc.handleOpenAIUpstreamTransportError(context.Background(), c, account, errors.New("temporary EOF"), false)
	require.Error(t, err)

	var failoverErr *UpstreamFailoverError
	require.True(t, errors.As(err, &failoverErr))
	require.True(t, failoverErr.RetryableOnSameAccount)
	require.Equal(t, 2, failoverErr.MaxSameAccountRetries)
	require.Equal(t, 0, rec.Body.Len())
}
