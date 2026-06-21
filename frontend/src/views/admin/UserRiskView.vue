<template>
  <AppLayout>
    <div class="space-y-6 pb-12">
      <section class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div class="flex items-start gap-4">
            <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-primary-500/10 text-primary-600 dark:text-primary-300">
              <Icon name="shield" size="lg" />
            </div>
            <div class="min-w-0">
              <h2 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('admin.userRisk.title') }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.userRisk.description') }}</p>
              <p class="mt-1 text-xs text-gray-400 dark:text-dark-400">
                {{ generatedAt ? `生成时间 ${formatDateTime(generatedAt)}` : '尚未生成' }}
              </p>
            </div>
          </div>

          <button
            type="button"
            class="btn btn-secondary"
            :disabled="loading"
            @click="reloadCurrent"
          >
            <Icon name="refresh" size="sm" class="mr-2" :class="{ 'animate-spin': loading }" />
            刷新
          </button>
        </div>

        <div v-if="errorMessage" class="mt-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/40 dark:text-red-300">
          {{ errorMessage }}
        </div>
      </section>

      <section class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-6">
        <SummaryCard title="用户总数" :value="formatInteger(summary.total_users)" icon="users" tone="neutral" />
        <SummaryCard title="需关注用户" :value="formatInteger(summary.risky_users)" icon="exclamationTriangle" tone="amber" />
        <SummaryCard title="高危/严重" :value="`${formatInteger(summary.high_risk_users)} / ${formatInteger(summary.critical_risk_users)}`" icon="fire" tone="red" />
        <SummaryCard title="窗口请求" :value="formatInteger(summary.request_count)" icon="chart" tone="blue" />
        <SummaryCard title="错误率" :value="formatPercent(summary.error_rate)" icon="bolt" tone="purple" />
        <SummaryCard title="共享 IP 用户" :value="formatInteger(summary.shared_ip_users)" icon="link" tone="teal" />
      </section>

      <TablePageLayout>
        <template #filters>
          <div class="flex flex-col justify-between gap-4 xl:flex-row xl:items-center">
            <div class="flex flex-1 flex-col gap-3 md:flex-row md:flex-wrap">
              <div class="relative w-full md:w-80">
                <Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                <input
                  v-model="filters.search"
                  type="text"
                  class="input pl-10"
                  placeholder="搜索邮箱、用户名、备注或 ID"
                  @keyup.enter="reloadFromFirstPage"
                />
              </div>

              <select v-model="filters.window" class="input w-full md:w-32" @change="reloadFromFirstPage">
                <option value="1h">近 1 小时</option>
                <option value="24h">近 24 小时</option>
                <option value="7d">近 7 天</option>
                <option value="30d">近 30 天</option>
              </select>

              <select v-model="filters.risk_level" class="input w-full md:w-36" @change="reloadFromFirstPage">
                <option value="">全部风险</option>
                <option value="critical">严重</option>
                <option value="high">高</option>
                <option value="medium">中</option>
                <option value="low">低</option>
              </select>

              <select v-model="filters.status" class="input w-full md:w-36" @change="reloadFromFirstPage">
                <option value="">全部状态</option>
                <option value="active">启用</option>
                <option value="disabled">禁用</option>
              </select>

              <label class="inline-flex h-11 items-center gap-2 rounded-lg border border-gray-300 px-3 text-sm text-gray-700 dark:border-dark-600 dark:text-gray-200">
                <input v-model="filters.only_risky" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" @change="reloadFromFirstPage" />
                只看异常
              </label>
            </div>

            <div class="flex flex-wrap justify-end gap-2">
              <button type="button" class="btn btn-secondary" :disabled="loading" @click="resetFilters">
                重置
              </button>
              <button type="button" class="btn btn-primary" :disabled="loading" @click="reloadFromFirstPage">
                <Icon name="search" size="sm" class="mr-2" />
                搜索
              </button>
            </div>
          </div>
        </template>

        <template #table>
          <DataTable
            :columns="columns"
            :data="items"
            :loading="loading"
            :row-key="row => row.user.id"
            :sticky-actions-column="false"
            :estimate-row-height="92"
          >
            <template #cell-risk="{ row }">
              <div class="flex min-w-[170px] items-center gap-3">
                <div class="score-ring" :class="levelRingClass(row.level)">
                  {{ row.score }}
                </div>
                <div>
                  <div :class="levelBadgeClass(row.level)">{{ levelLabel(row.level) }}</div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ row.reasons.length }} 条原因</div>
                  <button
                    type="button"
                    class="mt-2 inline-flex items-center text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-300 dark:hover:text-primary-200"
                    @click="openDetail(row)"
                  >
                    <Icon name="eye" size="xs" class="mr-1" />
                    查看
                  </button>
                </div>
              </div>
            </template>

            <template #cell-user="{ row }">
              <div class="min-w-[260px]">
                <div class="flex flex-wrap items-center gap-2">
                  <span class="font-medium text-gray-900 dark:text-white">{{ row.user.email }}</span>
                  <span class="text-xs text-gray-400">#{{ row.user.id }}</span>
                  <span :class="statusBadgeClass(row.user.status)">{{ statusLabel(row.user.status) }}</span>
                </div>
                <div class="mt-1 flex flex-wrap gap-2 text-xs text-gray-500 dark:text-dark-400">
                  <span>{{ row.user.username || row.user.notes || '-' }}</span>
                  <span>来源 {{ row.user.signup_source || '-' }}</span>
                  <span>余额 {{ formatMoney(row.user.balance) }}</span>
                </div>
              </div>
            </template>

            <template #cell-traffic="{ row }">
              <div class="min-w-[150px] space-y-1">
                <MetricLine label="请求" :value="formatInteger(row.metrics.request_count)" />
                <MetricLine label="Token" :value="formatCompact(row.metrics.token_count)" />
                <MetricLine label="消费" :value="formatMoney(row.metrics.cost)" />
              </div>
            </template>

            <template #cell-errors="{ row }">
              <div class="min-w-[170px] space-y-1">
                <MetricLine label="错误率" :value="formatPercent(row.metrics.error_rate)" />
                <MetricLine label="429/认证" :value="`${row.metrics.rate_limited_count}/${row.metrics.auth_error_count}`" />
                <MetricLine label="5xx/超时" :value="`${row.metrics.upstream_5xx_count}/${row.metrics.timeout_count}`" />
              </div>
            </template>

            <template #cell-sources="{ row }">
              <div class="min-w-[150px] space-y-1">
                <MetricLine label="IP" :value="formatInteger(row.metrics.unique_ips)" />
                <MetricLine label="关联账号" :value="formatInteger(row.ip_risk?.linked_user_count)" />
                <MetricLine label="同 IP 最多" :value="formatInteger(row.ip_risk?.max_users_on_same_ip)" />
                <MetricLine label="模型" :value="formatInteger(row.metrics.unique_models)" />
              </div>
            </template>

            <template #cell-concurrency="{ row }">
              <div class="min-w-[150px] space-y-1">
                <MetricLine label="并发" :value="`${row.concurrency.current_in_use}/${limitText(row.concurrency.max_capacity)}`" />
                <MetricLine label="排队" :value="formatInteger(row.concurrency.waiting_in_queue)" />
                <MetricLine label="RPM" :value="limitText(row.limits.user_rpm_limit_override ?? row.limits.legacy_rpm)" />
              </div>
            </template>

            <template #cell-reasons="{ row }">
              <div class="min-w-[240px]">
                <div v-if="row.reasons.length" class="flex flex-wrap gap-1.5">
                  <span v-for="reason in row.reasons.slice(0, 3)" :key="reason.code" :class="reasonBadgeClass(reason.severity)">
                    {{ reason.label }}
                  </span>
                  <span v-if="row.reasons.length > 3" class="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-700 dark:text-dark-300">
                    +{{ row.reasons.length - 3 }}
                  </span>
                </div>
                <span v-else class="text-sm text-gray-400">暂无异常</span>
                <div class="mt-2 text-xs text-gray-500 dark:text-dark-400">
                  最近请求 {{ formatDateTime(row.metrics.last_request_at) }}
                </div>
              </div>
            </template>

            <template #cell-actions="{ row }">
              <button type="button" class="btn btn-secondary btn-sm whitespace-nowrap" @click="openDetail(row)">
                <Icon name="eye" size="sm" class="mr-1" />
                详情
              </button>
            </template>

            <template #empty>
              <div class="flex flex-col items-center py-8">
                <Icon name="inbox" size="xl" class="mb-3 text-gray-400" />
                <p class="text-sm text-gray-500 dark:text-dark-400">没有匹配的用户风控数据</p>
              </div>
            </template>
          </DataTable>
        </template>

        <template #pagination>
          <Pagination
            :total="pagination.total"
            :page="pagination.page"
            :page-size="pagination.pageSize"
            @update:page="handlePageChange"
            @update:pageSize="handlePageSizeChange"
          />
        </template>
      </TablePageLayout>

      <BaseDialog
        :show="detailOpen"
        :title="selectedItem ? `${selectedItem.user.email} 风控明细` : '风控明细'"
        width="full"
        @close="closeDetail"
      >
        <div v-if="detailLoading" class="space-y-4">
          <div v-for="i in 4" :key="i" class="h-24 animate-pulse rounded-xl bg-gray-100 dark:bg-dark-800"></div>
        </div>
        <div v-else-if="detail" class="space-y-6">
          <section class="grid grid-cols-1 gap-3 md:grid-cols-4">
            <DetailMetric label="风险分" :value="String(detail.item.score)" :hint="levelLabel(detail.item.level)" />
            <DetailMetric label="请求/错误" :value="`${formatInteger(detail.item.metrics.request_count)} / ${formatInteger(detail.item.metrics.error_count)}`" :hint="formatPercent(detail.item.metrics.error_rate)" />
            <DetailMetric label="消费" :value="formatMoney(detail.item.metrics.cost)" :hint="`上期 ${formatMoney(detail.item.metrics.previous_cost)}`" />
            <DetailMetric label="并发" :value="`${detail.item.concurrency.current_in_use}/${limitText(detail.item.concurrency.max_capacity)}`" :hint="`排队 ${detail.item.concurrency.waiting_in_queue}`" />
            <DetailMetric label="IP 关联" :value="formatInteger(detail.item.ip_risk?.linked_user_count)" :hint="`共享 IP ${formatInteger(detail.item.ip_risk?.shared_ip_count)}`" />
          </section>

          <section class="grid grid-cols-1 gap-4 xl:grid-cols-2">
            <div class="detail-card">
              <h3 class="detail-title">风险原因</h3>
              <div v-if="detail.item.reasons.length" class="space-y-2">
                <div v-for="reason in detail.item.reasons" :key="reason.code" class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
                  <div class="flex items-center justify-between gap-3">
                    <span class="text-sm font-medium text-gray-900 dark:text-white">{{ reason.label }}</span>
                    <span :class="reasonBadgeClass(reason.severity)">{{ severityLabel(reason.severity) }}</span>
                  </div>
                  <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ reason.detail }}</p>
                </div>
              </div>
              <p v-else class="text-sm text-gray-500 dark:text-dark-400">暂无风险原因</p>
            </div>

            <div class="detail-card">
              <h3 class="detail-title">最近错误</h3>
              <div v-if="detail.recent_errors.length" class="space-y-2">
                <div v-for="err in detail.recent_errors" :key="err.id" class="rounded-lg bg-gray-50 p-3 text-sm dark:bg-dark-800">
                  <div class="flex flex-wrap items-center gap-2">
                    <span class="font-medium text-gray-900 dark:text-white">{{ err.type || '-' }}</span>
                    <span class="rounded bg-red-100 px-1.5 py-0.5 text-xs text-red-700 dark:bg-red-500/15 dark:text-red-300">{{ err.status_code || '-' }}</span>
                    <span class="text-xs text-gray-400">{{ formatDateTime(err.created_at) }}</span>
                  </div>
                  <p class="mt-1 line-clamp-2 text-xs text-gray-500 dark:text-dark-400">{{ err.message || '-' }}</p>
                </div>
              </div>
              <p v-else class="text-sm text-gray-500 dark:text-dark-400">窗口内暂无错误</p>
            </div>
          </section>

          <section class="grid grid-cols-1 gap-4 xl:grid-cols-3">
            <div class="detail-card xl:col-span-1">
              <h3 class="detail-title">API Key</h3>
              <div class="space-y-2">
                <div v-for="key in detail.api_keys" :key="key.id" class="rounded-lg bg-gray-50 p-3 text-sm dark:bg-dark-800">
                  <div class="flex items-center justify-between gap-3">
                    <span class="truncate font-medium text-gray-900 dark:text-white">{{ key.name }}</span>
                    <span :class="statusBadgeClass(key.status)">{{ statusLabel(key.status) }}</span>
                  </div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                    请求 {{ formatInteger(key.request_count) }} · 错误 {{ formatInteger(key.error_count) }} · {{ formatMoney(key.cost) }}
                  </div>
                </div>
                <p v-if="!detail.api_keys.length" class="text-sm text-gray-500 dark:text-dark-400">暂无 API Key</p>
              </div>
            </div>

            <div class="detail-card">
              <h3 class="detail-title">Top 模型</h3>
              <div class="space-y-2">
                <div v-for="model in detail.top_models" :key="model.model" class="flex items-center justify-between gap-3 text-sm">
                  <span class="truncate text-gray-900 dark:text-white">{{ model.model }}</span>
                  <span class="text-gray-500 dark:text-dark-400">{{ formatInteger(model.request_count) }} / {{ formatMoney(model.cost) }}</span>
                </div>
                <p v-if="!detail.top_models.length" class="text-sm text-gray-500 dark:text-dark-400">暂无模型数据</p>
              </div>
            </div>

            <div class="detail-card">
              <h3 class="detail-title">Top IP</h3>
              <div class="space-y-2">
                <div v-for="ip in detail.top_ips" :key="ip.ip" class="flex items-center justify-between gap-3 text-sm">
                  <span class="truncate text-gray-900 dark:text-white">{{ ip.ip }}</span>
                  <span class="text-gray-500 dark:text-dark-400">{{ formatInteger(ip.request_count) }} / {{ formatInteger(ip.error_count) }}</span>
                </div>
                <p v-if="!detail.top_ips.length" class="text-sm text-gray-500 dark:text-dark-400">暂无 IP 数据</p>
              </div>
            </div>
          </section>

          <section class="grid grid-cols-1 gap-4 xl:grid-cols-2">
            <div class="detail-card">
              <h3 class="detail-title">IP 关联分析</h3>
              <div v-if="detail.ip_links.length" class="space-y-3">
                <div v-for="link in detail.ip_links" :key="link.ip" class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
                  <div class="flex flex-wrap items-center justify-between gap-3">
                    <div>
                      <div class="font-mono text-sm font-semibold text-gray-900 dark:text-white">{{ link.ip }}</div>
                      <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                        关联 {{ formatInteger(link.other_user_count) }} 个其他账号 · 新账号 {{ formatInteger(link.new_user_count) }} · 注册事件 {{ formatInteger(link.register_event_count) }}
                      </div>
                    </div>
                    <span class="rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-500/15 dark:text-amber-300">
                      同 UA {{ formatInteger(link.same_ua_user_count) }}
                    </span>
                  </div>
                  <div class="mt-3 space-y-2">
                    <div v-for="linked in link.linked_users.slice(0, 6)" :key="`${link.ip}-${linked.id}`" class="flex flex-wrap items-center justify-between gap-2 rounded bg-gray-50 px-3 py-2 text-xs dark:bg-dark-800">
                      <div class="min-w-0">
                        <span class="font-medium text-gray-900 dark:text-white">{{ linked.email }}</span>
                        <span class="ml-1 text-gray-400">#{{ linked.id }}</span>
                        <span v-if="linked.shared_user_agent_hint" class="ml-2 rounded bg-cyan-100 px-1.5 py-0.5 text-cyan-700 dark:bg-cyan-500/15 dark:text-cyan-300">同 UA</span>
                      </div>
                      <span class="text-gray-500 dark:text-dark-400">
                        请求 {{ formatInteger(linked.request_count) }} · 认证 {{ formatInteger(linked.auth_event_count) }}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
              <p v-else class="text-sm text-gray-500 dark:text-dark-400">窗口内暂无跨账号 IP 关联</p>
            </div>

            <div class="detail-card">
              <h3 class="detail-title">最近认证事件</h3>
              <div v-if="detail.auth_events.length" class="space-y-2">
                <div v-for="event in detail.auth_events" :key="event.id" class="rounded-lg bg-gray-50 p-3 text-sm dark:bg-dark-800">
                  <div class="flex flex-wrap items-center justify-between gap-2">
                    <div class="flex flex-wrap items-center gap-2">
                      <span class="font-medium text-gray-900 dark:text-white">{{ authEventLabel(event.event_type) }}</span>
                      <span class="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-600 dark:bg-dark-700 dark:text-dark-300">{{ event.provider || '-' }}</span>
                      <span :class="event.success ? statusBadgeClass('active') : reasonBadgeClass('high')">{{ event.success ? '成功' : '失败' }}</span>
                    </div>
                    <span class="text-xs text-gray-400">{{ formatDateTime(event.created_at) }}</span>
                  </div>
                  <div class="mt-1 flex flex-wrap gap-2 text-xs text-gray-500 dark:text-dark-400">
                    <span class="font-mono">{{ event.ip_address || '-' }}</span>
                    <span v-if="event.reason">{{ event.reason }}</span>
                  </div>
                </div>
              </div>
              <p v-else class="text-sm text-gray-500 dark:text-dark-400">窗口内暂无认证事件</p>
            </div>
          </section>
        </div>
        <div v-else-if="detailError" class="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/40 dark:text-red-300">
          {{ detailError }}
        </div>
      </BaseDialog>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI, type UserRiskDetail, type UserRiskItem, type UserRiskLevel } from '@/api/admin'
import type { Column } from '@/components/common/types'

type WindowValue = '1h' | '24h' | '7d' | '30d'

const { t } = useI18n()

const loading = ref(false)
const errorMessage = ref('')
const items = ref<UserRiskItem[]>([])
const generatedAt = ref('')
const summary = reactive({
  total_users: 0,
  risky_users: 0,
  high_risk_users: 0,
  critical_risk_users: 0,
  request_count: 0,
  error_count: 0,
  error_rate: 0,
  active_concurrency: 0,
  waiting_in_queue: 0,
  shared_ip_users: 0,
  shared_ip_groups: 0
})

const filters = reactive<{
  search: string
  window: WindowValue
  risk_level: '' | UserRiskLevel
  status: '' | 'active' | 'disabled'
  only_risky: boolean
}>({
  search: '',
  window: '24h',
  risk_level: '',
  status: '',
  only_risky: false
})

const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

const detailOpen = ref(false)
const detailLoading = ref(false)
const detailError = ref('')
const selectedItem = ref<UserRiskItem | null>(null)
const detail = ref<UserRiskDetail | null>(null)

const columns = computed<Column[]>(() => [
  { key: 'risk', label: '风险' },
  { key: 'user', label: '用户' },
  { key: 'traffic', label: '流量' },
  { key: 'errors', label: '错误' },
  { key: 'sources', label: '来源' },
  { key: 'concurrency', label: '限流/并发' },
  { key: 'reasons', label: '原因' },
  { key: 'actions', label: '操作', class: 'w-24' }
])

async function loadList() {
  loading.value = true
  errorMessage.value = ''
  try {
    const result = await adminAPI.userRisk.list({
      page: pagination.page,
      page_size: pagination.pageSize,
      window: filters.window,
      search: filters.search.trim() || undefined,
      status: filters.status,
      risk_level: filters.risk_level,
      only_risky: filters.only_risky
    })
    items.value = result.items || []
    pagination.total = result.total || 0
    generatedAt.value = result.generated_at || ''
    Object.assign(summary, result.summary || {})
  } catch (error: any) {
    errorMessage.value = error?.message || '用户风控数据加载失败'
  } finally {
    loading.value = false
  }
}

function reloadFromFirstPage() {
  pagination.page = 1
  void loadList()
}

function reloadCurrent() {
  void loadList()
}

function resetFilters() {
  filters.search = ''
  filters.window = '24h'
  filters.risk_level = ''
  filters.status = ''
  filters.only_risky = false
  reloadFromFirstPage()
}

function handlePageChange(page: number) {
  pagination.page = page
  void loadList()
}

function handlePageSizeChange(pageSize: number) {
  pagination.pageSize = pageSize
  pagination.page = 1
  void loadList()
}

async function openDetail(item: UserRiskItem) {
  selectedItem.value = item
  detailOpen.value = true
  detailLoading.value = true
  detailError.value = ''
  detail.value = null
  try {
    detail.value = await adminAPI.userRisk.getDetail(item.user.id, filters.window)
  } catch (error: any) {
    detailError.value = error?.message || '用户风控明细加载失败'
  } finally {
    detailLoading.value = false
  }
}

function closeDetail() {
  detailOpen.value = false
  detail.value = null
  detailError.value = ''
  selectedItem.value = null
}

function levelLabel(level: UserRiskLevel | string): string {
  switch (level) {
    case 'critical':
      return '严重'
    case 'high':
      return '高'
    case 'medium':
      return '中'
    default:
      return '低'
  }
}

function statusLabel(status: string): string {
  if (status === 'active') return '启用'
  if (status === 'disabled') return '禁用'
  return status || '-'
}

function authEventLabel(type: string): string {
  switch (type) {
    case 'register':
      return '邮箱注册'
    case 'login':
      return '邮箱登录'
    case 'login_2fa':
      return '二次验证'
    case 'oauth_register':
      return 'OAuth 注册'
    case 'oauth_login':
      return 'OAuth 登录'
    case 'oauth_bind_login':
      return 'OAuth 绑定登录'
    default:
      return type || '-'
  }
}

function severityLabel(severity: string): string {
  if (severity === 'high') return '高'
  if (severity === 'medium') return '中'
  return '低'
}

function levelBadgeClass(level: UserRiskLevel | string) {
  return [
    'inline-flex rounded-full px-2 py-0.5 text-xs font-medium',
    level === 'critical' && 'bg-red-100 text-red-700 dark:bg-red-500/15 dark:text-red-300',
    level === 'high' && 'bg-orange-100 text-orange-700 dark:bg-orange-500/15 dark:text-orange-300',
    level === 'medium' && 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300',
    level === 'low' && 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
  ]
}

function levelRingClass(level: UserRiskLevel | string) {
  return {
    'border-red-300 bg-red-50 text-red-700 dark:border-red-500/40 dark:bg-red-500/10 dark:text-red-300': level === 'critical',
    'border-orange-300 bg-orange-50 text-orange-700 dark:border-orange-500/40 dark:bg-orange-500/10 dark:text-orange-300': level === 'high',
    'border-amber-300 bg-amber-50 text-amber-700 dark:border-amber-500/40 dark:bg-amber-500/10 dark:text-amber-300': level === 'medium',
    'border-emerald-300 bg-emerald-50 text-emerald-700 dark:border-emerald-500/40 dark:bg-emerald-500/10 dark:text-emerald-300': level === 'low'
  }
}

function statusBadgeClass(status: string) {
  return [
    'inline-flex rounded-full px-2 py-0.5 text-xs font-medium',
    status === 'active'
      ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
      : 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-dark-300'
  ]
}

function reasonBadgeClass(severity: string) {
  return [
    'inline-flex rounded-full px-2 py-0.5 text-xs font-medium',
    severity === 'high' && 'bg-red-100 text-red-700 dark:bg-red-500/15 dark:text-red-300',
    severity === 'medium' && 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300',
    (!severity || severity === 'low') && 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
  ]
}

function limitText(value: number | null | undefined): string {
  if (value === null || value === undefined) return '继承'
  if (value <= 0) return '不限'
  return formatInteger(value)
}

function formatInteger(value?: number | null): string {
  return Number(value || 0).toLocaleString()
}

function formatCompact(value?: number | null): string {
  const n = Number(value || 0)
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return n.toLocaleString()
}

function formatMoney(value?: number | null): string {
  return `$${Number(value || 0).toFixed(4)}`
}

function formatPercent(value?: number | null): string {
  return `${(Number(value || 0) * 100).toFixed(1)}%`
}

function formatDateTime(value?: string | null): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString('zh-CN', { hour12: false })
}

onMounted(() => {
  void loadList()
})

const SummaryCard = defineComponent({
  name: 'SummaryCard',
  props: {
    title: { type: String, required: true },
    value: { type: String, required: true },
    icon: { type: String, required: true },
    tone: { type: String, default: 'neutral' }
  },
  setup(props) {
    const toneClass = computed(() => ({
      neutral: 'bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-dark-300',
      amber: 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300',
      red: 'bg-red-100 text-red-700 dark:bg-red-500/15 dark:text-red-300',
      blue: 'bg-blue-100 text-blue-700 dark:bg-blue-500/15 dark:text-blue-300',
      purple: 'bg-violet-100 text-violet-700 dark:bg-violet-500/15 dark:text-violet-300',
      teal: 'bg-teal-100 text-teal-700 dark:bg-teal-500/15 dark:text-teal-300'
    })[props.tone] || 'bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-dark-300')
    return () => h('div', { class: 'rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-900' }, [
      h('div', { class: 'flex items-center justify-between gap-3' }, [
        h('div', [
          h('p', { class: 'text-sm text-gray-500 dark:text-dark-400' }, props.title),
          h('p', { class: 'mt-2 text-2xl font-semibold text-gray-900 dark:text-white' }, props.value)
        ]),
        h('div', { class: ['flex h-10 w-10 items-center justify-center rounded-lg', toneClass.value] }, [
          h(Icon, { name: props.icon as any, size: 'md' })
        ])
      ])
    ])
  }
})

const MetricLine = defineComponent({
  name: 'MetricLine',
  props: {
    label: { type: String, required: true },
    value: { type: String, required: true }
  },
  setup(props) {
    return () => h('div', { class: 'flex items-center justify-between gap-3 text-xs' }, [
      h('span', { class: 'text-gray-500 dark:text-dark-400' }, props.label),
      h('span', { class: 'font-medium text-gray-900 dark:text-white' }, props.value)
    ])
  }
})

const DetailMetric = defineComponent({
  name: 'DetailMetric',
  props: {
    label: { type: String, required: true },
    value: { type: String, required: true },
    hint: { type: String, default: '' }
  },
  setup(props) {
    return () => h('div', { class: 'rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800' }, [
      h('p', { class: 'text-xs text-gray-500 dark:text-dark-400' }, props.label),
      h('p', { class: 'mt-2 text-xl font-semibold text-gray-900 dark:text-white' }, props.value),
      props.hint ? h('p', { class: 'mt-1 text-xs text-gray-500 dark:text-dark-400' }, props.hint) : null
    ])
  }
})
</script>

<style scoped>
.score-ring {
  @apply flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full border text-sm font-bold;
}

.detail-card {
  @apply rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900;
}

.detail-title {
  @apply mb-3 text-sm font-semibold text-gray-900 dark:text-white;
}
</style>
