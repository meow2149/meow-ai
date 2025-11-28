import { useEffect, useState, useRef } from "react"
import { motion } from "motion/react"

const TypewriterText = ({ text }: { text: string }) => {
  const [displayedText, setDisplayedText] = useState("")
  const textRef = useRef(text) // 始终持有最新 text，避免闭包问题

  // 1. 同步 text 到 ref，并处理重置逻辑
  useEffect(() => {
    textRef.current = text
    // 如果新文本不包含已显示的文本（说明换了一句话），立即重置
    if (!text.startsWith(displayedText) && displayedText !== "") {
      // 使用 setTimeout 将状态更新推迟，避免 cascading render
      const t = setTimeout(() => setDisplayedText(""), 0)
      return () => clearTimeout(t)
    }
  }, [text, displayedText])

  // 2. 核心打字驱动循环
  // 这个 Effect 只依赖 displayedText，因此不会被 text 的高频更新打断（解决卡顿问题）
  useEffect(() => {
    const target = textRef.current

    // 如果还没追上，继续打字
    if (displayedText.length < target.length) {
      const distance = target.length - displayedText.length
      // 动态速度：积压多时加速(30ms)，积压少时保持优雅慢速(100ms)
      const speed = distance > 5 ? 30 : 100

      const timer = setTimeout(() => {
        setDisplayedText((prev) => {
          const currentTarget = textRef.current
          if (prev.length < currentTarget.length) {
            return currentTarget.slice(0, prev.length + 1)
          }
          return prev
        })
      }, speed)
      return () => clearTimeout(timer)
    }
    // 容错：如果显示的比目标还长，截断
    else if (displayedText.length > target.length) {
      const t = setTimeout(() => setDisplayedText(target), 0)
      return () => clearTimeout(t)
    }
  }, [displayedText])

  // 3. 监听 text 变化，唤醒打字机
  // 当 text 更新时，如果打字机已经停了（因为之前追平了），需要重新触发一下 setDisplayedText
  useEffect(() => {
    // 使用 setTimeout 将状态更新推迟到下一个 tick，避免 cascading renders
    const t = setTimeout(() => {
      setDisplayedText((prev) => {
        // 如果是新的一句话（reset case），交由第一个 effect 处理，这里保持原样
        if (!text.startsWith(prev) && prev !== "") {
          return prev
        }
        // 如果是追加内容，且显示落后于目标，推进一步以唤醒打字循环
        if (prev.length < text.length) {
          return text.slice(0, prev.length + 1)
        }
        return prev
      })
    }, 0)
    return () => clearTimeout(t)
  }, [text])

  return (
    <motion.div
      className="relative inline-block text-left"
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -10, transition: { duration: 0.2 } }}
      transition={{ duration: 0.5, ease: "easeOut" }}
    >
      <span className="font-serif text-lg leading-relaxed font-medium tracking-wide text-stone-600/90">
        {displayedText}
      </span>
      <motion.span
        animate={{ opacity: [0, 1, 0] }}
        transition={{ duration: 0.8, repeat: Infinity, ease: "linear" }}
        className="ml-1 inline-block h-[1.2em] w-[2px] bg-stone-400 align-middle"
      />
    </motion.div>
  )
}

export { TypewriterText }
