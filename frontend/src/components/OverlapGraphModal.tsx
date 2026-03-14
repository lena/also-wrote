import { useEffect, useRef, useState } from 'react'
import * as d3 from 'd3'
import { api } from '../api'
import type { OverlapGraphNode } from '../types'

interface GraphNode extends OverlapGraphNode {
  cardW: number
  cardH: number
  year: number | null
  x: number
  y: number
  targetY: number
}
interface GraphLink {
  source: GraphNode
  target: GraphNode
}

const IMAGE_BASE = 'https://image.tmdb.org/t/p/w154'
const writerLinkColors = ['#6366f1', '#8b5cf6', '#a855f7', '#d946ef', '#ec4899', '#f43f5e', '#0ea5e9', '#14b8a6', '#eab308', '#f97316']

function setupPanZoom(
  viewportEl: HTMLDivElement,
  containerEl: HTMLDivElement,
  graphW: number,
  graphH: number
): () => void {
  const cw = containerEl.offsetWidth
  const ch = containerEl.offsetHeight
  const minScale = Math.max(cw / graphW, ch / graphH)
  const maxScale = 3
  let scale = minScale
  let tx = (cw - graphW * scale) / (2 * scale)
  let ty = (ch - graphH * scale) / (2 * scale)

  function clampPan() {
    const viewW = graphW * scale
    const viewH = graphH * scale
    const marginTop = 20
    const curCw = containerEl.offsetWidth
    const rect = containerEl.getBoundingClientRect()
    const visibleBottom = typeof window !== 'undefined' ? Math.min(rect.bottom, window.innerHeight) : rect.bottom
    const visibleTop = typeof window !== 'undefined' ? Math.max(rect.top, 0) : rect.top
    const visibleCh = Math.max(0, visibleBottom - visibleTop)
    const maxTx = viewW >= curCw ? 0 : (curCw - viewW) / scale
    const minTx = viewW >= curCw ? (curCw - viewW) / scale : 0
    const maxTy = marginTop / scale
    // Use visible container height so when modal is scrolled we still allow pan to visible bottom
    const minTy = (visibleCh - viewH) / scale
    tx = Math.max(minTx, Math.min(maxTx, tx))
    ty = Math.max(minTy, Math.min(maxTy, ty))
  }

  function applyTransform() {
    clampPan()
    viewportEl.style.transform = `scale(${scale}) translate(${tx}px, ${ty}px)`
  }
  applyTransform()

  let lastDist = 0
  let lastCenterX = 0
  let lastCenterY = 0
  let lastSingleX = 0
  let lastSingleY = 0
  let totalDx = 0
  let totalDy = 0
  let panning = false
  const tapThreshold = 10

  function dist(t1: Touch, t2: Touch) {
    return Math.hypot(t2.clientX - t1.clientX, t2.clientY - t1.clientY)
  }
  function centerX(t1: Touch, t2: Touch) {
    return (t1.clientX + t2.clientX) / 2
  }
  function centerY(t1: Touch, t2: Touch) {
    return (t1.clientY + t2.clientY) / 2
  }

  const onTouchStart = (e: TouchEvent) => {
    if (e.touches.length === 2) {
      panning = true
      lastDist = dist(e.touches[0], e.touches[1])
      lastCenterX = centerX(e.touches[0], e.touches[1])
      lastCenterY = centerY(e.touches[0], e.touches[1])
    } else if (e.touches.length === 1) {
      totalDx = 0
      totalDy = 0
      panning = false
      lastSingleX = e.touches[0].clientX
      lastSingleY = e.touches[0].clientY
    }
  }

  const onTouchMove = (e: TouchEvent) => {
    if (e.touches.length === 2) {
      e.preventDefault()
      panning = true
      const d = dist(e.touches[0], e.touches[1])
      const cx = centerX(e.touches[0], e.touches[1])
      const cy = centerY(e.touches[0], e.touches[1])
      if (lastDist > 0) {
        const newScale = scale * (d / lastDist)
        scale = Math.max(minScale, Math.min(maxScale, newScale))
        const dx = (cx - lastCenterX) / scale
        const dy = (cy - lastCenterY) / scale
        tx += dx
        ty += dy
      }
      lastDist = d
      lastCenterX = cx
      lastCenterY = cy
      applyTransform()
    } else if (e.touches.length === 1) {
      const dx = e.touches[0].clientX - lastSingleX
      const dy = e.touches[0].clientY - lastSingleY
      totalDx += dx
      totalDy += dy
      if (!panning && (Math.abs(totalDx) > tapThreshold || Math.abs(totalDy) > tapThreshold)) {
        panning = true
        e.preventDefault()
        tx += totalDx / scale
        ty += totalDy / scale
        lastSingleX = e.touches[0].clientX
        lastSingleY = e.touches[0].clientY
        applyTransform()
      } else if (panning) {
        e.preventDefault()
        tx += dx / scale
        ty += dy / scale
        lastSingleX = e.touches[0].clientX
        lastSingleY = e.touches[0].clientY
        applyTransform()
      }
    }
  }

  const onTouchEnd = (e: TouchEvent) => {
    if (e.touches.length === 2) {
      lastDist = dist(e.touches[0], e.touches[1])
      lastCenterX = centerX(e.touches[0], e.touches[1])
      lastCenterY = centerY(e.touches[0], e.touches[1])
    } else if (e.touches.length === 1) {
      lastSingleX = e.touches[0].clientX
      lastSingleY = e.touches[0].clientY
    } else {
      lastDist = 0
    }
  }

  containerEl.addEventListener('touchstart', onTouchStart, { passive: true })
  containerEl.addEventListener('touchmove', onTouchMove, { passive: false })
  containerEl.addEventListener('touchend', onTouchEnd, { passive: true })

  return () => {
    containerEl.removeEventListener('touchstart', onTouchStart)
    containerEl.removeEventListener('touchmove', onTouchMove)
    containerEl.removeEventListener('touchend', onTouchEnd)
  }
}

interface OverlapGraphModalProps {
  onClose: () => void
  userId?: number
}

type GraphDataRef = {
  linkEls: d3.Selection<SVGLineElement, { source: GraphNode; target: GraphNode }, d3.BaseType, unknown>
  nodeWrap: d3.Selection<HTMLDivElement, GraphNode, d3.BaseType, unknown>
  writerColorScale: Record<string, string>
  showToWriters: Record<string, string[]>
  showWriterCount: Record<string, number>
  writerToShows: Record<string, string[]>
}

export default function OverlapGraphModal({ onClose, userId }: OverlapGraphModalProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const viewportRef = useRef<HTMLDivElement>(null)
  const linksSvgRef = useRef<SVGSVGElement>(null)
  const nodesDivRef = useRef<HTMLDivElement>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const simRef = useRef<d3.Simulation<GraphNode, GraphLink> | null>(null)
  const graphDataRef = useRef<GraphDataRef | null>(null)
  const panZoomCleanupRef = useRef<(() => void) | null>(null)
  const [writerList, setWriterList] = useState<{ id: string; name: string }[]>([])
  const [maxOverlap, setMaxOverlap] = useState(1)
  const [hiddenWriters, setHiddenWriters] = useState<string[]>([])
  const [minOverlap, setMinOverlap] = useState(1)

  useEffect(() => {
    if (!containerRef.current || !viewportRef.current || !linksSvgRef.current || !nodesDivRef.current) return

    setLoading(true)
    setError(null)
    api
      .overlapGraph(userId)
      .then((data) => {
        setLoading(false)
        const writerNodes = (data.nodes || []).filter((d) => d.type === 'writer')
        const showNodes = (data.nodes || []).filter((d) => d.type === 'show')
        const maxWriterCount = Math.max(...showNodes.map((d) => d.writer_count || 1), 1)
        const minWriterCount = Math.min(...showNodes.map((d) => d.writer_count || 1), 1)
        const writerColorScale: Record<string, string> = {}
        writerNodes.forEach((d, i) => {
          writerColorScale[d.id] = writerLinkColors[i % writerLinkColors.length]
        })

        const showCardMinW = 72,
          showCardMinH = 100,
          showCardMaxW = 120,
          showCardMaxH = 168,
          writerCardMinW = 72,
          writerCardMaxW = 240,
          writerCardH = 40,
          padding = 16,
          axisWidth = 44

        const nodes: GraphNode[] = (data.nodes || []).map((d) => {
          const isWriter = d.type === 'writer'
          let cardW: number, cardH: number
          if (isWriter) {
            const nameLen = (d.name || '').length
            cardW = Math.min(writerCardMaxW, Math.max(writerCardMinW, Math.ceil(nameLen * 7.5)))
            cardH = writerCardH
          } else {
            const wc = typeof d.writer_count === 'number' ? d.writer_count : 1
            const t = maxWriterCount <= minWriterCount ? 1 : (wc - minWriterCount) / (maxWriterCount - minWriterCount)
            cardW = Math.round(showCardMinW + t * (showCardMaxW - showCardMinW))
            cardH = Math.round(showCardMinH + t * (showCardMaxH - showCardMinH))
          }
          let year: number | null = null
          if (d.type === 'show' && d.first_air_date) {
            const match = String(d.first_air_date).match(/^(\d{4})/)
            if (match) year = parseInt(match[1], 10)
          }
          return {
            ...d,
            id: String(d.id),
            cardW,
            cardH,
            year,
            x: 0,
            y: 0,
            targetY: 0,
          }
        })

        const links: { source: string; target: string }[] = (data.edges || []).map((e) => ({
          source: String(e.source),
          target: String(e.target),
        }))

        if (nodes.length === 0) {
          setError('No shows to display. Add more favorite writers.')
          return
        }

        const container = containerRef.current
        const viewport = viewportRef.current
        const linksSvg = linksSvgRef.current
        const nodesDiv = nodesDivRef.current
        if (!container || !viewport || !linksSvg || !nodesDiv) return

        const isMobile = typeof window !== 'undefined' && window.matchMedia('(max-width: 767px)').matches
        let graphWidth: number
        let graphHeight: number
        if (isMobile) {
          graphWidth = 1200
          graphHeight = 900
          viewport.style.width = `${graphWidth}px`
          viewport.style.height = `${graphHeight}px`
          viewport.style.transformOrigin = '0 0'
        } else {
          graphWidth = container.offsetWidth
          graphHeight = container.offsetHeight
          viewport.style.width = '100%'
          viewport.style.height = '100%'
          viewport.style.transform = ''
        }
        const width = graphWidth
        const height = graphHeight
        const centerX = axisWidth + (width - axisWidth - 2 * padding) / 2
        const centerY = height / 2

        const showYears = nodes.filter((n) => n.type === 'show' && n.year != null).map((n) => n.year as number)
        const minYear = showYears.length ? Math.min(...showYears) : 1990
        const maxYear = showYears.length ? Math.max(...showYears) : new Date().getFullYear()
        const yearMin = Math.floor(minYear / 10) * 10
        const yearMax = Math.ceil(maxYear / 10) * 10
        const yearScale = d3.scaleLinear().domain([yearMax, yearMin]).range([padding, height - padding])
        nodes.forEach((n) => {
          n.targetY = n.type === 'show' && n.year != null ? yearScale(n.year) : centerY
        })

        const nodeById = new Map(nodes.map((n) => [n.id, n]))
        const linkData: { source: GraphNode; target: GraphNode }[] = links
          .map((l) => {
            const src = nodeById.get(l.source)
            const tgt = nodeById.get(l.target)
            return src && tgt ? { source: src, target: tgt } : null
          })
          .filter((x): x is { source: GraphNode; target: GraphNode } => x != null)

        const writerIdSet = new Set(nodes.filter((n) => n.type === 'writer').map((n) => n.id))
        const showToWriters: Record<string, string[]> = {}
        const showWriterCount: Record<string, number> = {}
        const writerToShows: Record<string, string[]> = {}
        nodes.forEach((n) => {
          if (n.type === 'show') showWriterCount[n.id] = n.writer_count || 0
        })
        linkData.forEach((link) => {
          const writerId = writerIdSet.has(link.source.id) ? link.source.id : link.target.id
          const showId = writerIdSet.has(link.source.id) ? link.target.id : link.source.id
          if (!showToWriters[showId]) showToWriters[showId] = []
          if (!showToWriters[showId].includes(writerId)) showToWriters[showId].push(writerId)
          if (!writerToShows[writerId]) writerToShows[writerId] = []
          if (!writerToShows[writerId].includes(showId)) writerToShows[writerId].push(showId)
        })

        setWriterList(writerNodes.map((w) => ({ id: w.id, name: w.name || '' })))
        setMaxOverlap(Math.max(1, maxWriterCount))
        setHiddenWriters([])
        setMinOverlap(1)

        const maxPriority = Math.max(...nodes.map((d) => d.priority || 0), 1)
        nodes.forEach((n, i) => {
          const cols = Math.ceil(Math.sqrt(nodes.length))
          n.x = centerX + (i % cols - cols / 2) * 160
          n.y = n.targetY ?? centerY
        })

        const linkForce = d3.forceLink<GraphNode, GraphLink>(linkData).id((d) => (d as GraphNode).id).distance(180).strength(0.2)
        const chargeForce = d3.forceManyBody().strength(-520)
        const centerForce = d3.forceCenter(centerX, centerY).strength(0.02)
        const pullStrength = 0.0004
        const xForce = d3.forceX(centerX).strength((d) => pullStrength * (((d as GraphNode).priority || 0) / maxPriority))
        const yForce = d3.forceY((d) => (d as GraphNode).targetY ?? centerY).strength((d) => ((d as GraphNode).type === 'show' ? 0.2 : 0.02))
        const collideForce = d3.forceCollide((d) => Math.max((d as GraphNode).cardW, (d as GraphNode).cardH) / 2 + 16).iterations(2)

        const sim = d3
          .forceSimulation(nodes)
          .force('link', linkForce)
          .force('charge', chargeForce)
          .force('center', centerForce)
          .force('x', xForce)
          .force('y', yForce)
          .force('collision', collideForce)
        simRef.current = sim

        if (!linksSvg) return
        linksSvg.setAttribute('viewBox', `0 0 ${width} ${height}`)
        linksSvg.innerHTML = ''
        const axisG = d3.select(linksSvg).append('g').attr('class', 'axis').attr('aria-hidden', 'true')
        for (let y = yearMax; y >= yearMin; y -= 10) {
          axisG.append('text').attr('x', 10).attr('y', yearScale(y)).attr('fill', '#64748b').attr('font-size', '11px').attr('font-weight', '500').attr('dominant-baseline', 'middle').text(String(y))
        }

        const linkEls = d3
          .select(linksSvg)
          .append('g')
          .selectAll<SVGLineElement, { source: GraphNode; target: GraphNode }>('line')
          .data(linkData)
          .join('line')
          .attr('stroke', (d) => {
            const writerId = d.source.type === 'writer' ? d.source.id : d.target.id
            return writerColorScale[writerId] || '#6366f1'
          })
          .attr('stroke-width', 2)
          .attr('stroke-opacity', 0.85)

        if (!nodesDiv) return
        nodesDiv.innerHTML = ''
        const nodeWrap = d3
          .select(nodesDiv)
          .selectAll<HTMLDivElement, GraphNode>('div')
          .data(nodes)
          .join('div')
          .attr('class', 'absolute pointer-events-auto')
          .style('width', (d) => `${d.cardW}px`)
          .style('height', (d) => `${d.cardH}px`)
          .style('left', (d) => `${d.x - d.cardW / 2}px`)
          .style('top', (d) => `${d.y - d.cardH / 2}px`)

        nodeWrap.each(function (d) {
          const el = d3.select(this)
          if (d.type === 'writer') {
            const personId = d.id.replace(/^w-/, '')
            el.append('a')
              .attr('href', `/writer?id=${personId}`)
              .attr('class', 'flex items-center justify-center w-full h-full rounded-xl px-2.5 py-1 bg-violet-500/20 border border-violet-300/50 shadow hover:bg-violet-500/30 hover:border-violet-400 transition text-white whitespace-nowrap')
              .append('span')
              .attr('class', 'text-xs font-semibold')
              .text(d.name || '')
          } else {
            const showId = d.id.replace(/^s-/, '')
            el.append('a')
              .attr('href', `/show?id=${showId}`)
              .attr('class', 'flex flex-col w-full h-full rounded-xl overflow-hidden bg-slate-700 shadow-lg hover:shadow-indigo-500/20 hover:ring-2 hover:ring-indigo-400 transition')
            const inner = el.select('a')
            if (d.poster_path) {
              inner.append('img').attr('src', IMAGE_BASE + d.poster_path).attr('alt', d.name || '').attr('class', 'w-full flex-1 min-h-0 object-cover')
            } else {
              inner.append('div').attr('class', 'w-full flex-1 min-h-0 bg-slate-600 flex items-center justify-center text-slate-400 text-2xl').text('?')
            }
            inner.append('p').attr('class', 'p-1.5 text-xs font-semibold text-white truncate shrink-0').text(d.name || '')
          }
        })

        graphDataRef.current = {
          linkEls,
          nodeWrap,
          writerColorScale,
          showToWriters,
          showWriterCount,
          writerToShows,
        }

        const clamp = (x: number, lo: number, hi: number) => Math.max(lo, Math.min(hi, x))
        const minCardLeftEdge = axisWidth + 20
        const bottomMargin = 24

        function ticked() {
          nodes.forEach((d) => {
            const halfW = d.cardW / 2,
              halfH = d.cardH / 2
            d.x = clamp(d.x, minCardLeftEdge + halfW, width - padding - halfW)
            d.y = clamp(d.y, padding + halfH, height - padding - halfH - bottomMargin)
          })
          linkEls.attr('x1', (d) => d.source.x).attr('y1', (d) => d.source.y).attr('x2', (d) => d.target.x).attr('y2', (d) => d.target.y)
          nodeWrap.style('left', (d) => `${d.x - d.cardW / 2}px`).style('top', (d) => `${d.y - d.cardH / 2}px`)
        }
        sim.on('tick', ticked)

        if (isMobile) {
          panZoomCleanupRef.current = setupPanZoom(viewport, container, graphWidth, graphHeight)
        }
      })
      .catch(() => {
        setLoading(false)
        setError('Could not load graph.')
      })

    return () => {
      panZoomCleanupRef.current?.()
      panZoomCleanupRef.current = null
      simRef.current?.stop()
      simRef.current = null
      graphDataRef.current = null
    }
  }, [userId])

  useEffect(() => {
    const g = graphDataRef.current
    if (!g) return
    const hiddenSet = new Set(hiddenWriters)
    const showPassesOverlap = (d: GraphNode) => ((d.writer_count ?? 0) as number) >= minOverlap
    const writerPassesOverlap = (writerId: string) => {
      const showIds = g.writerToShows[writerId]
      if (!showIds?.length) return false
      return showIds.some((showId) => (g.showWriterCount[showId] ?? 0) >= minOverlap)
    }
    g.linkEls.attr('stroke-opacity', (d) => {
      const writerId = d.source.type === 'writer' ? d.source.id : d.target.id
      const showNode = d.source.type === 'show' ? d.source : d.target
      if (hiddenSet.has(writerId)) return 0
      if (!showPassesOverlap(showNode) || !writerPassesOverlap(writerId)) return 0
      return 0.85
    })
    g.nodeWrap.style('opacity', (d) => {
      if (d.type === 'writer') {
        if (hiddenSet.has(d.id)) return '0'
        return writerPassesOverlap(d.id) ? '1' : '0'
      }
      if (!showPassesOverlap(d)) return '0'
      const writers = g.showToWriters[d.id]
      if (!writers?.length) return '1'
      const allHidden = writers.every((wid) => hiddenSet.has(wid))
      return allHidden ? '0' : '1'
    })
    g.nodeWrap.style('pointer-events', (d) => {
      if (d.type === 'writer') {
        if (hiddenSet.has(d.id)) return 'none'
        return writerPassesOverlap(d.id) ? 'auto' : 'none'
      }
      if (!showPassesOverlap(d)) return 'none'
      const writers = g.showToWriters[d.id]
      if (!writers?.length) return 'auto'
      const allHidden = writers.every((wid) => hiddenSet.has(wid))
      return allHidden ? 'none' : 'auto'
    })
  }, [hiddenWriters, minOverlap])

  useEffect(() => {
    document.body.classList.add('modal-open')
    return () => document.body.classList.remove('modal-open')
  }, [])

  const toggleWriter = (id: string) => {
    setHiddenWriters((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]))
  }

  const hiddenSet = new Set(hiddenWriters)
  const writerPassesOverlapForLegend = (writerId: string) => {
    const g = graphDataRef.current
    if (!g) return true
    const showIds = g.writerToShows[writerId]
    if (!showIds?.length) return false
    return showIds.some((showId) => (g.showWriterCount[showId] ?? 0) >= minOverlap)
  }
  const legendVisible = (writerId: string) => !hiddenSet.has(writerId) && writerPassesOverlapForLegend(writerId)

  return (
    <div className="fixed inset-0 z-[100]" aria-hidden="false">
      <div className="absolute inset-0 bg-black/70 backdrop-blur-sm" onClick={onClose} onKeyDown={(e) => e.key === 'Escape' && onClose()} aria-hidden="true" />
      <div className="absolute inset-x-4 inset-y-2 md:inset-x-8 md:inset-y-4 lg:inset-12 bg-slate-800 rounded-2xl shadow-2xl flex flex-col overflow-y-auto overflow-x-hidden min-h-0">
        <div className="flex items-center justify-between px-4 py-1.5 md:py-2 border-b border-slate-600 shrink-0 bg-slate-800">
          <h2 className="text-base md:text-xl font-bold text-white">Writer-Series Overlap Graph</h2>
          <button
            type="button"
            onClick={onClose}
            className="flex p-2 min-w-[40px] min-h-[40px] md:p-3 md:min-w-[48px] md:min-h-[48px] items-center justify-center rounded-lg bg-slate-600 text-white font-bold text-base md:text-lg border-2 border-slate-500 hover:bg-slate-500 active:bg-slate-400 transition touch-manipulation shrink-0"
            aria-label="Close"
          >
            ✕
          </button>
        </div>
        <div ref={containerRef} className="flex-1 min-h-[400px] relative w-full overflow-hidden touch-none">
          {loading && (
            <div className="absolute inset-0 flex items-center justify-center text-slate-400 bg-slate-800/90 z-10">Loading…</div>
          )}
          {error && (
            <div className="absolute inset-0 flex items-center justify-center text-red-400 bg-slate-800/90 z-10">{error}</div>
          )}
          <div ref={viewportRef} className="absolute left-0 top-0 w-full h-full will-change-transform">
            <svg ref={linksSvgRef} className="absolute inset-0 w-full h-full pointer-events-none" aria-hidden="true" />
            <div ref={nodesDivRef} className="absolute inset-0 pointer-events-none" />
          </div>
        </div>
        {writerList.length > 0 && (
          <div className="flex flex-col sm:flex-row sm:items-start gap-3 px-4 py-1.5 border-t border-slate-600 shrink-0 bg-slate-800/50 text-slate-300 text-xs">
            <div className="flex flex-wrap items-center gap-3 min-w-0 flex-1 max-h-[6rem] overflow-y-auto overflow-x-hidden episode-list-scroll md:max-h-none md:overflow-visible">
              {writerList.map((w) => {
                const color = graphDataRef.current?.writerColorScale[w.id] ?? '#6366f1'
                const visible = legendVisible(w.id)
                return (
                  <button
                    key={w.id}
                    type="button"
                    onClick={() => toggleWriter(w.id)}
                    className={`inline-flex items-center gap-1.5 cursor-pointer rounded px-1.5 py-0.5 hover:bg-slate-700 transition text-left border border-transparent hover:border-slate-600 shrink-0 ${!visible ? 'opacity-50' : ''}`}
                    aria-pressed={hiddenSet.has(w.id)}
                    title={hiddenSet.has(w.id) ? 'Show writer' : 'Hide writer'}
                  >
                    <span className="legend-dot rounded-full w-3 h-3 shrink-0" style={{ background: color }} />
                    <span className="legend-name truncate max-w-[120px]">{w.name}</span>
                  </button>
                )
              })}
            </div>
            <div className="flex flex-nowrap items-center gap-2 min-w-0 overflow-x-auto scrollbar-hide-x shrink-0">
              <span className="text-slate-400 shrink-0 font-medium">Overlap:</span>
              {Array.from({ length: maxOverlap }, (_, i) => i + 1).map((n) => (
                <label key={n} className="inline-flex items-center gap-1.5 cursor-pointer select-none shrink-0">
                  <input
                    type="radio"
                    name="overlap-min"
                    value={n}
                    checked={minOverlap === n}
                    onChange={() => setMinOverlap(n)}
                    className="rounded border-slate-500 text-indigo-600 focus:ring-indigo-500"
                  />
                  <span>{n}+</span>
                </label>
              ))}
            </div>
          </div>
        )}
        <p className="text-slate-400 text-sm px-4 py-2 border-t border-slate-600 shrink-0">
          Shows your favorite writers have worked on. Line color = writer. Larger show cards = more of your favorite writers worked on it.
        </p>
      </div>
    </div>
  )
}
