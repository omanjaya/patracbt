import { ref, onUnmounted } from 'vue'

export type WSEvent =
  | 'student_joined'
  | 'student_left'
  | 'answer_saved'
  | 'answer_batch'
  | 'violation_logged'
  | 'session_finished'
  | 'lock_client'
  | 'time_sync'
  | 'heartbeat'
  | 'force_finish'
  | 'time_extended'
  | 'chat_message'

export interface WSMessage<T = unknown> {
  event: WSEvent
  data: T
}

type EventHandler = (data: unknown) => void

const WS_BASE = import.meta.env.VITE_WS_URL ?? 'ws://localhost:8080'

export function useWebSocket(scheduleId: number, sessionId?: number) {
  const connected = ref(false)
  const handlers = new Map<WSEvent, EventHandler[]>()
  let ws: WebSocket | null = null
  let heartbeatTimer: ReturnType<typeof setInterval> | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let destroyed = false

  function getUrl() {
    const token = localStorage.getItem('access_token') ?? ''
    let url = `${WS_BASE}/api/v1/ws/exam/${scheduleId}?token=${token}`
    if (sessionId) url += `&session_id=${sessionId}`
    return url
  }

  function connect() {
    if (destroyed) return
    ws = new WebSocket(getUrl())

    // Connection timeout: if onopen hasn't fired within 10s, close and reconnect
    const connectTimeout = setTimeout(() => {
      if (ws && ws.readyState !== WebSocket.OPEN) {
        ws.close()
      }
    }, 10000)

    ws.onopen = () => {
      clearTimeout(connectTimeout)
      connected.value = true
      startHeartbeat()
      // Signal connection via time_sync-like event
      const list = handlers.get('time_sync') ?? []
      list.forEach((fn) => fn({ server_time: new Date().toISOString() }))
    }

    ws.onmessage = (ev) => {
      try {
        const msg: WSMessage = JSON.parse(ev.data)
        const list = handlers.get(msg.event) ?? []
        list.forEach((fn) => fn(msg.data))
      } catch (e) { console.warn('WS message parse failed:', e) }
    }

    ws.onclose = () => {
      connected.value = false
      stopHeartbeat()
      if (!destroyed) {
        reconnectTimer = setTimeout(connect, 3000)
      } else {
        connected.value = false
      }
    }

    ws.onerror = () => {
      ws?.close()
    }
  }

  function startHeartbeat() {
    stopHeartbeat()
    heartbeatTimer = setInterval(() => {
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ event: 'heartbeat' }))
      }
    }, 30000)
  }

  function stopHeartbeat() {
    if (heartbeatTimer) {
      clearInterval(heartbeatTimer)
      heartbeatTimer = null
    }
  }

  function on<T = any>(event: WSEvent, handler: (data: T) => void) {
    const list = handlers.get(event) ?? []
    list.push(handler as EventHandler)
    handlers.set(event, list)
  }

  function off(event: WSEvent, handler: EventHandler) {
    const list = (handlers.get(event) ?? []).filter((fn) => fn !== handler)
    if (list.length === 0) handlers.delete(event)
    else handlers.set(event, list)
  }

  function send(msg: WSMessage) {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(msg))
    }
  }

  function disconnect() {
    destroyed = true
    stopHeartbeat()
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    ws?.close()
    ws = null
  }

  onUnmounted(disconnect)

  return { connected, connect, disconnect, on, off, send }
}
