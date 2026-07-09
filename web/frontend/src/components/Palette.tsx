import { useEffect, useMemo, useRef, useState } from 'react'
import type { Session, Action } from '../types'
import { ago } from '../format'

export default function Palette({ sessions, runAction, onClose }: {
  sessions: Session[]
  runAction: (a: Action, s: Session) => Promise<void>
  onClose: () => void
}) {
  const [q, setQ] = useState('')
  const [sel, setSel] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)

  const list = useMemo(() => {
    const ql = q.toLowerCase().trim()
    return sessions
      .filter(s => !ql || (s.project + ' ' + (s.title || '') + ' ' + (s.ai_title || '') + ' ' + s.cwd + ' ' + (s.current_task || '')).toLowerCase().includes(ql))
      .sort((a, b) => (Number(b.status === 'running') - Number(a.status === 'running')) || (b.last_used_at - a.last_used_at))
      .slice(0, 40)
  }, [sessions, q])

  useEffect(() => { inputRef.current?.focus() }, [])
  useEffect(() => { if (sel >= list.length) setSel(0) }, [list, sel])

  const run = (s: Session | undefined) => {
    if (!s) return
    onClose()
    runAction(s.status === 'running' ? 'focus' : 'resume', s)
  }

  const onKey = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') onClose()
    else if (e.key === 'ArrowDown') { e.preventDefault(); setSel(i => Math.min(i + 1, list.length - 1)) }
    else if (e.key === 'ArrowUp') { e.preventDefault(); setSel(i => Math.max(i - 1, 0)) }
    else if (e.key === 'Enter') run(list[sel])
  }

  return (
    <div className="overlay" onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div className="pbox">
        <input ref={inputRef} placeholder="Jump to a session… (Enter = focus if running, else resume)"
          value={q} onChange={e => setQ(e.target.value)} onKeyDown={onKey} />
        <div className="presults">
          {list.length ? list.map((s, i) => (
            <div key={s.id} className={'pitem' + (i === sel ? ' sel' : '')} onMouseEnter={() => setSel(i)} onClick={() => run(s)}>
              <span className={'sdot ' + s.status} /><span className="pp">{s.title || s.project}</span>
              <span className="pc">{s.status === 'running' ? 'focus' : 'resume'} · {ago(s.last_used_at)}</span>
            </div>
          )) : <div className="pitem">no match</div>}
        </div>
      </div>
    </div>
  )
}
