<script setup lang="ts">
// (?) next to a source title that opens a step-by-step setup guide. Steps
// come from i18n as `${prefix}.step1..N`; `links` maps a step to a URL that
// opens in the system browser (the Wails webview would otherwise navigate away).
import { CircleQuestionMark, ExternalLink } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { BrowserOpenURL } from '../wailsjs/runtime/runtime'
import { t } from './i18n'

const props = defineProps<{ prefix: string; steps: number; links?: Record<number, string>; note?: string; vars?: Record<string, string | number> }>()
const stepKeys = Array.from({ length: props.steps }, (_, i) => i + 1)
</script>

<template>
  <Dialog>
    <DialogTrigger as-child>
      <button type="button" class="inline-flex text-muted-foreground hover:text-foreground" :aria-label="t('guide.open')">
        <CircleQuestionMark class="size-4" />
      </button>
    </DialogTrigger>
    <DialogContent class="max-h-[85vh] w-[calc(100vw-2rem)] overflow-y-auto sm:max-w-2xl">
      <DialogHeader>
        <DialogTitle>{{ t(`${prefix}.title`) }}</DialogTitle>
        <DialogDescription class="break-keep text-base leading-relaxed">{{ t(`${prefix}.intro`) }}</DialogDescription>
      </DialogHeader>
      <ol class="space-y-4 text-base leading-relaxed">
        <li v-for="n in stepKeys" :key="n" class="flex gap-3">
          <span class="flex size-6 shrink-0 items-center justify-center rounded-full bg-primary font-mono text-xs text-primary-foreground">{{ n }}</span>
          <div class="min-w-0 flex-1 space-y-1.5 pt-0.5">
            <p class="whitespace-pre-line break-keep">{{ t(`${prefix}.step${n}`, vars) }}</p>
            <Button v-if="links?.[n]" variant="outline" size="sm" class="max-w-full" @click="BrowserOpenURL(links[n])">
              <ExternalLink /><span class="truncate">{{ links[n].replace(/^https?:\/\//, '') }}</span>
            </Button>
          </div>
        </li>
      </ol>
      <p v-if="note" class="break-keep rounded-md border bg-muted px-4 py-3 text-sm leading-relaxed text-muted-foreground">{{ t(note, vars) }}</p>
    </DialogContent>
  </Dialog>
</template>
