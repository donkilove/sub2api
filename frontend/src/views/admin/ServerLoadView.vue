<template>
  <AppLayout>
    <div class="space-y-6 pb-12">
      <section class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div class="flex items-start gap-4">
            <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-primary-500/10 text-primary-600 dark:text-primary-300">
              <ServerIcon class="h-6 w-6" />
            </div>
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-3">
                <h2 class="text-xl font-semibold text-gray-900 dark:text-white">服务器负载</h2>
                <span class="status-pill" :class="statusClass">
                  {{ statusLabel }}
                </span>
              </div>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ collectedAtText }} · 自动刷新 5 秒 · 进程运行 {{ formatDuration(snapshot?.runtime.process_uptime_seconds ?? snapshot?.uptime_seconds ?? 0) }}
              </p>
            </div>
          </div>

          <button
            type="button"
            data-test="server-load-refresh"
            class="inline-flex h-11 items-center justify-center gap-2 rounded-lg border border-gray-300 px-4 text-sm font-medium text-gray-700 transition hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-dark-600 dark:text-gray-200 dark:hover:bg-dark-800"
            :disabled="loading"
            @click="fetchSnapshot"
          >
            <RefreshIcon class="h-4 w-4" :class="{ 'animate-spin': loading }" />
            刷新
          </button>
        </div>

        <div v-if="errorMessage" class="mt-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/40 dark:text-red-300">
          {{ errorMessage }}
        </div>
      </section>

      <div v-if="loading && !snapshot" class="grid grid-cols-1 gap-4 xl:grid-cols-3">
        <div v-for="i in 6" :key="i" class="h-56 animate-pulse rounded-2xl bg-gray-100 dark:bg-dark-800"></div>
      </div>

      <template v-else-if="snapshot">
        <div class="grid grid-cols-1 gap-4 xl:grid-cols-3">
          <MetricPanel title="CPU" :status="metricStatus(snapshot.cpu.usage_percent, snapshot.thresholds.cpu_warning_percent, snapshot.thresholds.cpu_critical_percent)">
            <template #icon><CpuIcon class="h-5 w-5" /></template>
            <ProgressMetric label="使用率" :value="snapshot.cpu.usage_percent" suffix="%" />
            <div class="metric-grid">
              <MetricItem label="核心数" :value="String(snapshot.cpu.cores)" />
              <MetricItem label="来源" :value="snapshot.cpu.source || '-'" />
              <MetricItem label="Load 1m" :value="formatNumber(snapshot.cpu.load1)" />
              <MetricItem label="Load 5m" :value="formatNumber(snapshot.cpu.load5)" />
              <MetricItem label="Load 15m" :value="formatNumber(snapshot.cpu.load15)" />
              <MetricItem label="容器 CPU" :value="formatOptionalPercent(snapshot.cpu.cgroup_usage_percent)" />
            </div>
          </MetricPanel>

          <MetricPanel title="内存" :status="metricStatus(snapshot.memory.usage_percent, snapshot.thresholds.memory_warning_percent, snapshot.thresholds.memory_critical_percent)">
            <template #icon><MemoryIcon class="h-5 w-5" /></template>
            <ProgressMetric label="使用率" :value="snapshot.memory.usage_percent" suffix="%" />
            <div class="metric-grid">
              <MetricItem label="已用" :value="formatBytes(snapshot.memory.used_bytes)" />
              <MetricItem label="总量" :value="formatBytes(snapshot.memory.total_bytes)" />
              <MetricItem label="可用" :value="formatBytes(snapshot.memory.available_bytes)" />
              <MetricItem label="Swap" :value="`${formatBytes(snapshot.memory.swap_used_bytes)} / ${formatBytes(snapshot.memory.swap_total_bytes)}`" />
              <MetricItem label="来源" :value="snapshot.memory.source || '-'" />
            </div>
          </MetricPanel>

          <MetricPanel title="硬盘" :status="diskStatus">
            <template #icon><DiskIcon class="h-5 w-5" /></template>
            <ProgressMetric label="根目录" :value="snapshot.disk.root.usage_percent" suffix="%" />
            <ProgressMetric label="数据目录" :value="snapshot.disk.data.usage_percent" suffix="%" compact />
            <div class="metric-grid">
              <MetricItem label="根目录容量" :value="`${formatBytes(snapshot.disk.root.used_bytes)} / ${formatBytes(snapshot.disk.root.total_bytes)}`" />
              <MetricItem label="数据目录容量" :value="`${formatBytes(snapshot.disk.data.used_bytes)} / ${formatBytes(snapshot.disk.data.total_bytes)}`" />
              <MetricItem label="根目录 inode" :value="`${formatPercent(snapshot.disk.root.inode_usage_percent)}`" />
              <MetricItem label="数据目录 inode" :value="`${formatPercent(snapshot.disk.data.inode_usage_percent)}`" />
              <MetricItem label="读取速率" :value="`${formatBytes(snapshot.disk.read_bytes_per_sec)}/s`" />
              <MetricItem label="写入速率" :value="`${formatBytes(snapshot.disk.write_bytes_per_sec)}/s`" />
            </div>
          </MetricPanel>

          <MetricPanel title="Docker" :status="snapshot.docker.available ? 'ok' : 'unknown'">
            <template #icon><ContainerIcon class="h-5 w-5" /></template>
            <div v-if="!snapshot.docker.available" class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-300">
              {{ snapshot.docker.unavailable_reason || 'docker unavailable' }}
            </div>
            <template v-else>
              <ProgressMetric label="容器 CPU" :value="snapshot.docker.cpu_usage_percent" suffix="%" />
              <div class="metric-grid">
                <MetricItem label="容器" :value="snapshot.docker.container_name || '-'" />
                <MetricItem label="镜像" :value="snapshot.docker.image || '-'" />
                <MetricItem label="状态" :value="snapshot.docker.status || '-'" />
                <MetricItem label="健康" :value="snapshot.docker.health || '-'" />
                <MetricItem label="容器数量" :value="`${snapshot.docker.containers_running}/${snapshot.docker.containers_total}`" />
                <MetricItem label="容器内存" :value="`${formatBytes(snapshot.docker.memory_usage_bytes)} / ${formatBytes(snapshot.docker.memory_limit_bytes)}`" />
                <MetricItem label="网络 IO" :value="`${formatBytes(snapshot.docker.network_rx_bytes)} / ${formatBytes(snapshot.docker.network_tx_bytes)}`" />
                <MetricItem label="块设备 IO" :value="`${formatBytes(snapshot.docker.block_read_bytes)} / ${formatBytes(snapshot.docker.block_write_bytes)}`" />
              </div>
            </template>
          </MetricPanel>

          <MetricPanel title="Go Runtime" :status="metricStatus(snapshot.runtime.goroutines, snapshot.thresholds.goroutines_warning, snapshot.thresholds.goroutines_critical)">
            <template #icon><RuntimeIcon class="h-5 w-5" /></template>
            <div class="metric-grid">
              <MetricItem label="Goroutines" :value="formatInteger(snapshot.runtime.goroutines)" />
              <MetricItem label="Heap Alloc" :value="formatBytes(snapshot.runtime.heap_alloc_bytes)" />
              <MetricItem label="Heap Sys" :value="formatBytes(snapshot.runtime.heap_sys_bytes)" />
              <MetricItem label="GC 次数" :value="formatInteger(snapshot.runtime.gc_count)" />
              <MetricItem label="最后 GC" :value="formatDateTime(snapshot.runtime.last_gc_at)" />
              <MetricItem label="运行时长" :value="formatDuration(snapshot.runtime.process_uptime_seconds)" />
            </div>
          </MetricPanel>

          <MetricPanel title="网络" status="ok">
            <template #icon><NetworkIcon class="h-5 w-5" /></template>
            <div class="metric-grid">
              <MetricItem label="主网卡" :value="snapshot.network.primary_interface || '-'" />
              <MetricItem label="接收速率" :value="`${formatBytes(snapshot.network.rx_bytes_per_sec)}/s`" />
              <MetricItem label="发送速率" :value="`${formatBytes(snapshot.network.tx_bytes_per_sec)}/s`" />
              <MetricItem label="累计接收" :value="formatBytes(snapshot.network.rx_bytes)" />
              <MetricItem label="累计发送" :value="formatBytes(snapshot.network.tx_bytes)" />
              <MetricItem label="TCP ESTABLISHED" :value="formatInteger(snapshot.network.tcp_established)" />
              <MetricItem label="TCP LISTEN" :value="formatInteger(snapshot.network.tcp_listen)" />
              <MetricItem label="TCP TIME_WAIT" :value="formatInteger(snapshot.network.tcp_time_wait)" />
            </div>
          </MetricPanel>
        </div>

        <section class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
          <div class="mb-4 flex items-center gap-3">
            <HealthIcon class="h-5 w-5 text-primary-500" />
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">依赖健康</h3>
          </div>
          <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
            <DependencyStatus label="Backend" :ok="snapshot.dependencies.backend_ok" />
            <DependencyStatus label="DB" :ok="snapshot.dependencies.db_ok" />
            <DependencyStatus label="Redis" :ok="snapshot.dependencies.redis_ok" />
          </div>
          <div v-if="snapshot.errors?.length" class="mt-4 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-300">
            {{ snapshot.errors.join('；') }}
          </div>
        </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, onUnmounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import serverLoadAPI, { type ServerLoadStatus, type ServerLoadSnapshot } from '@/api/admin/serverLoad'

const snapshot = ref<ServerLoadSnapshot | null>(null)
const loading = ref(false)
const errorMessage = ref('')
let refreshTimer: ReturnType<typeof setInterval> | null = null

const statusLabel = computed(() => {
  switch (snapshot.value?.status) {
    case 'ok':
      return '正常'
    case 'warning':
      return '预警'
    case 'critical':
      return '严重'
    default:
      return '未知'
  }
})

const statusClass = computed(() => statusClassFor(snapshot.value?.status ?? 'unknown'))

const collectedAtText = computed(() => {
  if (!snapshot.value?.collected_at) {
    return '尚未采集'
  }
  return `采集时间 ${formatDateTime(snapshot.value.collected_at)}`
})

const diskStatus = computed<ServerLoadStatus>(() => {
  if (!snapshot.value) return 'unknown'
  const usage = Math.max(snapshot.value.disk.root.usage_percent, snapshot.value.disk.data.usage_percent)
  return metricStatus(usage, snapshot.value.thresholds.disk_warning_percent, snapshot.value.thresholds.disk_critical_percent)
})

async function fetchSnapshot() {
  loading.value = true
  errorMessage.value = ''
  try {
    snapshot.value = await serverLoadAPI.getSnapshot()
  } catch (error: any) {
    errorMessage.value = error?.message || '服务器负载数据加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void fetchSnapshot()
  refreshTimer = setInterval(() => {
    void fetchSnapshot()
  }, 5000)
})

onUnmounted(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
})

function metricStatus(value: number, warning: number, critical: number): ServerLoadStatus {
  if (value > critical) return 'critical'
  if (value > warning) return 'warning'
  return 'ok'
}

function statusClassFor(status: ServerLoadStatus) {
  return {
    'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300': status === 'ok',
    'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300': status === 'warning',
    'bg-red-100 text-red-700 dark:bg-red-500/15 dark:text-red-300': status === 'critical',
    'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300': status === 'unknown',
  }
}

function statusText(status: ServerLoadStatus): string {
  switch (status) {
    case 'ok':
      return '正常'
    case 'warning':
      return '预警'
    case 'critical':
      return '严重'
    default:
      return '未知'
  }
}

function formatBytes(input: number | undefined): string {
  const value = Number(input || 0)
  if (value <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1)
  return `${(value / Math.pow(1024, index)).toFixed(index === 0 ? 0 : 1)} ${units[index]}`
}

function formatNumber(value: number): string {
  return Number(value || 0).toFixed(2)
}

function formatPercent(value: number): string {
  return `${Number(value || 0).toFixed(1)}%`
}

function formatOptionalPercent(value?: number): string {
  if (value === undefined || value === null) return '-'
  return formatPercent(value)
}

function formatInteger(value: number): string {
  return Number(value || 0).toLocaleString()
}

function formatDateTime(value?: string): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

function formatDuration(seconds: number): string {
  const total = Math.max(0, Math.floor(seconds || 0))
  const days = Math.floor(total / 86400)
  const hours = Math.floor((total % 86400) / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  if (days > 0) return `${days}天 ${hours}小时`
  if (hours > 0) return `${hours}小时 ${minutes}分钟`
  return `${minutes}分钟`
}

const MetricPanel = defineComponent({
  name: 'MetricPanel',
  props: {
    title: { type: String, required: true },
    status: { type: String as () => ServerLoadStatus, default: 'unknown' },
  },
  setup(props, { slots }) {
    return () =>
      h('section', { class: 'min-h-56 rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900' }, [
        h('div', { class: 'mb-4 flex items-center justify-between gap-3' }, [
          h('div', { class: 'flex min-w-0 items-center gap-3' }, [
            h('div', { class: 'flex h-9 w-9 items-center justify-center rounded-lg bg-primary-500/10 text-primary-600 dark:text-primary-300' }, slots.icon?.()),
            h('h3', { class: 'truncate text-lg font-semibold text-gray-900 dark:text-white' }, props.title),
          ]),
          h('span', { class: ['status-dot', statusClassFor(props.status)] }, statusText(props.status)),
        ]),
        slots.default?.(),
      ])
  },
})

const ProgressMetric = defineComponent({
  name: 'ProgressMetric',
  props: {
    label: { type: String, required: true },
    value: { type: Number, required: true },
    suffix: { type: String, default: '' },
    compact: { type: Boolean, default: false },
  },
  setup(props) {
    return () => {
      const value = Math.max(0, Math.min(100, props.value || 0))
      return h('div', { class: props.compact ? 'mt-3' : '' }, [
        h('div', { class: 'mb-2 flex items-center justify-between text-sm' }, [
          h('span', { class: 'text-gray-500 dark:text-gray-400' }, props.label),
          h('span', { class: 'font-semibold text-gray-900 dark:text-white' }, `${props.value.toFixed(1)}${props.suffix}`),
        ]),
        h('div', { class: 'h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700' }, [
          h('div', {
            class: 'h-full rounded-full bg-primary-500 transition-all',
            style: { width: `${value}%` },
          }),
        ]),
      ])
    }
  },
})

const MetricItem = defineComponent({
  name: 'MetricItem',
  props: {
    label: { type: String, required: true },
    value: { type: String, required: true },
  },
  setup(props) {
    return () =>
      h('div', { class: 'min-w-0 rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-800' }, [
        h('div', { class: 'truncate text-xs font-medium text-gray-500 dark:text-gray-400' }, props.label),
        h('div', { class: 'mt-1 truncate text-sm font-semibold text-gray-900 dark:text-white', title: props.value }, props.value),
      ])
  },
})

const DependencyStatus = defineComponent({
  name: 'DependencyStatus',
  props: {
    label: { type: String, required: true },
    ok: { type: Boolean, required: true },
  },
  setup(props) {
    return () =>
      h('div', { class: 'flex items-center justify-between rounded-lg bg-gray-50 px-4 py-3 dark:bg-dark-800' }, [
        h('span', { class: 'font-medium text-gray-800 dark:text-gray-100' }, props.label),
        h('span', { class: ['status-pill', props.ok ? statusClassFor('ok') : statusClassFor('warning')] }, props.ok ? '正常' : '异常'),
      ])
  },
})

function icon(path: string) {
  return defineComponent({
    render() {
      return h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [
        h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: path }),
      ])
    },
  })
}

const ServerIcon = icon('M5.25 14.25h13.5m-13.5 0a3 3 0 01-3-3m3 3a3 3 0 100 6h13.5a3 3 0 100-6m-16.5-3a3 3 0 013-3h13.5a3 3 0 013 3m-19.5 0a4.5 4.5 0 01.9-2.7L5.737 5.1a3.375 3.375 0 012.7-1.35h7.126c1.062 0 2.062.5 2.7 1.35l2.587 3.45a4.5 4.5 0 01.9 2.7m0 0a3 3 0 01-3 3')
const CpuIcon = icon('M3.75 4.875c0-.621.504-1.125 1.125-1.125h14.25c.621 0 1.125.504 1.125 1.125v14.25c0 .621-.504 1.125-1.125 1.125H4.875a1.125 1.125 0 01-1.125-1.125V4.875zM8.25 8.25h7.5v7.5h-7.5v-7.5zM9 1.5v2.25m6-2.25v2.25M9 20.25v2.25m6-2.25v2.25M1.5 9h2.25m-2.25 6h2.25M20.25 9h2.25m-2.25 6h2.25')
const MemoryIcon = icon('M6.75 7.5h10.5v9H6.75v-9zM4.5 9h2.25M4.5 15h2.25M17.25 9h2.25M17.25 15h2.25M9 4.5v3M15 4.5v3M9 16.5v3M15 16.5v3')
const DiskIcon = icon('M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375')
const ContainerIcon = icon('M21 7.5l-9-4.5-9 4.5m18 0l-9 4.5m9-4.5v9l-9 4.5m0-9L3 7.5m9 4.5v9m-9-13.5v9l9 4.5')
const RuntimeIcon = icon('M13.5 16.875h3.375m0 0h3.375m-3.375 0V13.5m0 3.375v3.375M6.75 6.75h10.5v4.5H6.75v-4.5zM4.5 4.5h15v17.25h-15V4.5z')
const NetworkIcon = icon('M12 3v6m0 0l4.5 4.5M12 9l-4.5 4.5M12 21v-6m-6 0h12')
const HealthIcon = icon('M9 12.75L11.25 15 15 9.75M21 12c0 4.556-3.08 8.394-7.267 9.545a1.5 1.5 0 01-.733 0C8.811 20.394 5.25 16.556 5.25 12V6.75A2.25 2.25 0 017.5 4.5h9A2.25 2.25 0 0118.75 6.75V12')
const RefreshIcon = icon('M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M20.015 4.356v4.992m0 0h-4.992m4.992 0l-3.181-3.183a8.25 8.25 0 00-13.803 3.7')
</script>

<style scoped>
.metric-grid {
  margin-top: 1rem;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
}

.status-pill,
.status-dot {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  padding: 0.25rem 0.625rem;
  font-size: 0.75rem;
  font-weight: 700;
  line-height: 1rem;
}

.status-dot {
  min-width: 4.5rem;
  text-transform: uppercase;
}
</style>
