package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newUniFedSettingsHandler() (*SettingHandler, *settingHandlerRepoStub) {
	repo := &settingHandlerRepoStub{values: map[string]string{
		service.SettingKeyPromoCodeEnabled:                     "true",
		service.SettingKeyUniFedConnectEnabled:                 "false",
		service.SettingKeyUniFedConnectInstanceURL:             "https://dc.hhhl.cc",
		service.SettingKeyUniFedConnectRedirectURL:             "",
		service.SettingKeyAuthSourceDefaultUniFedBalance:       "0",
		service.SettingKeyAuthSourceDefaultUniFedConcurrency:   "5",
		service.SettingKeyAuthSourceDefaultUniFedSubscriptions: "[]",
		service.SettingKeyAuthSourceDefaultUniFedGrantOnSignup: "false",
	}}
	svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil, nil)
	return handler, repo
}

func TestUpdateSettingsPersistsUniFedConfigAndDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, repo := newUniFedSettingsHandler()

	body := map[string]any{
		"promo_code_enabled":                             true,
		"unifed_connect_enabled":                         true,
		"unifed_connect_instance_url":                    "https://misskey.example",
		"unifed_connect_redirect_url":                    "https://app.example.com/api/v1/auth/oauth/unifed/callback",
		"auth_source_default_unifed_balance":             3.5,
		"auth_source_default_unifed_concurrency":         7,
		"auth_source_default_unifed_grant_on_signup":     true,
		"auth_source_default_unifed_grant_on_first_bind": true,
	}
	rawBody, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(rawBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateSettings(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "true", repo.values[service.SettingKeyUniFedConnectEnabled])
	require.Equal(t, "https://misskey.example", repo.values[service.SettingKeyUniFedConnectInstanceURL])
	require.Equal(t, "https://app.example.com/api/v1/auth/oauth/unifed/callback", repo.values[service.SettingKeyUniFedConnectRedirectURL])
	require.Equal(t, "3.50000000", repo.values[service.SettingKeyAuthSourceDefaultUniFedBalance])
	require.Equal(t, "7", repo.values[service.SettingKeyAuthSourceDefaultUniFedConcurrency])
	require.Equal(t, "true", repo.values[service.SettingKeyAuthSourceDefaultUniFedGrantOnSignup])
	require.Equal(t, "true", repo.values[service.SettingKeyAuthSourceDefaultUniFedGrantOnFirstBind])
}

func TestUpdateSettingsRejectsEnabledUniFedWithInvalidURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := newUniFedSettingsHandler()

	body := map[string]any{
		"promo_code_enabled":          true,
		"unifed_connect_enabled":      true,
		"unifed_connect_instance_url": "dc.hhhl.cc",
		"unifed_connect_redirect_url": "https://app.example.com/api/v1/auth/oauth/unifed/callback",
	}
	rawBody, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(rawBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateSettings(c)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
