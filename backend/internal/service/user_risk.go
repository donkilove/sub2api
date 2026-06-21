package service

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/lib/pq"
)

const (
	UserRiskLevelLow      = "low"
	UserRiskLevelMedium   = "medium"
	UserRiskLevelHigh     = "high"
	UserRiskLevelCritical = "critical"
)

var ErrUserRiskWindowInvalid = infraerrors.BadRequest(
	"USER_RISK_WINDOW_INVALID",
	"window must be one of 1h, 24h, 7d, 30d",
)

type UserRiskService struct {
	db                 *sql.DB
	concurrencyService *ConcurrencyService
	now                func() time.Time
}

func NewUserRiskService(db *sql.DB, concurrencyService *ConcurrencyService) *UserRiskService {
	return &UserRiskService{
		db:                 db,
		concurrencyService: concurrencyService,
		now:                func() time.Time { return time.Now().UTC() },
	}
}

type UserRiskListParams struct {
	Page      int
	PageSize  int
	Window    string
	Search    string
	Status    string
	RiskLevel string
	OnlyRisky bool
}

type UserRiskWindow struct {
	Label     string    `json:"label"`
	StartAt   time.Time `json:"start_at"`
	EndAt     time.Time `json:"end_at"`
	PrevStart time.Time `json:"prev_start_at"`
}

type UserRiskListResult struct {
	Items       []*UserRiskItem `json:"items"`
	Total       int64           `json:"total"`
	Page        int             `json:"page"`
	PageSize    int             `json:"page_size"`
	Pages       int             `json:"pages"`
	Summary     UserRiskSummary `json:"summary"`
	Window      UserRiskWindow  `json:"window"`
	GeneratedAt time.Time       `json:"generated_at"`
}

type UserRiskSummary struct {
	TotalUsers        int64   `json:"total_users"`
	RiskyUsers        int64   `json:"risky_users"`
	HighRiskUsers     int64   `json:"high_risk_users"`
	CriticalRiskUsers int64   `json:"critical_risk_users"`
	RequestCount      int64   `json:"request_count"`
	ErrorCount        int64   `json:"error_count"`
	ErrorRate         float64 `json:"error_rate"`
	ActiveConcurrency int64   `json:"active_concurrency"`
	WaitingInQueue    int64   `json:"waiting_in_queue"`
	SharedIPUsers     int64   `json:"shared_ip_users"`
	SharedIPGroups    int64   `json:"shared_ip_groups"`
}

type UserRiskItem struct {
	User        UserRiskUser         `json:"user"`
	Score       int                  `json:"score"`
	Level       string               `json:"level"`
	Reasons     []UserRiskReason     `json:"reasons"`
	Metrics     UserRiskMetrics      `json:"metrics"`
	Limits      UserRiskLimits       `json:"limits"`
	APIKeys     UserRiskAPIKeyStats  `json:"api_keys"`
	Concurrency UserRiskConcurrency  `json:"concurrency"`
	IPRisk      UserRiskIPRisk       `json:"ip_risk"`
	LastError   *UserRiskRecentError `json:"last_error,omitempty"`
}

type UserRiskUser struct {
	ID           int64      `json:"id"`
	Email        string     `json:"email"`
	Username     string     `json:"username"`
	Notes        string     `json:"notes"`
	Role         string     `json:"role"`
	Status       string     `json:"status"`
	SignupSource string     `json:"signup_source"`
	Balance      float64    `json:"balance"`
	CreatedAt    time.Time  `json:"created_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	LastActiveAt *time.Time `json:"last_active_at,omitempty"`
}

type UserRiskLimits struct {
	LegacyConcurrency       int  `json:"legacy_concurrency"`
	LegacyRPM               int  `json:"legacy_rpm"`
	UserConcurrencyOverride *int `json:"user_concurrency_override,omitempty"`
	UserRPMLimitOverride    *int `json:"user_rpm_limit_override,omitempty"`
}

type UserRiskAPIKeyStats struct {
	Total       int64      `json:"total"`
	Active      int64      `json:"active"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	ActiveRatio float64    `json:"active_ratio"`
}

type UserRiskConcurrency struct {
	CurrentInUse   int64   `json:"current_in_use"`
	WaitingInQueue int64   `json:"waiting_in_queue"`
	MaxCapacity    int64   `json:"max_capacity"`
	LoadPercentage float64 `json:"load_percentage"`
	Collected      bool    `json:"collected"`
}

type UserRiskMetrics struct {
	RequestCount       int64      `json:"request_count"`
	PreviousRequests   int64      `json:"previous_request_count"`
	RequestGrowthRatio float64    `json:"request_growth_ratio"`
	TokenCount         int64      `json:"token_count"`
	Cost               float64    `json:"cost"`
	PreviousCost       float64    `json:"previous_cost"`
	CostGrowthRatio    float64    `json:"cost_growth_ratio"`
	AverageLatencyMs   float64    `json:"avg_latency_ms"`
	UniqueIPs          int64      `json:"unique_ips"`
	UniqueModels       int64      `json:"unique_models"`
	ErrorCount         int64      `json:"error_count"`
	ErrorRate          float64    `json:"error_rate"`
	RateLimitedCount   int64      `json:"rate_limited_count"`
	AuthErrorCount     int64      `json:"auth_error_count"`
	Upstream5xxCount   int64      `json:"upstream_5xx_count"`
	TimeoutCount       int64      `json:"timeout_count"`
	LastRequestAt      *time.Time `json:"last_request_at,omitempty"`
	LastErrorAt        *time.Time `json:"last_error_at,omitempty"`
}

type UserRiskIPRisk struct {
	SharedIPCount      int64 `json:"shared_ip_count"`
	LinkedUserCount    int64 `json:"linked_user_count"`
	MaxUsersOnSameIP   int64 `json:"max_users_on_same_ip"`
	NewUsersOnSharedIP int64 `json:"new_users_on_shared_ip"`
	SameUAUserCount    int64 `json:"same_ua_user_count"`
	AuthEventCount     int64 `json:"auth_event_count"`
	RegisterEventCount int64 `json:"register_event_count"`
}

type UserRiskReason struct {
	Code     string `json:"code"`
	Label    string `json:"label"`
	Severity string `json:"severity"`
	Detail   string `json:"detail"`
}

type UserRiskDetail struct {
	Item         *UserRiskItem          `json:"item"`
	APIKeys      []UserRiskAPIKeyDetail `json:"api_keys"`
	RecentErrors []UserRiskRecentError  `json:"recent_errors"`
	TopModels    []UserRiskTopModel     `json:"top_models"`
	TopIPs       []UserRiskIPStat       `json:"top_ips"`
	IPLinks      []UserRiskIPLink       `json:"ip_links"`
	AuthEvents   []UserRiskAuthEvent    `json:"auth_events"`
	Window       UserRiskWindow         `json:"window"`
	GeneratedAt  time.Time              `json:"generated_at"`
}

type UserRiskAPIKeyDetail struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	Status        string     `json:"status"`
	GroupID       *int64     `json:"group_id,omitempty"`
	GroupName     string     `json:"group_name"`
	RequestCount  int64      `json:"request_count"`
	ErrorCount    int64      `json:"error_count"`
	Cost          float64    `json:"cost"`
	LastUsedAt    *time.Time `json:"last_used_at,omitempty"`
	LastRequestAt *time.Time `json:"last_request_at,omitempty"`
}

type UserRiskRecentError struct {
	ID         int64     `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	Phase      string    `json:"phase"`
	Type       string    `json:"type"`
	StatusCode int       `json:"status_code"`
	Message    string    `json:"message"`
	RequestID  string    `json:"request_id"`
	Model      string    `json:"model"`
}

type UserRiskTopModel struct {
	Model        string     `json:"model"`
	RequestCount int64      `json:"request_count"`
	TokenCount   int64      `json:"token_count"`
	Cost         float64    `json:"cost"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
}

type UserRiskIPStat struct {
	IP           string     `json:"ip"`
	RequestCount int64      `json:"request_count"`
	ErrorCount   int64      `json:"error_count"`
	LastSeenAt   *time.Time `json:"last_seen_at,omitempty"`
}

type UserRiskIPLink struct {
	IP                 string               `json:"ip"`
	Source             string               `json:"source"`
	UserCount          int64                `json:"user_count"`
	OtherUserCount     int64                `json:"other_user_count"`
	NewUserCount       int64                `json:"new_user_count"`
	SameUAUserCount    int64                `json:"same_ua_user_count"`
	RequestCount       int64                `json:"request_count"`
	ErrorCount         int64                `json:"error_count"`
	AuthEventCount     int64                `json:"auth_event_count"`
	RegisterEventCount int64                `json:"register_event_count"`
	Cost               float64              `json:"cost"`
	LastSeenAt         *time.Time           `json:"last_seen_at,omitempty"`
	LinkedUsers        []UserRiskLinkedUser `json:"linked_users"`
}

type UserRiskLinkedUser struct {
	ID                  int64      `json:"id"`
	Email               string     `json:"email"`
	Username            string     `json:"username"`
	Status              string     `json:"status"`
	SignupSource        string     `json:"signup_source"`
	Balance             float64    `json:"balance"`
	CreatedAt           time.Time  `json:"created_at"`
	RequestCount        int64      `json:"request_count"`
	AuthEventCount      int64      `json:"auth_event_count"`
	RegisterEventCount  int64      `json:"register_event_count"`
	LastSeenAt          *time.Time `json:"last_seen_at,omitempty"`
	LastAuthEventAt     *time.Time `json:"last_auth_event_at,omitempty"`
	SharedUserAgentHint bool       `json:"shared_user_agent_hint"`
}

type UserRiskAuthEvent struct {
	ID        int64     `json:"id"`
	UserID    *int64    `json:"user_id,omitempty"`
	Email     string    `json:"email"`
	EventType string    `json:"event_type"`
	Provider  string    `json:"provider"`
	IPAddress string    `json:"ip_address"`
	Success   bool      `json:"success"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

type userRiskAggregate struct {
	user        UserRiskUser
	limits      UserRiskLimits
	apiKeys     UserRiskAPIKeyStats
	metrics     UserRiskMetrics
	concurrency UserRiskConcurrency
	ipRisk      UserRiskIPRisk
	lastError   *UserRiskRecentError
}

type userRiskEvaluationInput struct {
	RequestCount       int64
	PreviousRequests   int64
	Cost               float64
	PreviousCost       float64
	ErrorCount         int64
	RateLimitedCount   int64
	AuthErrorCount     int64
	Upstream5xxCount   int64
	TimeoutCount       int64
	UniqueIPs          int64
	ActiveAPIKeys      int64
	Balance            float64
	CurrentConcurrency int64
	WaitingInQueue     int64
	MaxConcurrency     int64
	UserStatus         string
	IPRisk             UserRiskIPRisk
}

func (s *UserRiskService) List(ctx context.Context, params UserRiskListParams) (*UserRiskListResult, error) {
	params = normalizeUserRiskListParams(params)
	window, err := s.buildWindow(params.Window)
	if err != nil {
		return nil, err
	}

	aggregates, err := s.fetchRiskAggregates(ctx, params, window, 0)
	if err != nil {
		return nil, err
	}
	if err := s.enrichIPRisk(ctx, aggregates, window); err != nil {
		return nil, err
	}
	s.enrichConcurrency(ctx, aggregates)

	items := make([]*UserRiskItem, 0, len(aggregates))
	for i := range aggregates {
		items = append(items, buildUserRiskItem(&aggregates[i]))
	}
	sortUserRiskItems(items)

	filtered := filterUserRiskItems(items, params)
	summary := summarizeUserRisk(filtered)

	total := int64(len(filtered))
	pages := int(math.Ceil(float64(total) / float64(params.PageSize)))
	if pages < 1 {
		pages = 1
	}
	start := (params.Page - 1) * params.PageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + params.PageSize
	if end > len(filtered) {
		end = len(filtered)
	}

	return &UserRiskListResult{
		Items:       filtered[start:end],
		Total:       total,
		Page:        params.Page,
		PageSize:    params.PageSize,
		Pages:       pages,
		Summary:     summary,
		Window:      window,
		GeneratedAt: s.now(),
	}, nil
}

func (s *UserRiskService) Get(ctx context.Context, userID int64, windowLabel string) (*UserRiskDetail, error) {
	if userID <= 0 {
		return nil, infraerrors.BadRequest("USER_RISK_USER_ID_INVALID", "invalid user id")
	}
	window, err := s.buildWindow(windowLabel)
	if err != nil {
		return nil, err
	}
	aggregates, err := s.fetchRiskAggregates(ctx, UserRiskListParams{Page: 1, PageSize: 1}, window, userID)
	if err != nil {
		return nil, err
	}
	if len(aggregates) == 0 {
		return nil, infraerrors.NotFound("USER_RISK_USER_NOT_FOUND", "user not found")
	}
	if err := s.enrichIPRisk(ctx, aggregates, window); err != nil {
		return nil, err
	}
	s.enrichConcurrency(ctx, aggregates)
	item := buildUserRiskItem(&aggregates[0])

	apiKeys, err := s.fetchAPIKeyDetails(ctx, userID, window)
	if err != nil {
		return nil, err
	}
	recentErrors, err := s.fetchRecentErrors(ctx, userID, window)
	if err != nil {
		return nil, err
	}
	topModels, err := s.fetchTopModels(ctx, userID, window)
	if err != nil {
		return nil, err
	}
	topIPs, err := s.fetchTopIPs(ctx, userID, window)
	if err != nil {
		return nil, err
	}
	ipLinks, err := s.fetchIPLinks(ctx, userID, window)
	if err != nil {
		return nil, err
	}
	authEvents, err := s.fetchAuthEvents(ctx, userID, window)
	if err != nil {
		return nil, err
	}

	return &UserRiskDetail{
		Item:         item,
		APIKeys:      apiKeys,
		RecentErrors: recentErrors,
		TopModels:    topModels,
		TopIPs:       topIPs,
		IPLinks:      ipLinks,
		AuthEvents:   authEvents,
		Window:       window,
		GeneratedAt:  s.now(),
	}, nil
}

func (s *UserRiskService) buildWindow(label string) (UserRiskWindow, error) {
	duration, normalized, ok := ParseUserRiskWindow(label)
	if !ok {
		return UserRiskWindow{}, ErrUserRiskWindowInvalid
	}
	end := s.now()
	start := end.Add(-duration)
	return UserRiskWindow{
		Label:     normalized,
		StartAt:   start,
		EndAt:     end,
		PrevStart: start.Add(-duration),
	}, nil
}

func ParseUserRiskWindow(label string) (time.Duration, string, bool) {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "", "24h", "1d":
		return 24 * time.Hour, "24h", true
	case "1h", "60m", "60min":
		return time.Hour, "1h", true
	case "7d":
		return 7 * 24 * time.Hour, "7d", true
	case "30d":
		return 30 * 24 * time.Hour, "30d", true
	default:
		return 0, "", false
	}
}

func normalizeUserRiskListParams(params UserRiskListParams) UserRiskListParams {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}
	params.Search = strings.TrimSpace(params.Search)
	params.Status = strings.TrimSpace(params.Status)
	params.RiskLevel = strings.ToLower(strings.TrimSpace(params.RiskLevel))
	return params
}

func (s *UserRiskService) fetchRiskAggregates(ctx context.Context, params UserRiskListParams, window UserRiskWindow, userID int64) ([]userRiskAggregate, error) {
	if s == nil || s.db == nil {
		return []userRiskAggregate{}, nil
	}

	const q = `
WITH filtered_users AS (
  SELECT
    id, email, username, notes, role, status,
    balance::double precision AS balance,
    concurrency, rpm_limit, user_concurrency_override, user_rpm_limit_override,
    signup_source, last_login_at, last_active_at, created_at
  FROM users
  WHERE deleted_at IS NULL
    AND ($3 = '' OR status = $3)
    AND ($4 = '' OR lower(email) LIKE '%' || lower($4) || '%'
      OR lower(username) LIKE '%' || lower($4) || '%'
      OR lower(notes) LIKE '%' || lower($4) || '%'
      OR id::text = $4)
    AND ($6::bigint = 0 OR id = $6::bigint)
),
cur_usage AS (
  SELECT
    user_id,
    COUNT(*)::bigint AS request_count,
    COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens + cache_creation_5m_tokens + cache_creation_1h_tokens + image_output_tokens), 0)::bigint AS token_count,
    COALESCE(SUM(actual_cost)::double precision, 0) AS cost,
    COALESCE(AVG(duration_ms)::double precision, 0) AS avg_latency_ms,
    COUNT(DISTINCT NULLIF(ip_address, ''))::bigint AS unique_ips,
    COUNT(DISTINCT COALESCE(NULLIF(requested_model, ''), NULLIF(model, '')))::bigint AS unique_models,
    MAX(created_at) AS last_request_at
  FROM usage_logs
  WHERE created_at >= $1 AND created_at < $2
  GROUP BY user_id
),
prev_usage AS (
  SELECT
    user_id,
    COUNT(*)::bigint AS request_count,
    COALESCE(SUM(actual_cost)::double precision, 0) AS cost
  FROM usage_logs
  WHERE created_at >= $5 AND created_at < $1
  GROUP BY user_id
),
err_usage AS (
  SELECT
    COALESCE(user_id, deleted_key_owner_user_id) AS user_id,
    COUNT(*)::bigint AS error_count,
    COUNT(*) FILTER (
      WHERE COALESCE(status_code, upstream_status_code) = 429
        OR is_business_limited = true
        OR error_type ILIKE '%rate%'
        OR error_type ILIKE '%limit%'
    )::bigint AS rate_limited_count,
    COUNT(*) FILTER (
      WHERE error_phase = 'auth'
        OR COALESCE(status_code, upstream_status_code) IN (401, 403)
        OR error_type ILIKE '%auth%'
        OR error_type ILIKE '%permission%'
        OR error_type ILIKE '%invalid_api_key%'
    )::bigint AS auth_error_count,
    COUNT(*) FILTER (
      WHERE COALESCE(upstream_status_code, status_code) BETWEEN 500 AND 599
        OR error_type ILIKE '%upstream%'
        OR error_type ILIKE '%server%'
    )::bigint AS upstream_5xx_count,
    COUNT(*) FILTER (
      WHERE error_type ILIKE '%timeout%'
        OR network_error_type ILIKE '%timeout%'
        OR error_message ILIKE '%timeout%'
    )::bigint AS timeout_count,
    MAX(created_at) AS last_error_at
  FROM ops_error_logs
  WHERE created_at >= $1 AND created_at < $2
    AND COALESCE(user_id, deleted_key_owner_user_id) IS NOT NULL
  GROUP BY COALESCE(user_id, deleted_key_owner_user_id)
),
key_usage AS (
  SELECT
    user_id,
    COUNT(*)::bigint AS total_api_keys,
    COUNT(*) FILTER (WHERE status = 'active')::bigint AS active_api_keys,
    MAX(last_used_at) AS last_key_used_at
  FROM api_keys
  WHERE deleted_at IS NULL
  GROUP BY user_id
)
SELECT
  u.id, u.email, u.username, u.notes, u.role, u.status, u.balance,
  u.concurrency, u.rpm_limit, u.user_concurrency_override, u.user_rpm_limit_override,
  u.signup_source, u.last_login_at, u.last_active_at, u.created_at,
  COALESCE(k.total_api_keys, 0), COALESCE(k.active_api_keys, 0), k.last_key_used_at,
  COALESCE(c.request_count, 0), COALESCE(p.request_count, 0),
  COALESCE(c.token_count, 0), COALESCE(c.cost, 0), COALESCE(p.cost, 0),
  COALESCE(c.avg_latency_ms, 0), COALESCE(c.unique_ips, 0), COALESCE(c.unique_models, 0), c.last_request_at,
  COALESCE(e.error_count, 0), COALESCE(e.rate_limited_count, 0), COALESCE(e.auth_error_count, 0),
  COALESCE(e.upstream_5xx_count, 0), COALESCE(e.timeout_count, 0), e.last_error_at,
  le.id, le.created_at, le.error_phase, le.error_type, COALESCE(le.status_code, le.upstream_status_code),
  COALESCE(NULLIF(le.error_message, ''), NULLIF(le.upstream_error_message, ''), ''), le.request_id,
  COALESCE(NULLIF(le.requested_model, ''), NULLIF(le.model, ''), '')
FROM filtered_users u
LEFT JOIN key_usage k ON k.user_id = u.id
LEFT JOIN cur_usage c ON c.user_id = u.id
LEFT JOIN prev_usage p ON p.user_id = u.id
LEFT JOIN err_usage e ON e.user_id = u.id
LEFT JOIN LATERAL (
  SELECT id, created_at, error_phase, error_type, status_code, upstream_status_code,
         error_message, upstream_error_message, request_id, requested_model, model
  FROM ops_error_logs
  WHERE created_at >= $1 AND created_at < $2
    AND COALESCE(user_id, deleted_key_owner_user_id) = u.id
  ORDER BY created_at DESC
  LIMIT 1
) le ON true
ORDER BY u.id ASC`

	rows, err := s.db.QueryContext(
		ctx,
		q,
		window.StartAt,
		window.EndAt,
		params.Status,
		params.Search,
		window.PrevStart,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]userRiskAggregate, 0, 64)
	for rows.Next() {
		var row userRiskAggregate
		var userConcurrencyOverride, userRPMLimitOverride sql.NullInt64
		var lastLoginAt, lastActiveAt, keyLastUsedAt, lastRequestAt, lastErrorAt sql.NullTime
		var lastErrorID, lastErrorStatus sql.NullInt64
		var lastErrorCreatedAt sql.NullTime
		var lastErrorPhase, lastErrorType, lastErrorMessage, lastErrorRequestID, lastErrorModel sql.NullString

		if err := rows.Scan(
			&row.user.ID,
			&row.user.Email,
			&row.user.Username,
			&row.user.Notes,
			&row.user.Role,
			&row.user.Status,
			&row.user.Balance,
			&row.limits.LegacyConcurrency,
			&row.limits.LegacyRPM,
			&userConcurrencyOverride,
			&userRPMLimitOverride,
			&row.user.SignupSource,
			&lastLoginAt,
			&lastActiveAt,
			&row.user.CreatedAt,
			&row.apiKeys.Total,
			&row.apiKeys.Active,
			&keyLastUsedAt,
			&row.metrics.RequestCount,
			&row.metrics.PreviousRequests,
			&row.metrics.TokenCount,
			&row.metrics.Cost,
			&row.metrics.PreviousCost,
			&row.metrics.AverageLatencyMs,
			&row.metrics.UniqueIPs,
			&row.metrics.UniqueModels,
			&lastRequestAt,
			&row.metrics.ErrorCount,
			&row.metrics.RateLimitedCount,
			&row.metrics.AuthErrorCount,
			&row.metrics.Upstream5xxCount,
			&row.metrics.TimeoutCount,
			&lastErrorAt,
			&lastErrorID,
			&lastErrorCreatedAt,
			&lastErrorPhase,
			&lastErrorType,
			&lastErrorStatus,
			&lastErrorMessage,
			&lastErrorRequestID,
			&lastErrorModel,
		); err != nil {
			return nil, err
		}

		if lastLoginAt.Valid {
			row.user.LastLoginAt = &lastLoginAt.Time
		}
		if lastActiveAt.Valid {
			row.user.LastActiveAt = &lastActiveAt.Time
		}
		if userConcurrencyOverride.Valid {
			v := int(userConcurrencyOverride.Int64)
			row.limits.UserConcurrencyOverride = &v
		}
		if userRPMLimitOverride.Valid {
			v := int(userRPMLimitOverride.Int64)
			row.limits.UserRPMLimitOverride = &v
		}
		if keyLastUsedAt.Valid {
			row.apiKeys.LastUsedAt = &keyLastUsedAt.Time
		}
		if row.apiKeys.Total > 0 {
			row.apiKeys.ActiveRatio = float64(row.apiKeys.Active) / float64(row.apiKeys.Total)
		}
		if lastRequestAt.Valid {
			row.metrics.LastRequestAt = &lastRequestAt.Time
		}
		if lastErrorAt.Valid {
			row.metrics.LastErrorAt = &lastErrorAt.Time
		}
		row.metrics.ErrorRate = ratio(row.metrics.ErrorCount, row.metrics.RequestCount+row.metrics.ErrorCount)
		row.metrics.RequestGrowthRatio = growthRatio(float64(row.metrics.RequestCount), float64(row.metrics.PreviousRequests))
		row.metrics.CostGrowthRatio = growthRatio(row.metrics.Cost, row.metrics.PreviousCost)
		row.concurrency.MaxCapacity = int64(row.limits.LegacyConcurrency)
		if row.limits.UserConcurrencyOverride != nil {
			row.concurrency.MaxCapacity = int64(*row.limits.UserConcurrencyOverride)
		}

		if lastErrorID.Valid && lastErrorCreatedAt.Valid {
			row.lastError = &UserRiskRecentError{
				ID:         lastErrorID.Int64,
				CreatedAt:  lastErrorCreatedAt.Time,
				Phase:      lastErrorPhase.String,
				Type:       lastErrorType.String,
				StatusCode: int(lastErrorStatus.Int64),
				Message:    lastErrorMessage.String,
				RequestID:  lastErrorRequestID.String,
				Model:      lastErrorModel.String,
			}
		}

		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *UserRiskService) enrichConcurrency(ctx context.Context, rows []userRiskAggregate) {
	if s == nil || s.concurrencyService == nil || len(rows) == 0 {
		return
	}
	batch := make([]UserWithConcurrency, 0, len(rows))
	for i := range rows {
		max := int(rows[i].concurrency.MaxCapacity)
		batch = append(batch, UserWithConcurrency{ID: rows[i].user.ID, MaxConcurrency: max})
	}
	loadMap, err := s.concurrencyService.GetUsersLoadBatch(ctx, batch)
	if err != nil {
		return
	}
	for i := range rows {
		load := loadMap[rows[i].user.ID]
		if load == nil {
			continue
		}
		rows[i].concurrency.Collected = true
		rows[i].concurrency.CurrentInUse = int64(load.CurrentConcurrency)
		rows[i].concurrency.WaitingInQueue = int64(load.WaitingCount)
		if rows[i].concurrency.MaxCapacity > 0 {
			rows[i].concurrency.LoadPercentage = float64(rows[i].concurrency.CurrentInUse) / float64(rows[i].concurrency.MaxCapacity) * 100
		} else if load.LoadRate > 0 {
			rows[i].concurrency.LoadPercentage = float64(load.LoadRate)
		}
	}
}

func (s *UserRiskService) enrichIPRisk(ctx context.Context, rows []userRiskAggregate, window UserRiskWindow) error {
	if s == nil || s.db == nil || len(rows) == 0 {
		return nil
	}
	userIDs := make([]int64, 0, len(rows))
	indexByID := make(map[int64]int, len(rows))
	for i := range rows {
		if rows[i].user.ID <= 0 {
			continue
		}
		userIDs = append(userIDs, rows[i].user.ID)
		indexByID[rows[i].user.ID] = i
	}
	if len(userIDs) == 0 {
		return nil
	}

	const q = `
WITH selected_users AS (
  SELECT unnest($3::bigint[]) AS user_id
),
user_ip_ua AS (
  SELECT user_id, NULLIF(ip_address, '') AS ip, NULLIF(user_agent, '') AS ua, 'usage' AS source, created_at
  FROM usage_logs
  WHERE user_id = ANY($3::bigint[]) AND created_at >= $1 AND created_at < $2 AND NULLIF(ip_address, '') IS NOT NULL
  UNION ALL
  SELECT user_id, NULLIF(ip_address, '') AS ip, NULLIF(user_agent, '') AS ua, 'auth' AS source, created_at
  FROM user_security_events
  WHERE user_id = ANY($3::bigint[]) AND created_at >= $1 AND created_at < $2 AND NULLIF(ip_address, '') IS NOT NULL
),
candidate_ips AS (
  SELECT DISTINCT ip FROM user_ip_ua WHERE ip IS NOT NULL
),
all_ip_users AS (
  SELECT ul.user_id, NULLIF(ul.ip_address, '') AS ip, NULLIF(ul.user_agent, '') AS ua, 'usage' AS source, ul.created_at, false AS is_register
  FROM usage_logs ul
  JOIN candidate_ips ci ON ci.ip = NULLIF(ul.ip_address, '')
  WHERE ul.created_at >= $1 AND ul.created_at < $2
  UNION ALL
  SELECT se.user_id, NULLIF(se.ip_address, '') AS ip, NULLIF(se.user_agent, '') AS ua, 'auth' AS source, se.created_at,
         se.event_type IN ('register', 'oauth_register') AS is_register
  FROM user_security_events se
  JOIN candidate_ips ci ON ci.ip = NULLIF(se.ip_address, '')
  WHERE se.created_at >= $1 AND se.created_at < $2
),
ip_stats AS (
  SELECT
    ui.user_id,
    ui.ip,
    COUNT(DISTINCT au.user_id)::bigint AS user_count,
    COUNT(DISTINCT au.user_id) FILTER (WHERE au.user_id <> ui.user_id)::bigint AS other_user_count,
    COUNT(DISTINCT au.user_id) FILTER (WHERE u.created_at >= $4)::bigint AS new_user_count,
    COUNT(DISTINCT au.user_id) FILTER (WHERE ui.ua IS NOT NULL AND ui.ua <> '' AND au.ua = ui.ua)::bigint AS same_ua_user_count,
    COUNT(*) FILTER (WHERE au.source = 'auth')::bigint AS auth_event_count,
    COUNT(*) FILTER (WHERE au.is_register)::bigint AS register_event_count
  FROM user_ip_ua ui
  JOIN all_ip_users au ON au.ip = ui.ip
  LEFT JOIN users u ON u.id = au.user_id
  WHERE au.user_id IS NOT NULL
  GROUP BY ui.user_id, ui.ip
)
SELECT
  user_id,
  COUNT(*) FILTER (WHERE other_user_count > 0)::bigint AS shared_ip_count,
  COALESCE(SUM(other_user_count), 0)::bigint AS linked_user_count,
  COALESCE(MAX(user_count), 0)::bigint AS max_users_on_same_ip,
  COALESCE(SUM(new_user_count), 0)::bigint AS new_users_on_shared_ip,
  COALESCE(MAX(same_ua_user_count), 0)::bigint AS same_ua_user_count,
  COALESCE(SUM(auth_event_count), 0)::bigint AS auth_event_count,
  COALESCE(SUM(register_event_count), 0)::bigint AS register_event_count
FROM ip_stats
GROUP BY user_id`
	rowsQuery, err := s.db.QueryContext(ctx, q, window.StartAt, window.EndAt, pq.Array(userIDs), window.EndAt.Add(-7*24*time.Hour))
	if err != nil {
		return err
	}
	defer rowsQuery.Close()

	for rowsQuery.Next() {
		var userID int64
		var risk UserRiskIPRisk
		if err := rowsQuery.Scan(
			&userID,
			&risk.SharedIPCount,
			&risk.LinkedUserCount,
			&risk.MaxUsersOnSameIP,
			&risk.NewUsersOnSharedIP,
			&risk.SameUAUserCount,
			&risk.AuthEventCount,
			&risk.RegisterEventCount,
		); err != nil {
			return err
		}
		if idx, ok := indexByID[userID]; ok {
			rows[idx].ipRisk = risk
		}
	}
	return rowsQuery.Err()
}

func buildUserRiskItem(row *userRiskAggregate) *UserRiskItem {
	input := userRiskEvaluationInput{
		RequestCount:       row.metrics.RequestCount,
		PreviousRequests:   row.metrics.PreviousRequests,
		Cost:               row.metrics.Cost,
		PreviousCost:       row.metrics.PreviousCost,
		ErrorCount:         row.metrics.ErrorCount,
		RateLimitedCount:   row.metrics.RateLimitedCount,
		AuthErrorCount:     row.metrics.AuthErrorCount,
		Upstream5xxCount:   row.metrics.Upstream5xxCount,
		TimeoutCount:       row.metrics.TimeoutCount,
		UniqueIPs:          row.metrics.UniqueIPs,
		ActiveAPIKeys:      row.apiKeys.Active,
		Balance:            row.user.Balance,
		CurrentConcurrency: row.concurrency.CurrentInUse,
		WaitingInQueue:     row.concurrency.WaitingInQueue,
		MaxConcurrency:     row.concurrency.MaxCapacity,
		UserStatus:         row.user.Status,
		IPRisk:             row.ipRisk,
	}
	score, level, reasons := evaluateUserRisk(input)
	return &UserRiskItem{
		User:        row.user,
		Score:       score,
		Level:       level,
		Reasons:     reasons,
		Metrics:     row.metrics,
		Limits:      row.limits,
		APIKeys:     row.apiKeys,
		Concurrency: row.concurrency,
		IPRisk:      row.ipRisk,
		LastError:   row.lastError,
	}
}

func evaluateUserRisk(input userRiskEvaluationInput) (int, string, []UserRiskReason) {
	score := 0
	reasons := make([]UserRiskReason, 0, 8)
	add := func(points int, code, label, severity, detail string) {
		if points <= 0 {
			return
		}
		score += points
		reasons = append(reasons, UserRiskReason{Code: code, Label: label, Severity: severity, Detail: detail})
	}

	totalAttempts := input.RequestCount + input.ErrorCount
	errorRate := ratio(input.ErrorCount, totalAttempts)
	if totalAttempts >= 10 {
		switch {
		case errorRate >= 0.50:
			add(35, "high_error_rate", "错误率过高", "high", fmt.Sprintf("错误率 %.1f%%，共 %d 次错误", errorRate*100, input.ErrorCount))
		case errorRate >= 0.25:
			add(24, "elevated_error_rate", "错误率偏高", "medium", fmt.Sprintf("错误率 %.1f%%，共 %d 次错误", errorRate*100, input.ErrorCount))
		case errorRate >= 0.10:
			add(12, "noticeable_error_rate", "错误率上升", "low", fmt.Sprintf("错误率 %.1f%%，共 %d 次错误", errorRate*100, input.ErrorCount))
		}
	}

	if input.RateLimitedCount >= 20 || (totalAttempts >= 10 && ratio(input.RateLimitedCount, totalAttempts) >= 0.30) {
		add(20, "rate_limited", "频繁触发限流", "medium", fmt.Sprintf("限流相关错误 %d 次", input.RateLimitedCount))
	} else if input.RateLimitedCount >= 5 {
		add(10, "rate_limited_some", "出现多次限流", "low", fmt.Sprintf("限流相关错误 %d 次", input.RateLimitedCount))
	}

	if input.AuthErrorCount >= 5 {
		add(30, "auth_errors", "认证或权限错误集中", "high", fmt.Sprintf("认证/权限错误 %d 次", input.AuthErrorCount))
	} else if input.AuthErrorCount >= 2 {
		add(12, "auth_errors_some", "出现认证或权限错误", "medium", fmt.Sprintf("认证/权限错误 %d 次", input.AuthErrorCount))
	}

	if input.Upstream5xxCount >= 10 {
		add(15, "upstream_5xx", "上游错误偏多", "medium", fmt.Sprintf("上游 5xx/服务错误 %d 次", input.Upstream5xxCount))
	} else if input.Upstream5xxCount >= 3 {
		add(8, "upstream_5xx_some", "出现上游错误", "low", fmt.Sprintf("上游 5xx/服务错误 %d 次", input.Upstream5xxCount))
	}

	if input.TimeoutCount >= 5 {
		add(10, "timeout_errors", "请求超时偏多", "medium", fmt.Sprintf("超时错误 %d 次", input.TimeoutCount))
	}

	reqGrowth := growthRatio(float64(input.RequestCount), float64(input.PreviousRequests))
	if input.RequestCount >= 100 && reqGrowth >= 3 {
		add(20, "request_spike", "请求量突增", "medium", fmt.Sprintf("本窗口请求 %d，是上一窗口 %.1f 倍", input.RequestCount, reqGrowth))
	} else if input.RequestCount >= 50 && reqGrowth >= 2 {
		add(10, "request_growth", "请求量增长明显", "low", fmt.Sprintf("本窗口请求 %d，是上一窗口 %.1f 倍", input.RequestCount, reqGrowth))
	}

	costGrowth := growthRatio(input.Cost, input.PreviousCost)
	if input.Cost >= 1 && costGrowth >= 3 {
		add(15, "cost_spike", "消费突增", "medium", fmt.Sprintf("本窗口消费 $%.4f，是上一窗口 %.1f 倍", input.Cost, costGrowth))
	}

	if input.UniqueIPs >= 10 {
		add(15, "many_ips", "来源 IP 分散", "medium", fmt.Sprintf("成功请求出现 %d 个不同 IP", input.UniqueIPs))
	} else if input.UniqueIPs >= 5 {
		add(8, "several_ips", "来源 IP 较多", "low", fmt.Sprintf("成功请求出现 %d 个不同 IP", input.UniqueIPs))
	}

	if input.IPRisk.MaxUsersOnSameIP >= 8 || input.IPRisk.LinkedUserCount >= 12 {
		add(35, "shared_ip_many_accounts", "同 IP 关联账号过多", "high", fmt.Sprintf("共享 IP 关联 %d 个其他账号，单 IP 最多 %d 个账号", input.IPRisk.LinkedUserCount, input.IPRisk.MaxUsersOnSameIP))
	} else if input.IPRisk.MaxUsersOnSameIP >= 4 || input.IPRisk.LinkedUserCount >= 5 {
		add(22, "shared_ip_accounts", "同 IP 关联多个账号", "medium", fmt.Sprintf("共享 IP 关联 %d 个其他账号，单 IP 最多 %d 个账号", input.IPRisk.LinkedUserCount, input.IPRisk.MaxUsersOnSameIP))
	} else if input.IPRisk.LinkedUserCount >= 2 {
		add(10, "shared_ip_some_accounts", "存在同 IP 关联账号", "low", fmt.Sprintf("共享 IP 关联 %d 个其他账号", input.IPRisk.LinkedUserCount))
	}

	if input.IPRisk.NewUsersOnSharedIP >= 5 || input.IPRisk.RegisterEventCount >= 5 {
		add(25, "shared_ip_new_accounts", "同 IP 新账号密集", "high", fmt.Sprintf("共享 IP 下新账号 %d 个，注册事件 %d 次", input.IPRisk.NewUsersOnSharedIP, input.IPRisk.RegisterEventCount))
	} else if input.IPRisk.NewUsersOnSharedIP >= 3 || input.IPRisk.RegisterEventCount >= 3 {
		add(15, "shared_ip_new_accounts_some", "同 IP 新账号偏多", "medium", fmt.Sprintf("共享 IP 下新账号 %d 个，注册事件 %d 次", input.IPRisk.NewUsersOnSharedIP, input.IPRisk.RegisterEventCount))
	}

	if input.IPRisk.SameUAUserCount >= 4 {
		add(12, "shared_ip_same_ua", "同 IP 设备特征相似", "medium", fmt.Sprintf("同 IP 且 User-Agent 相同的账号最多 %d 个", input.IPRisk.SameUAUserCount))
	}

	if input.ActiveAPIKeys >= 5 && input.RequestCount >= 50 {
		add(8, "many_active_keys", "活跃 API Key 较多", "low", fmt.Sprintf("当前有 %d 个启用 Key", input.ActiveAPIKeys))
	}

	if input.WaitingInQueue > 0 {
		add(18, "concurrency_queue", "存在并发排队", "medium", fmt.Sprintf("当前排队 %d 个请求", input.WaitingInQueue))
	} else if input.MaxConcurrency > 0 && input.CurrentConcurrency >= input.MaxConcurrency && input.CurrentConcurrency > 0 {
		add(12, "concurrency_full", "并发已满", "medium", fmt.Sprintf("当前并发 %d/%d", input.CurrentConcurrency, input.MaxConcurrency))
	}

	if strings.EqualFold(input.UserStatus, StatusDisabled) && totalAttempts > 0 {
		add(25, "disabled_user_activity", "禁用用户仍有请求痕迹", "high", fmt.Sprintf("窗口内成功/失败尝试共 %d 次", totalAttempts))
	}

	if input.Balance < 0.1 && input.Cost >= 0.1 {
		add(10, "low_balance_usage", "余额偏低仍在消耗", "low", fmt.Sprintf("余额 $%.4f，本窗口消费 $%.4f", input.Balance, input.Cost))
	}

	if score > 100 {
		score = 100
	}

	level := UserRiskLevelLow
	switch {
	case score >= 75:
		level = UserRiskLevelCritical
	case score >= 50:
		level = UserRiskLevelHigh
	case score >= 25:
		level = UserRiskLevelMedium
	}
	return score, level, reasons
}

func filterUserRiskItems(items []*UserRiskItem, params UserRiskListParams) []*UserRiskItem {
	level := strings.ToLower(strings.TrimSpace(params.RiskLevel))
	if level == "" && !params.OnlyRisky {
		return items
	}
	filtered := make([]*UserRiskItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if params.OnlyRisky && item.Level == UserRiskLevelLow {
			continue
		}
		if level != "" && item.Level != level {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func sortUserRiskItems(items []*UserRiskItem) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.Metrics.ErrorRate != b.Metrics.ErrorRate {
			return a.Metrics.ErrorRate > b.Metrics.ErrorRate
		}
		if a.Metrics.RequestCount != b.Metrics.RequestCount {
			return a.Metrics.RequestCount > b.Metrics.RequestCount
		}
		return a.User.ID < b.User.ID
	})
}

func summarizeUserRisk(items []*UserRiskItem) UserRiskSummary {
	var out UserRiskSummary
	out.TotalUsers = int64(len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if item.Level != UserRiskLevelLow {
			out.RiskyUsers++
		}
		if item.Level == UserRiskLevelHigh {
			out.HighRiskUsers++
		}
		if item.Level == UserRiskLevelCritical {
			out.CriticalRiskUsers++
		}
		out.RequestCount += item.Metrics.RequestCount
		out.ErrorCount += item.Metrics.ErrorCount
		out.ActiveConcurrency += item.Concurrency.CurrentInUse
		out.WaitingInQueue += item.Concurrency.WaitingInQueue
		if item.IPRisk.LinkedUserCount > 0 {
			out.SharedIPUsers++
		}
		out.SharedIPGroups += item.IPRisk.SharedIPCount
	}
	out.ErrorRate = ratio(out.ErrorCount, out.RequestCount+out.ErrorCount)
	return out
}

func ratio(numerator, denominator int64) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func growthRatio(current, previous float64) float64 {
	if current <= 0 {
		return 0
	}
	if previous <= 0 {
		return current
	}
	return current / previous
}

func (s *UserRiskService) fetchAPIKeyDetails(ctx context.Context, userID int64, window UserRiskWindow) ([]UserRiskAPIKeyDetail, error) {
	const q = `
WITH usage_by_key AS (
  SELECT
    api_key_id,
    COUNT(*)::bigint AS request_count,
    COALESCE(SUM(actual_cost)::double precision, 0) AS cost,
    MAX(created_at) AS last_request_at
  FROM usage_logs
  WHERE user_id = $3 AND created_at >= $1 AND created_at < $2
  GROUP BY api_key_id
),
errors_by_key AS (
  SELECT
    api_key_id,
    COUNT(*)::bigint AS error_count
  FROM ops_error_logs
  WHERE COALESCE(user_id, deleted_key_owner_user_id) = $3
    AND api_key_id IS NOT NULL
    AND created_at >= $1 AND created_at < $2
  GROUP BY api_key_id
)
SELECT
  k.id, k.name, k.status, k.group_id, COALESCE(g.name, ''),
  COALESCE(u.request_count, 0), COALESCE(e.error_count, 0),
  COALESCE(u.cost, 0), k.last_used_at, u.last_request_at
FROM api_keys k
LEFT JOIN groups g ON g.id = k.group_id
LEFT JOIN usage_by_key u ON u.api_key_id = k.id
LEFT JOIN errors_by_key e ON e.api_key_id = k.id
WHERE k.user_id = $3 AND k.deleted_at IS NULL
ORDER BY COALESCE(u.request_count, 0) DESC, COALESCE(e.error_count, 0) DESC, k.id ASC
LIMIT 20`
	rows, err := s.db.QueryContext(ctx, q, window.StartAt, window.EndAt, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]UserRiskAPIKeyDetail, 0, 8)
	for rows.Next() {
		var item UserRiskAPIKeyDetail
		var groupID sql.NullInt64
		var lastUsedAt, lastRequestAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Status,
			&groupID,
			&item.GroupName,
			&item.RequestCount,
			&item.ErrorCount,
			&item.Cost,
			&lastUsedAt,
			&lastRequestAt,
		); err != nil {
			return nil, err
		}
		if groupID.Valid {
			item.GroupID = &groupID.Int64
		}
		if lastUsedAt.Valid {
			item.LastUsedAt = &lastUsedAt.Time
		}
		if lastRequestAt.Valid {
			item.LastRequestAt = &lastRequestAt.Time
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *UserRiskService) fetchRecentErrors(ctx context.Context, userID int64, window UserRiskWindow) ([]UserRiskRecentError, error) {
	const q = `
SELECT
  id, created_at, error_phase, error_type, COALESCE(status_code, upstream_status_code, 0),
  COALESCE(NULLIF(error_message, ''), NULLIF(upstream_error_message, ''), ''),
  COALESCE(request_id, ''),
  COALESCE(NULLIF(requested_model, ''), NULLIF(model, ''), '')
FROM ops_error_logs
WHERE COALESCE(user_id, deleted_key_owner_user_id) = $3
  AND created_at >= $1 AND created_at < $2
ORDER BY created_at DESC
LIMIT 12`
	rows, err := s.db.QueryContext(ctx, q, window.StartAt, window.EndAt, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]UserRiskRecentError, 0, 12)
	for rows.Next() {
		var item UserRiskRecentError
		if err := rows.Scan(
			&item.ID,
			&item.CreatedAt,
			&item.Phase,
			&item.Type,
			&item.StatusCode,
			&item.Message,
			&item.RequestID,
			&item.Model,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *UserRiskService) fetchTopModels(ctx context.Context, userID int64, window UserRiskWindow) ([]UserRiskTopModel, error) {
	const q = `
SELECT
  COALESCE(NULLIF(requested_model, ''), NULLIF(model, ''), '-') AS model,
  COUNT(*)::bigint AS request_count,
  COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens + cache_creation_5m_tokens + cache_creation_1h_tokens + image_output_tokens), 0)::bigint AS token_count,
  COALESCE(SUM(actual_cost)::double precision, 0) AS cost,
  MAX(created_at) AS last_used_at
FROM usage_logs
WHERE user_id = $3 AND created_at >= $1 AND created_at < $2
GROUP BY COALESCE(NULLIF(requested_model, ''), NULLIF(model, ''), '-')
ORDER BY request_count DESC, cost DESC
LIMIT 10`
	rows, err := s.db.QueryContext(ctx, q, window.StartAt, window.EndAt, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]UserRiskTopModel, 0, 10)
	for rows.Next() {
		var item UserRiskTopModel
		var lastUsedAt sql.NullTime
		if err := rows.Scan(&item.Model, &item.RequestCount, &item.TokenCount, &item.Cost, &lastUsedAt); err != nil {
			return nil, err
		}
		if lastUsedAt.Valid {
			item.LastUsedAt = &lastUsedAt.Time
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *UserRiskService) fetchTopIPs(ctx context.Context, userID int64, window UserRiskWindow) ([]UserRiskIPStat, error) {
	const q = `
SELECT
  ip,
  SUM(request_count)::bigint AS request_count,
  SUM(error_count)::bigint AS error_count,
  MAX(last_seen_at) AS last_seen_at
FROM (
  SELECT NULLIF(ip_address, '') AS ip, COUNT(*)::bigint AS request_count, 0::bigint AS error_count, MAX(created_at) AS last_seen_at
  FROM usage_logs
  WHERE user_id = $3 AND created_at >= $1 AND created_at < $2
  GROUP BY NULLIF(ip_address, '')
  UNION ALL
  SELECT client_ip::text AS ip, 0::bigint AS request_count, COUNT(*)::bigint AS error_count, MAX(created_at) AS last_seen_at
  FROM ops_error_logs
  WHERE COALESCE(user_id, deleted_key_owner_user_id) = $3
    AND created_at >= $1 AND created_at < $2
  GROUP BY client_ip::text
) x
WHERE ip IS NOT NULL AND ip <> ''
GROUP BY ip
ORDER BY (SUM(request_count) + SUM(error_count)) DESC, MAX(last_seen_at) DESC
LIMIT 10`
	rows, err := s.db.QueryContext(ctx, q, window.StartAt, window.EndAt, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]UserRiskIPStat, 0, 10)
	for rows.Next() {
		var item UserRiskIPStat
		var lastSeenAt sql.NullTime
		if err := rows.Scan(&item.IP, &item.RequestCount, &item.ErrorCount, &lastSeenAt); err != nil {
			return nil, err
		}
		if lastSeenAt.Valid {
			item.LastSeenAt = &lastSeenAt.Time
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *UserRiskService) fetchIPLinks(ctx context.Context, userID int64, window UserRiskWindow) ([]UserRiskIPLink, error) {
	const q = `
WITH user_ip_ua AS (
  SELECT NULLIF(ip_address, '') AS ip, NULLIF(user_agent, '') AS ua, 'usage' AS source, created_at
  FROM usage_logs
  WHERE user_id = $3 AND created_at >= $1 AND created_at < $2 AND NULLIF(ip_address, '') IS NOT NULL
  UNION ALL
  SELECT NULLIF(ip_address, '') AS ip, NULLIF(user_agent, '') AS ua, 'auth' AS source, created_at
  FROM user_security_events
  WHERE user_id = $3 AND created_at >= $1 AND created_at < $2 AND NULLIF(ip_address, '') IS NOT NULL
),
candidate_ips AS (
  SELECT DISTINCT ip FROM user_ip_ua WHERE ip IS NOT NULL
),
usage_by_ip_user AS (
  SELECT
    NULLIF(ip_address, '') AS ip,
    user_id,
    COUNT(*)::bigint AS request_count,
    COALESCE(SUM(actual_cost)::double precision, 0) AS cost,
    MAX(created_at) AS last_seen_at,
    BOOL_OR(NULLIF(user_agent, '') IN (SELECT ua FROM user_ip_ua WHERE ua IS NOT NULL AND ua <> '')) AS shared_ua
  FROM usage_logs
  WHERE created_at >= $1 AND created_at < $2
    AND NULLIF(ip_address, '') IN (SELECT ip FROM candidate_ips)
  GROUP BY NULLIF(ip_address, ''), user_id
),
auth_by_ip_user AS (
  SELECT
    NULLIF(ip_address, '') AS ip,
    user_id,
    COUNT(*)::bigint AS auth_event_count,
    COUNT(*) FILTER (WHERE event_type IN ('register', 'oauth_register'))::bigint AS register_event_count,
    MAX(created_at) AS last_auth_event_at,
    BOOL_OR(NULLIF(user_agent, '') IN (SELECT ua FROM user_ip_ua WHERE ua IS NOT NULL AND ua <> '')) AS shared_ua
  FROM user_security_events
  WHERE created_at >= $1 AND created_at < $2
    AND NULLIF(ip_address, '') IN (SELECT ip FROM candidate_ips)
    AND user_id IS NOT NULL
  GROUP BY NULLIF(ip_address, ''), user_id
),
combined AS (
  SELECT
    COALESCE(u.ip, a.ip) AS ip,
    COALESCE(u.user_id, a.user_id) AS user_id,
    COALESCE(u.request_count, 0)::bigint AS request_count,
    COALESCE(u.cost, 0)::double precision AS cost,
    u.last_seen_at,
    COALESCE(a.auth_event_count, 0)::bigint AS auth_event_count,
    COALESCE(a.register_event_count, 0)::bigint AS register_event_count,
    a.last_auth_event_at,
    COALESCE(u.shared_ua, false) OR COALESCE(a.shared_ua, false) AS shared_ua
  FROM usage_by_ip_user u
  FULL JOIN auth_by_ip_user a ON a.ip = u.ip AND a.user_id = u.user_id
  WHERE COALESCE(u.user_id, a.user_id) IS NOT NULL
),
ip_summary AS (
  SELECT
    c.ip,
    COUNT(DISTINCT c.user_id)::bigint AS user_count,
    COUNT(DISTINCT c.user_id) FILTER (WHERE c.user_id <> $3)::bigint AS other_user_count,
    COUNT(DISTINCT c.user_id) FILTER (WHERE usr.created_at >= $4)::bigint AS new_user_count,
    COUNT(DISTINCT c.user_id) FILTER (WHERE c.shared_ua)::bigint AS same_ua_user_count,
    SUM(c.request_count)::bigint AS request_count,
    0::bigint AS error_count,
    SUM(c.auth_event_count)::bigint AS auth_event_count,
    SUM(c.register_event_count)::bigint AS register_event_count,
    SUM(c.cost)::double precision AS cost,
    MAX(GREATEST(COALESCE(c.last_seen_at, '-infinity'::timestamptz), COALESCE(c.last_auth_event_at, '-infinity'::timestamptz))) AS last_seen_at
  FROM combined c
  LEFT JOIN users usr ON usr.id = c.user_id
  GROUP BY c.ip
)
SELECT
  ip, user_count, other_user_count, new_user_count, same_ua_user_count,
  request_count, error_count, auth_event_count, register_event_count, cost, last_seen_at
FROM ip_summary
WHERE other_user_count > 0 OR register_event_count > 0
ORDER BY other_user_count DESC, register_event_count DESC, request_count DESC, last_seen_at DESC
LIMIT 10`
	rows, err := s.db.QueryContext(ctx, q, window.StartAt, window.EndAt, userID, window.EndAt.Add(-7*24*time.Hour))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]UserRiskIPLink, 0, 10)
	for rows.Next() {
		var item UserRiskIPLink
		var lastSeenAt sql.NullTime
		if err := rows.Scan(
			&item.IP,
			&item.UserCount,
			&item.OtherUserCount,
			&item.NewUserCount,
			&item.SameUAUserCount,
			&item.RequestCount,
			&item.ErrorCount,
			&item.AuthEventCount,
			&item.RegisterEventCount,
			&item.Cost,
			&lastSeenAt,
		); err != nil {
			return nil, err
		}
		item.Source = "usage_auth"
		if lastSeenAt.Valid {
			item.LastSeenAt = &lastSeenAt.Time
		}
		item.LinkedUsers, err = s.fetchLinkedUsersForIP(ctx, item.IP, userID, window)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *UserRiskService) fetchLinkedUsersForIP(ctx context.Context, ipAddress string, userID int64, window UserRiskWindow) ([]UserRiskLinkedUser, error) {
	const q = `
WITH usage_by_user AS (
  SELECT
    user_id,
    COUNT(*)::bigint AS request_count,
    MAX(created_at) AS last_seen_at,
    BOOL_OR(NULLIF(user_agent, '') IN (
      SELECT NULLIF(user_agent, '')
      FROM usage_logs
      WHERE user_id = $4 AND created_at >= $2 AND created_at < $3 AND NULLIF(ip_address, '') = $1 AND NULLIF(user_agent, '') IS NOT NULL
    )) AS shared_ua
  FROM usage_logs
  WHERE created_at >= $2 AND created_at < $3 AND NULLIF(ip_address, '') = $1
  GROUP BY user_id
),
auth_by_user AS (
  SELECT
    user_id,
    COUNT(*)::bigint AS auth_event_count,
    COUNT(*) FILTER (WHERE event_type IN ('register', 'oauth_register'))::bigint AS register_event_count,
    MAX(created_at) AS last_auth_event_at,
    BOOL_OR(NULLIF(user_agent, '') IN (
      SELECT NULLIF(user_agent, '')
      FROM user_security_events
      WHERE user_id = $4 AND created_at >= $2 AND created_at < $3 AND NULLIF(ip_address, '') = $1 AND NULLIF(user_agent, '') IS NOT NULL
    )) AS shared_ua
  FROM user_security_events
  WHERE created_at >= $2 AND created_at < $3 AND NULLIF(ip_address, '') = $1 AND user_id IS NOT NULL
  GROUP BY user_id
),
combined AS (
  SELECT
    COALESCE(u.user_id, a.user_id) AS user_id,
    COALESCE(u.request_count, 0)::bigint AS request_count,
    u.last_seen_at,
    COALESCE(a.auth_event_count, 0)::bigint AS auth_event_count,
    COALESCE(a.register_event_count, 0)::bigint AS register_event_count,
    a.last_auth_event_at,
    COALESCE(u.shared_ua, false) OR COALESCE(a.shared_ua, false) AS shared_ua
  FROM usage_by_user u
  FULL JOIN auth_by_user a ON a.user_id = u.user_id
)
SELECT
  usr.id, usr.email, usr.username, usr.status, usr.signup_source,
  usr.balance::double precision, usr.created_at,
  c.request_count, c.auth_event_count, c.register_event_count, c.last_seen_at, c.last_auth_event_at, c.shared_ua
FROM combined c
JOIN users usr ON usr.id = c.user_id
WHERE usr.deleted_at IS NULL
ORDER BY (usr.id = $4) DESC, c.register_event_count DESC, c.auth_event_count DESC, c.request_count DESC, usr.id ASC
LIMIT 12`
	rows, err := s.db.QueryContext(ctx, q, ipAddress, window.StartAt, window.EndAt, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]UserRiskLinkedUser, 0, 12)
	for rows.Next() {
		var item UserRiskLinkedUser
		var lastSeenAt, lastAuthEventAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.Email,
			&item.Username,
			&item.Status,
			&item.SignupSource,
			&item.Balance,
			&item.CreatedAt,
			&item.RequestCount,
			&item.AuthEventCount,
			&item.RegisterEventCount,
			&lastSeenAt,
			&lastAuthEventAt,
			&item.SharedUserAgentHint,
		); err != nil {
			return nil, err
		}
		if lastSeenAt.Valid {
			item.LastSeenAt = &lastSeenAt.Time
		}
		if lastAuthEventAt.Valid {
			item.LastAuthEventAt = &lastAuthEventAt.Time
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *UserRiskService) fetchAuthEvents(ctx context.Context, userID int64, window UserRiskWindow) ([]UserRiskAuthEvent, error) {
	const q = `
SELECT id, user_id, email, event_type, provider, ip_address, success, reason, created_at
FROM user_security_events
WHERE user_id = $3 AND created_at >= $1 AND created_at < $2
ORDER BY created_at DESC
LIMIT 12`
	rows, err := s.db.QueryContext(ctx, q, window.StartAt, window.EndAt, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]UserRiskAuthEvent, 0, 12)
	for rows.Next() {
		var item UserRiskAuthEvent
		var eventUserID sql.NullInt64
		if err := rows.Scan(
			&item.ID,
			&eventUserID,
			&item.Email,
			&item.EventType,
			&item.Provider,
			&item.IPAddress,
			&item.Success,
			&item.Reason,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		if eventUserID.Valid {
			item.UserID = &eventUserID.Int64
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
