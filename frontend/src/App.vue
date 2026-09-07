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
  <main class="min-h-screen p-8 space-y-6">
    <header>
      <p class="eyebrow">admin</p>
      <h1 class="text-2xl font-semibold tracking-tight">vibe-music</h1>
    </header>

    <section class="card max-w-sm space-y-4">
      <p class="eyebrow">server</p>
      <label class="block text-sm text-body">
        Guest server port
        <input v-model.number="port" type="number" :disabled="running" class="input mt-1" />
      </label>
      <button @click="toggle" :class="running ? 'btn-ghost' : 'btn-primary'">
        {{ running ? 'Stop server' : 'Start server' }}
      </button>
      <p v-if="running" class="text-sm text-body">
        Guests join at <a :href="url" target="_blank" class="text-link underline">{{ url }}</a>
      </p>
      <p v-if="error" class="text-sm text-error">{{ error }}</p>
    </section>
  </main>
</template>
