import { useEffect, useRef, useState } from 'react'
import type { Session } from '../types'
import type { Handlers } from '../App'

const MODELS: [string, string][] = [['opus', 'Opus'], ['sonnet', 'Sonnet'], ['haiku', 'Haiku']]

export default function RowActions({ s, h }: { s: Session; h: Handlers }) {
  const [open, setOpen] = useState(false)
  const [perm, setPerm] = useState(() => localStorage.getItem('cd_perm') || 'auto')
  const ref = useRef<HTMLDivElement>(null)
  const run = s.status === 'running'
  const stop = (e: React.MouseEvent) => e.stopPropagation()
  const setPermPref = (p: string) => { setPerm(p); localStorage.setItem('cd_perm', p) }

  useEffect(() => {
    if (!open) return
    setPerm(localStorage.getItem('cd_perm') || 'auto') // re-sync in case it changed elsewhere
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
  const copyResume = () => {
    setOpen(false)
    const flag = perm === 'skip' ? ' --dangerously-skip-permissions' : perm === 'ask' ? ' --permission-mode default' : ''
    navigator.clipboard.writeText(`cd '${s.cwd}' && claude --resume '${s.id}'${flag}`)
  }
  const kill = () => { if (confirm('Kill this session? This terminates its claude process.')) { setOpen(false); h.runAction('kill', s) } }
  const bypass = () => {
    if (confirm('Enable skip-permissions for this session?\n\nIt exits and immediately resumes the same session with --dangerously-skip-permissions, which bypasses ALL permission prompts. Only do this for a session and directory you trust.')) {
      setOpen(false); h.runAction('bypass', s)
    }
  }
  const unbypass = () => {
    if (confirm('Turn skip-permissions off for this session?\n\nIt exits and resumes the same session in default mode, so permission prompts come back.')) {
      setOpen(false); h.runAction('unbypass', s)
    }
  }

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
          {!run && (
            <>
              <div className="midiv">Permissions on resume</div>
              <div className="mimodels">
                <button className={'permchip' + (perm === 'auto' ? ' on' : '')} onClick={() => setPermPref('auto')} title="Keep the session's own mode">Auto</button>
                <button className={'permchip' + (perm === 'ask' ? ' on' : '')} onClick={() => setPermPref('ask')} title="Force permission prompts on">Ask</button>
                <button className={'permchip skip' + (perm === 'skip' ? ' on' : '')} onClick={() => setPermPref('skip')} title="Skip all prompts">Skip</button>
              </div>
            </>
          )}
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
              {s.bypass
                ? <button className="mi" onClick={unbypass}>🛡 Turn skip-permissions off (restart)</button>
                : <button className="mi" onClick={bypass}>⚡ Skip-permissions (restart)</button>}
              <button className="mi danger" onClick={kill}>✕ Kill session</button>
            </>
          )}
        </div>
      )}
    </div>
  )
}
