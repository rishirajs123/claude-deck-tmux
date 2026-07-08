import { useMemo, useRef, useState } from 'react'
import type { Stats, Day } from '../types'
import { fmtN } from '../format'

export default function Hero({ stats, daily }: { stats: Stats | null; daily: Day[] }) {
  const svgRef = useRef<SVGSVGElement>(null)
  const [hover, setHover] = useState<number | null>(null)

  const wave = useMemo(() => {
    const pts = (daily || []).slice(-30)
    if (pts.length < 2) return null
    const W = 640, H = 150, pad = 10
    const max = Math.max(1, ...pts.map(p => p.prompts))
    const step = W / (pts.length - 1)
    const xy = pts.map((p, i) => [i * step, H - pad - (p.prompts / max) * (H - 2 * pad)] as const)
    const line = xy.map((c, i) => (i ? 'L' : 'M') + c[0].toFixed(1) + ',' + c[1].toFixed(1)).join(' ')
    const area = `${line} L${W},${H} L0,${H} Z`
    return { pts, xy, line, area }
  }, [daily])

  const onMove = (e: React.MouseEvent) => {
    if (!wave || !svgRef.current) return
    const r = svgRef.current.getBoundingClientRect()
    const frac = (e.clientX - r.left) / r.width
    setHover(Math.max(0, Math.min(wave.pts.length - 1, Math.round(frac * (wave.pts.length - 1)))))
  }

  const running = stats?.running ?? 0
  const total = stats?.total ?? 0
  const hp = wave && hover != null ? wave.pts[hover] : null
  const hxy = wave && hover != null ? wave.xy[hover] : null
  const fmtDay = (d: string) => new Date(d + 'T00:00:00').toLocaleDateString(undefined, { month: 'short', day: 'numeric' })

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
            <>
              <svg ref={svgRef} className="wave" viewBox="0 0 640 150" preserveAspectRatio="none"
                onMouseMove={onMove} onMouseLeave={() => setHover(null)}>
                <defs>
                  <linearGradient id="wg" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0" stopColor="#5fe0f0" stopOpacity="0.38" />
                    <stop offset="1" stopColor="#5fe0f0" stopOpacity="0" />
                  </linearGradient>
                </defs>
                <path d={wave.area} fill="url(#wg)" />
                <path d={wave.line} fill="none" stroke="#5fe0f0" strokeWidth="2.5"
                  style={{ filter: 'drop-shadow(0 0 5px rgba(95,224,240,.8))' }} />
                <circle className="nowdot" cx={wave.xy[wave.xy.length - 1][0]} cy={wave.xy[wave.xy.length - 1][1]} r="4.5" />
                {hxy && (
                  <>
                    <line x1={hxy[0]} y1="0" x2={hxy[0]} y2="150" stroke="rgba(255,255,255,.18)" strokeWidth="1" />
                    <circle cx={hxy[0]} cy={hxy[1]} r="5" fill="#5fe0f0" style={{ filter: 'drop-shadow(0 0 6px #5fe0f0)' }} />
                  </>
                )}
              </svg>
              {hp && hxy && (
                <div className="wtip" style={{ left: `${(hxy[0] / 640) * 100}%` }}>
                  <b>{hp.prompts}</b> prompt{hp.prompts === 1 ? '' : 's'} · {fmtDay(hp.day)}
                </div>
              )}
            </>
          ) : (
            <div style={{ color: 'var(--faint)', alignSelf: 'center', fontFamily: 'var(--mono)', fontSize: 12 }}>collecting activity…</div>
          )}
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
