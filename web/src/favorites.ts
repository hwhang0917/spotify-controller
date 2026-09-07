// Favorites live in this browser only (localStorage); the host never sees them.
import { useStorage } from '@vueuse/core'
import type { Track } from './types'

export const favorites = useStorage<Track[]>('vibe-music.favorites', [])
const key = (t: Track) => `${t.source}:${t.id}`
export const isFav = (t: Track) => favorites.value.some((f) => key(f) === key(t))
export function toggleFav(t: Track) {
  favorites.value = isFav(t) ? favorites.value.filter((f) => key(f) !== key(t)) : [...favorites.value, t]
}
