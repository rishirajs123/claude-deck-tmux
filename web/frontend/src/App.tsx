import { useCallback, useEffect, useRef, useState } from 'react'
import type { Session, Stats, Analytics as AnalyticsData, Action } from './types'
import { getSessions, getStats, getAnalytics, doAction, saveMeta } from './api'
import { isZombie } from './format'
import Hero from './components/Hero'
import Sessions from './components/Sessions'
import Analytics from './components/Analytics'
import Palette from './components/Palette'

export interface Handlers {
  runAction: (a: Action, s: Session, text?: string) => Promise<void>
  toggleFav: (s: Session) => Promise<void>
  saveNote: (s: Session, tags: string, notes: string) => Promise<void>
  killZombies: () => Promise<void>
}

export default function App() {
  const [sessions, setSessions] = useState<Session[]>([])
  const [stats, setStats] = useState<Stats | null>(null)
  const [analytics, setAnalytics] = useState<AnalyticsData | null>(null)
  const [view, setView] = useState<'sessions' | 'analytics'>('sessions')
  const [paused, setPaused] = useState(false)
  const [clock, setClock] = useState('')
  const [toast, setToast] = useState<{ msg: string; kind: string } | null>(null)
  const [paletteOpen, setPaletteOpen] = useState(false)
  const toastTimer = useRef<ReturnType<typeof setTimeout>>()

  const showToast = useCallback((msg: string, kind = '') => {
    setToast({ msg, kind })
    clearTimeout(toastTimer.current)
    toastTimer.current = setTimeout(() => setToast(null), 2600)
  }, [])

  const load = useCallback(async () => {
    try {
      const [s, st, an] = await Promise.all([getSessions(), getStats(), getAnalytics()])
      setSessions(s || [])
      setStats(st)
      setAnalytics(an)
      setClock(new Date().toLocaleTimeString([], { hour12: false }))
    } catch {
      setClock('offline')
    }
  }, [])

  useEffect(() => {
    load()
    if (paused) return
    const t = setInterval(load, 5000)
    return () => clearInterval(t)
  }, [load, paused])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') { e.preventDefault(); setPaletteOpen(true) }
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [])

  const runAction = useCallback(async (action: Action, s: Session, text?: string) => {
    const label = action === 'send' ? (text || 'command') : action
    showToast(label + '…')
    try {
      const r = await doAction(action, s, text)
      showToast(r.ok ? label + ' ✓' : label + ' failed: ' + r.error, r.ok ? 'ok' : 'err')
    } catch {
      showToast(label + ' failed', 'err')
    }
    setTimeout(load, 500)
  }, [load, showToast])

  const toggleFav = useCallback(async (s: Session) => {
    await saveMeta(s.id, !s.favorite, s.tags || '', s.notes || '')
    showToast(!s.favorite ? 'pinned ★' : 'unpinned', 'ok')
    load()
  }, [load, showToast])

  const saveNote = useCallback(async (s: Session, tags: string, notes: string) => {
    await saveMeta(s.id, s.favorite, tags, notes)
    showToast('saved ✓', 'ok')
    load()
  }, [load, showToast])

  const killZombies = useCallback(async () => {
    const z = sessions.filter(isZombie)
    if (!z.length) return
    if (!confirm(`Kill ${z.length} idle zombie session(s)? This terminates their claude processes.`)) return
    for (const s of z) await doAction('kill', s)
    showToast(`killed ${z.length} zombie(s)`, 'ok')
    setTimeout(load, 600)
  }, [sessions, load, showToast])

  const handlers: Handlers = { runAction, toggleFav, saveNote, killZombies }
  const zombies = sessions.filter(isZombie).length

  return (
    <div className="wrap">
      <div className="strip">
        <div className="brand">CLAUDE<b>DECK</b></div>
        <div className="eyebrow">Mission Control</div>
        <div className="nav">
          <button className={view === 'sessions' ? 'on' : ''} onClick={() => setView('sessions')}>Sessions</button>
          <button className={view === 'analytics' ? 'on' : ''} onClick={() => setView('analytics')}>Analytics</button>
        </div>
        <div className="right">
          <button className="iconbtn" onClick={() => setPaletteOpen(true)}>⌘K</button>
          <button className="iconbtn" onClick={() => setPaused(p => !p)}>{paused ? '▶' : '⏸'}</button>
          <button className="iconbtn" onClick={load}>⟳</button>
          {zombies > 0 && <span className="alert">⚠ {zombies} idle</span>}
          <span className="clock">{clock || '—'}{paused ? ' · paused' : ''}</span>
        </div>
      </div>

      {view === 'sessions' && <Hero stats={stats} daily={analytics?.daily || []} />}

      {view === 'sessions'
        ? <Sessions sessions={sessions} handlers={handlers} />
        : <Analytics data={analytics} />}

      {paletteOpen && <Palette sessions={sessions} runAction={runAction} onClose={() => setPaletteOpen(false)} />}
      {toast && <div className={'toast ' + toast.kind}>{toast.msg}</div>}
    </div>
  )
}
