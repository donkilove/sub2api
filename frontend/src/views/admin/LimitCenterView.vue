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
        <div class="flex flex-col justify-between gap-4 lg:flex-row lg:items-center">
          <div class="relative w-full lg:max-w-md">
            <Icon
              name="search"
              size="md"
              class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
            />
            <input
              v-model="search"
              type="text"
              class="input pl-10"
              :placeholder="activeTab === 'group'
                ? t('admin.limitCenter.searchGroups')
                : t('admin.limitCenter.searchUsers')"
              @keyup.enter="reloadFromFirstPage"
            />
          </div>

          <div class="flex w-full flex-shrink-0 flex-wrap justify-end gap-2 lg:w-auto">
            <button
              type="button"
              class="btn btn-secondary px-2 md:px-3"
              :title="t('common.refresh')"
              :disabled="activeLoading"
              @click="reloadCurrent"
            >
              <Icon name="refresh" size="md" :class="activeLoading ? 'animate-spin' : ''" />
            </button>
            <button type="button" class="btn btn-secondary" :disabled="activeLoading" @click="resetSearch">
              {{ t('common.reset') }}
            </button>
            <button type="button" class="btn btn-primary" :disabled="activeLoading" @click="reloadFromFirstPage">
              <Icon name="search" size="sm" class="mr-2" />
              {{ t('common.search') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          v-if="activeTab === 'group'"
          :columns="groupColumns"
          :data="groups"
          :loading="groupsLoading"
          row-key="id"
          :sticky-actions-column="false"
          :estimate-row-height="72"
        >
          <template #cell-group="{ row }">
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="font-medium text-gray-900 dark:text-white">{{ row.name }}</span>
                <span class="text-xs text-gray-400">#{{ row.id }}</span>
              </div>
              <div class="mt-0.5 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
                <span>{{ t(`admin.groups.platforms.${row.platform}`) }}</span>
                <span>·</span>
                <span>{{ row.description || '-' }}</span>
              </div>
            </div>
          </template>

          <template #cell-concurrency="{ row }">
            <LimitNumberInput
              :model-value="groupDraft(row.id).user_concurrency_limit"
              @update:model-value="value => groupDraft(row.id).user_concurrency_limit = value"
            />
          </template>

          <template #cell-rpm="{ row }">
            <LimitNumberInput
              :model-value="groupDraft(row.id).rpm_limit"
              @update:model-value="value => groupDraft(row.id).rpm_limit = value"
            />
          </template>

          <template #cell-status="{ row }">
            <span :class="statusBadgeClass(row.status)">
              {{ row.status === 'active' ? t('common.active') : t('common.inactive') }}
            </span>
          </template>

          <template #cell-actions="{ row }">
            <button
              type="button"
              class="btn btn-primary btn-sm"
              :disabled="savingGroupID === row.id || !groupDirty(row)"
              @click="saveGroup(row)"
            >
              <Icon
                :name="savingGroupID === row.id ? 'refresh' : 'check'"
                size="sm"
                class="mr-1"
                :class="savingGroupID === row.id ? 'animate-spin' : ''"
              />
              {{ t('common.save') }}
            </button>
          </template>

          <template #empty>
            <EmptyState :message="t('admin.limitCenter.emptyGroups')" />
          </template>
        </DataTable>

        <DataTable
          v-else
          :columns="userColumns"
          :data="users"
          :loading="usersLoading"
          row-key="id"
          :sticky-actions-column="false"
          :estimate-row-height="72"
        >
          <template #cell-user="{ row }">
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="font-medium text-gray-900 dark:text-white">{{ row.email }}</span>
                <span class="text-xs text-gray-400">#{{ row.id }}</span>
              </div>
              <div class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                {{ row.username || row.notes || '-' }}
              </div>
            </div>
          </template>

          <template #cell-concurrency="{ row }">
            <OverrideNumberInput
              :model-value="userDraft(row.id).user_concurrency_override"
              @update:model-value="value => userDraft(row.id).user_concurrency_override = value"
            />
          </template>

          <template #cell-rpm="{ row }">
            <OverrideNumberInput
              :model-value="userDraft(row.id).user_rpm_limit_override"
              @update:model-value="value => userDraft(row.id).user_rpm_limit_override = value"
            />
          </template>

          <template #cell-source="{ row }">
            <div class="flex flex-wrap gap-1.5">
              <span :class="sourceBadgeClass(row.user_concurrency_override)">
                {{ t('admin.limitCenter.source.concurrency', { source: sourceLabel(row.user_concurrency_override) }) }}
              </span>
              <span :class="sourceBadgeClass(row.user_rpm_limit_override)">
                {{ t('admin.limitCenter.source.rpm', { source: sourceLabel(row.user_rpm_limit_override) }) }}
              </span>
            </div>
          </template>

          <template #cell-status="{ row }">
            <span :class="statusBadgeClass(row.status)">
              {{ row.status === 'active' ? t('common.active') : t('admin.users.disabled') }}
            </span>
          </template>

          <template #cell-actions="{ row }">
            <button
              type="button"
              class="btn btn-primary btn-sm"
              :disabled="savingUserID === row.id || !userDirty(row)"
              @click="saveUser(row)"
            >
              <Icon
                :name="savingUserID === row.id ? 'refresh' : 'check'"
                size="sm"
                class="mr-1"
                :class="savingUserID === row.id ? 'animate-spin' : ''"
              />
              {{ t('common.save') }}
            </button>
          </template>

          <template #empty>
            <EmptyState :message="t('admin.limitCenter.emptyUsers')" />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          :total="activePagination.total"
          :page="activePagination.page"
          :page-size="activePagination.pageSize"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref, watch, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { AdminGroup, AdminUser } from '@/types'
import { useAppStore } from '@/stores/app'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'

type ActiveTab = 'group' | 'user'
type NullableLimit = number | null

interface GroupDraft {
  user_concurrency_limit: number
  rpm_limit: number
}

interface UserDraft {
  user_concurrency_override: NullableLimit
  user_rpm_limit_override: NullableLimit
}

const { t } = useI18n()
const appStore = useAppStore()

const activeTab = ref<ActiveTab>('group')
const search = ref('')
const groups = ref<AdminGroup[]>([])
const users = ref<AdminUser[]>([])
const groupsLoading = ref(false)
const usersLoading = ref(false)
const savingGroupID = ref<number | null>(null)
const savingUserID = ref<number | null>(null)
const groupDrafts = reactive<Record<number, GroupDraft>>({})
const userDrafts = reactive<Record<number, UserDraft>>({})

const groupPagination = reactive({ page: 1, pageSize: 20, total: 0 })
const userPagination = reactive({ page: 1, pageSize: 20, total: 0 })

const groupColumns = computed<Column[]>(() => [
  { key: 'group', label: t('admin.limitCenter.columns.group') },
  { key: 'concurrency', label: t('admin.limitCenter.columns.groupConcurrency') },
  { key: 'rpm', label: t('admin.limitCenter.columns.groupRpm') },
  { key: 'status', label: t('admin.limitCenter.columns.status') },
  { key: 'actions', label: t('common.actions') }
])

const userColumns = computed<Column[]>(() => [
  { key: 'user', label: t('admin.limitCenter.columns.user') },
  { key: 'concurrency', label: t('admin.limitCenter.columns.userConcurrency') },
  { key: 'rpm', label: t('admin.limitCenter.columns.userRpm') },
  { key: 'source', label: t('admin.limitCenter.columns.source') },
  { key: 'status', label: t('admin.limitCenter.columns.status') },
  { key: 'actions', label: t('common.actions') }
])

const activeLoading = computed(() => activeTab.value === 'group' ? groupsLoading.value : usersLoading.value)
const activePagination = computed(() => activeTab.value === 'group' ? groupPagination : userPagination)

const clampLimit = (value: number) => {
  if (!Number.isFinite(value) || value < 0) return 0
  return Math.floor(value)
}

const parseNullableLimit = (value: unknown): NullableLimit => {
  if (value === null || value === undefined || value === '') return null
  const numberValue = Number(value)
  if (!Number.isFinite(numberValue) || numberValue < 0) return null
  return Math.floor(numberValue)
}

const limitEquals = (a: NullableLimit | undefined, b: NullableLimit | undefined) => {
  return (a ?? null) === (b ?? null)
}

const syncGroupDraft = (group: AdminGroup) => {
  groupDrafts[group.id] = {
    user_concurrency_limit: clampLimit(group.user_concurrency_limit ?? 0),
    rpm_limit: clampLimit(group.rpm_limit ?? 0)
  }
}

const syncUserDraft = (user: AdminUser) => {
  userDrafts[user.id] = {
    user_concurrency_override: parseNullableLimit(user.user_concurrency_override),
    user_rpm_limit_override: parseNullableLimit(user.user_rpm_limit_override)
  }
}

const groupDraft = (id: number) => {
  if (!groupDrafts[id]) {
    groupDrafts[id] = { user_concurrency_limit: 0, rpm_limit: 0 }
  }
  return groupDrafts[id]
}

const userDraft = (id: number) => {
  if (!userDrafts[id]) {
    userDrafts[id] = { user_concurrency_override: null, user_rpm_limit_override: null }
  }
  return userDrafts[id]
}

const groupDirty = (group: AdminGroup) => {
  const draft = groupDraft(group.id)
  return draft.user_concurrency_limit !== clampLimit(group.user_concurrency_limit ?? 0) ||
    draft.rpm_limit !== clampLimit(group.rpm_limit ?? 0)
}

const userDirty = (user: AdminUser) => {
  const draft = userDraft(user.id)
  return !limitEquals(draft.user_concurrency_override, parseNullableLimit(user.user_concurrency_override)) ||
    !limitEquals(draft.user_rpm_limit_override, parseNullableLimit(user.user_rpm_limit_override))
}

const statusBadgeClass = (status: string) => [
  'inline-flex rounded-full px-2 py-0.5 text-xs font-medium',
  status === 'active'
    ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
    : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-400'
]

const sourceBadgeClass = (value: NullableLimit | undefined) => [
  'inline-flex rounded-full px-2 py-0.5 text-xs font-medium',
  value === null || value === undefined
    ? 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-400'
    : 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
]

const sourceLabel = (value: NullableLimit | undefined) => {
  if (value === null || value === undefined) return t('admin.limitCenter.source.group')
  if (value === 0) return t('admin.limitCenter.source.unlimited')
  return t('admin.limitCenter.source.user')
}

const loadGroups = async () => {
  groupsLoading.value = true
  try {
    const result = await adminAPI.groups.list(
      groupPagination.page,
      groupPagination.pageSize,
      {
        search: search.value.trim() || undefined,
        sort_by: 'id',
        sort_order: 'asc'
      }
    )
    groups.value = result.items
    groupPagination.total = result.total
    groupPagination.page = result.page
    groupPagination.pageSize = result.page_size
    for (const group of result.items) {
      syncGroupDraft(group)
    }
  } catch (error: any) {
    appStore.showError(error?.response?.data?.detail || t('admin.limitCenter.errors.loadGroups'))
  } finally {
    groupsLoading.value = false
  }
}

const loadUsers = async () => {
  usersLoading.value = true
  try {
    const result = await adminAPI.users.list(
      userPagination.page,
      userPagination.pageSize,
      {
        search: search.value.trim() || undefined,
        sort_by: 'id',
        sort_order: 'asc'
      }
    )
    users.value = result.items
    userPagination.total = result.total
    userPagination.page = result.page
    userPagination.pageSize = result.page_size
    for (const user of result.items) {
      syncUserDraft(user)
    }
  } catch (error: any) {
    appStore.showError(error?.response?.data?.detail || t('admin.limitCenter.errors.loadUsers'))
  } finally {
    usersLoading.value = false
  }
}

const reloadCurrent = () => {
  if (activeTab.value === 'group') {
    void loadGroups()
    return
  }
  void loadUsers()
}

const reloadFromFirstPage = () => {
  activePagination.value.page = 1
  reloadCurrent()
}

const resetSearch = () => {
  search.value = ''
  reloadFromFirstPage()
}

const handlePageChange = (page: number) => {
  activePagination.value.page = page
  reloadCurrent()
}

const handlePageSizeChange = (pageSize: number) => {
  activePagination.value.pageSize = pageSize
  activePagination.value.page = 1
  reloadCurrent()
}

const saveGroup = async (group: AdminGroup) => {
  const draft = groupDraft(group.id)
  savingGroupID.value = group.id
  try {
    const updated = await adminAPI.groups.update(group.id, {
      user_concurrency_limit: clampLimit(draft.user_concurrency_limit),
      rpm_limit: clampLimit(draft.rpm_limit)
    })
    const index = groups.value.findIndex(item => item.id === group.id)
    if (index >= 0) {
      groups.value[index] = updated
    }
    syncGroupDraft(updated)
    appStore.showSuccess(t('admin.limitCenter.messages.groupSaved'))
  } catch (error: any) {
    appStore.showError(error?.response?.data?.detail || t('admin.limitCenter.errors.saveGroup'))
  } finally {
    savingGroupID.value = null
  }
}

const saveUser = async (user: AdminUser) => {
  const draft = userDraft(user.id)
  savingUserID.value = user.id
  try {
    const updated = await adminAPI.users.update(user.id, {
      user_concurrency_override: draft.user_concurrency_override,
      user_rpm_limit_override: draft.user_rpm_limit_override
    })
    const index = users.value.findIndex(item => item.id === user.id)
    if (index >= 0) {
      users.value[index] = updated
    }
    syncUserDraft(updated)
    appStore.showSuccess(t('admin.limitCenter.messages.userSaved'))
  } catch (error: any) {
    appStore.showError(error?.response?.data?.detail || t('admin.limitCenter.errors.saveUser'))
  } finally {
    savingUserID.value = null
  }
}

watch(activeTab, () => {
  search.value = ''
  reloadCurrent()
})

onMounted(() => {
  void loadGroups()
})

const numberInputClasses = 'hide-spinner input w-28 font-mono text-sm'

const LimitNumberInput = defineComponent({
  name: 'LimitNumberInput',
  props: {
    modelValue: {
      type: Number,
      required: true
    }
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () => h('input', {
      class: numberInputClasses,
      type: 'number',
      min: 0,
      step: 1,
      value: props.modelValue,
      onInput: (event: Event) => {
        const target = event.target as HTMLInputElement
        emit('update:modelValue', clampLimit(Number(target.value)))
      }
    })
  }
})

const OverrideNumberInput = defineComponent({
  name: 'OverrideNumberInput',
  props: {
    modelValue: {
      type: Number as PropType<NullableLimit>,
      default: null
    }
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () => h('input', {
      class: numberInputClasses,
      type: 'number',
      min: 0,
      step: 1,
      placeholder: t('admin.limitCenter.inheritPlaceholder'),
      value: props.modelValue ?? '',
      onInput: (event: Event) => {
        const target = event.target as HTMLInputElement
        emit('update:modelValue', parseNullableLimit(target.value))
      }
    })
  }
})

const EmptyState = defineComponent({
  name: 'LimitCenterEmptyState',
  props: {
    message: {
      type: String,
      required: true
    }
  },
  setup(props) {
    return () => h('div', { class: 'flex flex-col items-center' }, [
      h(Icon, {
        name: 'inbox',
        size: 'xl',
        class: 'mb-4 h-12 w-12 text-gray-400 dark:text-dark-500'
      }),
      h('p', { class: 'text-lg font-medium text-gray-900 dark:text-gray-100' }, props.message)
    ])
  }
})
</script>
