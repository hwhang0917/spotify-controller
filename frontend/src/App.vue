<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { StartServer, StopServer, Status } from '../wailsjs/go/main/App'

const running = ref(false)
const port = ref(0)
const url = ref('')
const error = ref('')

async function refresh() {
  const s = await Status()
  running.value = s.running
  url.value = s.url
  if (!port.value) port.value = s.port
}

async function toggle() {
  error.value = ''
  try {
    if (running.value) await StopServer()
    else await StartServer(port.value)
  } catch (e) {
    error.value = String(e)
  }
  await refresh()
}

onMounted(refresh)
</script>

<template>
  <main class="min-h-screen bg-neutral-950 text-neutral-100 p-8 space-y-6">
    <h1 class="text-2xl font-bold">vibe-music admin</h1>

    <section class="space-y-3 max-w-sm">
      <label class="block text-sm text-neutral-400">
        Guest server port
        <input
          v-model.number="port"
          type="number"
          :disabled="running"
          class="mt-1 w-full rounded bg-neutral-900 border border-neutral-700 px-3 py-2 disabled:opacity-50"
        />
      </label>
      <button
        @click="toggle"
        class="rounded px-4 py-2 font-medium"
        :class="running ? 'bg-rose-600 hover:bg-rose-500' : 'bg-emerald-600 hover:bg-emerald-500'"
      >
        {{ running ? 'Stop server' : 'Start server' }}
      </button>
      <p v-if="running" class="text-sm">
        Guests join at <a :href="url" target="_blank" class="underline text-emerald-400">{{ url }}</a>
      </p>
      <p v-if="error" class="text-sm text-rose-400">{{ error }}</p>
    </section>
  </main>
</template>
