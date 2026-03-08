import { ref, onUnmounted } from 'vue'

export type ViolationType =
  | 'tab_switch'
  | 'blur_extended'
  | 'multi_tab'
  | 'popup_detected'
  | 'background_detected'
  | 'external_paste'
  | 'alt_tab'
  | 'fullscreen_exit'
  | 'window_resize'
  | 'focus_lost'

export interface ViolationEvent {
  type: ViolationType
  description: string
}

interface AntiCheatOptions {
  onViolation: (event: ViolationEvent) => void
  /** Called when tab becomes hidden — use to flush pending work */
  onBlur?: () => void
  /** Cooldown in ms between violations of the same type (default 5000) */
  cooldown?: number
  /** Enable fullscreen enforcement (default true). If false, skips fullscreen request and exit detection. */
  enableFullscreen?: boolean
}

/**
 * Anti-cheat composable with 9 detection methods:
 * 1. Visibility + Blur duration tracking
 * 2. Multi-tab detection via BroadcastChannel
 * 3. window.open / popup detection
 * 4. requestAnimationFrame drift (detects split-screen / background)
 * 5. Clipboard paste monitoring (external paste into text fields)
 * 6. Alt+Tab / Cmd+Tab keystroke detection
 * 7. Fullscreen enforcement (floating app exits fullscreen → detected)
 * 8. Window resize monitoring (floating app / split-screen shrinks viewport)
 * 9. Window focus/blur (floating app steals focus without hiding tab)
 */
export function useAntiCheat(options: AntiCheatOptions) {
  const cooldown = options.cooldown ?? 5000
  const enableFullscreen = options.enableFullscreen ?? true
  const lastViolationTimes: Partial<Record<ViolationType, number>> = {}
  const violationCount = ref(0)
  const violationWarning = ref(false)
  const latestMessage = ref('')
  const isFullscreen = ref(false)

  function emit(type: ViolationType, description: string) {
    const now = Date.now()
    if ((now - (lastViolationTimes[type] ?? 0)) < cooldown) return
    lastViolationTimes[type] = now
    violationCount.value++
    violationWarning.value = true
    latestMessage.value = description
    options.onViolation({ type, description })
  }

  // ── 1. Visibility + Blur Duration Tracking ──
  let blurStart = 0

  function onVisibilityChange() {
    if (document.hidden) {
      blurStart = Date.now()
      options.onBlur?.()
      emit('tab_switch', 'Peserta berpindah tab/window')
    } else if (blurStart > 0) {
      const duration = Math.round((Date.now() - blurStart) / 1000)
      if (duration >= 3) {
        emit('blur_extended', `Tab tidak aktif selama ${duration} detik`)
      }
      blurStart = 0
    }
  }

  // ── 2. Multi-Tab Detection via BroadcastChannel ──
  let bc: BroadcastChannel | null = null
  const tabId = `${Date.now()}-${Math.random().toString(36).slice(2)}`

  function setupBroadcastChannel() {
    if (typeof BroadcastChannel === 'undefined') return
    bc = new BroadcastChannel('patra_exam_tab')
    bc.postMessage({ type: 'tab_open', tabId })
    bc.onmessage = (e) => {
      if (e.data?.type === 'tab_open' && e.data.tabId !== tabId) {
        emit('multi_tab', 'Terdeteksi membuka tab ujian lain')
        bc?.postMessage({ type: 'tab_exists', tabId })
      }
      if (e.data?.type === 'tab_exists' && e.data.tabId !== tabId) {
        emit('multi_tab', 'Terdeteksi membuka tab ujian lain')
      }
    }
  }

  // ── 3. window.open / Popup Detection ──
  const origOpen = window.open
  function patchWindowOpen() {
    window.open = function (...args: Parameters<typeof window.open>) {
      emit('popup_detected', 'Terdeteksi membuka popup/window baru')
      return origOpen.apply(window, args)
    }
  }
  function restoreWindowOpen() {
    window.open = origOpen
  }

  // ── 4. requestAnimationFrame Drift Detection ──
  let rafId = 0
  let lastRafTime = 0
  const RAF_DRIFT_THRESHOLD = 2000

  function rafLoop(timestamp: number) {
    if (lastRafTime > 0) {
      const delta = timestamp - lastRafTime
      if (delta > RAF_DRIFT_THRESHOLD) {
        emit('background_detected', `Tab tidak aktif (drift ${Math.round(delta)}ms)`)
      }
    }
    lastRafTime = timestamp
    rafId = requestAnimationFrame(rafLoop)
  }

  // ── 5. Clipboard Paste Monitoring ──
  function onPaste(e: ClipboardEvent) {
    const target = e.target as HTMLElement
    if (target.tagName !== 'TEXTAREA' && target.tagName !== 'INPUT') return
    const text = e.clipboardData?.getData('text/plain') ?? ''
    if (text.length > 50) {
      emit('external_paste', `Paste teks eksternal (${text.length} karakter)`)
    }
  }

  // ── 6. Alt+Tab / Cmd+Tab Keystroke Detection ──
  function onKeydown(e: KeyboardEvent) {
    if ((e.altKey && e.key === 'Tab') || (e.metaKey && e.key === 'Tab')) {
      emit('alt_tab', 'Terdeteksi shortcut pindah window (Alt+Tab / Cmd+Tab)')
    }
    if (e.metaKey && e.key === '`') {
      emit('alt_tab', 'Terdeteksi shortcut pindah window (Cmd+`)')
    }
  }

  // ── 7. Fullscreen Enforcement (Android floating app detection) ──
  // When a floating app appears on Android, fullscreen mode gets forcibly exited.
  // We request fullscreen on start and detect when it's exited unexpectedly.
  let fullscreenUserExited = false

  async function requestFullscreen() {
    if (!enableFullscreen) return
    const el = document.documentElement
    try {
      if (el.requestFullscreen) {
        await el.requestFullscreen({ navigationUI: 'hide' })
      } else if ((el as any).webkitRequestFullscreen) {
        (el as any).webkitRequestFullscreen()
      }
      isFullscreen.value = true
      fullscreenUserExited = false
    } catch {
      // Fullscreen not supported or denied — skip
    }
  }

  function onFullscreenChange() {
    if (!enableFullscreen) return
    const active = !!(document.fullscreenElement || (document as any).webkitFullscreenElement)
    isFullscreen.value = active
    if (!active && !fullscreenUserExited) {
      emit('fullscreen_exit', 'Keluar dari mode fullscreen (kemungkinan floating app)')
      // Try to re-enter fullscreen after a short delay
      setTimeout(() => {
        if (!fullscreenUserExited) requestFullscreen()
      }, 1000)
    }
  }

  // ── 8. Window Resize Monitoring (floating app / split-screen shrinks viewport) ──
  // On Android, floating apps or split-screen reduce the available viewport.
  // We record the initial dimensions and flag significant shrinkage.
  let initialWidth = 0
  let initialHeight = 0
  const RESIZE_SHRINK_THRESHOLD = 0.75 // flag if viewport shrinks to <75% of original
  let resizeDebounce: ReturnType<typeof setTimeout> | null = null

  function onResize() {
    if (resizeDebounce) clearTimeout(resizeDebounce)
    resizeDebounce = setTimeout(() => {
      if (initialWidth === 0 || initialHeight === 0) return
      const wRatio = window.innerWidth / initialWidth
      const hRatio = window.innerHeight / initialHeight
      // Keyboard popup on mobile reduces height — only flag if width also shrinks,
      // or if height shrinks dramatically (split-screen)
      if (wRatio < RESIZE_SHRINK_THRESHOLD) {
        emit('window_resize', `Lebar layar menyusut ke ${Math.round(wRatio * 100)}% (split-screen / floating app)`)
      } else if (hRatio < 0.5 && wRatio < 0.95) {
        // Height halved AND width slightly changed — likely split-screen, not keyboard
        emit('window_resize', `Layar menyusut ke ${Math.round(wRatio * 100)}%x${Math.round(hRatio * 100)}% (split-screen)`)
      }
    }, 500)
  }

  // ── 9. Window Focus/Blur (floating app steals focus without hiding tab) ──
  // On Android, floating apps can steal window focus without triggering visibilitychange.
  // Grace period: 3 seconds before counting as a violation.
  let focusLostStart = 0
  let focusLostGraceTimer: ReturnType<typeof setTimeout> | null = null

  function onWindowBlur() {
    focusLostStart = Date.now()
    // Grace period: wait 3 seconds before emitting violation
    if (focusLostGraceTimer) clearTimeout(focusLostGraceTimer)
    focusLostGraceTimer = setTimeout(() => {
      // Only emit if still blurred after 3 seconds
      if (focusLostStart > 0) {
        emit('focus_lost', 'Window kehilangan fokus (kemungkinan floating app)')
      }
      focusLostGraceTimer = null
    }, 3000)
  }

  function onWindowFocus() {
    // If focus returns within grace period, cancel the violation
    if (focusLostGraceTimer) {
      clearTimeout(focusLostGraceTimer)
      focusLostGraceTimer = null
    }
    if (focusLostStart > 0) {
      const duration = Math.round((Date.now() - focusLostStart) / 1000)
      if (duration >= 3) {
        emit('focus_lost', `Window tidak fokus selama ${duration} detik`)
      }
      focusLostStart = 0
    }
  }

  // ── Lifecycle ──
  function start() {
    // Record initial viewport dimensions for resize detection
    initialWidth = window.innerWidth
    initialHeight = window.innerHeight

    document.addEventListener('visibilitychange', onVisibilityChange)
    setupBroadcastChannel()
    patchWindowOpen()
    rafId = requestAnimationFrame(rafLoop)
    document.addEventListener('paste', onPaste, true)
    window.addEventListener('keydown', onKeydown, true)
    if (enableFullscreen) {
      document.addEventListener('fullscreenchange', onFullscreenChange)
      document.addEventListener('webkitfullscreenchange', onFullscreenChange)
    }
    window.addEventListener('resize', onResize)
    window.addEventListener('blur', onWindowBlur)
    window.addEventListener('focus', onWindowFocus)

    // Request fullscreen on mobile devices (only if enabled)
    requestFullscreen()
  }

  function stop() {
    fullscreenUserExited = true
    document.removeEventListener('visibilitychange', onVisibilityChange)
    bc?.close()
    bc = null
    restoreWindowOpen()
    if (rafId) cancelAnimationFrame(rafId)
    rafId = 0
    lastRafTime = 0
    document.removeEventListener('paste', onPaste, true)
    window.removeEventListener('keydown', onKeydown, true)
    document.removeEventListener('fullscreenchange', onFullscreenChange)
    document.removeEventListener('webkitfullscreenchange', onFullscreenChange)
    window.removeEventListener('resize', onResize)
    window.removeEventListener('blur', onWindowBlur)
    window.removeEventListener('focus', onWindowFocus)
    if (resizeDebounce) clearTimeout(resizeDebounce)
    if (focusLostGraceTimer) { clearTimeout(focusLostGraceTimer); focusLostGraceTimer = null }

    // Exit fullscreen if still active
    if (document.fullscreenElement) {
      document.exitFullscreen().catch(() => {})
    }
  }

  onUnmounted(stop)

  return {
    violationCount,
    violationWarning,
    latestMessage,
    isFullscreen,
    start,
    stop,
  }
}
