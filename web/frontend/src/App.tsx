import { useCallback, useEffect, useRef, useState } from 'react'
import type { Session, Stats, Analytics as AnalyticsData, Action } from './types'
import { getSessions, getStats, getAnalytics, doAction, saveMeta } from './api'
import { fmtN, isZombie } from './format'
import Sessions from './components/Sessions'
import Analytics from './components/Analytics'
import Palette from './components/Palette'

export interface Handlers {
  runAction: (a: Action, s: Session) => Promise<void>
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
  const [pulse, setPulse] = useState('live')
  const [toast, setToast] = useState<{ msg: string; kind: string } | null>(null)
  const [paletteOpen, setPaletteOpen] = useState(false)
  const viewRef = useRef(view)
  viewRef.current = view
  const toastTimer = useRef<ReturnType<typeof setTimeout>>()

  const showToast = useCallback((msg: string, kind = '') => {
    setToast({ msg, kind })
    clearTimeout(toastTimer.current)
    toastTimer.current = setTimeout(() => setToast(null), 2600)
  }, [])

  const load = useCallback(async () => {
    try {
      const [s, st] = await Promise.all([getSessions(), getStats()])
      setSessions(s || [])
      setStats(st)
      if (viewRef.current === 'analytics') {
        try { setAnalytics(await getAnalytics()) } catch { /* ignore */ }
      }
      setPulse('live · ' + new Date().toLocaleTimeString())
    } catch {
      setPulse('backend offline')
    }
  }, [])

  useEffect(() => {
    load()
    if (paused) return
    const t = setInterval(load, 5000)
    return () => clearInterval(t)
  }, [load, paused])

  useEffect(() => {
    if (view === 'analytics') getAnalytics().then(setAnalytics).catch(() => {})
  }, [view])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') { e.preventDefault(); setPaletteOpen(true) }
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [])

  const runAction = useCallback(async (action: Action, s: Session) => {
    showToast(action + '…')
    try {
      const r = await doAction(action, s)
      showToast(r.ok ? action + ' ✓' : action + ' failed: ' + r.error, r.ok ? 'ok' : 'err')
    } catch {
      showToast(action + ' failed', 'err')
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

  const cards: [keyof Stats, string, string][] = [
    ['total', 'Sessions', ''], ['running', 'Running now', 'hot'],
    ['prompts', 'Total prompts', ''], ['projects', 'Projects', ''],
  ]

  return (
    <div className="wrap">
      <header>
        <h1>ClaudeDeck</h1>
        <div className="nav">
          <button className={view === 'sessions' ? 'on' : ''} onClick={() => setView('sessions')}>Sessions</button>
          <button className={view === 'analytics' ? 'on' : ''} onClick={() => setView('analytics')}>Analytics</button>
        </div>
        <span className="right">
          <button className="iconbtn" onClick={() => setPaletteOpen(true)}>⌘K</button>
          <button className="iconbtn" onClick={() => setPaused(p => !p)}>{paused ? '▶' : '⏸'}</button>
          <button className="iconbtn" onClick={load}>⟳</button>
          <span><span className="dot" /> <span>{pulse}</span></span>
        </span>
      </header>

      {stats && (
        <div className="stats">
          {cards.map(([k, l, c]) => (
            <div className={'card ' + c} key={k}><div className="n">{fmtN(stats[k] as number)}</div><div className="l">{l}</div></div>
          ))}
          <div className="card"><div className="n">{fmtN((stats.tokens_in || 0) + (stats.tokens_out || 0))}</div><div className="l">Tokens (in+out)</div></div>
        </div>
      )}

      {view === 'sessions'
        ? <Sessions sessions={sessions} handlers={handlers} />
        : <Analytics data={analytics} />}

      {paletteOpen && <Palette sessions={sessions} runAction={runAction} onClose={() => setPaletteOpen(false)} />}
      {toast && <div className={'toast ' + toast.kind}>{toast.msg}</div>}
    </div>
  )
}
