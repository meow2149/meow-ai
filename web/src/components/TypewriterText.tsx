import { useEffect, useState, useRef } from "react"
import { motion } from "motion/react"

const TypewriterText = ({ text }: { text: string }) => {
  const [displayedText, setDisplayedText] = useState("")
  const textRef = useRef(text)

  useEffect(() => {
    textRef.current = text
    if (displayedText && !text.startsWith(displayedText)) {
      const timer = setTimeout(() => setDisplayedText(""), 0)
      return () => clearTimeout(timer)
    }
  }, [text, displayedText])

  useEffect(() => {
    const target = textRef.current

    if (displayedText.length < target.length) {
      const distance = target.length - displayedText.length
      const speed = distance > 8 ? 26 : 90
      const step = distance > 10 ? 2 : 1

      const timer = setTimeout(() => {
        setDisplayedText((prev) => {
          const currentTarget = textRef.current
          if (prev === currentTarget) return prev
          const nextLength = Math.min(prev.length + step, currentTarget.length)
          return currentTarget.slice(0, nextLength)
        })
      }, speed)
      return () => clearTimeout(timer)
    }
    if (displayedText.length > target.length) {
      const timer = setTimeout(() => setDisplayedText(target), 0)
      return () => clearTimeout(timer)
    }
  }, [displayedText])

  useEffect(() => {
    const timer = setTimeout(() => {
      setDisplayedText((prev) => {
        const target = textRef.current
        if (prev && !target.startsWith(prev)) {
          return prev
        }
        if (prev.length < target.length) {
          return target.slice(0, prev.length + 1)
        }
        if (prev.length > target.length) {
          return target
        }
        return prev
      })
    }, 0)
    return () => clearTimeout(timer)
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
