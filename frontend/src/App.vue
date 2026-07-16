<script setup lang="ts">
  import { onMounted, ref } from 'vue'
  import { checkHealth, getStatus, type StatusResponse } from './api/client'
  import StatusCard from './components/StatusCard.vue'
  import type { ProbeStatus } from './types'

  const health = ref<ProbeStatus>('loading')
  const buildStatus = ref<ProbeStatus>('loading')
  const buildInfo = ref<StatusResponse | null>(null)

  onMounted(async () => {
    const [healthResult, statusResult] = await Promise.allSettled([checkHealth(), getStatus()])

    health.value = healthResult.status === 'fulfilled' ? 'ok' : 'error'

    if (statusResult.status === 'fulfilled') {
      buildInfo.value = statusResult.value
      buildStatus.value = 'ok'
    } else {
      buildStatus.value = 'error'
    }
  })
</script>

<template>
  <div class="min-h-screen bg-gray-50 flex items-center justify-center p-4">
    <StatusCard :health="health" :build-status="buildStatus" :build-info="buildInfo" />
  </div>
</template>
