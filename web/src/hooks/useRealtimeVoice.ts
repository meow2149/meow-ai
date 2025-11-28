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
  const [text, setText] = useState("")
  const textBufferRef = useRef("")

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

  const bufferText = useCallback((incoming?: string) => {
    if (!incoming) return
    textBufferRef.current += incoming
    setText(textBufferRef.current)
  }, [])

  const resetText = useCallback(() => {
    textBufferRef.current = ""
    setText("")
  }, [])

  const handleServerMessage = useCallback(
    (raw: string) => {
      const msg = JSON.parse(raw)

      if (msg.type === "ready") {
        setStatus("running")
        return
      }
      if (msg.type === "error") {
        setStatus("error")
        cleanup()
        return
      }
      if (msg.type === "event") {
        switch (msg.event_id) {
          case 1000:
          case 1001: {
            resetText()
            return
          }
          case 350: {
            // TTS 开始：重置并设置初始文本（如果有）
            resetText()
            if (msg.payload?.text) {
              textBufferRef.current = msg.payload.text
              setText(msg.payload.text)
            }
            return
          }
          case 351: {
            // TTS 句子结束：如果有 text 则显示
            if (msg.payload?.text) {
              textBufferRef.current = msg.payload.text
              setText(msg.payload.text)
            }
            return
          }
          case 550: {
            // AI 回复文本：累积流式文本并实时更新
            if (msg.payload?.content) {
              bufferText(msg.payload.content)
            }
            return
          }
        }
      }
    },
    [cleanup, bufferText, resetText],
  )

  const start = useCallback(async () => {
    if (status === "connecting" || status === "running") return
    setStatus("connecting")
    resetText()
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
          handleServerMessage(event.data)
          return
        }
        handleServerAudio(event.data as ArrayBuffer)
      }

      ws.onerror = () => {
        setStatus("error")
        cleanup()
      }

      ws.onclose = () => {
        cleanup()
        setStatus("idle")
      }

      worklet.port.onmessage = (event) => {
        if (ws.readyState !== WebSocket.OPEN) return
        const chunk = event.data as Float32Array
        if (!chunk.length) return
        const copy = chunk.slice()
        ws.send(copy.buffer)
      }

      wsRef.current = ws
      mediaRef.current = media
      audioCtxRef.current = audioCtx
      workletRef.current = worklet
    } catch {
      cleanup()
      setStatus("error")
    }
  }, [cleanup, handleServerAudio, handleServerMessage, status, resetText])

  const stop = useCallback(() => {
    if (!wsRef.current) {
      cleanup()
      setStatus("idle")
      resetText()
      return
    }
    if (wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type: "stop" }))
    }
    wsRef.current.close()
    cleanup()
    setStatus("idle")
    resetText()
  }, [cleanup, resetText])

  useEffect(() => {
    return () => {
      cleanup()
    }
  }, [cleanup])

  return {
    status,
    start,
    stop,
    isRunning: status === "running",
    text,
  }
}

export { useRealtimeVoice, type Status }
