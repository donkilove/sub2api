-- 用户安全事件：记录注册、登录、OAuth 创建/绑定等认证侧行为。
-- 仅用于管理员风控分析；当前版本不参与自动拦截。

CREATE TABLE IF NOT EXISTS user_security_events (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    email VARCHAR(255) NOT NULL DEFAULT '',
    event_type VARCHAR(64) NOT NULL,
    provider VARCHAR(64) NOT NULL DEFAULT '',
    ip_address VARCHAR(45) NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    success BOOLEAN NOT NULL DEFAULT true,
    reason TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_security_events_user_time
    ON user_security_events(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_security_events_email_time
    ON user_security_events(lower(email), created_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_security_events_ip_time
    ON user_security_events(ip_address, created_at DESC)
    WHERE ip_address <> '';

CREATE INDEX IF NOT EXISTS idx_user_security_events_type_time
    ON user_security_events(event_type, created_at DESC);

COMMENT ON TABLE user_security_events IS '用户认证侧安全事件，用于注册/登录/OAuth 风控分析';
COMMENT ON COLUMN user_security_events.event_type IS '事件类型：register/login/login_2fa/oauth_register/oauth_login/oauth_bind_login 等';
COMMENT ON COLUMN user_security_events.provider IS '认证来源：email/linuxdo/wechat/oidc/github/google/dingtalk/unifed 等';
COMMENT ON COLUMN user_security_events.ip_address IS '认证事件客户端 IP，空字符串表示未采集到';
