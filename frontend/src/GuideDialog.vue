<script setup lang="ts">
// (?) next to a source title that opens a step-by-step setup guide. Steps
// come from i18n as `${prefix}.step1..N`; `links` maps a step to a URL that
// opens in the system browser (the Wails webview would otherwise navigate away).
import { CircleQuestionMark, ExternalLink } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { BrowserOpenURL } from '../wailsjs/runtime/runtime'
import { t } from './i18n'

const props = defineProps<{ prefix: string; steps: number; links?: Record<number, string>; note?: string }>()
const stepKeys = Array.from({ length: props.steps }, (_, i) => i + 1)
</script>

<template>
  <Dialog>
    <DialogTrigger as-child>
      <button type="button" class="inline-flex text-muted-foreground hover:text-foreground" :aria-label="t('guide.open')">
        <CircleQuestionMark class="size-4" />
      </button>
    </DialogTrigger>
    <DialogContent class="max-w-lg">
      <DialogHeader>
        <DialogTitle>{{ t(`${prefix}.title`) }}</DialogTitle>
        <DialogDescription>{{ t(`${prefix}.intro`) }}</DialogDescription>
      </DialogHeader>
      <ol class="space-y-3 text-sm">
        <li v-for="n in stepKeys" :key="n" class="flex gap-3">
          <span class="flex size-6 shrink-0 items-center justify-center rounded-full bg-primary font-mono text-xs text-primary-foreground">{{ n }}</span>
          <div class="min-w-0 flex-1 space-y-1.5 pt-0.5">
            <p class="whitespace-pre-line">{{ t(`${prefix}.step${n}`) }}</p>
            <Button v-if="links?.[n]" variant="outline" size="xs" @click="BrowserOpenURL(links[n])">
              <ExternalLink />{{ links[n].replace(/^https?:\/\//, '') }}
            </Button>
          </div>
        </li>
      </ol>
      <p v-if="note" class="rounded-md border bg-muted px-3 py-2 text-xs text-muted-foreground">{{ t(note) }}</p>
    </DialogContent>
  </Dialog>
</template>
