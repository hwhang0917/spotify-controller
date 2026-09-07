<script setup lang="ts">
// (?) next to a card title; the explanation lives in the tooltip instead of
// cluttering the card body.
import { CircleQuestionMark } from '@lucide/vue'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
defineProps<{ text: string }>()
</script>

<template>
  <Tooltip>
    <TooltipTrigger as-child>
      <button type="button" class="inline-flex text-muted-foreground hover:text-foreground" aria-label="help">
        <CircleQuestionMark class="size-4" />
      </button>
    </TooltipTrigger>
    <TooltipContent class="max-w-xs whitespace-pre-line text-left">{{ text }}</TooltipContent>
  </Tooltip>
</template>
