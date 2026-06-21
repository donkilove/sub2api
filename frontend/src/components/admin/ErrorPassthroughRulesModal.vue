<template>
  <BaseDialog
    :show="show"
    :title="t('admin.errorPassthrough.title')"
    width="extra-wide"
    @close="$emit('close')"
  >
    <ErrorPolicyManager />

    <template #footer>
      <div class="flex justify-end">
        <button @click="$emit('close')" class="btn btn-secondary">
          {{ t('common.close') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ErrorPolicyManager from '@/components/admin/ErrorPolicyManager.vue'

defineProps<{
  show: boolean
}>()

defineEmits<{
  close: []
}>()

const { t } = useI18n()
</script>
