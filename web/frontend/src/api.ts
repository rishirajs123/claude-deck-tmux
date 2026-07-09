import type { Session, Stats, Analytics, Action } from './types'

const j = (r: Response) => r.json()

export const getSessions = (): Promise<Session[]> => fetch('/api/sessions').then(j)
export const getStats = (): Promise<Stats> => fetch('/api/stats').then(j)
export const getAnalytics = (): Promise<Analytics> => fetch('/api/analytics').then(j)

export function doAction(action: Action, s: Session, text?: string): Promise<{ ok: boolean; error?: string }> {
  return fetch('/api/action', {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ action, id: s.id, cwd: s.cwd, text: text || '' }),
  }).then(j)
}

export function launchSession(cwd: string, prompt: string): Promise<{ ok: boolean; error?: string }> {
  return fetch('/api/action', {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ action: 'new', id: '', cwd, text: prompt }),
  }).then(j)
}

export function saveMeta(id: string, favorite: boolean, tags: string, notes: string) {
  return fetch('/api/meta', {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ id, favorite, tags, notes }),
  }).then(j)
}
