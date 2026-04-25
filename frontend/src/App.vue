<script setup lang="ts">
  import { onMounted, ref } from 'vue'
  import { checkHealth, getStatus, type StatusResponse } from './api/client'

  type Status = 'loading' | 'ok' | 'error'

  const health = ref<Status>('loading')
  const buildStatus = ref<Status>('loading')
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
    <div class="bg-white rounded-2xl shadow-sm border border-gray-100 p-8 w-full max-w-sm">
      <h1 class="text-xl font-semibold text-gray-900 tracking-tight">your-project</h1>
      <p class="text-sm text-gray-400 mt-0.5 mb-6">Go + Vue 3 template</p>

      <div class="flex items-center gap-2.5">
        <span class="text-sm text-gray-500">API</span>

        <span v-if="health === 'loading'" class="h-2 w-2 rounded-full bg-gray-300 animate-pulse" />
        <span v-else-if="health === 'ok'" class="h-2 w-2 rounded-full bg-emerald-500" />
        <span v-else class="h-2 w-2 rounded-full bg-red-400" />

        <span class="text-sm text-gray-400">
          {{ health === 'loading' ? 'checking…' : health === 'ok' ? 'healthy' : 'unreachable' }}
        </span>
      </div>

      <div class="mt-6 pt-6 border-t border-gray-100">
        <h2 class="text-sm font-medium text-gray-700 mb-3">Build info</h2>

        <p v-if="buildStatus === 'loading'" class="text-sm text-gray-400 animate-pulse">loading…</p>
        <p v-else-if="buildStatus === 'error'" class="text-sm text-red-400">unavailable</p>

        <dl v-else class="space-y-1.5 text-sm">
          <div class="flex gap-2">
            <dt class="text-gray-400 w-20 shrink-0">Git SHA</dt>
            <dd class="font-mono text-gray-700 truncate">{{ buildInfo?.git_sha }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="text-gray-400 w-20 shrink-0">Built at</dt>
            <dd class="font-mono text-gray-700">{{ buildInfo?.build_time }}</dd>
          </div>
        </dl>
      </div>
    </div>
  </div>
</template>
