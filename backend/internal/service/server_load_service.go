package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	stdnet "net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/redis/go-redis/v9"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	gopsnet "github.com/shirou/gopsutil/v4/net"
)

const (
	ServerLoadStatusOK       = "ok"
	ServerLoadStatusWarning  = "warning"
	ServerLoadStatusCritical = "critical"
	ServerLoadStatusUnknown  = "unknown"

	defaultDockerSocketPath = "/var/run/docker.sock"
)

type ServerLoadThresholds struct {
	CPUWarningPercent     float64 `json:"cpu_warning_percent"`
	CPUCriticalPercent    float64 `json:"cpu_critical_percent"`
	MemoryWarningPercent  float64 `json:"memory_warning_percent"`
	MemoryCriticalPercent float64 `json:"memory_critical_percent"`
	DiskWarningPercent    float64 `json:"disk_warning_percent"`
	DiskCriticalPercent   float64 `json:"disk_critical_percent"`
	GoroutinesWarning     int     `json:"goroutines_warning"`
	GoroutinesCritical    int     `json:"goroutines_critical"`
}

type ServerLoadSnapshot struct {
	Status        string                 `json:"status"`
	CollectedAt   time.Time              `json:"collected_at"`
	UptimeSeconds int64                  `json:"uptime_seconds"`
	CPU           ServerLoadCPU          `json:"cpu"`
	Memory        ServerLoadMemory       `json:"memory"`
	Disk          ServerLoadDisk         `json:"disk"`
	Docker        ServerLoadDocker       `json:"docker"`
	Runtime       ServerLoadRuntime      `json:"runtime"`
	Network       ServerLoadNetwork      `json:"network"`
	Dependencies  ServerLoadDependencies `json:"dependencies"`
	Thresholds    ServerLoadThresholds   `json:"thresholds"`
	Errors        []string               `json:"errors,omitempty"`
}

type ServerLoadCPU struct {
	UsagePercent       float64  `json:"usage_percent"`
	Cores              int      `json:"cores"`
	Load1              float64  `json:"load1"`
	Load5              float64  `json:"load5"`
	Load15             float64  `json:"load15"`
	CgroupUsagePercent *float64 `json:"cgroup_usage_percent,omitempty"`
	Source             string   `json:"source"`
}

type ServerLoadMemory struct {
	UsedBytes      uint64  `json:"used_bytes"`
	TotalBytes     uint64  `json:"total_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	UsagePercent   float64 `json:"usage_percent"`
	SwapUsedBytes  uint64  `json:"swap_used_bytes"`
	SwapTotalBytes uint64  `json:"swap_total_bytes"`
	Source         string  `json:"source"`
}

type ServerLoadDisk struct {
	Root             ServerLoadDiskUsage `json:"root"`
	Data             ServerLoadDiskUsage `json:"data"`
	ReadBytesPerSec  float64             `json:"read_bytes_per_sec"`
	WriteBytesPerSec float64             `json:"write_bytes_per_sec"`
}

type ServerLoadDiskUsage struct {
	Path              string  `json:"path"`
	UsedBytes         uint64  `json:"used_bytes"`
	TotalBytes        uint64  `json:"total_bytes"`
	UsagePercent      float64 `json:"usage_percent"`
	InodeUsagePercent float64 `json:"inode_usage_percent"`
}

type ServerLoadDocker struct {
	Available         bool    `json:"available"`
	ContainerName     string  `json:"container_name"`
	Image             string  `json:"image"`
	Status            string  `json:"status"`
	Health            string  `json:"health"`
	UptimeSeconds     int64   `json:"uptime_seconds"`
	ContainersRunning int     `json:"containers_running"`
	ContainersTotal   int     `json:"containers_total"`
	CPUUsagePercent   float64 `json:"cpu_usage_percent"`
	MemoryUsageBytes  uint64  `json:"memory_usage_bytes"`
	MemoryLimitBytes  uint64  `json:"memory_limit_bytes"`
	NetworkRXBytes    uint64  `json:"network_rx_bytes"`
	NetworkTXBytes    uint64  `json:"network_tx_bytes"`
	BlockReadBytes    uint64  `json:"block_read_bytes"`
	BlockWriteBytes   uint64  `json:"block_write_bytes"`
	UnavailableReason string  `json:"unavailable_reason"`
}

type ServerLoadRuntime struct {
	Goroutines           int        `json:"goroutines"`
	HeapAllocBytes       uint64     `json:"heap_alloc_bytes"`
	HeapSysBytes         uint64     `json:"heap_sys_bytes"`
	GCCount              uint32     `json:"gc_count"`
	LastGCAt             *time.Time `json:"last_gc_at,omitempty"`
	ProcessUptimeSeconds int64      `json:"process_uptime_seconds"`
}

type ServerLoadNetwork struct {
	PrimaryInterface string  `json:"primary_interface"`
	RXBytes          uint64  `json:"rx_bytes"`
	TXBytes          uint64  `json:"tx_bytes"`
	RXBytesPerSec    float64 `json:"rx_bytes_per_sec"`
	TXBytesPerSec    float64 `json:"tx_bytes_per_sec"`
	TCPEstablished   int     `json:"tcp_established"`
	TCPListen        int     `json:"tcp_listen"`
	TCPTimeWait      int     `json:"tcp_time_wait"`
}

type ServerLoadDependencies struct {
	BackendOK bool `json:"backend_ok"`
	DBOK      bool `json:"db_ok"`
	RedisOK   bool `json:"redis_ok"`
}

type ServerLoadCollector interface {
	Collect(ctx context.Context) (*ServerLoadSnapshot, error)
}

type ServerLoadService struct {
	collector  ServerLoadCollector
	thresholds ServerLoadThresholds
}

func DefaultServerLoadThresholds() ServerLoadThresholds {
	return ServerLoadThresholds{
		CPUWarningPercent:     80,
		CPUCriticalPercent:    90,
		MemoryWarningPercent:  80,
		MemoryCriticalPercent: 90,
		DiskWarningPercent:    85,
		DiskCriticalPercent:   95,
		GoroutinesWarning:     8000,
		GoroutinesCritical:    15000,
	}
}

func NewServerLoadService(db *sql.DB, redisClient *redis.Client, cfg *config.Config) *ServerLoadService {
	return &ServerLoadService{
		collector:  newRuntimeServerLoadCollector(db, redisClient, cfg),
		thresholds: DefaultServerLoadThresholds(),
	}
}

func ProvideServerLoadService(db *sql.DB, redisClient *redis.Client, cfg *config.Config) *ServerLoadService {
	return NewServerLoadService(db, redisClient, cfg)
}

func NewServerLoadServiceWithCollector(collector ServerLoadCollector) *ServerLoadService {
	return &ServerLoadService{
		collector:  collector,
		thresholds: DefaultServerLoadThresholds(),
	}
}

func NewServerLoadServiceForTest(snapshot *ServerLoadSnapshot) *ServerLoadService {
	return NewServerLoadServiceWithCollector(staticServerLoadCollector{snapshot: snapshot})
}

func (s *ServerLoadService) Snapshot(ctx context.Context) (*ServerLoadSnapshot, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	thresholds := DefaultServerLoadThresholds()
	if s != nil && s.thresholds != (ServerLoadThresholds{}) {
		thresholds = s.thresholds
	}

	if s == nil || s.collector == nil {
		return &ServerLoadSnapshot{
			Status:      ServerLoadStatusUnknown,
			CollectedAt: time.Now().UTC(),
			Thresholds:  thresholds,
			Errors:      []string{"server load collector unavailable"},
		}, nil
	}

	snapshot, err := s.collector.Collect(ctx)
	if snapshot == nil {
		out := &ServerLoadSnapshot{
			Status:      ServerLoadStatusUnknown,
			CollectedAt: time.Now().UTC(),
			Thresholds:  thresholds,
		}
		if err != nil {
			out.Errors = append(out.Errors, err.Error())
		}
		return out, nil
	}

	if snapshot.CollectedAt.IsZero() {
		snapshot.CollectedAt = time.Now().UTC()
	}
	if err != nil {
		snapshot.Errors = appendServerLoadError(snapshot.Errors, err)
	}
	snapshot.Thresholds = thresholds
	snapshot.Status = classifyServerLoadStatus(snapshot, thresholds)
	return snapshot, nil
}

type staticServerLoadCollector struct {
	snapshot *ServerLoadSnapshot
}

func (c staticServerLoadCollector) Collect(context.Context) (*ServerLoadSnapshot, error) {
	return c.snapshot, nil
}

func classifyServerLoadStatus(snapshot *ServerLoadSnapshot, thresholds ServerLoadThresholds) string {
	if snapshot == nil {
		return ServerLoadStatusUnknown
	}

	if snapshot.CPU.UsagePercent > thresholds.CPUCriticalPercent ||
		snapshot.Memory.UsagePercent > thresholds.MemoryCriticalPercent ||
		snapshot.Disk.Root.UsagePercent > thresholds.DiskCriticalPercent ||
		snapshot.Disk.Data.UsagePercent > thresholds.DiskCriticalPercent ||
		snapshot.Runtime.Goroutines > thresholds.GoroutinesCritical {
		return ServerLoadStatusCritical
	}

	if snapshot.CPU.UsagePercent > thresholds.CPUWarningPercent ||
		snapshot.Memory.UsagePercent > thresholds.MemoryWarningPercent ||
		snapshot.Disk.Root.UsagePercent > thresholds.DiskWarningPercent ||
		snapshot.Disk.Data.UsagePercent > thresholds.DiskWarningPercent ||
		snapshot.Runtime.Goroutines > thresholds.GoroutinesWarning ||
		!snapshot.Dependencies.BackendOK ||
		!snapshot.Dependencies.DBOK ||
		!snapshot.Dependencies.RedisOK {
		return ServerLoadStatusWarning
	}

	return ServerLoadStatusOK
}

type runtimeServerLoadCollector struct {
	db          *sql.DB
	redis       *redis.Client
	cfg         *config.Config
	startedAt   time.Time
	docker      *dockerSocketCollector
	rateSampler *serverLoadRateSampler
}

func newRuntimeServerLoadCollector(db *sql.DB, redisClient *redis.Client, cfg *config.Config) *runtimeServerLoadCollector {
	socketPath := defaultDockerSocketPath
	if raw := strings.TrimSpace(os.Getenv("DOCKER_HOST")); strings.HasPrefix(raw, "unix://") {
		socketPath = strings.TrimPrefix(raw, "unix://")
	}
	return &runtimeServerLoadCollector{
		db:          db,
		redis:       redisClient,
		cfg:         cfg,
		startedAt:   time.Now().UTC(),
		docker:      newDockerSocketCollector(socketPath),
		rateSampler: &serverLoadRateSampler{},
	}
}

func (c *runtimeServerLoadCollector) Collect(parentCtx context.Context) (*ServerLoadSnapshot, error) {
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(parentCtx, 4*time.Second)
	defer cancel()

	now := time.Now().UTC()
	snapshot := &ServerLoadSnapshot{
		CollectedAt:   now,
		UptimeSeconds: int64(now.Sub(c.startedAt).Seconds()),
		Dependencies:  ServerLoadDependencies{BackendOK: true},
	}
	var errs []error

	if err := c.collectCPU(ctx, now, snapshot); err != nil {
		errs = append(errs, fmt.Errorf("cpu: %w", err))
	}
	if err := c.collectMemory(ctx, snapshot); err != nil {
		errs = append(errs, fmt.Errorf("memory: %w", err))
	}
	if err := c.collectDisk(ctx, now, snapshot); err != nil {
		errs = append(errs, fmt.Errorf("disk: %w", err))
	}
	if err := c.collectRuntime(now, snapshot); err != nil {
		errs = append(errs, fmt.Errorf("runtime: %w", err))
	}
	if err := c.collectNetwork(ctx, now, snapshot); err != nil {
		errs = append(errs, fmt.Errorf("network: %w", err))
	}
	if err := c.collectDependencies(ctx, snapshot); err != nil {
		errs = append(errs, fmt.Errorf("dependencies: %w", err))
	}
	if err := c.collectDocker(ctx, now, snapshot); err != nil {
		errs = append(errs, fmt.Errorf("docker: %w", err))
	}

	return snapshot, errors.Join(errs...)
}

func (c *runtimeServerLoadCollector) collectCPU(ctx context.Context, now time.Time, snapshot *ServerLoadSnapshot) error {
	var errs []error
	snapshot.CPU.Source = "host"

	if cores, err := cpu.CountsWithContext(ctx, true); err == nil {
		snapshot.CPU.Cores = cores
	} else {
		errs = append(errs, err)
	}

	if avg, err := load.AvgWithContext(ctx); err == nil && avg != nil {
		snapshot.CPU.Load1 = roundTo2DP(avg.Load1)
		snapshot.CPU.Load5 = roundTo2DP(avg.Load5)
		snapshot.CPU.Load15 = roundTo2DP(avg.Load15)
	} else if err != nil {
		errs = append(errs, err)
	}

	cgroupCPUCollected := false
	if c.rateSampler != nil {
		if pct := c.rateSampler.tryCgroupCPUPercent(now); pct != nil {
			snapshot.CPU.CgroupUsagePercent = pct
			snapshot.CPU.UsagePercent = *pct
			snapshot.CPU.Source = "cgroup"
			cgroupCPUCollected = true
		}
	}
	if !cgroupCPUCollected {
		if percents, err := cpu.PercentWithContext(ctx, 0, false); err == nil && len(percents) > 0 {
			snapshot.CPU.UsagePercent = roundTo1DP(percents[0])
		} else if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (c *runtimeServerLoadCollector) collectMemory(ctx context.Context, snapshot *ServerLoadSnapshot) error {
	var errs []error
	snapshot.Memory.Source = "host"

	memoryFromCgroup := false
	if used, total, ok := readCgroupMemoryBytes(); ok && total > 0 {
		snapshot.Memory.UsedBytes = used
		snapshot.Memory.TotalBytes = total
		if total >= used {
			snapshot.Memory.AvailableBytes = total - used
		}
		snapshot.Memory.UsagePercent = roundTo1DP(float64(used) / float64(total) * 100)
		snapshot.Memory.Source = "cgroup"
		memoryFromCgroup = true
	}

	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		errs = append(errs, err)
	} else if vm != nil && !memoryFromCgroup {
		snapshot.Memory.UsedBytes = vm.Used
		snapshot.Memory.TotalBytes = vm.Total
		snapshot.Memory.AvailableBytes = vm.Available
		snapshot.Memory.UsagePercent = roundTo1DP(vm.UsedPercent)
	}

	if swap, err := mem.SwapMemoryWithContext(ctx); err == nil && swap != nil {
		snapshot.Memory.SwapUsedBytes = swap.Used
		snapshot.Memory.SwapTotalBytes = swap.Total
	} else if err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (c *runtimeServerLoadCollector) collectDisk(ctx context.Context, now time.Time, snapshot *ServerLoadSnapshot) error {
	var errs []error
	if root, err := diskUsage(ctx, "/"); err == nil {
		snapshot.Disk.Root = root
	} else {
		errs = append(errs, err)
	}

	dataPath := resolveServerLoadDataPath(c.cfg)
	if data, err := diskUsage(ctx, dataPath); err == nil {
		snapshot.Disk.Data = data
	} else {
		errs = append(errs, err)
		snapshot.Disk.Data.Path = dataPath
	}

	if counters, err := disk.IOCountersWithContext(ctx); err == nil {
		var readBytes uint64
		var writeBytes uint64
		for _, counter := range counters {
			readBytes += counter.ReadBytes
			writeBytes += counter.WriteBytes
		}
		readRate, writeRate := c.rateSampler.diskRates(now, readBytes, writeBytes)
		snapshot.Disk.ReadBytesPerSec = readRate
		snapshot.Disk.WriteBytesPerSec = writeRate
	} else {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (c *runtimeServerLoadCollector) collectRuntime(now time.Time, snapshot *ServerLoadSnapshot) error {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	snapshot.Runtime.Goroutines = runtime.NumGoroutine()
	snapshot.Runtime.HeapAllocBytes = stats.HeapAlloc
	snapshot.Runtime.HeapSysBytes = stats.HeapSys
	snapshot.Runtime.GCCount = stats.NumGC
	if stats.LastGC > 0 {
		lastGC := time.Unix(0, int64(stats.LastGC)).UTC()
		snapshot.Runtime.LastGCAt = &lastGC
	}
	snapshot.Runtime.ProcessUptimeSeconds = int64(now.Sub(c.startedAt).Seconds())
	return nil
}

func (c *runtimeServerLoadCollector) collectNetwork(ctx context.Context, now time.Time, snapshot *ServerLoadSnapshot) error {
	var errs []error

	counters, err := gopsnet.IOCountersWithContext(ctx, true)
	if err != nil {
		errs = append(errs, err)
	} else {
		counter, ok := choosePrimaryNetworkCounter(counters)
		if ok {
			snapshot.Network.PrimaryInterface = counter.Name
			snapshot.Network.RXBytes = counter.BytesRecv
			snapshot.Network.TXBytes = counter.BytesSent
			rxRate, txRate := c.rateSampler.networkRates(now, counter.BytesRecv, counter.BytesSent)
			snapshot.Network.RXBytesPerSec = rxRate
			snapshot.Network.TXBytesPerSec = txRate
		}
	}

	connections, err := gopsnet.ConnectionsWithContext(ctx, "tcp")
	if err != nil {
		errs = append(errs, err)
	} else {
		for _, conn := range connections {
			switch strings.ToUpper(conn.Status) {
			case "ESTABLISHED":
				snapshot.Network.TCPEstablished++
			case "LISTEN":
				snapshot.Network.TCPListen++
			case "TIME_WAIT":
				snapshot.Network.TCPTimeWait++
			}
		}
	}

	return errors.Join(errs...)
}

func (c *runtimeServerLoadCollector) collectDependencies(ctx context.Context, snapshot *ServerLoadSnapshot) error {
	depCtx, cancel := context.WithTimeout(ctx, 1200*time.Millisecond)
	defer cancel()

	snapshot.Dependencies.BackendOK = true
	snapshot.Dependencies.DBOK = checkServerLoadDB(depCtx, c.db)
	snapshot.Dependencies.RedisOK = checkServerLoadRedis(depCtx, c.redis)
	return nil
}

func (c *runtimeServerLoadCollector) collectDocker(ctx context.Context, now time.Time, snapshot *ServerLoadSnapshot) error {
	if c.docker == nil {
		snapshot.Docker.Available = false
		snapshot.Docker.UnavailableReason = "docker collector unavailable"
		return nil
	}
	docker, err := c.docker.Collect(ctx, now)
	snapshot.Docker = docker
	return err
}

type serverLoadRateSampler struct {
	mu sync.Mutex

	lastCgroupCPUUsageNanos uint64
	lastCgroupCPUSampleAt   time.Time

	lastDiskReadBytes  uint64
	lastDiskWriteBytes uint64
	lastDiskSampleAt   time.Time

	lastNetworkRXBytes  uint64
	lastNetworkTXBytes  uint64
	lastNetworkSampleAt time.Time
}

func (s *serverLoadRateSampler) tryCgroupCPUPercent(now time.Time) *float64 {
	if s == nil {
		return nil
	}
	usageNanos, ok := readCgroupCPUUsageNanos()
	if !ok {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastCgroupCPUSampleAt.IsZero() {
		s.lastCgroupCPUUsageNanos = usageNanos
		s.lastCgroupCPUSampleAt = now
		return nil
	}

	elapsed := now.Sub(s.lastCgroupCPUSampleAt)
	prev := s.lastCgroupCPUUsageNanos
	s.lastCgroupCPUUsageNanos = usageNanos
	s.lastCgroupCPUSampleAt = now

	if elapsed <= 0 || usageNanos < prev {
		return nil
	}

	cores := readCgroupCPULimitCores()
	if cores <= 0 {
		return nil
	}
	pct := (float64(usageNanos-prev) / 1e9 / (elapsed.Seconds() * cores)) * 100
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	v := roundTo1DP(pct)
	return &v
}

func (s *serverLoadRateSampler) diskRates(now time.Time, readBytes, writeBytes uint64) (float64, float64) {
	if s == nil {
		return 0, 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastDiskSampleAt.IsZero() {
		s.lastDiskReadBytes = readBytes
		s.lastDiskWriteBytes = writeBytes
		s.lastDiskSampleAt = now
		return 0, 0
	}
	elapsed := now.Sub(s.lastDiskSampleAt).Seconds()
	prevRead := s.lastDiskReadBytes
	prevWrite := s.lastDiskWriteBytes
	s.lastDiskReadBytes = readBytes
	s.lastDiskWriteBytes = writeBytes
	s.lastDiskSampleAt = now
	if elapsed <= 0 || readBytes < prevRead || writeBytes < prevWrite {
		return 0, 0
	}
	return roundTo1DP(float64(readBytes-prevRead) / elapsed), roundTo1DP(float64(writeBytes-prevWrite) / elapsed)
}

func (s *serverLoadRateSampler) networkRates(now time.Time, rxBytes, txBytes uint64) (float64, float64) {
	if s == nil {
		return 0, 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastNetworkSampleAt.IsZero() {
		s.lastNetworkRXBytes = rxBytes
		s.lastNetworkTXBytes = txBytes
		s.lastNetworkSampleAt = now
		return 0, 0
	}
	elapsed := now.Sub(s.lastNetworkSampleAt).Seconds()
	prevRX := s.lastNetworkRXBytes
	prevTX := s.lastNetworkTXBytes
	s.lastNetworkRXBytes = rxBytes
	s.lastNetworkTXBytes = txBytes
	s.lastNetworkSampleAt = now
	if elapsed <= 0 || rxBytes < prevRX || txBytes < prevTX {
		return 0, 0
	}
	return roundTo1DP(float64(rxBytes-prevRX) / elapsed), roundTo1DP(float64(txBytes-prevTX) / elapsed)
}

func diskUsage(ctx context.Context, path string) (ServerLoadDiskUsage, error) {
	if strings.TrimSpace(path) == "" {
		path = "/"
	}
	usage, err := disk.UsageWithContext(ctx, path)
	if err != nil {
		return ServerLoadDiskUsage{Path: path}, err
	}
	out := ServerLoadDiskUsage{
		Path:              path,
		UsedBytes:         usage.Used,
		TotalBytes:        usage.Total,
		UsagePercent:      roundTo1DP(usage.UsedPercent),
		InodeUsagePercent: roundTo1DP(usage.InodesUsedPercent),
	}
	return out, nil
}

func resolveServerLoadDataPath(cfg *config.Config) string {
	if dir := strings.TrimSpace(os.Getenv("DATA_DIR")); dir != "" {
		return dir
	}
	if info, err := os.Stat("/app/data"); err == nil && info.IsDir() {
		return "/app/data"
	}
	if cfg != nil && strings.TrimSpace(cfg.Pricing.DataDir) != "" {
		return cfg.Pricing.DataDir
	}
	return "."
}

func choosePrimaryNetworkCounter(counters []gopsnet.IOCountersStat) (gopsnet.IOCountersStat, bool) {
	var best gopsnet.IOCountersStat
	var bestTotal uint64
	for _, counter := range counters {
		name := strings.ToLower(counter.Name)
		if name == "" || name == "lo" || strings.HasPrefix(name, "lo") {
			continue
		}
		total := counter.BytesRecv + counter.BytesSent
		if total >= bestTotal {
			best = counter
			bestTotal = total
		}
	}
	if best.Name != "" {
		return best, true
	}
	if len(counters) > 0 {
		return counters[0], true
	}
	return gopsnet.IOCountersStat{}, false
}

func checkServerLoadDB(ctx context.Context, db *sql.DB) bool {
	if db == nil {
		return false
	}
	var one int
	if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&one); err != nil {
		return false
	}
	return one == 1
}

func checkServerLoadRedis(ctx context.Context, redisClient *redis.Client) bool {
	if redisClient == nil {
		return false
	}
	return redisClient.Ping(ctx).Err() == nil
}

func appendServerLoadError(existing []string, err error) []string {
	if err == nil {
		return existing
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return existing
	}
	return append(existing, msg)
}

func roundTo2DP(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

type dockerSocketCollector struct {
	socketPath string
	client     *http.Client
}

func newDockerSocketCollector(socketPath string) *dockerSocketCollector {
	if strings.TrimSpace(socketPath) == "" {
		socketPath = defaultDockerSocketPath
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (stdnet.Conn, error) {
			return (&stdnet.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}
	return &dockerSocketCollector{
		socketPath: socketPath,
		client: &http.Client{
			Transport: transport,
			Timeout:   2 * time.Second,
		},
	}
}

func (c *dockerSocketCollector) Collect(ctx context.Context, now time.Time) (ServerLoadDocker, error) {
	out := ServerLoadDocker{}
	if c == nil || c.client == nil {
		out.Available = false
		out.UnavailableReason = "docker collector unavailable"
		return out, nil
	}
	if _, err := os.Stat(c.socketPath); err != nil {
		out.Available = false
		out.UnavailableReason = "docker socket unavailable"
		return out, nil
	}

	var containers []dockerContainerListItem
	if err := c.getJSON(ctx, "/containers/json?all=1", &containers); err != nil {
		out.Available = false
		out.UnavailableReason = "docker api unavailable"
		return out, err
	}

	out.Available = true
	out.ContainersTotal = len(containers)
	for _, item := range containers {
		if strings.EqualFold(item.State, "running") {
			out.ContainersRunning++
		}
	}

	containerID := detectCurrentContainerID(containers)
	if containerID == "" {
		out.UnavailableReason = "current container not found"
		return out, nil
	}

	var inspect dockerContainerInspect
	if err := c.getJSON(ctx, "/containers/"+url.PathEscape(containerID)+"/json", &inspect); err != nil {
		out.UnavailableReason = "current container inspect unavailable"
		return out, err
	}
	out.ContainerName = strings.TrimPrefix(inspect.Name, "/")
	if out.ContainerName == "" && len(inspect.Config.Image) > 0 {
		out.ContainerName = containerID
	}
	out.Image = inspect.Config.Image
	out.Status = inspect.State.Status
	if inspect.State.Health.Status != "" {
		out.Health = inspect.State.Health.Status
	}
	if startedAt, err := time.Parse(time.RFC3339Nano, inspect.State.StartedAt); err == nil && !startedAt.IsZero() {
		out.UptimeSeconds = int64(now.Sub(startedAt).Seconds())
	}

	var stats dockerContainerStats
	if err := c.getJSON(ctx, "/containers/"+url.PathEscape(containerID)+"/stats?stream=false", &stats); err != nil {
		out.UnavailableReason = "current container stats unavailable"
		return out, err
	}
	out.CPUUsagePercent = computeDockerCPUPercent(stats)
	out.MemoryUsageBytes = stats.MemoryStats.Usage
	out.MemoryLimitBytes = stats.MemoryStats.Limit
	for _, network := range stats.Networks {
		out.NetworkRXBytes += network.RXBytes
		out.NetworkTXBytes += network.TXBytes
	}
	for _, item := range stats.BlkioStats.IoServiceBytesRecursive {
		switch strings.ToLower(item.Op) {
		case "read":
			out.BlockReadBytes += item.Value
		case "write":
			out.BlockWriteBytes += item.Value
		}
	}

	return out, nil
}

func (c *dockerSocketCollector) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://docker"+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("docker api returned %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

type dockerContainerListItem struct {
	ID     string   `json:"Id"`
	Image  string   `json:"Image"`
	State  string   `json:"State"`
	Status string   `json:"Status"`
	Names  []string `json:"Names"`
}

type dockerContainerInspect struct {
	Name   string `json:"Name"`
	Config struct {
		Image string `json:"Image"`
	} `json:"Config"`
	State struct {
		Status    string `json:"Status"`
		StartedAt string `json:"StartedAt"`
		Health    struct {
			Status string `json:"Status"`
		} `json:"Health"`
	} `json:"State"`
}

type dockerContainerStats struct {
	CPUStats struct {
		CPUUsage struct {
			TotalUsage  uint64   `json:"total_usage"`
			PercpuUsage []uint64 `json:"percpu_usage"`
		} `json:"cpu_usage"`
		SystemUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs  uint32 `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64 `json:"usage"`
		Limit uint64 `json:"limit"`
	} `json:"memory_stats"`
	Networks map[string]struct {
		RXBytes uint64 `json:"rx_bytes"`
		TXBytes uint64 `json:"tx_bytes"`
	} `json:"networks"`
	BlkioStats struct {
		IoServiceBytesRecursive []struct {
			Op    string `json:"op"`
			Value uint64 `json:"value"`
		} `json:"io_service_bytes_recursive"`
	} `json:"blkio_stats"`
}

func detectCurrentContainerID(containers []dockerContainerListItem) string {
	hostname, _ := os.Hostname()
	hostname = strings.TrimSpace(hostname)
	if hostname != "" {
		for _, item := range containers {
			if strings.HasPrefix(item.ID, hostname) || strings.HasPrefix(hostname, item.ID) {
				return item.ID
			}
			for _, name := range item.Names {
				if strings.EqualFold(strings.Trim(name, "/"), hostname) {
					return item.ID
				}
			}
		}
	}

	if id := readContainerIDFromCgroup(); id != "" {
		for _, item := range containers {
			if strings.HasPrefix(item.ID, id) || strings.HasPrefix(id, item.ID) {
				return item.ID
			}
		}
	}

	return ""
}

func readContainerIDFromCgroup() string {
	raw, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return ""
	}
	lines := strings.Split(string(raw), "\n")
	for _, line := range lines {
		fields := strings.Split(line, "/")
		for i := len(fields) - 1; i >= 0; i-- {
			part := strings.TrimSpace(fields[i])
			part = strings.TrimSuffix(part, ".scope")
			part = strings.TrimPrefix(part, "docker-")
			part = strings.TrimPrefix(part, "cri-containerd-")
			if len(part) >= 12 && isHexish(part) {
				return part
			}
		}
	}
	return ""
}

func isHexish(s string) bool {
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

func computeDockerCPUPercent(stats dockerContainerStats) float64 {
	if stats.CPUStats.CPUUsage.TotalUsage < stats.PreCPUStats.CPUUsage.TotalUsage ||
		stats.CPUStats.SystemUsage < stats.PreCPUStats.SystemUsage {
		return 0
	}
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	if cpuDelta <= 0 || systemDelta <= 0 {
		return 0
	}
	onlineCPUs := float64(stats.CPUStats.OnlineCPUs)
	if onlineCPUs <= 0 {
		onlineCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	}
	if onlineCPUs <= 0 {
		onlineCPUs = 1
	}
	return roundTo1DP((cpuDelta / systemDelta) * onlineCPUs * 100)
}
