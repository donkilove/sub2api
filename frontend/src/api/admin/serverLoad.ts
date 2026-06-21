import { apiClient } from '../client'

export type ServerLoadStatus = 'ok' | 'warning' | 'critical' | 'unknown'

export interface ServerLoadDiskUsage {
  path: string
  used_bytes: number
  total_bytes: number
  usage_percent: number
  inode_usage_percent: number
}

export interface ServerLoadSnapshot {
  status: ServerLoadStatus
  collected_at: string
  uptime_seconds: number
  cpu: {
    usage_percent: number
    cores: number
    load1: number
    load5: number
    load15: number
    cgroup_usage_percent?: number
    source: string
  }
  memory: {
    used_bytes: number
    total_bytes: number
    available_bytes: number
    usage_percent: number
    swap_used_bytes: number
    swap_total_bytes: number
    source: string
  }
  disk: {
    root: ServerLoadDiskUsage
    data: ServerLoadDiskUsage
    read_bytes_per_sec: number
    write_bytes_per_sec: number
  }
  docker: {
    available: boolean
    container_name: string
    image: string
    status: string
    health: string
    uptime_seconds: number
    containers_running: number
    containers_total: number
    cpu_usage_percent: number
    memory_usage_bytes: number
    memory_limit_bytes: number
    network_rx_bytes: number
    network_tx_bytes: number
    block_read_bytes: number
    block_write_bytes: number
    unavailable_reason: string
  }
  runtime: {
    goroutines: number
    heap_alloc_bytes: number
    heap_sys_bytes: number
    gc_count: number
    last_gc_at?: string
    process_uptime_seconds: number
  }
  network: {
    primary_interface: string
    rx_bytes: number
    tx_bytes: number
    rx_bytes_per_sec: number
    tx_bytes_per_sec: number
    tcp_established: number
    tcp_listen: number
    tcp_time_wait: number
  }
  dependencies: {
    backend_ok: boolean
    db_ok: boolean
    redis_ok: boolean
  }
  thresholds: {
    cpu_warning_percent: number
    cpu_critical_percent: number
    memory_warning_percent: number
    memory_critical_percent: number
    disk_warning_percent: number
    disk_critical_percent: number
    goroutines_warning: number
    goroutines_critical: number
  }
  errors?: string[]
}

export async function getSnapshot(): Promise<ServerLoadSnapshot> {
  const { data } = await apiClient.get<ServerLoadSnapshot>('/admin/server-load')
  return data
}

const serverLoadAPI = {
  getSnapshot,
}

export default serverLoadAPI
