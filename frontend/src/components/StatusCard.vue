<script setup lang="ts">
  import { CollapsibleContent, CollapsibleRoot, CollapsibleTrigger } from 'reka-ui'
  import type { StatusResponse } from '../api/client'
  import type { ProbeStatus } from '../types'

  defineProps<{
    health: ProbeStatus
    buildStatus: ProbeStatus
    buildInfo: StatusResponse | null
  }>()
</script>

<template>
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

    <CollapsibleRoot :default-open="true" class="mt-6 pt-6 border-t border-gray-100">
      <CollapsibleTrigger
        class="group flex w-full items-center justify-between text-sm font-medium text-gray-700 mb-3"
      >
        Build info
        <span
          class="text-gray-400 transition-transform group-data-[state=open]:rotate-180"
          aria-hidden="true"
        >
          ▾
        </span>
      </CollapsibleTrigger>

      <CollapsibleContent>
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
      </CollapsibleContent>
    </CollapsibleRoot>
  </div>
</template>
