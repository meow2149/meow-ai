import { useRealtimeVoice } from "./hooks/useRealtimeVoice"

function App() {
  const { status, info, start, stop, isRunning } = useRealtimeVoice()
  return (
    <div>
      <button onClick={isRunning ? stop : start}>{isRunning ? "结束对话" : "开始对话"}</button>
      <div>状态：{status}</div>
      {info && <div>提示：{info}</div>}
    </div>
  )
}

export default App
