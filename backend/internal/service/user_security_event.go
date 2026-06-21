package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	UserSecurityEventRegister       = "register"
	UserSecurityEventLogin          = "login"
	UserSecurityEventLogin2FA       = "login_2fa"
	UserSecurityEventOAuthRegister  = "oauth_register"
	UserSecurityEventOAuthLogin     = "oauth_login"
	UserSecurityEventOAuthBindLogin = "oauth_bind_login"
)

// UserSecurityEventService records auth-side security signals for admin risk analysis.
// Recording is intentionally best-effort; callers should not fail auth flows when it fails.
type UserSecurityEventService struct {
	db  *sql.DB
	now func() time.Time
}

func NewUserSecurityEventService(db *sql.DB) *UserSecurityEventService {
	return &UserSecurityEventService{
		db:  db,
		now: func() time.Time { return time.Now().UTC() },
	}
}

type UserSecurityEventInput struct {
	UserID    int64
	Email     string
	EventType string
	Provider  string
	IPAddress string
	UserAgent string
	Success   bool
	Reason    string
	Metadata  map[string]any
}

func (s *UserSecurityEventService) Record(ctx context.Context, input UserSecurityEventInput) error {
	if s == nil || s.db == nil {
		return nil
	}
	eventType := normalizeUserSecurityEventToken(input.EventType)
	if eventType == "" {
		return nil
	}
	provider := normalizeUserSecurityEventToken(input.Provider)
	if provider == "" {
		provider = "email"
	}
	email := strings.ToLower(strings.TrimSpace(input.Email))
	ipAddress := trimSecurityEventString(input.IPAddress, 45)
	userAgent := trimSecurityEventString(input.UserAgent, 1024)
	reason := trimSecurityEventString(input.Reason, 512)

	metadata := input.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		logger.LegacyPrintf("service.user_security_event", "[UserSecurityEvent] metadata marshal failed: %v", err)
		metadataJSON = []byte(`{}`)
	}

	var userID any
	if input.UserID > 0 {
		userID = input.UserID
	}

	_, err = s.db.ExecContext(ctx, `
INSERT INTO user_security_events (
  user_id, email, event_type, provider, ip_address, user_agent, success, reason, metadata, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10)`,
		userID,
		email,
		eventType,
		provider,
		ipAddress,
		userAgent,
		input.Success,
		reason,
		string(metadataJSON),
		s.now(),
	)
	return err
}

func normalizeUserSecurityEventToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) > 64 {
		value = value[:64]
	}
	return value
}

func trimSecurityEventString(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max]
}
