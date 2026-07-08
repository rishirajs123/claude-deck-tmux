import { useMemo } from 'react'
import type { Stats, Day } from '../types'
import { fmtN } from '../format'

export default function Hero({ stats, daily }: { stats: Stats | null; daily: Day[] }) {
  const wave = useMemo(() => {
    const pts = (daily || []).slice(-30)
    if (pts.length < 2) return null
    const W = 640, H = 150, pad = 10
    const max = Math.max(1, ...pts.map(p => p.prompts))
    const step = W / (pts.length - 1)
    const xy = pts.map((p, i) => [i * step, H - pad - (p.prompts / max) * (H - 2 * pad)] as const)
    const line = xy.map((c, i) => (i ? 'L' : 'M') + c[0].toFixed(1) + ',' + c[1].toFixed(1)).join(' ')
    const area = `${line} L${W},${H} L0,${H} Z`
    const last = xy[xy.length - 1]
    return { line, area, lastX: last[0], lastY: last[1] }
  }, [daily])

  const running = stats?.running ?? 0
  const total = stats?.total ?? 0

  return (
    <>
      <div className="hero">
        <div className="lead">
          <div className="livebadge"><span className="beacon" /> Sessions live</div>
          <div className="bignum">{running}</div>
          <div className="subline">of <b>{total}</b> sessions on the deck</div>
        </div>
        <div className="wavewrap">
          <span className="wcap">Activity · last 30 days</span>
          {wave ? (
            <svg className="wave" viewBox="0 0 640 150" preserveAspectRatio="none">
              <defs>
                <linearGradient id="wg" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0" stopColor="#5fe0f0" stopOpacity="0.38" />
                  <stop offset="1" stopColor="#5fe0f0" stopOpacity="0" />
                </linearGradient>
              </defs>
              <path d={wave.area} fill="url(#wg)" />
              <path d={wave.line} fill="none" stroke="#5fe0f0" strokeWidth="2.5"
                style={{ filter: 'drop-shadow(0 0 5px rgba(95,224,240,.8))' }} />
              <circle className="nowdot" cx={wave.lastX} cy={wave.lastY} r="4.5" />
            </svg>
          ) : <div style={{ color: 'var(--faint)', alignSelf: 'center', fontFamily: 'var(--mono)', fontSize: 12 }}>collecting activity…</div>}
        </div>
      </div>
      <div className="reads">
        <div className="read"><span className="k">Prompts</span><span className="v">{fmtN(stats?.prompts ?? 0)}</span></div>
        <div className="read"><span className="k">Tokens</span><span className="v">{fmtN((stats?.tokens_in ?? 0) + (stats?.tokens_out ?? 0))}</span></div>
        <div className="read"><span className="k">Projects</span><span className="v">{stats?.projects ?? 0}</span></div>
      </div>
    </>
  )
}
