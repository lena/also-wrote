import { useCallback, useEffect, useRef, useState } from 'react'

export default function EpisodeListWithFades({ children }: { children: React.ReactNode }) {
  const scrollRef = useRef<HTMLUListElement>(null)
  const [showTopFade, setShowTopFade] = useState(false)
  const [showBottomFade, setShowBottomFade] = useState(false)

  const updateFades = useCallback(() => {
    const el = scrollRef.current
    if (!el) return
    const atTop = el.scrollTop <= 1
    const atBottom = el.scrollTop + el.clientHeight >= el.scrollHeight - 1
    setShowTopFade(!atTop)
    setShowBottomFade(!atBottom)
  }, [])

  useEffect(() => {
    const el = scrollRef.current
    if (!el) return
    el.addEventListener('scroll', updateFades)
    const ro = typeof ResizeObserver !== 'undefined' ? new ResizeObserver(updateFades) : null
    ro?.observe(el)
    updateFades()
    return () => {
      el.removeEventListener('scroll', updateFades)
      ro?.disconnect()
    }
  }, [updateFades])

  return (
    <div className="relative max-h-48 episode-list-wrap">
      <ul ref={scrollRef} className="space-y-1 max-h-48 overflow-y-auto episode-list-scroll pr-2">
        {children}
      </ul>
      <div
        className="episode-fade episode-fade-top absolute top-0 left-0 right-2 h-8 pointer-events-none bg-gradient-to-b from-white to-transparent transition-opacity duration-200"
        style={{ opacity: showTopFade ? 1 : 0 }}
        aria-hidden="true"
      />
      <div
        className="episode-fade episode-fade-bottom absolute bottom-0 left-0 right-2 h-8 pointer-events-none bg-gradient-to-t from-white to-transparent transition-opacity duration-200"
        style={{ opacity: showBottomFade ? 1 : 0 }}
        aria-hidden="true"
      />
    </div>
  )
}
