package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// UserRiskHandler 处理用户风控诊断页的只读接口。
type UserRiskHandler struct {
	service *service.UserRiskService
}

func NewUserRiskHandler(service *service.UserRiskService) *UserRiskHandler {
	return &UserRiskHandler{service: service}
}

// List 返回用户风控诊断列表。
// GET /api/v1/admin/user-risk
func (h *UserRiskHandler) List(c *gin.Context) {
	if h.service == nil {
		response.InternalError(c, "User risk service not available")
		return
	}

	page, pageSize := response.ParsePagination(c)
	params := service.UserRiskListParams{
		Page:      page,
		PageSize:  pageSize,
		Window:    strings.TrimSpace(c.DefaultQuery("window", "24h")),
		Search:    strings.TrimSpace(c.Query("search")),
		Status:    strings.TrimSpace(c.Query("status")),
		RiskLevel: strings.TrimSpace(c.Query("risk_level")),
		OnlyRisky: parseBoolQuery(c.Query("only_risky")),
	}

	result, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// Get 返回单个用户的风控诊断明细。
// GET /api/v1/admin/user-risk/:id
func (h *UserRiskHandler) Get(c *gin.Context) {
	if h.service == nil {
		response.InternalError(c, "User risk service not available")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	result, err := h.service.Get(c.Request.Context(), id, strings.TrimSpace(c.DefaultQuery("window", "24h")))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func parseBoolQuery(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
