import { useEffect, useRef, useState } from 'react'
import type { Session, Action } from '../types'

const QUICK = ['/status', '/context', '/cost', '/model opus', '/model sonnet']

export default function Composer({ session, runAction, onClose }: {
  session: Session
  runAction: (a: Action, s: Session, text?: string) => Promise<void>
  onClose: () => void
}) {
  const [text, setText] = useState('')
  const ref = useRef<HTMLTextAreaElement>(null)

  useEffect(() => { ref.current?.focus() }, [])

  const send = () => {
    const t = text.trim()
    if (!t) return
    onClose()
    runAction('message', session, t)
  }
  const insert = (cmd: string) => {
    setText(t => (t && !t.endsWith('\n') ? t + '\n' : t) + cmd)
    ref.current?.focus()
  }
  const onKey = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') { e.preventDefault(); onClose() }
    else if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) { e.preventDefault(); send() }
  }

  return (
    <div className="overlay" onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div className="composer" onKeyDown={onKey}>
        <div className="cmphead">
          <div className="cmpid">
            <div className="cmpeyebrow">Send to session</div>
            <div className="cmptitle"><span className={'sdot ' + session.status} />{session.project || '(unknown)'}</div>
          </div>
          <div className="cmpcwd">{session.cwd}</div>
        </div>
        <textarea ref={ref} className="cmparea" value={text} onChange={e => setText(e.target.value)}
          placeholder="Write or paste a message, or a /command. It's delivered to the terminal session as one message — much easier than typing in the terminal." />
        <div className="cmpquick">
          <span className="cmpquicklabel">Insert</span>
          {QUICK.map(c => <button key={c} className="qchip" onClick={() => insert(c)}>{c}</button>)}
        </div>
        <div className="cmpfoot">
          <span className="cmphint">⌘↵ send · Esc close · pasted newlines stay in one message</span>
          <div className="cmpbtns">
            <button className="btn" onClick={onClose}>Cancel</button>
            <button className="btn go" onClick={send} disabled={!text.trim()}>Send to session</button>
          </div>
        </div>
      </div>
    </div>
  )
}
