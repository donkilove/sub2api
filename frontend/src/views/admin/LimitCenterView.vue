<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <div class="mb-1 flex gap-2 border-b border-gray-200 dark:border-dark-700">
          <button
            class="tab"
            :class="{ 'tab-active': activeTab === 'group' }"
            type="button"
            @click="activeTab = 'group'"
          >
            {{ t('admin.limitCenter.tabs.group') }}
          </button>
          <button
            class="tab"
            :class="{ 'tab-active': activeTab === 'user' }"
            type="button"
            @click="activeTab = 'user'"
          >
            {{ t('admin.limitCenter.tabs.user') }}
          </button>
        </div>
      </template>

      <template #filters>
        <div v-if="activeTab === 'group'" class="space-y-3">
          <div class="flex flex-col justify-between gap-4 lg:flex-row lg:items-center">
            <div class="flex flex-1 flex-wrap items-center gap-3">
              <div class="w-full sm:w-72">
                <Select
                  v-model="selectedGroupID"
                  :options="groupOptions"
                  searchable
                  :placeholder="t('admin.limitCenter.groupPicker')"
                  @change="handleGroupChange"
                />
              </div>

              <div class="relative w-full sm:w-64">
                <Icon
                  name="search"
                  size="md"
                  class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
                />
                <input
                  v-model="userSearch"
                  type="text"
                  class="input pl-10"
                  :placeholder="t('admin.limitCenter.searchPlaceholder')"
                  @keyup.enter="loadGroupUsers"
                />
              </div>

              <div class="flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
                <span class="rounded-full bg-gray-100 px-2.5 py-1 dark:bg-dark-700">
                  {{ t('admin.limitCenter.groupRpm') }}: {{ selectedGroup?.rpm_limit ?? 0 }}
                </span>
                <span class="rounded-full bg-gray-100 px-2.5 py-1 dark:bg-dark-700">
                  {{ t('admin.limitCenter.stats.usersInGroup') }}: {{ usersPagination.total }}
                </span>
                <span class="rounded-full bg-gray-100 px-2.5 py-1 dark:bg-dark-700">
                  {{ t('admin.limitCenter.stats.rpmOverrides') }}: {{ rpmLoading ? '-' : rpmEntries.length }}
                </span>
                <span v-if="selectedCapacity" class="rounded-full bg-gray-100 px-2.5 py-1 dark:bg-dark-700">
                  {{ t('admin.limitCenter.capacityConcurrency') }}:
                  {{ selectedCapacity.concurrency_used }}/{{ selectedCapacity.concurrency_max }}
                </span>
              </div>
            </div>

            <div class="flex w-full flex-shrink-0 flex-wrap items-center justify-end gap-2 lg:w-auto">
              <button
                type="button"
                class="btn btn-secondary px-2 md:px-3"
                :title="t('common.refresh')"
                :disabled="usersLoading || rpmLoading"
                @click="refreshGroupView"
              >
                <Icon
                  name="refresh"
                  size="md"
                  :class="usersLoading || rpmLoading ? 'animate-spin' : ''"
                />
              </button>
              <button type="button" class="btn btn-secondary" @click="resetGroupUserSearch">
                {{ t('common.reset') }}
              </button>
              <button type="button" class="btn btn-primary" :disabled="usersLoading" @click="loadGroupUsers">
                <Icon name="search" size="sm" class="mr-2" />
                {{ t('common.search') }}
              </button>
            </div>
          </div>

          <div
            v-if="selectedUserIDs.length > 0"
            class="flex flex-col gap-3 rounded-xl border border-primary-200 bg-primary-50/80 p-3 dark:border-primary-800/60 dark:bg-primary-900/20 lg:flex-row lg:items-center lg:justify-between"
          >
            <div class="text-sm text-primary-800 dark:text-primary-200">
              {{ t('admin.limitCenter.bulkPanel.selected') }}:
              <span class="font-semibold">{{ selectedUserIDs.length }}</span>
            </div>
            <div class="flex flex-1 flex-wrap items-center justify-end gap-2">
              <Select
                v-model="concurrencyMode"
                :options="concurrencyModeOptions"
                class="w-28"
              />
              <input v-model.number="concurrencyValue" type="number" min="1" step="1" class="input w-28" />
              <button
                type="button"
                class="btn btn-primary"
                :disabled="bulkSaving || concurrencyValue < 1"
                @click="applyConcurrency"
              >
                {{ t('admin.limitCenter.bulkPanel.applyConcurrency') }}
              </button>
              <input v-model.number="rpmOverrideValue" type="number" min="0" step="1" class="input w-32" />
              <button
                type="button"
                class="btn btn-secondary"
                :disabled="bulkSaving || rpmOverrideValue < 0"
                @click="applyRPMOverride"
              >
                {{ t('admin.limitCenter.bulkPanel.applyRpm') }}
              </button>
              <button
                type="button"
                class="btn btn-secondary"
                :disabled="bulkSaving"
                @click="clearSelectedRPMOverride"
              >
                {{ t('admin.limitCenter.bulkPanel.clearRpm') }}
              </button>
              <button type="button" class="btn btn-secondary px-2 md:px-3" @click="clearSelection">
                <Icon name="x" size="sm" />
              </button>
            </div>
          </div>
        </div>

        <div v-else class="flex flex-col justify-between gap-4 lg:flex-row lg:items-center">
          <div class="relative w-full lg:max-w-md">
            <Icon
              name="search"
              size="md"
              class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
            />
            <input
              v-model="inspectUserSearch"
              type="text"
              class="input pl-10"
              :placeholder="t('admin.limitCenter.inspect.searchPlaceholder')"
              @keyup.enter="searchInspectUsers"
            />
          </div>
          <div class="flex w-full flex-shrink-0 justify-end gap-2 lg:w-auto">
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="inspectLoading"
              @click="clearInspectSearch"
            >
              {{ t('common.reset') }}
            </button>
            <button
              type="button"
              class="btn btn-primary"
              :disabled="inspectLoading"
              @click="searchInspectUsers"
            >
              <Icon name="search" size="sm" class="mr-2" />
              {{ t('common.search') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          v-if="activeTab === 'group'"
          :columns="groupUserColumns"
          :data="groupUsers"
          :loading="usersLoading"
          row-key="id"
          :sticky-actions-column="false"
          :estimate-row-height="64"
        >
          <template #header-select>
            <input
              ref="pageSelectCheckboxRef"
              type="checkbox"
              class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="pageAllSelected"
              @change="togglePageSelection(($event.target as HTMLInputElement).checked)"
            />
          </template>

          <template #cell-select="{ row }">
            <input
              type="checkbox"
              class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="selectedUserIDSet.has(row.id)"
              @change="toggleUserSelection(row, ($event.target as HTMLInputElement).checked)"
            />
          </template>

          <template #cell-user="{ row }">
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="font-medium text-gray-900 dark:text-white">{{ row.email }}</span>
                <span class="text-xs text-gray-400">#{{ row.id }}</span>
              </div>
              <div class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                {{ row.username || '-' }}
              </div>
            </div>
          </template>

          <template #cell-concurrency="{ row }">
            <span class="font-mono text-gray-900 dark:text-white">
              {{ row.current_concurrency ?? 0 }}/{{ row.concurrency }}
            </span>
          </template>

          <template #cell-userRpm="{ row }">
            <span class="font-mono text-gray-700 dark:text-gray-300">{{ row.rpm_limit ?? 0 }}</span>
          </template>

          <template #cell-groupRpm="{ row }">
            <div class="space-y-0.5">
              <span class="font-mono text-gray-900 dark:text-white">{{ effectiveGroupRPM(row.id) }}</span>
              <div class="text-xs text-gray-500 dark:text-dark-400">
                {{ groupRPMSourceLabel(row.id) }}
              </div>
            </div>
          </template>

          <template #cell-status="{ row }">
            <span :class="statusBadgeClass(row.status)">
              {{ row.status === 'active' ? t('common.active') : t('admin.users.disabled') }}
            </span>
          </template>

          <template #empty>
            <div class="flex flex-col items-center">
              <Icon name="inbox" size="xl" class="mb-4 h-12 w-12 text-gray-400 dark:text-dark-500" />
              <p class="text-lg font-medium text-gray-900 dark:text-gray-100">
                {{ t('admin.limitCenter.emptyUsers') }}
              </p>
            </div>
          </template>
        </DataTable>

        <DataTable
          v-else
          :columns="inspectColumns"
          :data="inspectRows"
          :loading="inspectLoading || rpmStatusLoading"
          row-key="rowKey"
          :sticky-actions-column="false"
          :estimate-row-height="64"
        >
          <template #cell-user="{ row }">
            <button
              v-if="row.kind === 'user'"
              type="button"
              class="text-left"
              @click="selectInspectUser(row.user)"
            >
              <span class="block font-medium text-gray-900 hover:text-primary-600 dark:text-white dark:hover:text-primary-400">
                {{ row.user.email }}
              </span>
              <span class="mt-0.5 block text-xs text-gray-500 dark:text-dark-400">
                #{{ row.user.id }} · {{ row.user.username || '-' }}
              </span>
            </button>
            <div v-else-if="inspectedUser">
              <span class="block font-medium text-gray-900 dark:text-white">{{ inspectedUser.email }}</span>
              <span class="mt-0.5 block text-xs text-gray-500 dark:text-dark-400">
                {{ row.groupName || `#${row.groupID}` }}
              </span>
            </div>
          </template>

          <template #cell-concurrency="{ row }">
            <span v-if="row.kind === 'user'" class="font-mono">
              {{ row.user.current_concurrency ?? 0 }}/{{ row.user.concurrency }}
            </span>
            <span v-else class="text-gray-400">-</span>
          </template>

          <template #cell-userRpm="{ row }">
            <span v-if="row.kind === 'user'" class="font-mono">
              {{ row.user.rpm_limit ?? 0 }}
            </span>
            <span v-else class="font-mono">
              {{ rpmStatus?.user_rpm_used ?? 0 }}/{{ rpmStatus?.user_rpm_limit ?? 0 }}
            </span>
          </template>

          <template #cell-groupRpm="{ row }">
            <span v-if="row.kind === 'group'" class="font-mono">{{ row.limit }}</span>
            <span v-else class="text-gray-400">-</span>
          </template>

          <template #cell-source="{ row }">
            <span v-if="row.kind === 'group'">{{ rpmSourceText(row.source) }}</span>
            <span v-else-if="inspectedUser?.id === row.user.id" class="badge badge-primary">
              {{ t('admin.limitCenter.inspect.results') }}
            </span>
            <span v-else class="text-gray-400">-</span>
          </template>

          <template #cell-actions="{ row }">
            <button
              v-if="row.kind === 'user'"
              type="button"
              class="btn btn-secondary btn-sm"
              @click="selectInspectUser(row.user)"
            >
              {{ t('admin.limitCenter.inspect.pickUser') }}
            </button>
            <span v-else class="text-xs text-gray-500 dark:text-dark-400">
              {{ row.used }} / {{ row.limit }}
            </span>
          </template>

          <template #empty>
            <div class="flex flex-col items-center">
              <Icon name="inbox" size="xl" class="mb-4 h-12 w-12 text-gray-400 dark:text-dark-500" />
              <p class="text-lg font-medium text-gray-900 dark:text-gray-100">
                {{ inspectUserSearch.trim() ? t('admin.limitCenter.inspect.empty') : t('admin.limitCenter.inspect.pickUser') }}
              </p>
            </div>
          </template>
        </DataTable>
      </template>

      <template v-if="activeTab === 'group'" #pagination>
        <Pagination
          :total="usersPagination.total"
          :page="usersPagination.page"
          :page-size="usersPagination.pageSize"
          @update:page="handleUserPageChange"
          @update:pageSize="handleUserPageSizeChange"
        />
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { GroupRPMOverrideEntry } from '@/api/admin/groups'
import type { UserRPMStatus } from '@/api/admin/users'
import type { AdminGroup, AdminUser } from '@/types'
import { useAppStore } from '@/stores/app'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'

interface CapacityRow {
  group_id: number
  concurrency_used: number
  concurrency_max: number
  sessions_used: number
  sessions_max: number
  rpm_used: number
  rpm_max: number
}

type ActiveTab = 'group' | 'user'
type ConcurrencyMode = 'set' | 'add'

type InspectRow =
  | { rowKey: string; kind: 'user'; user: AdminUser }
  | {
      rowKey: string
      kind: 'group'
      groupID: number
      groupName?: string
      used: number
      limit: number
      source: string
    }

const { t } = useI18n()
const appStore = useAppStore()

const activeTab = ref<ActiveTab>('group')
const groups = ref<AdminGroup[]>([])
const capacities = ref<CapacityRow[]>([])
const selectedGroupID = ref<number | null>(null)
const groupUsers = ref<AdminUser[]>([])
const usersLoading = ref(false)
const rpmLoading = ref(false)
const bulkSaving = ref(false)
const userSearch = ref('')
const selectedUsers = ref<Map<number, AdminUser>>(new Map())
const rpmEntries = ref<GroupRPMOverrideEntry[]>([])

const usersPagination = ref({ page: 1, pageSize: 20, total: 0 })
const concurrencyMode = ref<ConcurrencyMode>('set')
const concurrencyValue = ref(1)
const rpmOverrideValue = ref(0)

const inspectUserSearch = ref('')
const inspectUsers = ref<AdminUser[]>([])
const inspectedUser = ref<AdminUser | null>(null)
const inspectLoading = ref(false)
const rpmStatus = ref<UserRPMStatus | null>(null)
const rpmStatusLoading = ref(false)
const pageSelectCheckboxRef = ref<HTMLInputElement | null>(null)

const groupUserColumns = computed<Column[]>(() => [
  { key: 'select', label: '', class: 'w-12' },
  { key: 'user', label: t('admin.limitCenter.columns.user'), sortable: true },
  { key: 'concurrency', label: t('admin.limitCenter.columns.concurrency') },
  { key: 'userRpm', label: t('admin.limitCenter.columns.userRpm') },
  { key: 'groupRpm', label: t('admin.limitCenter.columns.groupRpm') },
  { key: 'status', label: t('admin.limitCenter.columns.status') }
])

const inspectColumns = computed<Column[]>(() => [
  { key: 'user', label: t('admin.limitCenter.columns.user') },
  { key: 'concurrency', label: t('admin.limitCenter.columns.concurrency') },
  { key: 'userRpm', label: t('admin.limitCenter.columns.userRpm') },
  { key: 'groupRpm', label: t('admin.limitCenter.columns.groupRpm') },
  { key: 'source', label: t('admin.limitCenter.columns.source') },
  { key: 'actions', label: t('admin.limitCenter.columns.status') }
])

const concurrencyModeOptions = computed(() => [
  { value: 'set', label: t('admin.limitCenter.bulkPanel.set') },
  { value: 'add', label: t('admin.limitCenter.bulkPanel.add') }
])

const groupOptions = computed(() => groups.value.map(group => ({
  value: group.id,
  label: `${group.name} · ${t('admin.groups.platforms.' + group.platform)}`
})))

const selectedGroup = computed(() => groups.value.find(group => group.id === selectedGroupID.value) || null)
const selectedCapacity = computed(() => capacities.value.find(row => row.group_id === selectedGroupID.value) || null)
const selectedUserIDs = computed(() => Array.from(selectedUsers.value.keys()))
const selectedUserIDSet = computed(() => new Set(selectedUserIDs.value))
const pageAllSelected = computed(() => groupUsers.value.length > 0 && groupUsers.value.every(user => selectedUserIDSet.value.has(user.id)))
const pagePartiallySelected = computed(() => groupUsers.value.some(user => selectedUserIDSet.value.has(user.id)) && !pageAllSelected.value)

const inspectRows = computed<InspectRow[]>(() => {
  if (inspectedUser.value && rpmStatus.value?.per_group?.length) {
    return rpmStatus.value.per_group.map(row => ({
      rowKey: `group-${row.group_id}`,
      kind: 'group',
      groupID: row.group_id,
      groupName: row.group_name,
      used: row.used,
      limit: row.limit,
      source: row.source
    }))
  }
  return inspectUsers.value.map(user => ({
    rowKey: `user-${user.id}`,
    kind: 'user',
    user
  }))
})

const rpmMap = computed(() => {
  const map = new Map<number, number>()
  for (const entry of rpmEntries.value) {
    map.set(entry.user_id, entry.rpm_override)
  }
  return map
})

const statusBadgeClass = (status: string) => [
  'inline-flex rounded-full px-2 py-0.5 text-xs font-medium',
  status === 'active'
    ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
    : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-400'
]

const loadGroups = async () => {
  try {
    const [groupList, capacityList] = await Promise.all([
      adminAPI.groups.getAllIncludingInactive(),
      adminAPI.groups.getCapacitySummary()
    ])
    groups.value = groupList
    capacities.value = capacityList
    if (!selectedGroupID.value && groupList.length > 0) {
      selectedGroupID.value = groupList[0].id
    }
  } catch (error: any) {
    appStore.showError(error?.response?.data?.detail || t('admin.limitCenter.errors.loadGroups'))
  }
}

const loadRPMEntries = async () => {
  if (!selectedGroupID.value) return
  rpmLoading.value = true
  try {
    rpmEntries.value = await adminAPI.groups.getGroupRPMOverrides(selectedGroupID.value)
  } catch (error: any) {
    appStore.showError(error?.response?.data?.detail || t('admin.limitCenter.errors.loadRpm'))
  } finally {
    rpmLoading.value = false
  }
}

const loadGroupUsers = async () => {
  if (!selectedGroupID.value) return
  usersLoading.value = true
  try {
    const result = await adminAPI.users.list(
      usersPagination.value.page,
      usersPagination.value.pageSize,
      {
        allowed_group_id: selectedGroupID.value,
        search: userSearch.value.trim() || undefined,
        sort_by: 'id',
        sort_order: 'asc'
      }
    )
    groupUsers.value = result.items
    usersPagination.value.total = result.total
    usersPagination.value.page = result.page
    usersPagination.value.pageSize = result.page_size
  } catch (error: any) {
    appStore.showError(error?.response?.data?.detail || t('admin.limitCenter.errors.loadUsers'))
  } finally {
    usersLoading.value = false
  }
}

const handleGroupChange = async () => {
  usersPagination.value.page = 1
  selectedUsers.value = new Map()
  await Promise.all([loadRPMEntries(), loadGroupUsers()])
}

const refreshGroupView = async () => {
  await Promise.all([loadGroups(), loadRPMEntries(), loadGroupUsers()])
}

const resetGroupUserSearch = () => {
  userSearch.value = ''
  usersPagination.value.page = 1
  loadGroupUsers()
}

const handleUserPageChange = (page: number) => {
  usersPagination.value.page = page
  loadGroupUsers()
}

const handleUserPageSizeChange = (pageSize: number) => {
  usersPagination.value.pageSize = pageSize
  usersPagination.value.page = 1
  loadGroupUsers()
}

const toggleUserSelection = (user: AdminUser, checked: boolean) => {
  const next = new Map(selectedUsers.value)
  if (checked) {
    next.set(user.id, user)
  } else {
    next.delete(user.id)
  }
  selectedUsers.value = next
}

const togglePageSelection = (checked: boolean) => {
  const next = new Map(selectedUsers.value)
  for (const user of groupUsers.value) {
    if (checked) {
      next.set(user.id, user)
    } else {
      next.delete(user.id)
    }
  }
  selectedUsers.value = next
}

const clearSelection = () => {
  selectedUsers.value = new Map()
}

const effectiveGroupRPM = (userID: number) => {
  if (rpmMap.value.has(userID)) return rpmMap.value.get(userID) ?? 0
  return selectedGroup.value?.rpm_limit ?? 0
}

const groupRPMSourceLabel = (userID: number) => {
  return rpmMap.value.has(userID)
    ? t('admin.limitCenter.source.override')
    : t('admin.limitCenter.source.group')
}

const rpmSourceText = (source: string) => {
  if (source === 'override') return t('admin.limitCenter.source.override')
  if (source === 'group') return t('admin.limitCenter.source.group')
  return t('admin.limitCenter.source.none')
}

const buildMergedRPMEntries = (updates: Map<number, number | null>) => {
  const userInfo = new Map<number, AdminUser>()
  for (const user of selectedUsers.value.values()) {
    userInfo.set(user.id, user)
  }

  const merged = new Map<number, GroupRPMOverrideEntry>()
  for (const entry of rpmEntries.value) {
    merged.set(entry.user_id, { ...entry })
  }
  for (const [userID, value] of updates) {
    if (value == null) {
      merged.delete(userID)
      continue
    }
    const user = userInfo.get(userID)
    const existing = merged.get(userID)
    merged.set(userID, {
      user_id: userID,
      user_name: user?.username || existing?.user_name || '',
      user_email: user?.email || existing?.user_email || '',
      user_notes: user?.notes || existing?.user_notes || '',
      user_status: user?.status || existing?.user_status || 'active',
      rpm_override: value
    })
  }
  return Array.from(merged.values()).sort((a, b) => a.user_id - b.user_id)
}

const saveRPMEntries = async (entries: GroupRPMOverrideEntry[]) => {
  if (!selectedGroupID.value) return
  await adminAPI.groups.batchSetGroupRPMOverrides(
    selectedGroupID.value,
    entries.map(entry => ({ user_id: entry.user_id, rpm_override: entry.rpm_override }))
  )
  rpmEntries.value = entries
}

const applyConcurrency = async () => {
  if (selectedUserIDs.value.length === 0 || concurrencyValue.value < 1) return
  bulkSaving.value = true
  try {
    const result = await adminAPI.users.batchUpdateConcurrency({
      user_ids: selectedUserIDs.value,
      concurrency: concurrencyValue.value,
      mode: concurrencyMode.value
    })
    appStore.showSuccess(t('admin.limitCenter.messages.concurrencySaved', { count: result.affected }))
    await loadGroupUsers()
  } catch (error: any) {
    appStore.showError(error?.response?.data?.detail || t('admin.limitCenter.errors.saveConcurrency'))
  } finally {
    bulkSaving.value = false
  }
}

const applyRPMOverride = async () => {
  if (selectedUserIDs.value.length === 0 || rpmOverrideValue.value < 0) return
  bulkSaving.value = true
  try {
    const updates = new Map<number, number | null>()
    for (const userID of selectedUserIDs.value) updates.set(userID, rpmOverrideValue.value)
    const merged = buildMergedRPMEntries(updates)
    await saveRPMEntries(merged)
    appStore.showSuccess(t('admin.limitCenter.messages.rpmSaved'))
  } catch (error: any) {
    appStore.showError(error?.response?.data?.detail || t('admin.limitCenter.errors.saveRpm'))
  } finally {
    bulkSaving.value = false
  }
}

const clearSelectedRPMOverride = async () => {
  if (selectedUserIDs.value.length === 0) return
  bulkSaving.value = true
  try {
    const updates = new Map<number, number | null>()
    for (const userID of selectedUserIDs.value) updates.set(userID, null)
    const merged = buildMergedRPMEntries(updates)
    await saveRPMEntries(merged)
    appStore.showSuccess(t('admin.limitCenter.messages.rpmCleared'))
  } catch (error: any) {
    appStore.showError(error?.response?.data?.detail || t('admin.limitCenter.errors.saveRpm'))
  } finally {
    bulkSaving.value = false
  }
}

const searchInspectUsers = async () => {
  const query = inspectUserSearch.value.trim()
  if (!query) return
  inspectLoading.value = true
  inspectedUser.value = null
  rpmStatus.value = null
  try {
    const result = await adminAPI.users.list(1, 20, { search: query, sort_by: 'id', sort_order: 'asc' })
    inspectUsers.value = result.items
  } catch (error: any) {
    appStore.showError(error?.response?.data?.detail || t('admin.limitCenter.errors.loadUsers'))
  } finally {
    inspectLoading.value = false
  }
}

const clearInspectSearch = () => {
  inspectUserSearch.value = ''
  inspectUsers.value = []
  inspectedUser.value = null
  rpmStatus.value = null
}

const selectInspectUser = async (user: AdminUser) => {
  inspectedUser.value = user
  rpmStatus.value = null
  rpmStatusLoading.value = true
  try {
    rpmStatus.value = await adminAPI.users.getRPMStatus(user.id)
  } catch (error: any) {
    appStore.showError(error?.response?.data?.detail || t('admin.limitCenter.errors.loadRpmStatus'))
  } finally {
    rpmStatusLoading.value = false
  }
}

watch(selectedGroupID, (next, prev) => {
  if (next && next !== prev) {
    void handleGroupChange()
  }
})

watch([pagePartiallySelected, pageAllSelected], ([partiallySelected]) => {
  if (pageSelectCheckboxRef.value) {
    pageSelectCheckboxRef.value.indeterminate = partiallySelected
  }
}, { immediate: true })

onMounted(async () => {
  await loadGroups()
})
</script>
