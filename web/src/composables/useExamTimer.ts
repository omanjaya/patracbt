import { ref, computed, watch, onUnmounted, type Ref } from 'vue'

/**
 * Play a short beep using Web Audio API (no external audio file needed).
 * @param frequency - Hz (e.g. 440 for A4, 880 for A5)
 * @param duration - milliseconds
 */
function playBeep(frequency: number, duration: number) {
  try {
    const ctx = new (window.AudioContext || (window as any).webkitAudioContext)()
    const oscillator = ctx.createOscillator()
    const gain = ctx.createGain()
    oscillator.type = 'sine'
    oscillator.frequency.setValueAtTime(frequency, ctx.currentTime)
    gain.gain.setValueAtTime(0.3, ctx.currentTime)
    gain.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + duration / 1000)
    oscillator.connect(gain)
    gain.connect(ctx.destination)
    oscillator.start()
    oscillator.stop(ctx.currentTime + duration / 1000)
    oscillator.onended = () => ctx.close()
  } catch {
    // Web Audio API not available — silently skip
  }
}

export interface ExamTimerCallbacks {
  onExpire: () => void
  onWarning?: () => void   // fires once when remaining first hits <= 300s
  onDanger?: () => void    // fires once when remaining first hits <= 60s
}

/**
 * IMPORTANT: The `callbacks` parameter is captured once at construction time.
 * Callers must pass stable (non-reactive) function references. If callbacks
 * need to close over reactive state, use refs inside the callback body rather
 * than relying on the closure being refreshed.
 */
export function useExamTimer(endTimeRef: Ref<string | null>, callbacks: (() => void) | ExamTimerCallbacks) {
  const remaining = ref(0)
  let interval: ReturnType<typeof setInterval> | null = null

  // Normalize callbacks — stored as ref so they can be updated if needed
  const cbsRef = ref<ExamTimerCallbacks>(typeof callbacks === 'function'
    ? { onExpire: callbacks }
    : { ...callbacks })
  const cbs = cbsRef.value

  // Track whether callbacks have already fired (fire once only)
  let warningFired = false
  let dangerFired = false

  // --- Offline pause/resume logic ---
  const OFFLINE_PAUSE_THRESHOLD = 30 // seconds
  const isPaused = ref(false)
  let offlineSince: number | null = null
  let pauseCheckTimer: ReturnType<typeof setTimeout> | null = null
  // Accumulated compensation in seconds (added to remaining time calculation)
  let compensationSeconds = 0

  function onOfflineTimer() {
    offlineSince = Date.now()
    // After 30s offline, pause the timer
    pauseCheckTimer = setTimeout(() => {
      if (offlineSince !== null) {
        isPaused.value = true
        // Stop the countdown interval while paused
        if (interval) clearInterval(interval)
        interval = null
      }
    }, OFFLINE_PAUSE_THRESHOLD * 1000)
  }

  function onOnlineTimer() {
    if (pauseCheckTimer) {
      clearTimeout(pauseCheckTimer)
      pauseCheckTimer = null
    }

    if (offlineSince !== null) {
      const offlineDurationSec = (Date.now() - offlineSince) / 1000

      if (offlineDurationSec >= OFFLINE_PAUSE_THRESHOLD) {
        // Compensate: add back (offlineDuration - 30s) to remaining time
        // The first 30 seconds still count against the exam time
        const extraCompensation = Math.floor(offlineDurationSec - OFFLINE_PAUSE_THRESHOLD)
        compensationSeconds += extraCompensation
      }
      // If offline < 30s: no compensation needed, time elapsed normally

      offlineSince = null
      isPaused.value = false

      // Restart the countdown interval
      if (endTimeRef.value) {
        update()
        if (interval) clearInterval(interval)
        interval = setInterval(update, 1000)
      }
    }
  }

  function setupOfflineListeners() {
    window.addEventListener('offline', onOfflineTimer)
    window.addEventListener('online', onOnlineTimer)
    // If already offline at start
    if (!navigator.onLine) {
      onOfflineTimer()
    }
  }

  function cleanupOfflineListeners() {
    window.removeEventListener('offline', onOfflineTimer)
    window.removeEventListener('online', onOnlineTimer)
    if (pauseCheckTimer) {
      clearTimeout(pauseCheckTimer)
      pauseCheckTimer = null
    }
  }
  // --- End offline pause/resume logic ---

  function update() {
    const endTimeStr = endTimeRef.value
    if (!endTimeStr) return
    const diff = Math.floor((new Date(endTimeStr).getTime() - Date.now()) / 1000) + compensationSeconds
    if (diff <= 0) {
      remaining.value = 0
      stop()
      cbs.onExpire()
    } else {
      remaining.value = diff

      // Fire warning callback once when remaining first drops to <= 300s
      if (!warningFired && diff <= 300 && cbs.onWarning) {
        warningFired = true
        playBeep(660, 300) // short beep at 660Hz for warning (5 min)
        cbs.onWarning()
      }

      // Fire danger callback once when remaining first drops to <= 60s
      if (!dangerFired && diff <= 60 && cbs.onDanger) {
        dangerFired = true
        playBeep(880, 500) // higher pitch, longer beep for danger (1 min)
        cbs.onDanger()
      }
    }
  }

  function start() {
    // Reset fired flags when restarting (e.g. time extended)
    warningFired = false
    dangerFired = false
    update()
    if (interval) clearInterval(interval)
    interval = setInterval(update, 1000)
    setupOfflineListeners()
  }

  function stop() {
    if (interval) clearInterval(interval)
    interval = null
    cleanupOfflineListeners()
  }

  // Auto-start when endTime becomes available
  watch(endTimeRef, (val) => {
    if (val) start()
  })

  const hours = computed(() => Math.floor(remaining.value / 3600))
  const minutes = computed(() => Math.floor((remaining.value % 3600) / 60))
  const seconds = computed(() => remaining.value % 60)

  const formatted = computed(() => {
    const h = String(hours.value).padStart(2, '0')
    const m = String(minutes.value).padStart(2, '0')
    const s = String(seconds.value).padStart(2, '0')
    return hours.value > 0 ? `${h}:${m}:${s}` : `${m}:${s}`
  })

  const isWarning = computed(() => remaining.value > 0 && remaining.value <= 300)
  const isDanger = computed(() => remaining.value > 0 && remaining.value <= 60)
  const isCritical = computed(() => remaining.value > 0 && remaining.value <= 30)
  const isExpired = computed(() => remaining.value <= 0)

  onUnmounted(stop)

  return { remaining, formatted, isWarning, isDanger, isCritical, isExpired, isPaused, start, stop }
}
