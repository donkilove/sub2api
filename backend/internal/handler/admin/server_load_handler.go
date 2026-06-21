package admin

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// GetServerLoad returns a realtime server load snapshot.
// GET /api/v1/admin/server-load
func (h *OpsHandler) GetServerLoad(c *gin.Context) {
	if h == nil || h.serverLoadService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Server load service not available")
		return
	}

	snapshot, err := h.serverLoadService.Snapshot(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, snapshot)
}
