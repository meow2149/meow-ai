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
  const activeSourcesRef = useRef<Set<AudioBufferSourceNode>>(new Set())

  const stopAllAudio = useCallback(() => {
    // 停止所有正在播放的音频源
    activeSourcesRef.current.forEach((source) => {
      source.stop()
    })
    activeSourcesRef.current.clear()
    // 重置播放时间，以便新音频可以立即开始
    if (playbackCtxRef.current) {
      playbackTimeRef.current = playbackCtxRef.current.currentTime
    }
  }, [])

  const cleanup = useCallback(() => {
    stopAllAudio()
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
  }, [stopAllAudio])

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

    // 将音频源添加到活动集合中
    activeSourcesRef.current.add(source)

    // 当音频播放结束时，从集合中移除
    source.onended = () => {
      activeSourcesRef.current.delete(source)
    }

    const startAt = Math.max(playbackTimeRef.current, ctx.currentTime)
    source.start(startAt)
    playbackTimeRef.current = startAt + buffer.duration
  }, [])

  const bufferText = useCallback((incoming: string) => {
    textBufferRef.current += incoming
    setText(textBufferRef.current)
  }, [])

  const resetText = useCallback(() => {
    textBufferRef.current = ""
    setText("")
  }, [])

  const handleServerMessage = useCallback(
    (raw: string) => {
      const message = JSON.parse(raw)

      if (message.type === "ready") {
        setStatus("running")
        return
      }
      if (message.type === "error") {
        setStatus("error")
        cleanup()
        return
      }
      if (message.type === "event") {
        switch (message.event_id) {
          case 154: {
            // 每一轮交互对应的用量信息
            return
          }
          case 350: {
            // 合成音频的起始事件
            stopAllAudio()
            resetText()
            const text = message.payload.text
            if (text) {
              textBufferRef.current = text
              setText(text)
            }
            return
          }
          case 351: {
            // 合成音频的分句结束事件
            return
          }
          case 359: {
            // 模型一轮音频合成结束事件
            return
          }
          case 450: {
            // 模型识别出音频流中的首字返回的事件，用于打断客户端的播报
            stopAllAudio()
            return
          }
          case 451: {
            // 模型识别出用户说话的文本内容
            return
          }
          case 459: {
            // 模型认为用户说话结束的事件
            return
          }
          case 550: {
            // 模型回复的文本内容
            bufferText(message.payload.content)
            return
          }
          case 559: {
            // 模型回复文本结束事件
            return
          }
        }
      }
    },
    [cleanup, bufferText, resetText, stopAllAudio],
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
