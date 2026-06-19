-- 154_add_announcement_is_pinned.sql
-- 添加公告置顶字段

-- 添加 is_pinned 列
ALTER TABLE announcements ADD COLUMN IF NOT EXISTS is_pinned boolean NOT NULL DEFAULT false;

-- 创建部分唯一索引，确保最多只有一条公告被置顶
-- PostgreSQL 部分索引：只对 is_pinned=true 的行创建唯一约束
CREATE UNIQUE INDEX IF NOT EXISTS idx_announcements_single_pinned
    ON announcements (is_pinned)
    WHERE is_pinned = true;