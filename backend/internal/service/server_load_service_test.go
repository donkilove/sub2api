package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeServerLoadCollector struct {
	snapshot *ServerLoadSnapshot
	err      error
}

func (f fakeServerLoadCollector) Collect(context.Context) (*ServerLoadSnapshot, error) {
	return f.snapshot, f.err
}

func TestServerLoadServiceSnapshotClassifiesStatus(t *testing.T) {
	now := time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		snapshot *ServerLoadSnapshot
		want     string
	}{
		{
			name: "healthy snapshot is ok",
			snapshot: &ServerLoadSnapshot{
				CPU:          ServerLoadCPU{UsagePercent: 35},
				Memory:       ServerLoadMemory{UsagePercent: 45},
				Disk:         ServerLoadDisk{Root: ServerLoadDiskUsage{UsagePercent: 50}, Data: ServerLoadDiskUsage{UsagePercent: 40}},
				Runtime:      ServerLoadRuntime{Goroutines: 128},
				Dependencies: ServerLoadDependencies{BackendOK: true, DBOK: true, RedisOK: true},
			},
			want: ServerLoadStatusOK,
		},
		{
			name: "dependency failure is warning",
			snapshot: &ServerLoadSnapshot{
				CPU:          ServerLoadCPU{UsagePercent: 35},
				Memory:       ServerLoadMemory{UsagePercent: 45},
				Disk:         ServerLoadDisk{Root: ServerLoadDiskUsage{UsagePercent: 50}, Data: ServerLoadDiskUsage{UsagePercent: 40}},
				Runtime:      ServerLoadRuntime{Goroutines: 128},
				Dependencies: ServerLoadDependencies{BackendOK: true, DBOK: false, RedisOK: true},
			},
			want: ServerLoadStatusWarning,
		},
		{
			name: "memory warning threshold returns warning",
			snapshot: &ServerLoadSnapshot{
				CPU:          ServerLoadCPU{UsagePercent: 35},
				Memory:       ServerLoadMemory{UsagePercent: 81},
				Disk:         ServerLoadDisk{Root: ServerLoadDiskUsage{UsagePercent: 50}, Data: ServerLoadDiskUsage{UsagePercent: 40}},
				Runtime:      ServerLoadRuntime{Goroutines: 128},
				Dependencies: ServerLoadDependencies{BackendOK: true, DBOK: true, RedisOK: true},
			},
			want: ServerLoadStatusWarning,
		},
		{
			name: "disk critical threshold returns critical",
			snapshot: &ServerLoadSnapshot{
				CPU:          ServerLoadCPU{UsagePercent: 35},
				Memory:       ServerLoadMemory{UsagePercent: 45},
				Disk:         ServerLoadDisk{Root: ServerLoadDiskUsage{UsagePercent: 96}, Data: ServerLoadDiskUsage{UsagePercent: 40}},
				Runtime:      ServerLoadRuntime{Goroutines: 128},
				Dependencies: ServerLoadDependencies{BackendOK: true, DBOK: true, RedisOK: true},
			},
			want: ServerLoadStatusCritical,
		},
		{
			name: "goroutine critical threshold returns critical",
			snapshot: &ServerLoadSnapshot{
				CPU:          ServerLoadCPU{UsagePercent: 35},
				Memory:       ServerLoadMemory{UsagePercent: 45},
				Disk:         ServerLoadDisk{Root: ServerLoadDiskUsage{UsagePercent: 50}, Data: ServerLoadDiskUsage{UsagePercent: 40}},
				Runtime:      ServerLoadRuntime{Goroutines: 15001},
				Dependencies: ServerLoadDependencies{BackendOK: true, DBOK: true, RedisOK: true},
			},
			want: ServerLoadStatusCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.snapshot.CollectedAt = now
			svc := &ServerLoadService{
				collector:  fakeServerLoadCollector{snapshot: tt.snapshot},
				thresholds: DefaultServerLoadThresholds(),
			}

			got, err := svc.Snapshot(context.Background())
			require.NoError(t, err)
			require.Equal(t, tt.want, got.Status)
			require.Equal(t, DefaultServerLoadThresholds(), got.Thresholds)
			require.Equal(t, now, got.CollectedAt)
		})
	}
}

func TestServerLoadServiceSnapshotKeepsPartialDataOnCollectorError(t *testing.T) {
	svc := &ServerLoadService{
		collector: fakeServerLoadCollector{
			snapshot: &ServerLoadSnapshot{
				CollectedAt:  time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC),
				CPU:          ServerLoadCPU{UsagePercent: 20},
				Memory:       ServerLoadMemory{UsagePercent: 30},
				Disk:         ServerLoadDisk{Root: ServerLoadDiskUsage{UsagePercent: 40}},
				Runtime:      ServerLoadRuntime{Goroutines: 10},
				Dependencies: ServerLoadDependencies{BackendOK: true, DBOK: true, RedisOK: true},
			},
			err: errors.New("docker socket unavailable"),
		},
		thresholds: DefaultServerLoadThresholds(),
	}

	got, err := svc.Snapshot(context.Background())
	require.NoError(t, err)
	require.Equal(t, ServerLoadStatusOK, got.Status)
	require.Contains(t, got.Errors, "docker socket unavailable")
}

func TestServerLoadServiceSnapshotReturnsUnknownWhenCollectorHasNoData(t *testing.T) {
	svc := &ServerLoadService{
		collector:  fakeServerLoadCollector{err: errors.New("collector failed")},
		thresholds: DefaultServerLoadThresholds(),
	}

	got, err := svc.Snapshot(context.Background())
	require.NoError(t, err)
	require.Equal(t, ServerLoadStatusUnknown, got.Status)
	require.Contains(t, got.Errors, "collector failed")
}
