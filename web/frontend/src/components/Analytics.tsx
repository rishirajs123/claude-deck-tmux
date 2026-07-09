import { useEffect, useMemo, useState } from 'react'
import type { Analytics as AnalyticsData, EnvStats } from '../types'
import { fmtN, ago } from '../format'
import { getEnvironment } from '../api'

const fmtDay = (d: string) => new Date(d + 'T00:00:00').toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })

export default function Analytics({ data }: { data: AnalyticsData | null }) {
  const [hc, setHc] = useState<{ day: string; n: number; left: number; top: number } | null>(null)
  const [env, setEnv] = useState<EnvStats | null>(null)

  useEffect(() => {
    let alive = true
    const load = () => getEnvironment().then(e => alive && setEnv(e)).catch(() => {})
    load()
    const t = setInterval(load, 30000)
    return () => { alive = false; clearInterval(t) }
  }, [])

  const heat = useMemo(() => {
    if (!data) return { cells: [] as { key: string; n: number; lv: number }[], total: 0 }
    const map: Record<string, number> = {}
    let total = 0
    for (const d of data.daily || []) { map[d.day] = d.prompts; total += d.prompts }
    const end = new Date()
    const start = new Date(end); start.setDate(end.getDate() - 181); start.setDate(start.getDate() - start.getDay())
    const cells: { key: string; n: number; lv: number }[] = []
    for (const dt = new Date(start); dt <= end; dt.setDate(dt.getDate() + 1)) {
      const key = dt.toISOString().slice(0, 10)
      const n = map[key] || 0
      const lv = n === 0 ? 0 : n < 6 ? 1 : n < 21 ? 2 : n < 51 ? 3 : 4
      cells.push({ key, n, lv })
    }
    return { cells, total }
  }, [data])

  if (!data) return <div className="empty">Loading analytics…</div>
  const projs = data.topProjects || []
  const pmax = Math.max(1, ...projs.map(p => p.prompts))
  const models = data.models || []
  const mmax = Math.max(1, ...models.map(m => m.tokens))
  const skills = env?.skills || []
  const smax = Math.max(1, ...skills.map(s => s.count))
  const agentsByProj = env?.agent_by_project || []
  const amax = Math.max(1, ...agentsByProj.map(a => a.count))

  return (
    <section>
      <div className="panel">
        <h3>Activity — prompts per day</h3>
        <div className="heatwrap" style={{ position: 'relative' }} onMouseLeave={() => setHc(null)}>
          <div className="heat">
            {heat.cells.map(c => (
              <div key={c.key} className={'c' + (c.lv ? ' l' + c.lv : '')}
                onMouseEnter={e => {
                  const el = e.currentTarget
                  setHc({ day: c.key, n: c.n, left: el.offsetLeft + el.offsetWidth / 2, top: el.offsetTop })
                }} />
            ))}
          </div>
          {hc && (
            <div className="wtip" style={{ left: hc.left, top: hc.top, transform: 'translate(-50%,-130%)' }}>
              <b>{hc.n}</b> prompt{hc.n === 1 ? '' : 's'} · {fmtDay(hc.day)}
            </div>
          )}
        </div>
        <div className="legend">Less
          <span className="c" style={{ background: '#151c26' }} /><span className="c l1" /><span className="c l2" /><span className="c l3" /><span className="c l4" /> More
          <span style={{ marginLeft: 14 }}>{fmtN(heat.total)} prompts over last ~26 weeks</span>
        </div>
      </div>
      <div className="cols">
        <div className="panel">
          <h3>Top projects (by prompts)</h3>
          <div className="bars">
            {projs.length ? projs.map(p => (
              <div className="row" key={p.key}>
                <span className="lab">{p.key}</span>
                <div className="track"><div className="fill" style={{ width: Math.max(2, 100 * p.prompts / pmax) + '%' }} /></div>
                <span className="val">{p.prompts} · {p.sessions}s</span>
              </div>
            )) : <div className="empty">no data</div>}
          </div>
        </div>
        <div className="panel">
          <h3>Models (by tokens)</h3>
          <div className="bars">
            {models.length ? models.map(m => (
              <div className="row" key={m.key}>
                <span className="lab">{m.key.replace('claude-', '')}</span>
                <div className="track"><div className="fill g" style={{ width: Math.max(2, 100 * m.tokens / mmax) + '%' }} /></div>
                <span className="val">{fmtN(m.tokens)} · {m.sessions}s</span>
              </div>
            )) : <div className="empty">no data</div>}
          </div>
        </div>
      </div>

      <div className="cols">
        <div className="panel">
          <h3>Skills used</h3>
          <div className="subhead">Read from your session transcripts — no setup.</div>
          {skills.length ? (
            <div className="bars">
              {skills.map(s => (
                <div className="row" key={s.name}>
                  <span className="lab">{s.name}</span>
                  <div className="track"><div className="fill" style={{ width: Math.max(2, 100 * s.count / smax) + '%' }} /></div>
                  <span className="val">{s.count}× · {ago(s.last_used)}</span>
                </div>
              ))}
            </div>
          ) : <div className="empty">No skill runs found yet.</div>}
        </div>
        <div className="panel">
          <h3>Subagent runs</h3>
          <div className="subhead">{env ? `${env.agent_runs} runs across ${env.agent_projects} project${env.agent_projects === 1 ? '' : 's'}` : ' '}</div>
          {agentsByProj.length ? (
            <div className="bars">
              {agentsByProj.map(a => (
                <div className="row" key={a.key}>
                  <span className="lab">{a.key}</span>
                  <div className="track"><div className="fill g" style={{ width: Math.max(2, 100 * a.count / amax) + '%' }} /></div>
                  <span className="val">{a.count}</span>
                </div>
              ))}
            </div>
          ) : <div className="empty">No subagents launched yet.</div>}
        </div>
      </div>
    </section>
  )
}
