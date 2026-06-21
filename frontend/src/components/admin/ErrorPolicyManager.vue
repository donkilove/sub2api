<template>
  <div class="space-y-4">
    <!-- Header -->
    <div v-if="showDescription || activeTab === 'rules'" class="flex items-center justify-between">
      <p v-if="showDescription" class="text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.errorPassthrough.description') }}
      </p>
      <button v-if="activeTab === 'rules'" @click="showCreateModal = true" class="btn btn-primary btn-sm">
        <Icon name="plus" size="sm" class="mr-1" />
        {{ t('admin.errorPassthrough.createRule') }}
      </button>
    </div>

      <div class="inline-flex rounded-lg bg-gray-100 p-1 dark:bg-dark-700">
        <button
          type="button"
          :class="[
            'rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
            activeTab === 'policies'
              ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-600 dark:text-white'
              : 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'
          ]"
          @click="activeTab = 'policies'"
        >
          {{ t('admin.errorPassthrough.tabs.policies') }}
        </button>
        <button
          type="button"
          :class="[
            'rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
            activeTab === 'rules'
              ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-600 dark:text-white'
              : 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'
          ]"
          @click="activeTab = 'rules'"
        >
          {{ t('admin.errorPassthrough.tabs.rules') }}
        </button>
      </div>

      <div v-if="activeTab === 'policies'" class="space-y-4">
        <div v-if="policiesLoading" class="flex items-center justify-center py-8">
          <Icon name="refresh" size="lg" class="animate-spin text-gray-400" />
        </div>

        <div v-else class="min-h-[24rem] max-h-[calc(100vh-20rem)] overflow-auto rounded-lg border border-gray-200 dark:border-dark-600">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="sticky top-0 bg-gray-50 dark:bg-dark-700">
              <tr>
                <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                  {{ t('admin.errorPassthrough.policyColumns.category') }}
                </th>
                <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                  {{ t('admin.errorPassthrough.policyColumns.defaultResponse') }}
                </th>
                <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                  {{ t('admin.errorPassthrough.policyColumns.effectiveResponse') }}
                </th>
                <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                  {{ t('admin.errorPassthrough.policyColumns.retry') }}
                </th>
                <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                  {{ t('admin.errorPassthrough.columns.actions') }}
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-800">
              <tr v-for="policy in policies" :key="policy.category" class="hover:bg-gray-50 dark:hover:bg-dark-700">
                <td class="px-3 py-3 align-top">
                  <div class="flex items-start gap-2">
                    <span
                      :class="[
                        'mt-0.5 inline-flex h-2 w-2 flex-shrink-0 rounded-full',
                        policy.custom_enabled || policy.retry_enabled ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-500'
                      ]"
                    />
                    <div>
                      <div class="text-sm font-medium text-gray-900 dark:text-white">
                        {{ policy.label }}
                      </div>
                      <div class="mt-0.5 font-mono text-xs text-gray-500 dark:text-gray-400">
                        {{ policy.category }}
                      </div>
                      <div class="mt-1 max-w-xs text-xs text-gray-500 dark:text-gray-400">
                        {{ policy.description }}
                      </div>
                    </div>
                  </div>
                </td>
                <td class="px-3 py-3 align-top">
                  <div class="space-y-1 text-xs">
                    <div>
                      <span class="badge badge-gray">{{ policy.default_status_code }}</span>
                    </div>
                    <div class="font-mono text-gray-700 dark:text-gray-300">{{ policy.default_error_type }}</div>
                    <div class="max-w-xs text-gray-500 dark:text-gray-400">{{ policy.default_message }}</div>
                  </div>
                </td>
                <td class="px-3 py-3 align-top">
                  <div class="space-y-1 text-xs">
                    <div class="flex flex-wrap items-center gap-1">
                      <span :class="policy.custom_enabled ? 'badge badge-primary' : 'badge badge-gray'">
                        {{ policy.effective_status_code }}
                      </span>
                      <span v-if="policy.custom_enabled" class="badge badge-success">
                        {{ t('admin.errorPassthrough.custom') }}
                      </span>
                    </div>
                    <div class="font-mono text-gray-700 dark:text-gray-300">{{ policy.effective_error_type }}</div>
                    <div class="max-w-xs text-gray-500 dark:text-gray-400">{{ policy.effective_message }}</div>
                  </div>
                </td>
                <td class="px-3 py-3 align-top">
                  <div v-if="policy.retry_enabled" class="space-y-1">
                    <span class="badge badge-warning">{{ t('admin.errorPassthrough.retryEnabled') }}</span>
                    <div class="text-xs text-gray-500 dark:text-gray-400">
                      {{ t('admin.errorPassthrough.maxRetries', { count: policy.max_retries }) }}
                    </div>
                  </div>
                  <div v-else-if="policy.default_retryable" class="text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.errorPassthrough.retryDisabled') }}
                  </div>
                  <div v-else class="text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.errorPassthrough.notRetryable') }}
                  </div>
                </td>
                <td class="px-3 py-3 align-top">
                  <button
                    type="button"
                    @click="handleEditPolicy(policy)"
                    class="p-1 text-gray-500 hover:text-primary-600 dark:hover:text-primary-400"
                    :title="t('common.edit')"
                  >
                    <Icon name="edit" size="sm" />
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div v-else class="space-y-4">
      <!-- Rules Table -->
      <div v-if="loading" class="flex items-center justify-center py-8">
        <Icon name="refresh" size="lg" class="animate-spin text-gray-400" />
      </div>

      <div v-else-if="rules.length === 0" class="py-8 text-center">
        <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700">
          <Icon name="shield" size="lg" class="text-gray-400" />
        </div>
        <h4 class="mb-1 text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.errorPassthrough.noRules') }}
        </h4>
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.errorPassthrough.createFirstRule') }}
        </p>
      </div>

      <div v-else class="min-h-[24rem] max-h-[calc(100vh-20rem)] overflow-auto rounded-lg border border-gray-200 dark:border-dark-600">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
          <thead class="sticky top-0 bg-gray-50 dark:bg-dark-700">
            <tr>
              <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.errorPassthrough.columns.priority') }}
              </th>
              <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.errorPassthrough.columns.name') }}
              </th>
              <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.errorPassthrough.columns.conditions') }}
              </th>
              <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.errorPassthrough.columns.platforms') }}
              </th>
              <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.errorPassthrough.columns.behavior') }}
              </th>
              <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.errorPassthrough.columns.status') }}
              </th>
              <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.errorPassthrough.columns.actions') }}
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-800">
            <tr v-for="rule in rules" :key="rule.id" class="hover:bg-gray-50 dark:hover:bg-dark-700">
              <td class="whitespace-nowrap px-3 py-2">
                <span class="inline-flex h-5 w-5 items-center justify-center rounded bg-gray-100 text-xs font-medium text-gray-700 dark:bg-dark-600 dark:text-gray-300">
                  {{ rule.priority }}
                </span>
              </td>
              <td class="px-3 py-2">
                <div class="font-medium text-gray-900 dark:text-white text-sm">{{ rule.name }}</div>
                <div v-if="rule.description" class="mt-0.5 text-xs text-gray-500 dark:text-gray-400 max-w-xs truncate">
                  {{ rule.description }}
                </div>
              </td>
              <td class="px-3 py-2">
                <div class="flex flex-wrap gap-1 max-w-48">
                  <span
                    v-for="code in rule.error_codes.slice(0, 3)"
                    :key="code"
                    class="badge badge-danger text-xs"
                  >
                    {{ code }}
                  </span>
                  <span
                    v-if="rule.error_codes.length > 3"
                    class="text-xs text-gray-500"
                  >
                    +{{ rule.error_codes.length - 3 }}
                  </span>
                  <span
                    v-for="keyword in rule.keywords.slice(0, 1)"
                    :key="keyword"
                    class="badge badge-gray text-xs"
                  >
                    "{{ keyword.length > 10 ? keyword.substring(0, 10) + '...' : keyword }}"
                  </span>
                  <span
                    v-if="rule.keywords.length > 1"
                    class="text-xs text-gray-500"
                  >
                    +{{ rule.keywords.length - 1 }}
                  </span>
                </div>
                <div class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.errorPassthrough.matchMode.' + rule.match_mode) }}
                </div>
              </td>
              <td class="px-3 py-2">
                <div v-if="rule.platforms.length === 0" class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.errorPassthrough.allPlatforms') }}
                </div>
                <div v-else class="flex flex-wrap gap-1">
                  <span
                    v-for="platform in rule.platforms.slice(0, 2)"
                    :key="platform"
                    class="badge badge-primary text-xs"
                  >
                    {{ platform }}
                  </span>
                  <span v-if="rule.platforms.length > 2" class="text-xs text-gray-500">
                    +{{ rule.platforms.length - 2 }}
                  </span>
                </div>
              </td>
              <td class="px-3 py-2">
                <div class="text-xs space-y-0.5">
                  <div class="flex items-center gap-1">
                    <Icon
                      :name="rule.passthrough_code ? 'checkCircle' : 'xCircle'"
                      size="xs"
                      :class="rule.passthrough_code ? 'text-green-500' : 'text-gray-400'"
                    />
                    <span class="text-gray-600 dark:text-gray-400">
                      {{ t('admin.errorPassthrough.code') }}:
                      {{ rule.passthrough_code ? t('admin.errorPassthrough.passthrough') : (rule.response_code || '-') }}
                    </span>
                  </div>
                  <div class="flex items-center gap-1">
                    <Icon
                      :name="rule.passthrough_body ? 'checkCircle' : 'xCircle'"
                      size="xs"
                      :class="rule.passthrough_body ? 'text-green-500' : 'text-gray-400'"
                    />
                    <span class="text-gray-600 dark:text-gray-400">
                      {{ t('admin.errorPassthrough.body') }}:
                      {{ rule.passthrough_body ? t('admin.errorPassthrough.passthrough') : t('admin.errorPassthrough.custom') }}
                    </span>
                  </div>
                  <div v-if="rule.skip_monitoring" class="flex items-center gap-1">
                    <Icon
                      name="checkCircle"
                      size="xs"
                      class="text-yellow-500"
                    />
                    <span class="text-gray-600 dark:text-gray-400">
                      {{ t('admin.errorPassthrough.skipMonitoring') }}
                    </span>
                  </div>
                </div>
              </td>
              <td class="px-3 py-2">
                <button
                  @click="toggleEnabled(rule)"
                  :class="[
                    'relative inline-flex h-4 w-7 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
                    rule.enabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
                  ]"
                >
                  <span
                    :class="[
                      'pointer-events-none inline-block h-3 w-3 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                      rule.enabled ? 'translate-x-3' : 'translate-x-0'
                    ]"
                  />
                </button>
              </td>
              <td class="px-3 py-2">
                <div class="flex items-center gap-1">
                  <button
                    @click="handleEdit(rule)"
                    class="p-1 text-gray-500 hover:text-primary-600 dark:hover:text-primary-400"
                    :title="t('common.edit')"
                  >
                    <Icon name="edit" size="sm" />
                  </button>
                  <button
                    @click="handleDelete(rule)"
                    class="p-1 text-gray-500 hover:text-red-600 dark:hover:text-red-400"
                    :title="t('common.delete')"
                  >
                    <Icon name="trash" size="sm" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      </div>
  </div>

    <!-- Create/Edit Modal -->
    <BaseDialog
      :show="showCreateModal || showEditModal"
      :title="showEditModal ? t('admin.errorPassthrough.editRule') : t('admin.errorPassthrough.createRule')"
      width="wide"
      :z-index="60"
      @close="closeFormModal"
    >
      <form @submit.prevent="handleSubmit" class="space-y-4">
        <!-- Basic Info -->
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="input-label">{{ t('admin.errorPassthrough.form.name') }}</label>
            <input
              v-model="form.name"
              type="text"
              required
              class="input"
              :placeholder="t('admin.errorPassthrough.form.namePlaceholder')"
            />
          </div>
          <div>
            <label class="input-label">{{ t('admin.errorPassthrough.form.priority') }}</label>
            <input
              v-model.number="form.priority"
              type="number"
              min="0"
              class="input"
            />
            <p class="input-hint">{{ t('admin.errorPassthrough.form.priorityHint') }}</p>
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('admin.errorPassthrough.form.description') }}</label>
          <input
            v-model="form.description"
            type="text"
            class="input"
            :placeholder="t('admin.errorPassthrough.form.descriptionPlaceholder')"
          />
        </div>

        <!-- Match Conditions -->
        <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <h4 class="mb-2 text-sm font-medium text-gray-900 dark:text-white">
            {{ t('admin.errorPassthrough.form.matchConditions') }}
          </h4>

          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="input-label text-xs">{{ t('admin.errorPassthrough.form.errorCodes') }}</label>
              <input
                v-model="errorCodesInput"
                type="text"
                class="input text-sm"
                :placeholder="t('admin.errorPassthrough.form.errorCodesPlaceholder')"
              />
              <p class="input-hint text-xs">{{ t('admin.errorPassthrough.form.errorCodesHint') }}</p>
            </div>
            <div>
              <label class="input-label text-xs">{{ t('admin.errorPassthrough.form.keywords') }}</label>
              <textarea
                v-model="keywordsInput"
                rows="2"
                class="input font-mono text-xs"
                :placeholder="t('admin.errorPassthrough.form.keywordsPlaceholder')"
              />
              <p class="input-hint text-xs">{{ t('admin.errorPassthrough.form.keywordsHint') }}</p>
            </div>
          </div>

          <div class="mt-3">
            <label class="input-label text-xs">{{ t('admin.errorPassthrough.form.matchMode') }}</label>
            <div class="mt-1 space-y-2">
              <label
                v-for="option in matchModeOptions"
                :key="option.value"
                class="flex items-start gap-2 cursor-pointer"
              >
                <input
                  type="radio"
                  :value="option.value"
                  v-model="form.match_mode"
                  class="mt-0.5 h-3.5 w-3.5 border-gray-300 text-primary-600 focus:ring-primary-500"
                />
                <div class="flex-1">
                  <span class="text-xs font-medium text-gray-700 dark:text-gray-300">{{ option.label }}</span>
                  <p class="text-xs text-gray-500 dark:text-gray-400">{{ option.description }}</p>
                </div>
              </label>
            </div>
          </div>

          <div class="mt-3">
            <label class="input-label text-xs">{{ t('admin.errorPassthrough.form.platforms') }}</label>
            <div class="flex flex-wrap gap-3">
              <label
                v-for="platform in platformOptions"
                :key="platform.value"
                class="inline-flex items-center gap-1.5"
              >
                <input
                  type="checkbox"
                  :value="platform.value"
                  v-model="form.platforms"
                  class="h-3.5 w-3.5 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                />
                <span class="text-xs text-gray-700 dark:text-gray-300">{{ platform.label }}</span>
              </label>
            </div>
            <p class="input-hint text-xs mt-1">{{ t('admin.errorPassthrough.form.platformsHint') }}</p>
          </div>
        </div>

        <!-- Response Behavior -->
        <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <h4 class="mb-2 text-sm font-medium text-gray-900 dark:text-white">
            {{ t('admin.errorPassthrough.form.responseBehavior') }}
          </h4>

          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="flex items-center gap-1.5">
                <input
                  type="checkbox"
                  v-model="form.passthrough_code"
                  class="h-3.5 w-3.5 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                />
                <span class="text-xs font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.errorPassthrough.form.passthroughCode') }}
                </span>
              </label>
              <div v-if="!form.passthrough_code" class="mt-2">
                <label class="input-label text-xs">{{ t('admin.errorPassthrough.form.responseCode') }}</label>
                <input
                  v-model.number="form.response_code"
                  type="number"
                  min="100"
                  max="599"
                  class="input text-sm"
                  placeholder="422"
                />
              </div>
            </div>
            <div>
              <label class="flex items-center gap-1.5">
                <input
                  type="checkbox"
                  v-model="form.passthrough_body"
                  class="h-3.5 w-3.5 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                />
                <span class="text-xs font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.errorPassthrough.form.passthroughBody') }}
                </span>
              </label>
              <div v-if="!form.passthrough_body" class="mt-2">
                <label class="input-label text-xs">{{ t('admin.errorPassthrough.form.customMessage') }}</label>
                <input
                  v-model="form.custom_message"
                  type="text"
                  class="input text-sm"
                  :placeholder="t('admin.errorPassthrough.form.customMessagePlaceholder')"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- Skip Monitoring -->
        <div class="flex items-center gap-1.5">
          <input
            type="checkbox"
            v-model="form.skip_monitoring"
            class="h-3.5 w-3.5 rounded border-gray-300 text-yellow-600 focus:ring-yellow-500"
          />
          <span class="text-xs font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.errorPassthrough.form.skipMonitoring') }}
          </span>
        </div>
        <p class="input-hint text-xs -mt-3">{{ t('admin.errorPassthrough.form.skipMonitoringHint') }}</p>

        <!-- Enabled -->
        <div class="flex items-center gap-1.5">
          <input
            type="checkbox"
            v-model="form.enabled"
            class="h-3.5 w-3.5 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
          <span class="text-xs font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.errorPassthrough.form.enabled') }}
          </span>
        </div>
      </form>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button @click="closeFormModal" type="button" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button @click="handleSubmit" :disabled="submitting" class="btn btn-primary">
            <Icon v-if="submitting" name="refresh" size="sm" class="mr-1 animate-spin" />
            {{ showEditModal ? t('common.update') : t('common.create') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Policy Edit Modal -->
    <BaseDialog
      :show="showPolicyEditModal"
      :title="t('admin.errorPassthrough.editPolicy')"
      width="wide"
      :z-index="60"
      @close="closePolicyModal"
    >
      <form v-if="editingPolicy" @submit.prevent="handlePolicySubmit" class="space-y-4">
        <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <div class="text-sm font-medium text-gray-900 dark:text-white">
            {{ editingPolicy.label }}
          </div>
          <div class="mt-1 font-mono text-xs text-gray-500 dark:text-gray-400">
            {{ editingPolicy.category }}
          </div>
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            {{ editingPolicy.description }}
          </p>
        </div>

        <div class="space-y-3 rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <label class="flex items-center gap-2">
            <input
              v-model="policyForm.custom_enabled"
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
            />
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.errorPassthrough.policyForm.customEnabled') }}
            </span>
          </label>

          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="input-label text-xs">{{ t('admin.errorPassthrough.policyForm.statusCode') }}</label>
              <input
                v-model.number="policyForm.status_code"
                type="number"
                min="400"
                max="599"
                class="input text-sm"
                :disabled="!policyForm.custom_enabled"
              />
            </div>
            <div>
              <label class="input-label text-xs">{{ t('admin.errorPassthrough.policyForm.errorType') }}</label>
              <input
                v-model="policyForm.error_type"
                type="text"
                class="input font-mono text-sm"
                :disabled="!policyForm.custom_enabled"
              />
            </div>
          </div>

          <div>
            <label class="input-label text-xs">{{ t('admin.errorPassthrough.policyForm.message') }}</label>
            <textarea
              v-model="policyForm.message"
              rows="3"
              class="input text-sm"
              :disabled="!policyForm.custom_enabled"
            />
          </div>
        </div>

        <div class="space-y-3 rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <label class="flex items-center gap-2">
            <input
              v-model="policyForm.retry_enabled"
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 disabled:opacity-50"
              :disabled="!editingPolicy.default_retryable"
            />
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.errorPassthrough.policyForm.retryEnabled') }}
            </span>
          </label>
          <p v-if="!editingPolicy.default_retryable" class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.errorPassthrough.policyForm.notRetryableHint') }}
          </p>
          <div class="max-w-xs">
            <label class="input-label text-xs">{{ t('admin.errorPassthrough.policyForm.maxRetries') }}</label>
            <input
              v-model.number="policyForm.max_retries"
              type="number"
              min="1"
              max="5"
              class="input text-sm"
              :disabled="!policyForm.retry_enabled || !editingPolicy.default_retryable"
            />
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('admin.errorPassthrough.policyForm.note') }}</label>
          <input
            v-model="policyForm.note"
            type="text"
            class="input"
            :placeholder="t('admin.errorPassthrough.policyForm.notePlaceholder')"
          />
        </div>
      </form>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button @click="closePolicyModal" type="button" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button @click="handlePolicySubmit" :disabled="policySubmitting" class="btn btn-primary">
            <Icon v-if="policySubmitting" name="refresh" size="sm" class="mr-1 animate-spin" />
            {{ t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Delete Confirmation -->
    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('admin.errorPassthrough.deleteRule')"
      :message="t('admin.errorPassthrough.deleteConfirm', { name: deletingRule?.name })"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      :z-index="60"
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { ErrorPassthroughRule, UpstreamErrorPolicy } from '@/api/admin/errorPassthrough'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'

withDefaults(defineProps<{
  showDescription?: boolean
}>(), {
  showDescription: true
})

const { t } = useI18n()
const appStore = useAppStore()

const activeTab = ref<'policies' | 'rules'>('policies')
const rules = ref<ErrorPassthroughRule[]>([])
const policies = ref<UpstreamErrorPolicy[]>([])
const loading = ref(false)
const policiesLoading = ref(false)
const submitting = ref(false)
const policySubmitting = ref(false)
const showCreateModal = ref(false)
const showEditModal = ref(false)
const showPolicyEditModal = ref(false)
const showDeleteDialog = ref(false)
const editingRule = ref<ErrorPassthroughRule | null>(null)
const editingPolicy = ref<UpstreamErrorPolicy | null>(null)
const deletingRule = ref<ErrorPassthroughRule | null>(null)

// Form inputs for arrays
const errorCodesInput = ref('')
const keywordsInput = ref('')

const form = reactive({
  name: '',
  enabled: true,
  priority: 0,
  match_mode: 'any' as 'any' | 'all',
  platforms: [] as string[],
  passthrough_code: true,
  response_code: null as number | null,
  passthrough_body: true,
  custom_message: null as string | null,
  skip_monitoring: false,
  description: null as string | null
})

const policyForm = reactive({
  custom_enabled: false,
  status_code: 502 as number | null,
  error_type: '',
  message: '',
  retry_enabled: false,
  max_retries: 1,
  note: ''
})

const matchModeOptions = computed(() => [
  { value: 'any', label: t('admin.errorPassthrough.matchMode.any'), description: t('admin.errorPassthrough.matchMode.anyHint') },
  { value: 'all', label: t('admin.errorPassthrough.matchMode.all'), description: t('admin.errorPassthrough.matchMode.allHint') }
])

const platformOptions = [
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'antigravity', label: 'Antigravity' }
]

onMounted(() => {
  loadPolicies()
})

watch(activeTab, (tab) => {
  if (tab === 'policies') {
    loadPolicies()
    return
  }
  loadRules()
})

const loadPolicies = async () => {
  policiesLoading.value = true
  try {
    policies.value = await adminAPI.errorPassthrough.listPolicies()
  } catch (error) {
    appStore.showError(t('admin.errorPassthrough.failedToLoadPolicies'))
    console.error('Error loading upstream error policies:', error)
  } finally {
    policiesLoading.value = false
  }
}

const loadRules = async () => {
  loading.value = true
  try {
    rules.value = await adminAPI.errorPassthrough.list()
  } catch (error) {
    appStore.showError(t('admin.errorPassthrough.failedToLoad'))
    console.error('Error loading rules:', error)
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  form.name = ''
  form.enabled = true
  form.priority = 0
  form.match_mode = 'any'
  form.platforms = []
  form.passthrough_code = true
  form.response_code = null
  form.passthrough_body = true
  form.custom_message = null
  form.skip_monitoring = false
  form.description = null
  errorCodesInput.value = ''
  keywordsInput.value = ''
}

const closeFormModal = () => {
  showCreateModal.value = false
  showEditModal.value = false
  editingRule.value = null
  resetForm()
}

const handleEdit = (rule: ErrorPassthroughRule) => {
  editingRule.value = rule
  form.name = rule.name
  form.enabled = rule.enabled
  form.priority = rule.priority
  form.match_mode = rule.match_mode
  form.platforms = [...rule.platforms]
  form.passthrough_code = rule.passthrough_code
  form.response_code = rule.response_code
  form.passthrough_body = rule.passthrough_body
  form.custom_message = rule.custom_message
  form.skip_monitoring = rule.skip_monitoring
  form.description = rule.description
  errorCodesInput.value = rule.error_codes.join(', ')
  keywordsInput.value = rule.keywords.join('\n')
  showEditModal.value = true
}

const handleDelete = (rule: ErrorPassthroughRule) => {
  deletingRule.value = rule
  showDeleteDialog.value = true
}

const handleEditPolicy = (policy: UpstreamErrorPolicy) => {
  editingPolicy.value = policy
  policyForm.custom_enabled = policy.custom_enabled
  policyForm.status_code = policy.status_code ?? policy.default_status_code
  policyForm.error_type = policy.error_type || policy.default_error_type
  policyForm.message = policy.message || policy.default_message
  policyForm.retry_enabled = policy.default_retryable && policy.retry_enabled
  policyForm.max_retries = policy.max_retries > 0 ? policy.max_retries : 1
  policyForm.note = policy.note || ''
  showPolicyEditModal.value = true
}

const closePolicyModal = () => {
  showPolicyEditModal.value = false
  editingPolicy.value = null
  policySubmitting.value = false
}

const handlePolicySubmit = async () => {
  if (!editingPolicy.value) return
  if (policyForm.custom_enabled) {
    if (!policyForm.status_code || policyForm.status_code < 400 || policyForm.status_code > 599) {
      appStore.showError(t('admin.errorPassthrough.policyForm.invalidStatusCode'))
      return
    }
    if (!policyForm.error_type.trim() || !policyForm.message.trim()) {
      appStore.showError(t('admin.errorPassthrough.policyForm.responseRequired'))
      return
    }
  }

  policySubmitting.value = true
  try {
    const updated = await adminAPI.errorPassthrough.updatePolicy(editingPolicy.value.category, {
      custom_enabled: policyForm.custom_enabled,
      status_code: policyForm.custom_enabled ? policyForm.status_code : null,
      error_type: policyForm.custom_enabled ? policyForm.error_type.trim() : '',
      message: policyForm.custom_enabled ? policyForm.message.trim() : '',
      retry_enabled: editingPolicy.value.default_retryable ? policyForm.retry_enabled : false,
      max_retries: policyForm.retry_enabled ? policyForm.max_retries : 0,
      note: policyForm.note.trim()
    })
    const index = policies.value.findIndex(policy => policy.category === updated.category)
    if (index >= 0) {
      policies.value[index] = updated
    }
    appStore.showSuccess(t('admin.errorPassthrough.policyUpdated'))
    closePolicyModal()
  } catch (error: any) {
    appStore.showError(error.response?.data?.message || error.response?.data?.detail || t('admin.errorPassthrough.failedToSavePolicy'))
    console.error('Error saving upstream error policy:', error)
  } finally {
    policySubmitting.value = false
  }
}

const parseErrorCodes = (): number[] => {
  if (!errorCodesInput.value.trim()) return []
  return errorCodesInput.value
    .split(/[,\s]+/)
    .map(s => parseInt(s.trim(), 10))
    .filter(n => !isNaN(n) && n > 0)
}

const parseKeywords = (): string[] => {
  if (!keywordsInput.value.trim()) return []
  return keywordsInput.value
    .split('\n')
    .map(s => s.trim())
    .filter(s => s.length > 0)
}

const handleSubmit = async () => {
  if (!form.name.trim()) {
    appStore.showError(t('admin.errorPassthrough.nameRequired'))
    return
  }

  const errorCodes = parseErrorCodes()
  const keywords = parseKeywords()

  if (errorCodes.length === 0 && keywords.length === 0) {
    appStore.showError(t('admin.errorPassthrough.conditionsRequired'))
    return
  }

  submitting.value = true
  try {
    const data = {
      name: form.name.trim(),
      enabled: form.enabled,
      priority: form.priority,
      error_codes: errorCodes,
      keywords: keywords,
      match_mode: form.match_mode,
      platforms: form.platforms,
      passthrough_code: form.passthrough_code,
      response_code: form.passthrough_code ? null : form.response_code,
      passthrough_body: form.passthrough_body,
      custom_message: form.passthrough_body ? null : form.custom_message,
      skip_monitoring: form.skip_monitoring,
      description: form.description?.trim() || null
    }

    if (showEditModal.value && editingRule.value) {
      await adminAPI.errorPassthrough.update(editingRule.value.id, data)
      appStore.showSuccess(t('admin.errorPassthrough.ruleUpdated'))
    } else {
      await adminAPI.errorPassthrough.create(data)
      appStore.showSuccess(t('admin.errorPassthrough.ruleCreated'))
    }

    closeFormModal()
    loadRules()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.errorPassthrough.failedToSave'))
    console.error('Error saving rule:', error)
  } finally {
    submitting.value = false
  }
}

const toggleEnabled = async (rule: ErrorPassthroughRule) => {
  try {
    await adminAPI.errorPassthrough.toggleEnabled(rule.id, !rule.enabled)
    rule.enabled = !rule.enabled
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.errorPassthrough.failedToToggle'))
    console.error('Error toggling rule:', error)
  }
}

const confirmDelete = async () => {
  if (!deletingRule.value) return

  try {
    await adminAPI.errorPassthrough.delete(deletingRule.value.id)
    appStore.showSuccess(t('admin.errorPassthrough.ruleDeleted'))
    showDeleteDialog.value = false
    deletingRule.value = null
    loadRules()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.errorPassthrough.failedToDelete'))
    console.error('Error deleting rule:', error)
  }
}
</script>
