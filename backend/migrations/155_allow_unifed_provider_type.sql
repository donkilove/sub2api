-- 允许 Universe Federation 作为登录/注册来源。
-- 代码层面的 validator 已支持 unifed，但历史数据库约束仍停留在 dingtalk 版本，
-- 会导致首次登录自动注册后的身份绑定失败。

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_signup_source_check;

ALTER TABLE users
    ADD CONSTRAINT users_signup_source_check
    CHECK (signup_source IN ('email', 'linuxdo', 'wechat', 'oidc', 'github', 'google', 'dingtalk', 'unifed'));

ALTER TABLE auth_identities
    DROP CONSTRAINT IF EXISTS auth_identities_provider_type_check;

ALTER TABLE auth_identities
    ADD CONSTRAINT auth_identities_provider_type_check
    CHECK (provider_type IN ('email', 'linuxdo', 'wechat', 'oidc', 'github', 'google', 'dingtalk', 'unifed'));

ALTER TABLE auth_identity_channels
    DROP CONSTRAINT IF EXISTS auth_identity_channels_provider_type_check;

ALTER TABLE auth_identity_channels
    ADD CONSTRAINT auth_identity_channels_provider_type_check
    CHECK (provider_type IN ('email', 'linuxdo', 'wechat', 'oidc', 'github', 'google', 'dingtalk', 'unifed'));

ALTER TABLE pending_auth_sessions
    DROP CONSTRAINT IF EXISTS pending_auth_sessions_provider_type_check;

ALTER TABLE pending_auth_sessions
    ADD CONSTRAINT pending_auth_sessions_provider_type_check
    CHECK (provider_type IN ('email', 'linuxdo', 'wechat', 'oidc', 'github', 'google', 'dingtalk', 'unifed'));

ALTER TABLE user_provider_default_grants
    DROP CONSTRAINT IF EXISTS user_provider_default_grants_provider_type_check;

ALTER TABLE user_provider_default_grants
    ADD CONSTRAINT user_provider_default_grants_provider_type_check
    CHECK (provider_type IN ('email', 'linuxdo', 'wechat', 'oidc', 'github', 'google', 'dingtalk', 'unifed'));

INSERT INTO settings (key, value)
VALUES
    ('auth_source_default_unifed_balance', '0'),
    ('auth_source_default_unifed_concurrency', '5'),
    ('auth_source_default_unifed_subscriptions', '[]'),
    ('auth_source_default_unifed_grant_on_signup', 'false'),
    ('auth_source_default_unifed_grant_on_first_bind', 'false')
ON CONFLICT (key) DO NOTHING;
