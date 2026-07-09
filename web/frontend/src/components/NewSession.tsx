import { useEffect, useMemo, useRef, useState } from 'react'
import type { Session } from '../types'

export default function NewSession({ sessions, onClose, onLaunch }: {
  sessions: Session[]
  onClose: () => void
  onLaunch: (dir: string, prompt: string) => Promise<void>
}) {
  const [dir, setDir] = useState('')
  const [prompt, setPrompt] = useState('')
  const dirRef = useRef<HTMLInputElement>(null)

  const dirs = useMemo(
    () => Array.from(new Set(sessions.map(s => s.cwd).filter(Boolean))).sort(),
    [sessions],
  )

  useEffect(() => { dirRef.current?.focus() }, [])

  const launch = () => {
    const d = dir.trim()
    if (!d) return
    onClose()
    onLaunch(d, prompt.trim())
  }
  const onKey = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') { e.preventDefault(); onClose() }
    else if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) { e.preventDefault(); launch() }
  }

  return (
    <div className="overlay" onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div className="composer" onKeyDown={onKey}>
        <div className="cmphead">
          <div className="cmpid">
            <div className="cmpeyebrow">New session</div>
            <div className="cmptitle">Start Claude Code in a directory</div>
          </div>
        </div>
        <div className="nsbody">
          <div className="nsfield">
            <label>Directory</label>
            <input ref={dirRef} list="cd-dirs" value={dir} onChange={e => setDir(e.target.value)}
              placeholder="~/path/to/project — type a path or pick a recent one" />
            <datalist id="cd-dirs">{dirs.map(d => <option key={d} value={d} />)}</datalist>
          </div>
          <div className="nsfield">
            <label>First prompt <span className="opt">optional</span></label>
            <textarea className="nsarea" value={prompt} onChange={e => setPrompt(e.target.value)}
              placeholder="Sent as the session's first message — e.g. “Review the auth module and list the top risks.” Leave blank to just open Claude." />
          </div>
        </div>
        <div className="cmpfoot">
          <span className="cmphint">⌘↵ launch · Esc close · opens a new iTerm2 tab</span>
          <div className="cmpbtns">
            <button className="btn" onClick={onClose}>Cancel</button>
            <button className="btn go" onClick={launch} disabled={!dir.trim()}>Launch session</button>
          </div>
        </div>
      </div>
    </div>
  )
}
