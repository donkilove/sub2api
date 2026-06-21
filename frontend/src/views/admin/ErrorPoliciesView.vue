<template>
  <AppLayout>
    <div class="space-y-6 pb-12">
      <section class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div class="flex items-start gap-4">
            <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-primary-500/10 text-primary-600 dark:text-primary-300">
              <ShieldIcon class="h-6 w-6" />
            </div>
            <div class="min-w-0">
              <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
                {{ t('admin.errorPassthrough.title') }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.errorPassthrough.description') }}
              </p>
            </div>
          </div>
        </div>
      </section>

      <section class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <ErrorPolicyManager :show-description="false" />
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { defineComponent, h } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ErrorPolicyManager from '@/components/admin/ErrorPolicyManager.vue'

const { t } = useI18n()

const ShieldIcon = defineComponent({
  name: 'ShieldIcon',
  setup() {
    return () =>
      h(
        'svg',
        { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
        [
          h('path', {
            'stroke-linecap': 'round',
            'stroke-linejoin': 'round',
            d: 'M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z'
          })
        ]
      )
  }
})
</script>
