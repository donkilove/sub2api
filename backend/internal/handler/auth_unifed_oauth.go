package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/imroc/req/v3"
	"github.com/tidwall/gjson"
)

const (
	unifedOAuthCookiePath         = "/api/v1/auth/oauth/unifed"
	unifedOAuthStateCookieName    = "unifed_oauth_state"
	unifedOAuthRedirectCookie     = "unifed_oauth_redirect"
	unifedOAuthIntentCookieName   = "unifed_oauth_intent"
	unifedOAuthBindUserCookieName = "unifed_oauth_bind_user"
	unifedOAuthMiAuthSessionName  = "unifed_miauth_session"
	unifedOAuthCookieMaxAgeSec    = 10 * 60 // 10 minutes
	unifedOAuthDefaultRedirectTo  = "/dashboard"
	unifedOAuthDefaultFrontendCB  = "/auth/unifed/callback"
	unifedOAuthMaxSubjectLen      = 64 - len("unifed-")
)

type unifedTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type unifedUserInfo struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Name        string `json:"name,omitempty"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

// UniFedOAuthStart 启动 Universe Federation (Sharkey MiAuth) OAuth 登录流程。
// GET /api/v1/auth/oauth/unifed/start?redirect=/dashboard
func (h *AuthHandler) UniFedOAuthStart(c *gin.Context) {
	cfg, err := h.getUniFedOAuthConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	state, err := generateUniFedState()
	if err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_STATE_GEN_FAILED", "failed to generate oauth state").WithCause(err))
		return
	}

	redirectTo := sanitizeFrontendRedirectPath(c.Query("redirect"))
	if redirectTo == "" {
		redirectTo = unifedOAuthDefaultRedirectTo
	}

	browserSessionKey, err := generateOAuthPendingBrowserSession()
	if err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_BROWSER_SESSION_GEN_FAILED", "failed to generate oauth browser session").WithCause(err))
		return
	}

	sessionUUID, err := generateUniFedSessionUUID()
	if err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_SESSION_GEN_FAILED", "failed to generate miauth session").WithCause(err))
		return
	}

	secureCookie := isRequestHTTPS(c)
	unifedSetCookie(c, unifedOAuthStateCookieName, encodeCookieValue(state), unifedOAuthCookieMaxAgeSec, secureCookie)
	unifedSetCookie(c, unifedOAuthRedirectCookie, encodeCookieValue(redirectTo), unifedOAuthCookieMaxAgeSec, secureCookie)
	intent := normalizeOAuthIntent(c.Query("intent"))
	unifedSetCookie(c, unifedOAuthIntentCookieName, encodeCookieValue(intent), unifedOAuthCookieMaxAgeSec, secureCookie)
	captureOAuthPromoCode(c, secureCookie)
	setOAuthPendingBrowserCookie(c, browserSessionKey, secureCookie)
	clearOAuthPendingSessionCookie(c, secureCookie)
	if intent == oauthIntentBindCurrentUser {
		bindCookieValue, err := h.buildOAuthBindUserCookieFromContext(c)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		unifedSetCookie(c, unifedOAuthBindUserCookieName, encodeCookieValue(bindCookieValue), unifedOAuthCookieMaxAgeSec, secureCookie)
	} else {
		unifedClearCookie(c, unifedOAuthBindUserCookieName, secureCookie)
	}

	instanceURL := strings.TrimRight(cfg.InstanceURL, "/")
	callbackURL, err := url.Parse(strings.TrimSpace(cfg.RedirectURL))
	if err != nil || callbackURL.String() == "" {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_REDIRECT_URL_INVALID", "unifed redirect_url is invalid"))
		return
	}
	callbackQuery := callbackURL.Query()
	callbackQuery.Set("state", state)
	callbackURL.RawQuery = callbackQuery.Encode()

	miAuthURL := fmt.Sprintf("%s/miauth/%s?name=%s&callback=%s&permission=read:account",
		instanceURL,
		sessionUUID,
		url.QueryEscape("Sub2API"),
		url.QueryEscape(callbackURL.String()),
	)

	// 将 session UUID 也存到 cookie 用于 callback 验证
	unifedSetCookie(c, unifedOAuthMiAuthSessionName, encodeCookieValue(sessionUUID), unifedOAuthCookieMaxAgeSec, secureCookie)

	c.Redirect(http.StatusFound, miAuthURL)
}

// UniFedOAuthCallback 处理 Universe Federation MiAuth 回调。
// GET /api/v1/auth/oauth/unifed/callback?session=...&state=...
func (h *AuthHandler) UniFedOAuthCallback(c *gin.Context) {
	cfg, cfgErr := h.getUniFedOAuthConfig(c.Request.Context())
	if cfgErr != nil {
		response.ErrorFrom(c, cfgErr)
		return
	}

	frontendCallback := unifedOAuthDefaultFrontendCB

	sessionUUID := strings.TrimSpace(c.Query("session"))
	state := strings.TrimSpace(c.Query("state"))

	if providerErr := strings.TrimSpace(c.Query("error")); providerErr != "" {
		redirectOAuthError(c, frontendCallback, "provider_error", providerErr, c.Query("error_description"))
		return
	}
	if sessionUUID == "" || state == "" {
		redirectOAuthError(c, frontendCallback, "missing_params", "missing session/state", "")
		return
	}

	secureCookie := isRequestHTTPS(c)
	defer func() {
		unifedClearCookie(c, unifedOAuthStateCookieName, secureCookie)
		unifedClearCookie(c, unifedOAuthRedirectCookie, secureCookie)
		unifedClearCookie(c, unifedOAuthIntentCookieName, secureCookie)
		unifedClearCookie(c, unifedOAuthBindUserCookieName, secureCookie)
		unifedClearCookie(c, unifedOAuthMiAuthSessionName, secureCookie)
		clearOAuthPromoCodeCookie(c, secureCookie)
	}()

	expectedState, err := readCookieDecoded(c, unifedOAuthStateCookieName)
	if err != nil || expectedState == "" || state != expectedState {
		redirectOAuthError(c, frontendCallback, "invalid_state", "invalid oauth state", "")
		return
	}
	expectedSession, err := readCookieDecoded(c, unifedOAuthMiAuthSessionName)
	if err != nil || expectedSession == "" || sessionUUID != expectedSession {
		redirectOAuthError(c, frontendCallback, "invalid_state", "invalid miauth session", "")
		return
	}

	redirectTo, _ := readCookieDecoded(c, unifedOAuthRedirectCookie)
	redirectTo = sanitizeFrontendRedirectPath(redirectTo)
	if redirectTo == "" {
		redirectTo = unifedOAuthDefaultRedirectTo
	}
	browserSessionKey, _ := readOAuthPendingBrowserCookie(c)
	if strings.TrimSpace(browserSessionKey) == "" {
		redirectOAuthError(c, frontendCallback, "missing_browser_session", "missing oauth browser session", "")
		return
	}
	intent, _ := readCookieDecoded(c, unifedOAuthIntentCookieName)
	intent = normalizeOAuthIntent(intent)

	// 通过 MiAuth session 获取 token 和用户信息
	instanceURL := strings.TrimRight(cfg.InstanceURL, "/")
	_, userInfo, err := unifedExchangeMiAuth(c.Request.Context(), instanceURL, sessionUUID)
	if err != nil {
		log.Printf("[UniFed OAuth] MiAuth exchange failed: %v", err)
		redirectOAuthError(c, frontendCallback, "token_exchange_failed", "failed to authenticate with universe federation", "")
		return
	}

	// 构造 identity
	subject := strings.TrimSpace(userInfo.ID)
	if subject == "" {
		redirectOAuthError(c, frontendCallback, "missing_subject", "missing user id from provider", "")
		return
	}
	if !isSafeUniFedSubject(subject) {
		redirectOAuthError(c, frontendCallback, "invalid_subject", "invalid user id from provider", "")
		return
	}

	email := unifedSyntheticEmail(subject)
	username := strings.TrimSpace(userInfo.Username)
	if username == "" {
		username = "unifed_" + subject
	}
	displayName := strings.TrimSpace(firstNonEmpty(
		userInfo.DisplayName,
		userInfo.Name,
		username,
	))
	if displayName == "" {
		displayName = username
	}

	identityKey := service.PendingAuthIdentityKey{
		ProviderType:    "unifed",
		ProviderKey:     "unifed",
		ProviderSubject: subject,
	}

	upstreamClaims := map[string]any{
		"email":                  email,
		"username":               username,
		"subject":                subject,
		"suggested_display_name": displayName,
		"suggested_avatar_url":   strings.TrimSpace(userInfo.AvatarURL),
	}

	if intent == oauthIntentBindCurrentUser {
		targetUserID, err := h.readOAuthBindUserIDFromCookie(c, unifedOAuthBindUserCookieName)
		if err != nil {
			redirectOAuthError(c, frontendCallback, "invalid_state", "invalid oauth bind target", "")
			return
		}
		if err := h.createOAuthPendingSession(c, oauthPendingSessionPayload{
			Intent:                 oauthIntentBindCurrentUser,
			Identity:               identityKey,
			TargetUserID:           &targetUserID,
			ResolvedEmail:          email,
			RedirectTo:             redirectTo,
			BrowserSessionKey:      browserSessionKey,
			UpstreamIdentityClaims: upstreamClaims,
			CompletionResponse: map[string]any{
				"redirect": redirectTo,
			},
		}); err != nil {
			redirectOAuthError(c, frontendCallback, "session_error", "failed to continue oauth bind", "")
			return
		}
		redirectToFrontendCallback(c, frontendCallback)
		return
	}

	// 检查是否已有绑定 identity
	existingIdentityUser, err := h.findOAuthIdentityUser(c.Request.Context(), identityKey)
	if err != nil {
		redirectOAuthError(c, frontendCallback, "session_error", infraerrors.Reason(err), infraerrors.Message(err))
		return
	}
	if existingIdentityUser != nil {
		if err := h.createOAuthPendingSession(c, oauthPendingSessionPayload{
			Intent:                 oauthIntentLogin,
			Identity:               identityKey,
			TargetUserID:           &existingIdentityUser.ID,
			ResolvedEmail:          existingIdentityUser.Email,
			RedirectTo:             redirectTo,
			BrowserSessionKey:      browserSessionKey,
			UpstreamIdentityClaims: upstreamClaims,
			CompletionResponse: map[string]any{
				"redirect": redirectTo,
			},
		}); err != nil {
			redirectOAuthError(c, frontendCallback, "session_error", "failed to continue oauth login", "")
			return
		}
		redirectToFrontendCallback(c, frontendCallback)
		return
	}

	// 新用户：尝试快速注册或进入 choice 页面
	if err := h.ensureBackendModeAllowsNewUserLogin(c.Request.Context()); err != nil {
		redirectOAuthError(c, frontendCallback, "login_blocked", infraerrors.Reason(err), infraerrors.Message(err))
		return
	}

	forceEmailOnSignup := h.isForceEmailOnThirdPartySignup(c.Request.Context())

	// 尝试直接注册（当不需要邮箱验证/邀请码时）
	if !forceEmailOnSignup {
		securityEventBaseline := time.Now().UTC()
		tokenPair, user, err := h.authService.LoginOrRegisterOAuthWithTokenPairAndPromoCode(
			c.Request.Context(),
			email,
			username,
			"",
			"",
			readOAuthPromoCode(c),
			"unifed",
		)
		if err == nil {
			if err := applyPendingOAuthBinding(
				c.Request.Context(),
				h.entClient(),
				h.authService,
				h.userService,
				&dbent.PendingAuthSession{
					Intent:                 oauthIntentLogin,
					ProviderType:           identityKey.ProviderType,
					ProviderKey:            identityKey.ProviderKey,
					ProviderSubject:        identityKey.ProviderSubject,
					ResolvedEmail:          email,
					UpstreamIdentityClaims: upstreamClaims,
				},
				nil,
				&user.ID,
				true,
				false,
			); err != nil {
				log.Printf("[UniFed OAuth] bind identity failed: %v", err)
				redirectOAuthError(c, frontendCallback, "session_error", "failed to bind oauth identity", "")
				return
			}
			h.authService.RecordSuccessfulLogin(c.Request.Context(), user.ID)
			h.recordSecurityEvent(c, service.UserSecurityEventInput{
				UserID:    user.ID,
				Email:     user.Email,
				EventType: oauthSecurityEventType(user, securityEventBaseline),
				Provider:  "unifed",
				Success:   true,
			})
			clearOAuthPendingSessionCookie(c, secureCookie)
			clearOAuthPendingBrowserCookie(c, secureCookie)
			redirectOAuthTokenPair(c, frontendCallback, tokenPair, redirectTo)
			return
		}
		if !errors.Is(err, service.ErrOAuthInvitationRequired) {
			redirectOAuthError(c, frontendCallback, "session_error", infraerrors.Reason(err), infraerrors.Message(err))
			return
		}
	}

	// 需要 choice 页面
	if err := h.createUniFedOAuthChoicePendingSession(
		c,
		identityKey,
		email,
		email,
		redirectTo,
		browserSessionKey,
		upstreamClaims,
		"",
		nil,
		forceEmailOnSignup,
	); err != nil {
		redirectOAuthError(c, frontendCallback, "session_error", "failed to continue oauth login", "")
		return
	}
	redirectToFrontendCallback(c, frontendCallback)
}

// createUniFedOAuthChoicePendingSession 创建 Universe Federation 的 choice pending session。
func (h *AuthHandler) createUniFedOAuthChoicePendingSession(
	c *gin.Context,
	identity service.PendingAuthIdentityKey,
	suggestedEmail string,
	resolvedEmail string,
	redirectTo string,
	browserSessionKey string,
	upstreamClaims map[string]any,
	_ string,
	_ *any,
	forceEmailOnSignup bool,
) error {
	suggestionEmail := strings.TrimSpace(suggestedEmail)
	canonicalEmail := strings.TrimSpace(resolvedEmail)
	if suggestionEmail == "" {
		suggestionEmail = canonicalEmail
	}

	completionResponse := map[string]any{
		"step":                      oauthPendingChoiceStep,
		"adoption_required":         true,
		"redirect":                  strings.TrimSpace(redirectTo),
		"email":                     suggestionEmail,
		"resolved_email":            canonicalEmail,
		"existing_account_email":    "",
		"existing_account_bindable": false,
		"create_account_allowed":    true,
		"force_email_on_signup":     forceEmailOnSignup,
		"choice_reason":             "third_party_signup",
	}
	if forceEmailOnSignup {
		completionResponse["choice_reason"] = "force_email_on_signup"
	}

	return h.createOAuthPendingSession(c, oauthPendingSessionPayload{
		Intent:                 oauthIntentLogin,
		Identity:               identity,
		ResolvedEmail:          suggestionEmail,
		RedirectTo:             redirectTo,
		BrowserSessionKey:      browserSessionKey,
		UpstreamIdentityClaims: upstreamClaims,
		CompletionResponse:     completionResponse,
	})
}

type completeUniFedOAuthRequest struct {
	InvitationCode   string `json:"invitation_code" binding:"required"`
	AffCode          string `json:"aff_code,omitempty"`
	AdoptDisplayName *bool  `json:"adopt_display_name,omitempty"`
	AdoptAvatar      *bool  `json:"adopt_avatar,omitempty"`
}

// CompleteUniFedOAuthRegistration completes a pending UniFed OAuth registration.
// POST /api/v1/auth/oauth/unifed/complete-registration
func (h *AuthHandler) CompleteUniFedOAuthRegistration(c *gin.Context) {
	var req completeUniFedOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_REQUEST", "message": err.Error()})
		return
	}

	secureCookie := isRequestHTTPS(c)
	sessionToken, err := readOAuthPendingSessionCookie(c)
	if err != nil {
		clearOAuthPendingSessionCookie(c, secureCookie)
		clearOAuthPendingBrowserCookie(c, secureCookie)
		response.ErrorFrom(c, service.ErrPendingAuthSessionNotFound)
		return
	}
	browserSessionKey, err := readOAuthPendingBrowserCookie(c)
	if err != nil {
		clearOAuthPendingSessionCookie(c, secureCookie)
		clearOAuthPendingBrowserCookie(c, secureCookie)
		response.ErrorFrom(c, service.ErrPendingAuthBrowserMismatch)
		return
	}
	pendingSvc, err := h.pendingIdentityService()
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	session, err := pendingSvc.GetBrowserSession(c.Request.Context(), sessionToken, browserSessionKey)
	if err != nil {
		clearOAuthPendingSessionCookie(c, secureCookie)
		clearOAuthPendingBrowserCookie(c, secureCookie)
		response.ErrorFrom(c, err)
		return
	}
	if err := ensurePendingOAuthCompleteRegistrationSession(session); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if updatedSession, handled, err := h.legacyCompleteRegistrationSessionStatus(c, session); err != nil {
		response.ErrorFrom(c, err)
		return
	} else if handled {
		c.JSON(http.StatusOK, buildPendingOAuthSessionStatusPayload(updatedSession))
		return
	} else {
		session = updatedSession
	}
	if err := h.ensureBackendModeAllowsNewUserLogin(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	email := strings.TrimSpace(session.ResolvedEmail)
	username := pendingSessionStringValue(session.UpstreamIdentityClaims, "username")
	if email == "" || username == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("PENDING_AUTH_SESSION_INVALID", "pending auth registration context is invalid"))
		return
	}

	client := h.entClient()
	if client == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("PENDING_AUTH_NOT_READY", "pending auth service is not ready"))
		return
	}
	if err := ensurePendingOAuthRegistrationIdentityAvailable(c.Request.Context(), client, session); err != nil {
		respondPendingOAuthBindingApplyError(c, err)
		return
	}
	decision, err := h.ensurePendingOAuthAdoptionDecision(c, session.ID, oauthAdoptionDecisionRequest{
		AdoptDisplayName: req.AdoptDisplayName,
		AdoptAvatar:      req.AdoptAvatar,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	securityEventBaseline := time.Now().UTC()
	tokenPair, user, err := h.authService.LoginOrRegisterOAuthWithTokenPairAndPromoCode(
		c.Request.Context(),
		email,
		username,
		req.InvitationCode,
		req.AffCode,
		pendingOAuthPromoCode(session),
		"unifed",
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if err := applyPendingOAuthAdoptionAndConsumeSession(c.Request.Context(), client, h.authService, h.userService, session, decision, user.ID); err != nil {
		respondPendingOAuthBindingApplyError(c, err)
		return
	}
	h.authService.RecordSuccessfulLogin(c.Request.Context(), user.ID)
	h.recordSecurityEvent(c, service.UserSecurityEventInput{
		UserID:    user.ID,
		Email:     user.Email,
		EventType: oauthSecurityEventType(user, securityEventBaseline),
		Provider:  "unifed",
		Success:   true,
	})
	clearOAuthPendingSessionCookie(c, secureCookie)
	clearOAuthPendingBrowserCookie(c, secureCookie)

	c.JSON(http.StatusOK, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
		"token_type":    "Bearer",
	})
}

func (h *AuthHandler) getUniFedOAuthConfig(ctx context.Context) (config.UniFedConnectConfig, error) {
	if h != nil && h.settingSvc != nil {
		return h.settingSvc.GetUniFedConnectOAuthConfig(ctx)
	}
	if h == nil || h.cfg == nil {
		return config.UniFedConnectConfig{}, infraerrors.ServiceUnavailable("CONFIG_NOT_READY", "config not loaded")
	}
	if !h.cfg.UniFed.Enabled {
		return config.UniFedConnectConfig{}, infraerrors.NotFound("OAUTH_DISABLED", "oauth login is disabled")
	}
	return h.cfg.UniFed, nil
}

// unifedExchangeMiAuth 通过 MiAuth 协议交换 session 获取 token 和用户信息。
func unifedExchangeMiAuth(ctx context.Context, instanceURL, sessionUUID string) (*unifedTokenResponse, *unifedUserInfo, error) {
	instanceURL = strings.TrimRight(instanceURL, "/")

	client := req.C().SetTimeout(30 * time.Second)

	// Step 1: POST /api/miauth/{session}/check 换取 token
	checkURL := fmt.Sprintf("%s/api/miauth/%s/check", instanceURL, sessionUUID)
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36").
		SetBody(map[string]any{}).
		Post(checkURL)
	if err != nil {
		return nil, nil, fmt.Errorf("miauth check request failed: %w", err)
	}
	if !resp.IsSuccessState() {
		return nil, nil, fmt.Errorf("miauth check status=%d body=%s", resp.StatusCode, truncateLogValue(resp.String(), 1024))
	}

	body := resp.String()
	ok := gjson.Get(body, "ok").Bool()
	if !ok {
		return nil, nil, fmt.Errorf("miauth check returned not ok: body=%s", truncateLogValue(body, 512))
	}

	token := strings.TrimSpace(gjson.Get(body, "token").String())
	if token == "" {
		return nil, nil, fmt.Errorf("miauth check missing token")
	}

	tokenResp := &unifedTokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
	}

	// 尝试从 check 响应中直接获取 user 信息（MiAuth 可能直接返回）
	userID := strings.TrimSpace(gjson.Get(body, "user.id").String())
	if userID != "" {
		username := strings.TrimSpace(firstNonEmpty(
			gjson.Get(body, "user.username").String(),
			gjson.Get(body, "user.usernameLower").String(),
		))
		name := strings.TrimSpace(gjson.Get(body, "user.name").String())
		avatarURL := strings.TrimSpace(gjson.Get(body, "user.avatarUrl").String())

		return tokenResp, &unifedUserInfo{
			ID:          userID,
			Username:    username,
			Name:        name,
			AvatarURL:   avatarURL,
			DisplayName: name,
		}, nil
	}

	// Step 2: 如果 check 没有返回 user 信息，通过 POST /api/i 获取
	userInfo, err := unifedFetchUserInfo(ctx, instanceURL, token)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch userinfo after miauth: %w", err)
	}

	return tokenResp, userInfo, nil
}

// unifedFetchUserInfo 通过 Misskey /api/i 获取用户信息。
func unifedFetchUserInfo(ctx context.Context, instanceURL, token string) (*unifedUserInfo, error) {
	instanceURL = strings.TrimRight(instanceURL, "/")
	client := req.C().SetTimeout(30 * time.Second)

	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36").
		SetBody(map[string]string{"i": token}).
		Post(instanceURL + "/api/i")
	if err != nil {
		return nil, fmt.Errorf("request userinfo: %w", err)
	}
	if !resp.IsSuccessState() {
		return nil, fmt.Errorf("userinfo status=%d", resp.StatusCode)
	}

	body := resp.String()
	userID := strings.TrimSpace(gjson.Get(body, "id").String())
	if userID == "" {
		return nil, errors.New("userinfo missing id field")
	}

	username := strings.TrimSpace(firstNonEmpty(
		gjson.Get(body, "username").String(),
		gjson.Get(body, "usernameLower").String(),
	))
	name := strings.TrimSpace(gjson.Get(body, "name").String())
	avatarURL := strings.TrimSpace(gjson.Get(body, "avatarUrl").String())

	return &unifedUserInfo{
		ID:          userID,
		Username:    username,
		Name:        name,
		AvatarURL:   avatarURL,
		DisplayName: name,
	}, nil
}

func generateUniFedState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func generateUniFedSessionUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	uuid := hex.EncodeToString(b[:4]) + "-" +
		hex.EncodeToString(b[4:6]) + "-" +
		hex.EncodeToString(b[6:8]) + "-" +
		hex.EncodeToString(b[8:10]) + "-" +
		hex.EncodeToString(b[10:])
	return uuid, nil
}

func unifedSyntheticEmail(subject string) string {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return ""
	}
	return "unifed-" + subject + service.UniFedConnectSyntheticEmailDomain
}

func isSafeUniFedSubject(subject string) bool {
	subject = strings.TrimSpace(subject)
	if subject == "" || len(subject) > unifedOAuthMaxSubjectLen {
		return false
	}
	for _, r := range subject {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r == '_' || r == '-' || r == '.':
		default:
			return false
		}
	}
	return true
}

func unifedSetCookie(c *gin.Context, name, value string, maxAgeSec int, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     unifedOAuthCookiePath,
		MaxAge:   maxAgeSec,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func unifedClearCookie(c *gin.Context, name string, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     unifedOAuthCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
