import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

const { getSnapshot } = vi.hoisted(() => ({
  getSnapshot: vi.fn(),
}))

vi.mock('@/api/admin/serverLoad', () => ({
  default: {
    getSnapshot,
  },
}))

import ServerLoadView from '../ServerLoadView.vue'
import type { ServerLoadSnapshot } from '@/api/admin/serverLoad'

function makeSnapshot(overrides: Partial<ServerLoadSnapshot> = {}): ServerLoadSnapshot {
  return {
    status: 'ok',
    collected_at: '2026-06-21T10:00:00Z',
    uptime_seconds: 3600,
    cpu: {
      usage_percent: 12.5,
      cores: 4,
      load1: 0.1,
      load5: 0.2,
      load15: 0.3,
      cgroup_usage_percent: 11.5,
      source: 'cgroup',
    },
    memory: {
      used_bytes: 2147483648,
      total_bytes: 8589934592,
      available_bytes: 6442450944,
      usage_percent: 25,
      swap_used_bytes: 0,
      swap_total_bytes: 0,
      source: 'cgroup',
    },
    disk: {
      root: {
        path: '/',
        used_bytes: 10737418240,
        total_bytes: 53687091200,
        usage_percent: 20,
        inode_usage_percent: 8,
      },
      data: {
        path: '/app/data',
        used_bytes: 1073741824,
        total_bytes: 53687091200,
        usage_percent: 2,
        inode_usage_percent: 1,
      },
      read_bytes_per_sec: 1024,
      write_bytes_per_sec: 2048,
    },
    docker: {
      available: false,
      container_name: '',
      image: '',
      status: '',
      health: '',
      uptime_seconds: 0,
      containers_running: 0,
      containers_total: 0,
      cpu_usage_percent: 0,
      memory_usage_bytes: 0,
      memory_limit_bytes: 0,
      network_rx_bytes: 0,
      network_tx_bytes: 0,
      block_read_bytes: 0,
      block_write_bytes: 0,
      unavailable_reason: 'docker socket unavailable',
    },
    runtime: {
      goroutines: 128,
      heap_alloc_bytes: 67108864,
      heap_sys_bytes: 134217728,
      gc_count: 42,
      last_gc_at: '2026-06-21T09:59:58Z',
      process_uptime_seconds: 3600,
    },
    network: {
      primary_interface: 'eth0',
      rx_bytes: 104857600,
      tx_bytes: 52428800,
      rx_bytes_per_sec: 2048,
      tx_bytes_per_sec: 1024,
      tcp_established: 12,
      tcp_listen: 6,
      tcp_time_wait: 3,
    },
    dependencies: {
      backend_ok: true,
      db_ok: true,
      redis_ok: true,
    },
    thresholds: {
      cpu_warning_percent: 80,
      cpu_critical_percent: 90,
      memory_warning_percent: 80,
      memory_critical_percent: 90,
      disk_warning_percent: 85,
      disk_critical_percent: 95,
      goroutines_warning: 8000,
      goroutines_critical: 15000,
    },
    ...overrides,
  }
}

describe('ServerLoadView', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    getSnapshot.mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('loads and renders server load sections', async () => {
    getSnapshot.mockResolvedValue(makeSnapshot())

    const wrapper = mount(ServerLoadView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
        },
      },
    })
    await flushPromises()

    expect(getSnapshot).toHaveBeenCalledTimes(1)
    const text = wrapper.text()
    expect(text).toContain('CPU')
    expect(text).toContain('12.5%')
    expect(text).toContain('内存')
    expect(text).toContain('25.0%')
    expect(text).toContain('Docker')
    expect(text).toContain('docker socket unavailable')
    expect(text).toContain('Go Runtime')
    expect(text).toContain('网络')
    expect(text).toContain('依赖健康')
  })

  it('refreshes snapshot when clicking manual refresh', async () => {
    getSnapshot.mockResolvedValue(makeSnapshot())

    const wrapper = mount(ServerLoadView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
        },
      },
    })
    await flushPromises()

    await wrapper.get('[data-test="server-load-refresh"]').trigger('click')
    await flushPromises()

    expect(getSnapshot).toHaveBeenCalledTimes(2)
  })
})
