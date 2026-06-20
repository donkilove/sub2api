-- 新版限流中心：分组默认用户限流 + 用户独立覆盖。
-- 语义：
--   groups.user_concurrency_limit: 分组默认用户并发上限，0 = 不限流。
--   users.user_concurrency_override: NULL = 继承分组，0 = 不限流，>0 = 用户全局并发覆盖。
--   users.user_rpm_limit_override: NULL = 继承分组，0 = 不限流，>0 = 用户全局 RPM 覆盖。

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS user_concurrency_limit integer NOT NULL DEFAULT 5;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS user_concurrency_override integer NULL;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS user_rpm_limit_override integer NULL;

COMMENT ON COLUMN groups.user_concurrency_limit IS '分组默认用户并发上限，0 表示不限制';
COMMENT ON COLUMN users.user_concurrency_override IS '用户独立并发覆盖：NULL 继承分组，0 不限流，>0 限制';
COMMENT ON COLUMN users.user_rpm_limit_override IS '用户独立 RPM 覆盖：NULL 继承分组，0 不限流，>0 限制';
