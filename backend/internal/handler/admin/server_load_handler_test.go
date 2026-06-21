package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newServerLoadTestRouter(handler *OpsHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/server-load", handler.GetServerLoad)
	return r
}

func TestServerLoadHandlerUnavailable(t *testing.T) {
	h := NewOpsHandler(nil)
	r := newServerLoadTestRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/server-load", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestServerLoadHandlerSuccess(t *testing.T) {
	snapshot := &service.ServerLoadSnapshot{
		Status:      service.ServerLoadStatusOK,
		CollectedAt: time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC),
		CPU: service.ServerLoadCPU{
			UsagePercent: 12.5,
			Cores:        4,
			Source:       "host",
		},
		Memory: service.ServerLoadMemory{
			UsedBytes:    1024,
			TotalBytes:   4096,
			UsagePercent: 25,
			Source:       "host",
		},
		Disk: service.ServerLoadDisk{
			Root: ServerLoadDiskUsageForTest("/", 20),
		},
		Docker: service.ServerLoadDocker{
			Available:         false,
			UnavailableReason: "docker socket unavailable",
		},
		Runtime: service.ServerLoadRuntime{
			Goroutines: 64,
		},
		Network: service.ServerLoadNetwork{
			PrimaryInterface: "eth0",
		},
		Dependencies: service.ServerLoadDependencies{
			BackendOK: true,
			DBOK:      true,
			RedisOK:   true,
		},
		Thresholds: service.DefaultServerLoadThresholds(),
	}

	svc := service.NewServerLoadServiceForTest(snapshot)
	h := NewOpsHandler(nil)
	h.SetServerLoadService(svc)
	r := newServerLoadTestRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/server-load", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Status      string `json:"status"`
			CollectedAt string `json:"collected_at"`
			CPU         struct {
				UsagePercent float64 `json:"usage_percent"`
				Cores        int     `json:"cores"`
			} `json:"cpu"`
			Docker struct {
				Available         bool   `json:"available"`
				UnavailableReason string `json:"unavailable_reason"`
			} `json:"docker"`
			Dependencies struct {
				BackendOK bool `json:"backend_ok"`
				DBOK      bool `json:"db_ok"`
				RedisOK   bool `json:"redis_ok"`
			} `json:"dependencies"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, service.ServerLoadStatusOK, resp.Data.Status)
	require.Equal(t, "2026-06-21T10:00:00Z", resp.Data.CollectedAt)
	require.Equal(t, 12.5, resp.Data.CPU.UsagePercent)
	require.Equal(t, 4, resp.Data.CPU.Cores)
	require.False(t, resp.Data.Docker.Available)
	require.Equal(t, "docker socket unavailable", resp.Data.Docker.UnavailableReason)
	require.True(t, resp.Data.Dependencies.BackendOK)
	require.True(t, resp.Data.Dependencies.DBOK)
	require.True(t, resp.Data.Dependencies.RedisOK)
}

func ServerLoadDiskUsageForTest(path string, usage float64) service.ServerLoadDiskUsage {
	return service.ServerLoadDiskUsage{
		Path:         path,
		UsedBytes:    100,
		TotalBytes:   500,
		UsagePercent: usage,
	}
}
