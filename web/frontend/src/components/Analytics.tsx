import { useMemo } from 'react'
import type { Analytics as AnalyticsData } from '../types'
import { fmtN } from '../format'

export default function Analytics({ data }: { data: AnalyticsData | null }) {
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

  return (
    <section>
      <div className="panel">
        <h3>Activity — prompts per day</h3>
        <div className="heat">
          {heat.cells.map(c => <div key={c.key} className={'c' + (c.lv ? ' l' + c.lv : '')} title={`${c.key}: ${c.n} prompts`} />)}
        </div>
        <div className="legend">Less
          <span className="c" style={{ background: '#1a1f2b' }} /><span className="c l1" /><span className="c l2" /><span className="c l3" /><span className="c l4" /> More
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
    </section>
  )
}
