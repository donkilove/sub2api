package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUniFedSyntheticEmailAndSubjectValidation(t *testing.T) {
	require.Equal(t, "unifed-abc_123@unifed-connect.invalid", unifedSyntheticEmail("abc_123"))
	require.True(t, isSafeUniFedSubject("abc-123.X_y"))
	require.False(t, isSafeUniFedSubject("abc@123"))
	require.False(t, isSafeUniFedSubject(""))
}

func TestUniFedOAuthStartRedirectsToMiAuthAndSetsCookies(t *testing.T) {
	handler := newUniFedOAuthTestHandler(t, config.UniFedConnectConfig{
		Enabled:     true,
		InstanceURL: "https://dc.hhhl.cc",
		RedirectURL: "https://app.example.com/api/v1/auth/oauth/unifed/callback",
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/unifed/start?redirect=/dashboard", nil)

	handler.UniFedOAuthStart(c)

	require.Equal(t, http.StatusFound, rec.Code)
	location := rec.Header().Get("Location")
	parsed, err := url.Parse(location)
	require.NoError(t, err)
	require.Equal(t, "https", parsed.Scheme)
	require.Equal(t, "dc.hhhl.cc", parsed.Host)
	require.Contains(t, parsed.Path, "/miauth/")
	require.Equal(t, "Sub2API", parsed.Query().Get("name"))
	require.Equal(t, "read:account", parsed.Query().Get("permission"))
	require.Contains(t, parsed.Query().Get("callback"), "/api/v1/auth/oauth/unifed/callback")

	require.NotNil(t, findCookie(rec.Result().Cookies(), unifedOAuthStateCookieName))
	require.NotNil(t, findCookie(rec.Result().Cookies(), unifedOAuthRedirectCookie))
	require.NotNil(t, findCookie(rec.Result().Cookies(), unifedOAuthMiAuthSessionName))
	require.NotNil(t, findCookie(rec.Result().Cookies(), oauthPendingBrowserCookieName))
}

func TestUniFedOAuthBindStartSetsBindCookie(t *testing.T) {
	handler := newUniFedOAuthTestHandler(t, config.UniFedConnectConfig{
		Enabled:     true,
		InstanceURL: "https://dc.hhhl.cc",
		RedirectURL: "https://app.example.com/api/v1/auth/oauth/unifed/callback",
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/unifed/bind/start?intent=bind_current_user&redirect=/settings/profile", nil)
	c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 42})

	handler.UniFedOAuthStart(c)

	require.Equal(t, http.StatusFound, rec.Code)
	intentCookie := findCookie(rec.Result().Cookies(), unifedOAuthIntentCookieName)
	require.NotNil(t, intentCookie)
	require.Equal(t, oauthIntentBindCurrentUser, decodeCookieValueForTest(t, intentCookie.Value))
	require.NotNil(t, findCookie(rec.Result().Cookies(), unifedOAuthBindUserCookieName))
}

func TestUniFedExchangeMiAuthReadsUserFromCheckResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/miauth/session-123/check", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"token":"token-1","user":{"id":"user-1","username":"alice","name":"Alice","avatarUrl":"https://cdn.example/a.png"}}`))
	}))
	defer upstream.Close()

	token, user, err := unifedExchangeMiAuth(context.Background(), upstream.URL, "session-123")

	require.NoError(t, err)
	require.Equal(t, "token-1", token.AccessToken)
	require.Equal(t, "user-1", user.ID)
	require.Equal(t, "alice", user.Username)
	require.Equal(t, "Alice", user.DisplayName)
	require.Equal(t, "https://cdn.example/a.png", user.AvatarURL)
}

func TestUniFedOAuthCallbackRejectsInvalidState(t *testing.T) {
	handler := newUniFedOAuthTestHandler(t, config.UniFedConnectConfig{
		Enabled:     true,
		InstanceURL: "https://dc.hhhl.cc",
		RedirectURL: "https://app.example.com/api/v1/auth/oauth/unifed/callback",
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/unifed/callback?session=s1&state=bad", nil)
	req.AddCookie(encodedCookie(unifedOAuthStateCookieName, "expected"))
	req.AddCookie(encodedCookie(unifedOAuthMiAuthSessionName, "s1"))
	c.Request = req

	handler.UniFedOAuthCallback(c)

	require.Equal(t, http.StatusFound, rec.Code)
	require.Contains(t, rec.Header().Get("Location"), "error=invalid_state")
}

func newUniFedOAuthTestHandler(t *testing.T, cfg config.UniFedConnectConfig) *AuthHandler {
	t.Helper()
	return &AuthHandler{
		cfg: &config.Config{
			JWT: config.JWTConfig{
				Secret:                   "test-secret",
				ExpireHour:               1,
				AccessTokenExpireMinutes: 60,
				RefreshTokenExpireDays:   7,
			},
			UniFed: cfg,
		},
	}
}
