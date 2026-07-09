import { useEffect, useRef, useState } from 'react'
import type { Session } from '../types'
import type { Handlers } from '../App'

const MODELS: [string, string][] = [['opus', 'Opus'], ['sonnet', 'Sonnet'], ['haiku', 'Haiku']]

export default function RowActions({ s, h }: { s: Session; h: Handlers }) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const run = s.status === 'running'
  const stop = (e: React.MouseEvent) => e.stopPropagation()

  useEffect(() => {
    if (!open) return
    const onDoc = (e: MouseEvent) => { if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false) }
    const onEsc = (e: KeyboardEvent) => { if (e.key === 'Escape') setOpen(false) }
    document.addEventListener('mousedown', onDoc)
    document.addEventListener('keydown', onEsc)
    return () => { document.removeEventListener('mousedown', onDoc); document.removeEventListener('keydown', onEsc) }
  }, [open])

  const act = (action: Parameters<Handlers['runAction']>[0]) => { setOpen(false); h.runAction(action, s) }
  const send = (text: string, confirmMsg?: string) => {
    if (confirmMsg && !confirm(confirmMsg)) return
    setOpen(false); h.runAction('send', s, text)
  }
  const copyResume = () => { setOpen(false); navigator.clipboard.writeText(`cd '${s.cwd}' && claude --resume '${s.id}'`) }
  const kill = () => { if (confirm('Kill this session? This terminates its claude process.')) { setOpen(false); h.runAction('kill', s) } }

  return (
    <div className={'rowactions' + (open ? ' open' : '')} ref={ref} onClick={stop}>
      {run
        ? <>
            <button className="btn go" onClick={() => act('focus')}>▶ Focus</button>
            <button className="btn cmp" title="Compose & send a message to this session" onClick={() => h.openCompose(s)}>✎ Compose</button>
          </>
        : <button className="btn res" onClick={() => act('resume')}>↻ Resume</button>}
      <button className="menutrig" title="More actions" onClick={() => setOpen(o => !o)}>⋯</button>
      {open && (
        <div className="menu">
          {run
            ? <button className="mi" onClick={() => act('focus')}>▶ Focus tab</button>
            : <button className="mi" onClick={() => act('resume')}>↻ Resume session</button>}
          <button className="mi" onClick={() => act('reveal')}>📁 Reveal in Finder</button>
          <button className="mi" onClick={copyResume}>⧉ Copy resume cmd</button>
          {run && (
            <>
              <button className="mi" onClick={() => { setOpen(false); h.openCompose(s) }}>⌨ Compose &amp; send…</button>
              <div className="midiv">Switch model</div>
              <div className="mimodels">
                {MODELS.map(([id, label]) => (
                  <button key={id} className="mchip" onClick={() => send('/model ' + id)}>{label}</button>
                ))}
              </div>
              <div className="midiv">Session control</div>
              <button className="mi" onClick={() => send('/compact', 'Compact this session? It summarizes the conversation and drops detail.')}>«» Compact</button>
              <button className="mi" onClick={() => send('/clear', 'Clear this session? Starts fresh (recoverable via /resume).')}>⌫ Clear</button>
              <button className="mi danger" onClick={kill}>✕ Kill session</button>
            </>
          )}
        </div>
      )}
    </div>
  )
}
