# UniFed MiAuth Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `UniFed / Universe Federation / Sharkey-Misskey MiAuth` 接入现有第三方登录体系，并以测试驱动方式收口当前半成品。

**Architecture:** UniFed 作为独立 provider，provider 标识固定为 `unifed`，复用现有 pending OAuth、auth identity、默认权益和前端 callback 流程。MiAuth 协议差异封装在 `backend/internal/handler/auth_unifed_oauth.go`，系统设置通过 `SettingService` 暴露，前端只消费公开开关和统一 pending OAuth API。

**Tech Stack:** Go 1.26、Gin、Ent、testify、httptest、Vue 3、Vite、Vitest、Vue Test Utils。

---

## File Structure

- Modify: `backend/internal/config/config.go` — 定义 `UniFedConnectConfig`，提供默认实例地址来源。
- Modify: `backend/internal/service/domain_constants.go` — 增加 UniFed 设置键、默认权益键、合成邮箱域。
- Modify: `backend/internal/service/setting_service.go` — 初始化默认设置、解析系统设置、公开设置、最终 UniFed 配置。
- Modify: `backend/internal/service/settings_view.go` — 扩展服务层 settings DTO。
- Modify: `backend/internal/handler/dto/settings.go` — 扩展 handler DTO。
- Modify: `backend/internal/handler/admin/setting_handler.go` — 后台 settings API 保存、校验、返回 UniFed 设置和默认权益。
- Modify: `backend/ent/schema/auth_identity.go`、`backend/ent/schema/user.go` — 允许 `unifed` provider 和 signup source。
- Modify: `backend/internal/service/auth_service.go`、`backend/internal/service/user_service.go`、`backend/internal/service/admin_service.go`、`backend/internal/handler/user_handler.go` — 接入注册来源、保留邮箱、绑定状态、管理员 provider 过滤。
- Create/Modify: `backend/internal/handler/auth_unifed_oauth.go` — UniFed start/callback/MiAuth exchange/complete registration。
- Modify: `backend/internal/server/routes/auth.go` — 注册 UniFed 路由。
- Modify: `backend/internal/server/api_contract_test.go` — 更新 API contract。
- Modify/Create tests under `backend/internal/...` — 覆盖配置、handler、service、contract。
- Modify: `frontend/src/api/admin/settings.ts`、`frontend/src/api/auth.ts`、`frontend/src/types/index.ts`、`frontend/src/stores/app.ts` — 扩展类型和 API。
- Create/Modify: `frontend/src/components/auth/UniFedOAuthSection.vue`、`frontend/src/views/auth/UniFedCallbackView.vue` — 登录按钮和 callback 页面。
- Modify: `frontend/src/views/auth/LoginView.vue`、`frontend/src/views/auth/RegisterView.vue`、`frontend/src/views/auth/EmailVerifyView.vue`、`frontend/src/router/index.ts` — 入口、路由和 pending flow 返回路径。
- Modify: `frontend/src/views/user/ProfileView.vue`、`frontend/src/components/user/profile/*.vue` — 个人资料绑定状态。
- Modify: `frontend/src/views/admin/SettingsView.vue`、`frontend/src/i18n/locales/en.ts`、`frontend/src/i18n/locales/zh.ts` — 后台设置 UI 与文案。
- Modify/Create frontend tests under `frontend/src/**/__tests__` — 覆盖 settings helper、路由、入口显示、callback 行为、profile binding。

---

### Task 1: 后端 UniFed 设置默认值与公开设置

**Files:**
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/service/domain_constants.go`
- Modify: `backend/internal/service/setting_service.go`
- Modify: `backend/internal/service/settings_view.go`
- Modify: `backend/internal/handler/dto/settings.go`
- Test: `backend/internal/service/setting_service_unifed_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/service/setting_service_unifed_test.go`:

```go
package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type unifedSettingsRepoStub struct {
	values map[string]string
}

func (r *unifedSettingsRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	if value, ok := r.values[key]; ok {
		return &Setting{Key: key, Value: value}, nil
	}
	return nil, ErrSettingNotFound
}

func (r *unifedSettingsRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if value, ok := r.values[key]; ok {
		return value, nil
	}
	return "", nil
}

func (r *unifedSettingsRepoStub) Set(ctx context.Context, key string, value string) error {
	if r.values == nil {
		r.values = map[string]string{}
	}
	r.values[key] = value
	return nil
}

func (r *unifedSettingsRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (r *unifedSettingsRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	if r.values == nil {
		r.values = map[string]string{}
	}
	for key, value := range settings {
		r.values[key] = value
	}
	return nil
}

func (r *unifedSettingsRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(r.values))
	for key, value := range r.values {
		out[key] = value
	}
	return out, nil
}

func (r *unifedSettingsRepoStub) Delete(ctx context.Context, key string) error {
	delete(r.values, key)
	return nil
}

func TestGetUniFedConnectOAuthConfigUsesDefaultInstanceAndDBOverride(t *testing.T) {
	repo := &unifedSettingsRepoStub{values: map[string]string{
		SettingKeyUniFedConnectEnabled:     "true",
		SettingKeyUniFedConnectRedirectURL: "https://app.example.com/api/v1/auth/oauth/unifed/callback",
	}}
	svc := &SettingService{
		cfg: &config.Config{
			UniFed: config.UniFedConnectConfig{
				Enabled:     false,
				InstanceURL: "https://dc.hhhl.cc",
				RedirectURL: "",
			},
		},
		settingRepo: repo,
	}

	cfg, err := svc.GetUniFedConnectOAuthConfig(context.Background())

	require.NoError(t, err)
	require.True(t, cfg.Enabled)
	require.Equal(t, "https://dc.hhhl.cc", cfg.InstanceURL)
	require.Equal(t, "https://app.example.com/api/v1/auth/oauth/unifed/callback", cfg.RedirectURL)

	repo.values[SettingKeyUniFedConnectInstanceURL] = "https://misskey.example"
	cfg, err = svc.GetUniFedConnectOAuthConfig(context.Background())

	require.NoError(t, err)
	require.Equal(t, "https://misskey.example", cfg.InstanceURL)
}

func TestGetUniFedConnectOAuthConfigRejectsDisabledAndInvalidURLs(t *testing.T) {
	svc := &SettingService{
		cfg: &config.Config{
			UniFed: config.UniFedConnectConfig{
				Enabled:     true,
				InstanceURL: "https://dc.hhhl.cc",
				RedirectURL: "https://app.example.com/api/v1/auth/oauth/unifed/callback",
			},
		},
		settingRepo: &unifedSettingsRepoStub{values: map[string]string{
			SettingKeyUniFedConnectEnabled: "false",
		}},
	}

	_, err := svc.GetUniFedConnectOAuthConfig(context.Background())
	require.Error(t, err)
	require.Equal(t, "OAUTH_DISABLED", infraerrors.Reason(err))

	svc.settingRepo = &unifedSettingsRepoStub{values: map[string]string{
		SettingKeyUniFedConnectEnabled:     "true",
		SettingKeyUniFedConnectInstanceURL: "dc.hhhl.cc",
		SettingKeyUniFedConnectRedirectURL: "https://app.example.com/api/v1/auth/oauth/unifed/callback",
	}}
	_, err = svc.GetUniFedConnectOAuthConfig(context.Background())
	require.Error(t, err)
	require.Equal(t, "OAUTH_CONFIG_INVALID", infraerrors.Reason(err))
}

func TestPublicSettingsExposeUniFedOAuthEnabled(t *testing.T) {
	svc := &SettingService{
		cfg: &config.Config{
			UniFed: config.UniFedConnectConfig{
				Enabled:     false,
				InstanceURL: "https://dc.hhhl.cc",
			},
		},
		settingRepo: &unifedSettingsRepoStub{values: map[string]string{
			SettingKeyUniFedConnectEnabled: "true",
		}},
	}

	settings, err := svc.GetPublicSettings(context.Background())

	require.NoError(t, err)
	require.True(t, settings.UniFedOAuthEnabled)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd backend && go test ./internal/service -run 'TestGetUniFedConnectOAuthConfig|TestPublicSettingsExposeUniFedOAuthEnabled' -count=1
```

Expected: FAIL because `UniFedConnectConfig` / setting keys / public field / `GetUniFedConnectOAuthConfig` are missing or incomplete.

- [ ] **Step 3: Write minimal implementation**

Implement:

```go
// backend/internal/config/config.go
type Config struct {
	// existing fields...
	UniFed UniFedConnectConfig `mapstructure:"unifed_connect"`
}

type UniFedConnectConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	InstanceURL string `mapstructure:"instance_url"`
	RedirectURL string `mapstructure:"redirect_url"`
}
```

Add constants:

```go
const UniFedConnectSyntheticEmailDomain = "@unifed-connect.invalid"

const (
	SettingKeyUniFedConnectEnabled     = "unifed_connect_enabled"
	SettingKeyUniFedConnectInstanceURL = "unifed_connect_instance_url"
	SettingKeyUniFedConnectRedirectURL = "unifed_connect_redirect_url"
)
```

In `InitializeDefaultSettings`, set:

```go
SettingKeyUniFedConnectEnabled:     strconv.FormatBool(s.cfg.UniFed.Enabled),
SettingKeyUniFedConnectInstanceURL: firstNonEmpty(strings.TrimSpace(s.cfg.UniFed.InstanceURL), "https://dc.hhhl.cc"),
SettingKeyUniFedConnectRedirectURL: strings.TrimSpace(s.cfg.UniFed.RedirectURL),
```

In `GetPublicSettings`, include `SettingKeyUniFedConnectEnabled` in the loaded keys and populate:

```go
unifedEnabled := false
if raw, ok := settings[SettingKeyUniFedConnectEnabled]; ok {
	unifedEnabled = raw == "true"
} else {
	unifedEnabled = s.cfg != nil && s.cfg.UniFed.Enabled
}
```

In public settings structs add:

```go
UniFedOAuthEnabled bool `json:"unifed_oauth_enabled"`
```

Implement:

```go
func (s *SettingService) GetUniFedConnectOAuthConfig(ctx context.Context) (config.UniFedConnectConfig, error) {
	if s == nil || s.cfg == nil {
		return config.UniFedConnectConfig{}, infraerrors.ServiceUnavailable("CONFIG_NOT_READY", "config not loaded")
	}
	effective := s.cfg.UniFed
	if strings.TrimSpace(effective.InstanceURL) == "" {
		effective.InstanceURL = "https://dc.hhhl.cc"
	}

	settings, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyUniFedConnectEnabled,
		SettingKeyUniFedConnectInstanceURL,
		SettingKeyUniFedConnectRedirectURL,
	})
	if err != nil {
		return config.UniFedConnectConfig{}, fmt.Errorf("get unifed connect settings: %w", err)
	}
	if raw, ok := settings[SettingKeyUniFedConnectEnabled]; ok {
		effective.Enabled = raw == "true"
	}
	if value, ok := settings[SettingKeyUniFedConnectInstanceURL]; ok && strings.TrimSpace(value) != "" {
		effective.InstanceURL = strings.TrimSpace(value)
	}
	if value, ok := settings[SettingKeyUniFedConnectRedirectURL]; ok && strings.TrimSpace(value) != "" {
		effective.RedirectURL = strings.TrimSpace(value)
	}
	if !effective.Enabled {
		return config.UniFedConnectConfig{}, infraerrors.NotFound("OAUTH_DISABLED", "oauth login is disabled")
	}
	if err := config.ValidateAbsoluteHTTPURL(strings.TrimRight(effective.InstanceURL, "/")); err != nil {
		return config.UniFedConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "unifed instance url must be an absolute http(s) URL").WithCause(err)
	}
	if err := config.ValidateAbsoluteHTTPURL(strings.TrimSpace(effective.RedirectURL)); err != nil {
		return config.UniFedConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "unifed redirect url must be an absolute http(s) URL").WithCause(err)
	}
	effective.InstanceURL = strings.TrimRight(strings.TrimSpace(effective.InstanceURL), "/")
	effective.RedirectURL = strings.TrimSpace(effective.RedirectURL)
	return effective, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
cd backend && go test ./internal/service -run 'TestGetUniFedConnectOAuthConfig|TestPublicSettingsExposeUniFedOAuthEnabled' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/config/config.go backend/internal/service/domain_constants.go backend/internal/service/setting_service.go backend/internal/service/settings_view.go backend/internal/handler/dto/settings.go backend/internal/service/setting_service_unifed_test.go
git commit -m "feat: 增加 UniFed 登录设置"
```

---

### Task 2: 后台 settings API 保存 UniFed 配置与默认权益

**Files:**
- Modify: `backend/internal/handler/admin/setting_handler.go`
- Modify: `backend/internal/service/domain_constants.go`
- Modify: `backend/internal/service/setting_service.go`
- Test: `backend/internal/handler/admin/setting_handler_unifed_test.go`
- Test: `frontend/src/api/__tests__/settings.authSourceDefaults.spec.ts`
- Modify: `frontend/src/api/admin/settings.ts`

- [ ] **Step 1: Write the failing backend test**

Create `backend/internal/handler/admin/setting_handler_unifed_test.go`:

```go
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

func TestUpdateSettingsPersistsUniFedConfigAndDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
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

	body := map[string]any{
		"promo_code_enabled": true,
	}
	body["unifed_connect_enabled"] = true
	body["unifed_connect_instance_url"] = "https://misskey.example"
	body["unifed_connect_redirect_url"] = "https://app.example.com/api/v1/auth/oauth/unifed/callback"
	body["auth_source_default_unifed_balance"] = 3.5
	body["auth_source_default_unifed_concurrency"] = 7
	body["auth_source_default_unifed_grant_on_signup"] = true

	data, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(data))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateSettings(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "true", repo.values[service.SettingKeyUniFedConnectEnabled])
	require.Equal(t, "https://misskey.example", repo.values[service.SettingKeyUniFedConnectInstanceURL])
	require.Equal(t, "https://app.example.com/api/v1/auth/oauth/unifed/callback", repo.values[service.SettingKeyUniFedConnectRedirectURL])
	require.Equal(t, "3.50000000", repo.values[service.SettingKeyAuthSourceDefaultUniFedBalance])
	require.Equal(t, "7", repo.values[service.SettingKeyAuthSourceDefaultUniFedConcurrency])
	require.Equal(t, "true", repo.values[service.SettingKeyAuthSourceDefaultUniFedGrantOnSignup])
}

func TestUpdateSettingsRejectsEnabledUniFedWithInvalidURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{values: map[string]string{
		service.SettingKeyPromoCodeEnabled:        "true",
		service.SettingKeyUniFedConnectEnabled:    "false",
		service.SettingKeyUniFedConnectInstanceURL: "https://dc.hhhl.cc",
	}}
	svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil, nil)

	body := map[string]any{
		"promo_code_enabled": true,
	}
	body["unifed_connect_enabled"] = true
	body["unifed_connect_instance_url"] = "dc.hhhl.cc"
	body["unifed_connect_redirect_url"] = "https://app.example.com/api/v1/auth/oauth/unifed/callback"

	data, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(data))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateSettings(c)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
```

- [ ] **Step 2: Run backend test to verify it fails**

Run:

```bash
cd backend && go test ./internal/handler/admin -run 'TestUpdateSettings.*UniFed' -count=1
```

Expected: FAIL because handler fields, constants, or test helpers do not yet include UniFed.

- [ ] **Step 3: Write minimal backend implementation**

Add UniFed fields to `UpdateSettingsRequest`, `dto.SystemSettings`, `service.SystemSettings`, `buildSystemSettingsUpdates`, `parseSettings`, `GetSettings`, `UpdateSettings` mapping, `systemSettingsResponseData`, and change tracking.

Add default auth source keys:

```go
SettingKeyAuthSourceDefaultUniFedBalance
SettingKeyAuthSourceDefaultUniFedConcurrency
SettingKeyAuthSourceDefaultUniFedSubscriptions
SettingKeyAuthSourceDefaultUniFedGrantOnSignup
SettingKeyAuthSourceDefaultUniFedGrantOnFirstBind
SettingKeyAuthSourcePlatformQuotas("unifed")
```

Add `UniFed ProviderDefaultGrantSettings` to `AuthSourceDefaultSettings` and include it in parse/update/default initialization.

In `UpdateSettings`, trim and validate:

```go
req.UniFedConnectInstanceURL = strings.TrimSpace(req.UniFedConnectInstanceURL)
req.UniFedConnectRedirectURL = strings.TrimSpace(req.UniFedConnectRedirectURL)
if req.UniFedConnectEnabled {
	if req.UniFedConnectInstanceURL == "" {
		req.UniFedConnectInstanceURL = previousSettings.UniFedConnectInstanceURL
	}
	if req.UniFedConnectRedirectURL == "" {
		req.UniFedConnectRedirectURL = previousSettings.UniFedConnectRedirectURL
	}
	if err := config.ValidateAbsoluteHTTPURL(req.UniFedConnectInstanceURL); err != nil {
		response.BadRequest(c, "UniFed Instance URL must be an absolute http(s) URL")
		return
	}
	if err := config.ValidateAbsoluteHTTPURL(req.UniFedConnectRedirectURL); err != nil {
		response.BadRequest(c, "UniFed Redirect URL must be an absolute http(s) URL")
		return
	}
}
```

- [ ] **Step 4: Run backend test to verify it passes**

Run:

```bash
cd backend && go test ./internal/handler/admin -run 'TestUpdateSettings.*UniFed' -count=1
```

Expected: PASS.

- [ ] **Step 5: Write the failing frontend helper test**

In `frontend/src/api/__tests__/settings.authSourceDefaults.spec.ts`, add assertions if absent:

```ts
expect(state.unifed).toEqual({
  balance: 0,
  concurrency: 5,
  subscriptions: [],
  grant_on_signup: false,
  grant_on_first_bind: false,
  platform_quotas: allNullQuotas,
});
```

In the payload test, include:

```ts
unifed: {
  balance: 2,
  concurrency: 4,
  subscriptions: [{ group_id: 10, validity_days: 15 }],
  grant_on_signup: true,
  grant_on_first_bind: true,
  platform_quotas: {},
},
```

Assert:

```ts
expect(payload).toMatchObject({
  auth_source_default_unifed_balance: 2,
  auth_source_default_unifed_concurrency: 4,
  auth_source_default_unifed_subscriptions: [{ group_id: 10, validity_days: 15 }],
  auth_source_default_unifed_grant_on_signup: true,
  auth_source_default_unifed_grant_on_first_bind: true,
  auth_source_default_unifed_platform_quotas: allNullQuotas,
});
```

- [ ] **Step 6: Run frontend helper test to verify it fails**

Run:

```bash
pnpm --dir frontend test:run src/api/__tests__/settings.authSourceDefaults.spec.ts
```

Expected: FAIL because `AuthSourceType` / `AUTH_SOURCE_TYPES` / settings payload do not include UniFed.

- [ ] **Step 7: Write minimal frontend helper implementation**

In `frontend/src/api/admin/settings.ts`, add `unifed` to `AuthSourceType`, `AUTH_SOURCE_TYPES`, `SystemSettings`, and `UpdateSettingsRequest`.

- [ ] **Step 8: Run frontend helper test to verify it passes**

Run:

```bash
pnpm --dir frontend test:run src/api/__tests__/settings.authSourceDefaults.spec.ts
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add backend/internal/handler/admin/setting_handler.go backend/internal/service/domain_constants.go backend/internal/service/setting_service.go backend/internal/service/settings_view.go backend/internal/handler/dto/settings.go backend/internal/handler/admin/setting_handler_unifed_test.go frontend/src/api/admin/settings.ts frontend/src/api/__tests__/settings.authSourceDefaults.spec.ts
git commit -m "feat: 接入 UniFed 后台设置"
```

---

### Task 3: 后端身份模型与绑定状态接入

**Files:**
- Modify: `backend/ent/schema/auth_identity.go`
- Modify: `backend/ent/schema/user.go`
- Modify: `backend/ent/schema/auth_identity_schema_test.go`
- Modify: `backend/internal/service/auth_service.go`
- Modify: `backend/internal/service/auth_service_test.go`
- Modify: `backend/internal/service/user_service.go`
- Modify: `backend/internal/handler/user_handler.go`
- Modify: `backend/internal/service/admin_service.go`
- Test: existing schema/service/user handler tests

- [ ] **Step 1: Write the failing tests**

In `backend/internal/service/auth_service_test.go`, add:

```go
func TestUniFedReservedEmailAndLegacySignupSource(t *testing.T) {
	require.True(t, isReservedEmail("unifed-abc123@unifed-connect.invalid"))
	require.False(t, isReservedEmail("real@unifed.example"))
	require.Equal(t, "unifed", inferLegacySignupSource("unifed-abc123@unifed-connect.invalid"))
}
```

In `backend/ent/schema/auth_identity_schema_test.go`, ensure the valid values list includes `"unifed"`.

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd backend && go test ./ent/schema ./internal/service -run 'TestAuthIdentityFoundationSchemas|TestUniFedReservedEmailAndLegacySignupSource' -count=1
```

Expected: FAIL because `unifed` is not accepted or reserved.

- [ ] **Step 3: Write minimal implementation**

Add `"unifed"` to:

- `authProviderTypes` in `auth_identity.go`
- `signup_source` validator in `user.go`
- `authSourceSignupSettings` in `auth_service.go`
- `inferLegacySignupSource`
- `isReservedEmail`
- `UserIdentitySummarySet`
- `GetProfileIdentitySummaries`
- `applyExplicitProviderAvailability`
- `canUnbindProvider`
- `buildUserIdentityBindAuthorizeURL`
- `normalizeUserIdentityProvider`
- user profile response and identity map
- admin provider normalization / validation

- [ ] **Step 4: Run tests to verify they pass**

Run:

```bash
cd backend && go test ./ent/schema ./internal/service ./internal/handler -run 'TestAuthIdentityFoundationSchemas|TestUniFedReservedEmailAndLegacySignupSource|TestUserProfile' -count=1
```

Expected: PASS for targeted tests.

- [ ] **Step 5: Commit**

```bash
git add backend/ent/schema/auth_identity.go backend/ent/schema/user.go backend/ent/schema/auth_identity_schema_test.go backend/internal/service/auth_service.go backend/internal/service/auth_service_test.go backend/internal/service/user_service.go backend/internal/handler/user_handler.go backend/internal/service/admin_service.go
git commit -m "feat: 支持 UniFed 身份来源"
```

---

### Task 4: UniFed MiAuth 后端 start/callback/complete-registration

**Files:**
- Create/Modify: `backend/internal/handler/auth_unifed_oauth.go`
- Modify: `backend/internal/server/routes/auth.go`
- Test: `backend/internal/handler/auth_unifed_oauth_test.go`

- [ ] **Step 1: Write failing tests for helpers and start URL**

Create `backend/internal/handler/auth_unifed_oauth_test.go`:

```go
package handler

import (
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
```

- [ ] **Step 2: Add test helper**

In the same file, add:

```go
func newUniFedOAuthTestHandler(t *testing.T, cfg config.UniFedConnectConfig) *AuthHandler {
	t.Helper()
	return &AuthHandler{
		cfg: &config.Config{
			UniFed: cfg,
			SessionSecret: "test-secret",
		},
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run:

```bash
cd backend && go test ./internal/handler -run 'TestUniFed.*' -count=1
```

Expected: FAIL because UniFed handler/helper functions are missing or incomplete.

- [ ] **Step 4: Write minimal implementation for start/helpers/routes**

Implement in `auth_unifed_oauth.go`:

- constants for cookie names and callback path
- `UniFedOAuthStart`
- `getUniFedOAuthConfig`
- `generateUniFedState`
- `generateUniFedSessionUUID`
- `unifedSyntheticEmail`
- `isSafeUniFedSubject`
- `unifedSetCookie`
- `unifedClearCookie`

Register routes in `backend/internal/server/routes/auth.go`:

```go
auth.GET("/oauth/unifed/start", h.Auth.UniFedOAuthStart)
auth.GET("/oauth/unifed/bind/start", func(c *gin.Context) {
	query := c.Request.URL.Query()
	query.Set("intent", "bind_current_user")
	c.Request.URL.RawQuery = query.Encode()
	h.Auth.UniFedOAuthStart(c)
})
auth.GET("/oauth/unifed/callback", h.Auth.UniFedOAuthCallback)
auth.POST("/oauth/unifed/complete-registration",
	rateLimiter.LimitWithOptions("oauth-unifed-complete", 10, time.Minute, middleware.RateLimitOptions{
		FailureMode: middleware.RateLimitFailClose,
	}),
	h.Auth.CompleteUniFedOAuthRegistration,
)
```

- [ ] **Step 5: Run start/helper tests to verify they pass**

Run:

```bash
cd backend && go test ./internal/handler -run 'TestUniFedSyntheticEmail|TestUniFedOAuthStart|TestUniFedOAuthBindStart' -count=1
```

Expected: PASS.

- [ ] **Step 6: Write failing tests for MiAuth exchange and invalid callback state**

Append:

```go
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
```

Also add imports `context`.

- [ ] **Step 7: Run callback/exchange tests to verify they fail**

Run:

```bash
cd backend && go test ./internal/handler -run 'TestUniFedExchangeMiAuth|TestUniFedOAuthCallbackRejectsInvalidState' -count=1
```

Expected: FAIL because exchange/callback are missing or incomplete.

- [ ] **Step 8: Write minimal callback/exchange implementation**

Implement:

- `UniFedOAuthCallback`
- `unifedExchangeMiAuth`
- `unifedFetchUserInfo`
- `createUniFedOAuthChoicePendingSession`
- `CompleteUniFedOAuthRegistration`

Use existing helpers:

- `createOAuthPendingSession`
- `findOAuthIdentityUser`
- `ensureBackendModeAllowsNewUserLogin`
- `isForceEmailOnThirdPartySignup`
- `LoginOrRegisterOAuthWithTokenPairAndPromoCode`
- `applyPendingOAuthBinding`
- `applyPendingOAuthAdoptionAndConsumeSession`
- `redirectOAuthError`
- `redirectToFrontendCallback`

- [ ] **Step 9: Run UniFed handler tests to verify they pass**

Run:

```bash
cd backend && go test ./internal/handler -run 'TestUniFed.*' -count=1
```

Expected: PASS.

- [ ] **Step 10: Commit**

```bash
git add backend/internal/handler/auth_unifed_oauth.go backend/internal/handler/auth_unifed_oauth_test.go backend/internal/server/routes/auth.go
git commit -m "feat: 实现 UniFed MiAuth 登录回调"
```

---

### Task 5: 前端 UniFed 入口、回调页和路由

**Files:**
- Modify: `frontend/src/api/auth.ts`
- Create/Modify: `frontend/src/components/auth/UniFedOAuthSection.vue`
- Create/Modify: `frontend/src/views/auth/UniFedCallbackView.vue`
- Modify: `frontend/src/views/auth/LoginView.vue`
- Modify: `frontend/src/views/auth/RegisterView.vue`
- Modify: `frontend/src/views/auth/EmailVerifyView.vue`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/stores/app.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Test: `frontend/src/components/auth/__tests__/UniFedOAuthSection.spec.ts`
- Test: `frontend/src/views/auth/__tests__/UniFedCallbackView.spec.ts`
- Test: `frontend/src/router/__tests__/guards.spec.ts`

- [ ] **Step 1: Write failing UniFed button test**

Create `frontend/src/components/auth/__tests__/UniFedOAuthSection.spec.ts`:

```ts
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import UniFedOAuthSection from '@/components/auth/UniFedOAuthSection.vue'

const routeState = {
  query: {} as Record<string, string>,
}

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

vi.mock('@/utils/oauthAffiliate', () => ({
  resolveAffiliateReferralCode: vi.fn(() => 'aff-1'),
  storeOAuthAffiliateCode: vi.fn(),
}))

describe('UniFedOAuthSection', () => {
  beforeEach(() => {
    routeState.query = { redirect: '/settings/profile' }
    Object.defineProperty(window, 'location', {
      value: { href: 'http://localhost/login' },
      writable: true,
    })
  })

  it('starts UniFed OAuth with the current redirect path', async () => {
    const wrapper = mount(UniFedOAuthSection, {
      props: { showDivider: false },
    })

    await wrapper.get('button').trigger('click')

    expect(window.location.href).toBe('/api/v1/auth/oauth/unifed/start?redirect=%2Fsettings%2Fprofile')
  })
})
```

- [ ] **Step 2: Run button test to verify it fails**

Run:

```bash
pnpm --dir frontend test:run src/components/auth/__tests__/UniFedOAuthSection.spec.ts
```

Expected: FAIL because component does not exist or start URL is wrong.

- [ ] **Step 3: Write minimal button implementation**

Create `frontend/src/components/auth/UniFedOAuthSection.vue`:

```vue
<template>
  <div class="space-y-4">
    <button type="button" :disabled="disabled" class="btn btn-secondary w-full" @click="startLogin">
      <span class="mr-2 inline-flex h-5 w-5 items-center justify-center rounded-full bg-indigo-600 text-[10px] font-bold text-white">UF</span>
      {{ t('auth.unifed.signIn') }}
    </button>
    <div v-if="showDivider" class="flex items-center gap-3">
      <div class="h-px flex-1 bg-gray-200 dark:bg-dark-700"></div>
      <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('auth.oauthOrContinue') }}</span>
      <div class="h-px flex-1 bg-gray-200 dark:bg-dark-700"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { resolveAffiliateReferralCode, storeOAuthAffiliateCode } from '@/utils/oauthAffiliate'

const props = withDefaults(defineProps<{
  disabled?: boolean
  affCode?: string
  showDivider?: boolean
}>(), {
  showDivider: true,
})

const route = useRoute()
const { t } = useI18n()

function startLogin(): void {
  const redirectTo = (route.query.redirect as string) || '/dashboard'
  storeOAuthAffiliateCode(resolveAffiliateReferralCode(props.affCode, route.query.aff, route.query.aff_code))
  const apiBase = (import.meta.env.VITE_API_BASE_URL as string | undefined) || '/api/v1'
  const normalized = apiBase.replace(/\/$/, '')
  window.location.href = `${normalized}/auth/oauth/unifed/start?redirect=${encodeURIComponent(redirectTo)}`
}
</script>
```

- [ ] **Step 4: Run button test to verify it passes**

Run:

```bash
pnpm --dir frontend test:run src/components/auth/__tests__/UniFedOAuthSection.spec.ts
```

Expected: PASS.

- [ ] **Step 5: Write failing callback/API/router tests**

In `frontend/src/api/auth.ts` tests or a new `frontend/src/api/__tests__/auth.unifed.spec.ts`, add:

```ts
import { describe, expect, it, vi } from 'vitest'
import { createPendingUniFedOAuthAccount } from '@/api/auth'
import api from '@/api'

vi.mock('@/api', () => ({
  default: {
    post: vi.fn(async () => ({ data: { access_token: 'a', refresh_token: 'r', expires_in: 3600, token_type: 'Bearer' } })),
  },
}))

describe('UniFed auth API', () => {
  it('posts pending account creation to the UniFed endpoint', async () => {
    await createPendingUniFedOAuthAccount('invite-1')
    expect(api.post).toHaveBeenCalledWith('/auth/oauth/unifed/create-account', expect.objectContaining({
      invitation_code: 'invite-1',
    }))
  })
})
```

In `frontend/src/router/__tests__/guards.spec.ts`, include `/auth/unifed/callback` in callback path expectations.

- [ ] **Step 6: Run tests to verify they fail**

Run:

```bash
pnpm --dir frontend test:run src/api/__tests__/auth.unifed.spec.ts src/router/__tests__/guards.spec.ts
```

Expected: FAIL because UniFed API/router are missing.

- [ ] **Step 7: Write minimal frontend API/router/callback implementation**

In `frontend/src/api/auth.ts`, extend provider union:

```ts
provider: 'linuxdo' | 'oidc' | 'wechat' | 'dingtalk' | 'unifed'
```

Add:

```ts
export async function createPendingUniFedOAuthAccount(
  invitationCode: string,
  decision?: OAuthAdoptionDecision,
  affiliateCode?: string
): Promise<PendingOAuthCreateAccountResponse> {
  return createPendingOAuthAccount('unifed', invitationCode, decision, affiliateCode)
}

export async function completeUniFedOAuthRegistration(
  invitationCode: string,
  decision?: OAuthAdoptionDecision,
  affiliateCode?: string
): Promise<OAuthTokenResponse> {
  return createPendingUniFedOAuthAccount(invitationCode, decision, affiliateCode)
}
```

Create `UniFedCallbackView.vue` by adapting `OidcCallbackView.vue` / `WechatCallbackView.vue`, with provider name `Universe Federation`, test id prefix `unifed`, and UniFed API functions.

Add route:

```ts
{
  path: '/auth/unifed/callback',
  name: 'UniFedOAuthCallback',
  component: () => import('@/views/auth/UniFedCallbackView.vue'),
  meta: {
    requiresAuth: false,
    title: 'Universe Federation OAuth Callback',
    titleKey: 'auth.unifedCallbackPageTitle',
  },
}
```

Include `/auth/unifed/callback` in backend mode callback paths.

Add UniFed translations under `auth.unifed` and `auth.unifedCallbackPageTitle`.

- [ ] **Step 8: Run tests to verify they pass**

Run:

```bash
pnpm --dir frontend test:run src/components/auth/__tests__/UniFedOAuthSection.spec.ts src/api/__tests__/auth.unifed.spec.ts src/router/__tests__/guards.spec.ts
```

Expected: PASS.

- [ ] **Step 9: Wire Login/Register/EmailVerify**

In `LoginView.vue` and `RegisterView.vue`, import `UniFedOAuthSection`, add `unifedOAuthEnabled`, read `settings.unifed_oauth_enabled`, include it in `showOAuthLogin`, and render the button.

In `EmailVerifyView.vue`, add:

```ts
case 'unifed':
  return '/auth/unifed/callback'
```

In `types/index.ts` and `stores/app.ts`, add `unifed_oauth_enabled` and provider type `unifed`.

- [ ] **Step 10: Run focused frontend tests**

Run:

```bash
pnpm --dir frontend test:run src/views/auth/__tests__/EmailVerifyView.spec.ts src/components/auth/__tests__/UniFedOAuthSection.spec.ts src/router/__tests__/guards.spec.ts
pnpm --dir frontend typecheck
```

Expected: PASS.

- [ ] **Step 11: Commit**

```bash
git add frontend/src/api/auth.ts frontend/src/api/__tests__/auth.unifed.spec.ts frontend/src/components/auth/UniFedOAuthSection.vue frontend/src/components/auth/__tests__/UniFedOAuthSection.spec.ts frontend/src/views/auth/UniFedCallbackView.vue frontend/src/views/auth/LoginView.vue frontend/src/views/auth/RegisterView.vue frontend/src/views/auth/EmailVerifyView.vue frontend/src/router/index.ts frontend/src/router/__tests__/guards.spec.ts frontend/src/types/index.ts frontend/src/stores/app.ts frontend/src/i18n/locales/en.ts frontend/src/i18n/locales/zh.ts
git commit -m "feat: 接入 UniFed 前端登录入口"
```

---

### Task 6: 个人资料绑定与后台设置 UI

**Files:**
- Modify: `frontend/src/views/user/ProfileView.vue`
- Modify: `frontend/src/components/user/profile/ProfileAccountBindingsCard.vue`
- Modify: `frontend/src/components/user/profile/ProfileIdentityBindingsSection.vue`
- Modify: `frontend/src/components/user/profile/ProfileInfoCard.vue`
- Modify: `frontend/src/views/admin/SettingsView.vue`
- Modify: `frontend/src/views/admin/__tests__/SettingsView.spec.ts`
- Test: `frontend/src/components/user/profile/__tests__/ProfileIdentityBindingsSection.spec.ts`
- Test: `frontend/src/views/user/__tests__/ProfileView.spec.ts`

- [ ] **Step 1: Write failing profile binding test**

In `frontend/src/components/user/profile/__tests__/ProfileIdentityBindingsSection.spec.ts`, add:

```ts
it('shows UniFed as bindable when enabled and unbound', () => {
  const wrapper = mount(ProfileIdentityBindingsSection, {
    global: {
      plugins: [pinia],
    },
    props: {
      user: createUser({
        identities: {
          unifed: {
            provider: 'unifed',
            bound: false,
            bound_count: 0,
            can_bind: true,
            can_unbind: false,
            bind_start_path: '/api/v1/auth/oauth/unifed/bind/start?intent=bind_current_user&redirect=%2Fsettings%2Fprofile',
          },
        },
      }),
      unifedEnabled: true,
    },
  })

  expect(wrapper.text()).toContain('Universe Federation')
  expect(wrapper.text()).toContain('Bind Universe Federation')
})
```

- [ ] **Step 2: Run profile test to verify it fails**

Run:

```bash
pnpm --dir frontend test:run src/components/user/profile/__tests__/ProfileIdentityBindingsSection.spec.ts
```

Expected: FAIL because props/provider labels/types do not include UniFed.

- [ ] **Step 3: Write minimal profile implementation**

Add `unifedEnabled` prop through `ProfileView` -> `ProfileInfoCard` -> `ProfileAccountBindingsCard` -> `ProfileIdentityBindingsSection`.

Add provider item:

```ts
{
  provider: 'unifed' as const,
  label: t('profile.authBindings.providers.unifed'),
  bound: getBindingStatus('unifed'),
  canBind: !getBindingStatus('unifed') && isProviderEnabledForBinding('unifed') && (getBindingDetails('unifed')?.can_bind ?? true),
  canUnbind: Boolean(getBindingStatus('unifed') && getBindingDetails('unifed')?.can_unbind),
  details: getBindingDetails('unifed'),
}
```

Add label translations under `profile.authBindings.providers.unifed`.

- [ ] **Step 4: Run profile test to verify it passes**

Run:

```bash
pnpm --dir frontend test:run src/components/user/profile/__tests__/ProfileIdentityBindingsSection.spec.ts src/views/user/__tests__/ProfileView.spec.ts
```

Expected: PASS.

- [ ] **Step 5: Write failing SettingsView test**

In `frontend/src/views/admin/__tests__/SettingsView.spec.ts`, add a test:

```ts
it('renders and submits UniFed settings', async () => {
  const wrapper = mountView()
  await flushPromises()
  await openSecurityTab(wrapper)

  expect(wrapper.text()).toContain('Universe Federation')

  const instanceInput = wrapper.find('input[placeholder="https://dc.hhhl.cc"]')
  await instanceInput.setValue('https://misskey.example')

  await wrapper.find('button[type="submit"]').trigger('click')
  await flushPromises()

  expect(updateSettings).toHaveBeenCalledWith(expect.objectContaining({
    unifed_connect_instance_url: 'https://misskey.example',
  }))
})
```

- [ ] **Step 6: Run SettingsView test to verify it fails**

Run:

```bash
pnpm --dir frontend test:run src/views/admin/__tests__/SettingsView.spec.ts
```

Expected: FAIL because SettingsView does not render or submit UniFed settings.

- [ ] **Step 7: Write minimal SettingsView implementation**

Add a UniFed card with:

- `Toggle v-model="form.unifed_connect_enabled"`
- `input v-model="form.unifed_connect_instance_url"`
- `input v-model="form.unifed_connect_redirect_url"`
- computed `unifedRedirectUrlSuggestion`
- `setAndCopyUniFedRedirectUrl`

Initialize form:

```ts
unifed_connect_enabled: false,
unifed_connect_instance_url: "https://dc.hhhl.cc",
unifed_connect_redirect_url: "",
```

Include fields in `saveSettings()` payload.

Add UniFed auth source defaults meta item.

- [ ] **Step 8: Run SettingsView test to verify it passes**

Run:

```bash
pnpm --dir frontend test:run src/views/admin/__tests__/SettingsView.spec.ts
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add frontend/src/views/user/ProfileView.vue frontend/src/components/user/profile/ProfileAccountBindingsCard.vue frontend/src/components/user/profile/ProfileIdentityBindingsSection.vue frontend/src/components/user/profile/ProfileInfoCard.vue frontend/src/components/user/profile/__tests__/ProfileIdentityBindingsSection.spec.ts frontend/src/views/user/__tests__/ProfileView.spec.ts frontend/src/views/admin/SettingsView.vue frontend/src/views/admin/__tests__/SettingsView.spec.ts frontend/src/i18n/locales/en.ts frontend/src/i18n/locales/zh.ts
git commit -m "feat: 增加 UniFed 前端设置与绑定"
```

---

### Task 7: API contract 与全量验证

**Files:**
- Modify: `backend/internal/server/api_contract_test.go`
- Test: backend and frontend full targeted suites

- [ ] **Step 1: Write/update failing contract expectations**

In `backend/internal/server/api_contract_test.go`, add expected fields:

```json
"unifed_bound": false
```

Under `identities`, `identity_bindings`, and `auth_bindings`:

```json
"unifed": {
  "provider": "unifed",
  "bound": false,
  "bound_count": 0,
  "can_bind": true,
  "can_unbind": false,
  "bind_start_path": "/api/v1/auth/oauth/unifed/bind/start?intent=bind_current_user&redirect=%2Fsettings%2Fprofile"
}
```

Under settings response:

```json
"unifed_connect_enabled": false,
"unifed_connect_instance_url": "https://dc.hhhl.cc",
"unifed_connect_redirect_url": "",
"auth_source_default_unifed_balance": 0,
"auth_source_default_unifed_concurrency": 5,
"auth_source_default_unifed_subscriptions": [],
"auth_source_default_unifed_grant_on_signup": false,
"auth_source_default_unifed_grant_on_first_bind": false,
"auth_source_default_unifed_platform_quotas": null
```

- [ ] **Step 2: Run contract test to verify it fails**

Run:

```bash
cd backend && go test ./internal/server -run TestAPIContracts -count=1
```

Expected: FAIL until all backend DTOs and defaults align.

- [ ] **Step 3: Fix contract mismatches minimally**

Adjust implementation, not the test, unless the implementation already matches the approved spec and the expected JSON is stale. Keep default instance URL as `https://dc.hhhl.cc`.

- [ ] **Step 4: Run contract test to verify it passes**

Run:

```bash
cd backend && go test ./internal/server -run TestAPIContracts -count=1
```

Expected: PASS.

- [ ] **Step 5: Run backend focused suites**

Run:

```bash
cd backend && go test ./ent/schema ./internal/service ./internal/handler ./internal/handler/admin ./internal/server -count=1
```

Expected: PASS.

- [ ] **Step 6: Run frontend focused suites**

Run:

```bash
pnpm --dir frontend test:run src/api/__tests__/settings.authSourceDefaults.spec.ts src/api/__tests__/auth.unifed.spec.ts src/components/auth/__tests__/UniFedOAuthSection.spec.ts src/components/user/profile/__tests__/ProfileIdentityBindingsSection.spec.ts src/views/auth/__tests__/EmailVerifyView.spec.ts src/views/admin/__tests__/SettingsView.spec.ts src/router/__tests__/guards.spec.ts
pnpm --dir frontend typecheck
```

Expected: PASS.

- [ ] **Step 7: Run final repository checks**

Run:

```bash
git status --short --untracked-files=all
git diff --check
```

Expected: only intended files modified or clean after final commit; `git diff --check` exits 0.

- [ ] **Step 8: Commit contract and verification fixes**

```bash
git add backend/internal/server/api_contract_test.go
git commit -m "test: 补充 UniFed API 契约"
```

If Step 3 required implementation changes, include those files in the same commit only if they are contract alignment changes.

---

## Self-Review

- Spec coverage: plan covers settings defaults, MiAuth flow, identity model, user paths, default grants, frontend entry/callback, profile binding, security checks, and verification.
- Placeholder scan: no incomplete requirement markers or vague test placeholders remain.
- Type consistency: provider key is consistently `unifed`; public setting is `unifed_oauth_enabled`; system settings use `unifed_connect_*`; default grant fields use `auth_source_default_unifed_*`.
