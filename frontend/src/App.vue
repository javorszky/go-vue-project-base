<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { checkHealth } from './api/client'

type Status = 'loading' | 'ok' | 'error'

const status = ref<Status>('loading')

onMounted(async () => {
  try {
    await checkHealth()
    status.value = 'ok'
  } catch {
    status.value = 'error'
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

        <span
          v-if="status === 'loading'"
          class="h-2 w-2 rounded-full bg-gray-300 animate-pulse"
        />
        <span
          v-else-if="status === 'ok'"
          class="h-2 w-2 rounded-full bg-emerald-500"
        />
        <span
          v-else
          class="h-2 w-2 rounded-full bg-red-400"
        />

        <span class="text-sm text-gray-400">
          {{ status === 'loading' ? 'checking…' : status === 'ok' ? 'healthy' : 'unreachable' }}
        </span>
      </div>
    </div>
  </div>
</template>
