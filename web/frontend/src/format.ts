import type { Session } from './types'

export const fmtN = (n: number): string =>
  n >= 1000 ? (n / 1000).toFixed(n >= 10000 ? 0 : 1) + 'k' : '' + n

export function ago(ms: number): string {
  if (!ms) return '—'
  const s = (Date.now() - ms) / 1000
  if (s < 60) return Math.round(s) + 's'
  if (s < 3600) return Math.round(s / 60) + 'm'
  if (s < 86400) return Math.round(s / 3600) + 'h'
  return Math.round(s / 86400) + 'd'
}

export function dur(sec: number): string {
  if (!sec) return '—'
  if (sec < 3600) return Math.round(sec / 60) + 'm'
  if (sec < 86400) return (sec / 3600).toFixed(1) + 'h'
  return (sec / 86400).toFixed(1) + 'd'
}

export const mem = (mb: number): string =>
  mb >= 1024 ? (mb / 1024).toFixed(1) + 'G' : Math.round(mb) + 'M'

export const ZOMBIE_MS = 2 * 3600 * 1000
export const isZombie = (s: Session): boolean =>
  s.status === 'running' && !!s.last_used_at && Date.now() - s.last_used_at > ZOMBIE_MS
