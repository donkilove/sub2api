# UniFed MiAuth 登录设计

## 目标

在现有第三方登录体系中新增 `UniFed / Universe Federation / Sharkey-Misskey MiAuth` 登录能力。功能需要支持登录、注册、邀请码补全、绑定现有账号、个人资料绑定入口、后台配置、公开设置和第三方注册默认权益。

## 范围

本次只实现单个 UniFed provider，provider 标识固定为 `unifed`。默认实例地址为 `https://dc.hhhl.cc`，但管理员可以在后台系统设置中改为其他 Sharkey/Misskey MiAuth 兼容实例。

不在本次范围内：

- 多个 UniFed 实例同时登录。
- 标准 OAuth2 抽象改造。
- Sharkey/Misskey 除 MiAuth 之外的认证协议。
- UniFed 用户资料的后台同步任务。

## 方案选择

采用独立 provider，并复用现有 pending OAuth 框架。

UniFed 与 LinuxDo、OIDC、WeChat、DingTalk 一样进入 `pending_auth_sessions` 和 `auth_identities` 流程。MiAuth 协议差异只封装在 UniFed handler 内：启动 URL、session check、用户信息解析和合成邮箱生成。

此方案的收益：

- 复用邀请码、强制邮箱补全、绑定已有账号、TOTP、资料采纳和默认权益逻辑。
- 改动边界清晰，避免为了 MiAuth 引入过大的 OAuth2 抽象重构。
- 前端体验与现有第三方登录保持一致。

## 后台设置

新增系统设置字段：

- `unifed_connect_enabled`
- `unifed_connect_instance_url`
- `unifed_connect_redirect_url`

初始化默认值：

- `unifed_connect_enabled=false`
- `unifed_connect_instance_url=https://dc.hhhl.cc`
- `unifed_connect_redirect_url=""`

生效优先级：

1. 数据库中的系统设置。
2. `config.yaml` 或环境变量中的 `unifed_connect`。
3. 代码默认值。

启用时必须校验：

- `unifed_connect_instance_url` 是绝对 `http(s)` URL。
- `unifed_connect_redirect_url` 是绝对 `http(s)` URL。

公开设置返回 `unifed_oauth_enabled`。登录页和注册页只根据这个公开开关显示 UniFed 登录入口，不因为默认实例地址存在而自动显示。

## MiAuth 数据流

### 启动登录

`GET /api/v1/auth/oauth/unifed/start?redirect=/dashboard`

后端生成：

- CSRF `state`
- browser session key
- MiAuth session UUID
- 可选绑定意图 `intent=bind_current_user`

后端写入短期 cookie，并重定向到：

```text
{instance}/miauth/{session}?name=Sub2API&callback={callback}&permission=read:account
```

其中 `{instance}` 来自最终生效的 `unifed_connect_instance_url`，`{callback}` 来自 `unifed_connect_redirect_url`，并追加 `state` 查询参数。

### 回调处理

`GET /api/v1/auth/oauth/unifed/callback?session=...&state=...`

后端必须校验：

- query 中存在 `session` 和 `state`。
- query `state` 与 cookie 中的 `state` 一致。
- query `session` 与 cookie 中的 MiAuth session UUID 一致。
- browser session cookie 存在。

校验通过后调用：

```text
POST {instance}/api/miauth/{session}/check
```

请求成功后读取 `token` 和 `user`。如果 check 响应未带 `user`，使用 token 调用：

```text
POST {instance}/api/i
```

请求体：

```json
{"i":"<token>"}
```

用户资料至少需要 `id`。`username` 优先使用 `username`，其次可使用 `usernameLower`。显示名优先使用 `name`，头像使用 `avatarUrl`。

## 身份模型

UniFed 身份写入：

- `provider_type=unifed`
- `provider_key=unifed`
- `provider_subject=<MiAuth user.id>`

合成邮箱格式：

```text
unifed-{subject}@unifed-connect.invalid
```

`subject` 必须满足：

- 非空。
- 长度不超过合成邮箱本地部分限制。
- 只包含 ASCII 字母、数字、`_`、`-`、`.`。

`signup_source`、管理员用户筛选、个人资料绑定状态和保留邮箱判断都需要识别 `unifed`。

## 用户路径

### 已绑定身份

如果 `auth_identities` 中已有匹配 UniFed identity，回调创建 pending login session，前端 `/auth/unifed/callback` 兑换完成后登录该用户。

### 新身份直接注册

当站点允许第三方直接注册，且未启用强制第三方邮箱补全时，可以用合成邮箱直接注册，注册来源为 `unifed`，随后绑定 UniFed identity。

### 需要补全

当邀请码、强制邮箱补全或账号选择逻辑需要用户交互时，回调创建 pending OAuth session。前端回调页展示现有 pending OAuth UI：

- 邀请码补全。
- 创建新账号。
- 绑定已有账号。
- TOTP 验证。
- 显示名和头像采纳。

### 当前用户绑定

`GET /api/v1/auth/oauth/unifed/bind/start` 设置 `intent=bind_current_user` 并复用 start 流程。回调创建绑定当前用户的 pending session，前端兑换后完成绑定。

个人资料页在 `unifed_oauth_enabled=true` 时显示可绑定入口；当用户已绑定 UniFed identity 时显示已绑定状态和解绑能力。

## 默认权益

UniFed 加入第三方来源默认权益配置：

- `auth_source_default_unifed_balance`
- `auth_source_default_unifed_concurrency`
- `auth_source_default_unifed_subscriptions`
- `auth_source_default_unifed_grant_on_signup`
- `auth_source_default_unifed_grant_on_first_bind`
- `auth_source_default_unifed_platform_quotas`

默认值：

- balance: `0`
- concurrency: `5`
- subscriptions: `[]`
- grant_on_signup: `false`
- grant_on_first_bind: `false`
- platform_quotas: `null`

后台设置页需要显示、保存和回显这些字段。

## 前端体验

登录页和注册页在 `unifed_oauth_enabled=true` 时展示 UniFed 登录按钮，文案为中文“使用 Universe Federation 登录”，英文“Continue with Universe Federation”。

新增路由：

```text
/auth/unifed/callback
```

回调页复用现有 pending OAuth completion API 和交互模型。页面应处理：

- 后端错误 query。
- pending session 兑换。
- 邀请码提交。
- 创建账号。
- 绑定已有账号。
- TOTP。
- 显示名和头像采纳。
- 成功后跳转原始 `redirect`。

后台设置页新增 UniFed 配置卡片，包含：

- 启用开关。
- 实例地址输入框，默认显示 `https://dc.hhhl.cc`。
- 后端回调地址输入框。
- “使用当前站点生成并复制”按钮。

## 错误处理与安全

- start 和 callback 的 cookie 使用短有效期、`HttpOnly`、`SameSite=Lax`。
- state 和 MiAuth session 都必须双重校验。
- redirect 只允许前端相对路径。
- 上游错误不直接暴露敏感响应体给前端。
- 日志可以记录状态码和截断后的响应体，不能输出 token。
- UniFed 合成邮箱域加入 reserved email 检查，禁止普通注册占用。

## 测试与验证

实施必须按 TDD：

1. 先写失败测试。
2. 运行并确认失败原因符合预期。
3. 写最小实现。
4. 运行测试确认通过。
5. 需要时再重构，并保持测试通过。

优先测试范围：

- `GetUniFedConnectOAuthConfig` 的默认值、数据库覆盖、禁用状态和 URL 校验。
- UniFed start 路由生成正确 MiAuth URL，并设置 state、browser session 和 MiAuth session cookie。
- UniFed callback 在 state/session 不匹配时拒绝。
- UniFed callback 能通过 mock MiAuth check 创建 pending session。
- `unifed` 加入 `signup_source`、reserved email、identity binding 和默认权益逻辑。
- 后台设置 API 能保存、返回 UniFed 配置和默认权益字段。
- 前端 settings helper 能处理 UniFed 默认权益。
- 登录页、注册页、个人资料页根据 `unifed_oauth_enabled` 显示 UniFed 入口。
- `/auth/unifed/callback` 能调用正确 API 并处理 pending OAuth 状态。

完成前至少运行：

```bash
git diff --check
go test ./internal/service ./internal/handler ./internal/server
pnpm --dir frontend test:run
pnpm --dir frontend typecheck
```

如某些命令受环境限制无法运行，需要记录原因、影响范围和替代检查。

## 验收标准

- 管理员可启用 UniFed，并配置实例地址与回调地址。
- 默认实例地址为 `https://dc.hhhl.cc`，但可被管理员覆盖。
- 用户可通过 UniFed 登录或注册。
- 用户可在个人资料页绑定 UniFed。
- UniFed 注册用户可获得对应 auth source defaults。
- 禁用 UniFed 后公开设置关闭入口，个人资料页不允许新绑定。
- 核心后端和前端测试覆盖新增行为并通过。
