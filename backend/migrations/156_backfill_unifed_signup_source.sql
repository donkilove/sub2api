-- 修复 UniFed 约束放开前失败登录留下的半成品用户：
-- 这类用户邮箱是系统生成的 @unifed-connect.invalid，但 signup_source 仍为 email。
UPDATE users
SET signup_source = 'unifed'
WHERE signup_source <> 'unifed'
  AND lower(email) LIKE 'unifed-%@unifed-connect.invalid';
