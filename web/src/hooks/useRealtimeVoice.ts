import { useCallback, useEffect, useRef, useState } from "react"

type Status = "idle" | "connecting" | "running" | "error"

const WS_PATH = "/ws/realtime"
const DEFAULT_SAMPLE_RATE = 48000

const getBackendURL = () => {
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
  return `${protocol}//${window.location.host}${WS_PATH}`
}

const workletURL = new URL("../worklets/pcm-processor.js", import.meta.url)

const useRealtimeVoice = () => {
  const [status, setStatus] = useState<Status>("idle")
  const [info, setInfo] = useState("")
  const [aiText, setAiText] = useState("")
  const lastUpdateTimeRef = useRef(0) // 记录上次收到有效 AI 文本的时间

  const wsRef = useRef<WebSocket | null>(null)
  const mediaRef = useRef<MediaStream | null>(null)
  const audioCtxRef = useRef<AudioContext | null>(null)
  const workletRef = useRef<AudioWorkletNode | null>(null)
  const playbackCtxRef = useRef<AudioContext | null>(null)
  const playbackTimeRef = useRef(0)

  const cleanup = useCallback(() => {
    wsRef.current?.close()
    wsRef.current = null
    workletRef.current?.disconnect()
    workletRef.current = null
    mediaRef.current?.getTracks().forEach((track) => track.stop())
    mediaRef.current = null
    audioCtxRef.current?.close()
    audioCtxRef.current = null
    playbackCtxRef.current?.close()
    playbackCtxRef.current = null
    playbackTimeRef.current = 0
  }, [])

  const handleServerAudio = useCallback(async (payload: ArrayBuffer) => {
    if (!payload.byteLength) return
    let ctx = playbackCtxRef.current
    if (!ctx) {
      ctx = new AudioContext()
      playbackCtxRef.current = ctx
      playbackTimeRef.current = ctx.currentTime
    }
    if (ctx.state === "suspended") {
      await ctx.resume()
    }
    const samples = new Float32Array(payload)
    const buffer = ctx.createBuffer(1, samples.length, 24000)
    buffer.copyToChannel(samples, 0)
    const source = ctx.createBufferSource()
    source.buffer = buffer
    source.connect(ctx.destination)
    const startAt = Math.max(playbackTimeRef.current, ctx.currentTime)
    source.start(startAt)
    playbackTimeRef.current = startAt + buffer.duration
  }, [])

  const start = useCallback(async () => {
    if (status === "connecting" || status === "running") return
    setStatus("connecting")
    setInfo("")
    setAiText("")
    try {
      const media = await navigator.mediaDevices.getUserMedia({ audio: true })
      const audioCtx = new AudioContext({ sampleRate: DEFAULT_SAMPLE_RATE })
      await audioCtx.audioWorklet.addModule(workletURL)
      await audioCtx.resume()
      const source = audioCtx.createMediaStreamSource(media)
      const worklet = new AudioWorkletNode(audioCtx, "pcm-processor")
      const gain = audioCtx.createGain()
      gain.gain.value = 0
      source.connect(worklet)
      worklet.connect(gain).connect(audioCtx.destination)

      const ws = new WebSocket(getBackendURL())
      ws.binaryType = "arraybuffer"

      ws.onopen = () => {
        ws.send(
          JSON.stringify({
            type: "start",
            sampleRate: audioCtx.sampleRate,
            encoding: "f32le",
          }),
        )
      }

      ws.onmessage = (event) => {
        if (typeof event.data === "string") {
          try {
            const msg = JSON.parse(event.data)
            if (msg.type === "ready") {
              setStatus("running")
              setInfo("")
            } else if (msg.type === "error") {
              setStatus("error")
              setInfo(msg.message ?? "发生未知错误")
              cleanup()
            } else if (msg.type === "event") {
              const payload = msg.payload

              // 兼容多种可能的字段名
              let text = ""
              if (payload.content) text = payload.content
              else if (payload.text) text = payload.text
              else if (payload.result?.text) text = payload.result.text
              else if (payload.display_text) text = payload.display_text

              if (text && typeof text === "string" && text.trim().length > 0) {
                const now = Date.now()
                // Time-based Reset: 如果距离上次收到 AI 文本超过 3 秒，则视为新的一轮，清空旧文本
                const isTimeReset = now - lastUpdateTimeRef.current > 3000
                lastUpdateTimeRef.current = now

                setAiText((prev) => {
                  if (isTimeReset) {
                    return text
                  }

                  // 1. 全量流式更新检测
                  if (text.length > prev.length && text.startsWith(prev)) {
                    return text
                  }

                  // 2. 增量追加
                  if (!text.startsWith(prev)) {
                    return prev + text
                  }

                  return text
                })
              }

              // Event-based Reset: ASR Started
              if (msg.event_id === 1000 || msg.event_id === 1001) {
                setAiText("")
              }
            }
          } catch (err) {
            console.error("parse message error", err)
          }
          return
        }
        handleServerAudio(event.data as ArrayBuffer).catch((err) => {
          console.error("playback error", err)
        })
      }

      ws.onerror = () => {
        setStatus("error")
        setInfo("WebSocket 连接失败")
        cleanup()
      }

      ws.onclose = () => {
        cleanup()
        setStatus("idle")
      }

      worklet.port.onmessage = (event) => {
        if (ws.readyState !== WebSocket.OPEN) return
        const chunk = event.data as Float32Array
        if (!chunk?.length) return
        const copy = chunk.slice()
        ws.send(copy.buffer)
      }

      wsRef.current = ws
      mediaRef.current = media
      audioCtxRef.current = audioCtx
      workletRef.current = worklet
    } catch (err) {
      console.error(err)
      cleanup()
      setStatus("error")
      setInfo("无法访问麦克风或建立连接")
    }
  }, [cleanup, handleServerAudio, status])

  const stop = useCallback(() => {
    if (!wsRef.current) {
      cleanup()
      setStatus("idle")
      return
    }
    if (wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type: "stop" }))
    }
    wsRef.current.close()
    cleanup()
    setStatus("idle")
  }, [cleanup])

  useEffect(() => {
    return () => {
      cleanup()
    }
  }, [cleanup])

  return {
    status,
    info,
    start,
    stop,
    isRunning: status === "running",
    aiText,
  }
}

export { useRealtimeVoice, type Status }
