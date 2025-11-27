import { useRealtimeVoice } from "./hooks/useRealtimeVoice"
import { motion, AnimatePresence } from "motion/react"

function App() {
  const { start, stop, isRunning } = useRealtimeVoice()

  // 噪点纹理图 (Base64 SVG) - 增加纸质/胶片质感
  const noiseTexture = `url("data:image/svg+xml,%3Csvg viewBox='0 0 200 200' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noiseFilter'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.65' numOctaves='3' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23noiseFilter)' opacity='0.4'/%3E%3C/svg%3E")`

  return (
    <div className="relative flex min-h-screen w-full flex-col items-center justify-between overflow-hidden bg-[#FFFCF8] px-6 py-12 font-sans text-stone-600 select-none">
      {/* Global Noise Overlay - 质感层 */}
      <div
        className="pointer-events-none absolute inset-0 z-0 opacity-[0.03] mix-blend-overlay"
        style={{ backgroundImage: noiseTexture }}
      ></div>

      {/* Background Gradients - 更通透的混合模式 */}
      <div className="pointer-events-none absolute top-0 left-0 z-0 h-full w-full overflow-hidden">
        <motion.div
          animate={{
            scale: [1, 1.2, 1],
            opacity: [0.4, 0.6, 0.4],
            rotate: [0, 45, 0],
            x: [0, 20, 0],
            y: [0, -20, 0],
          }}
          transition={{ duration: 20, repeat: Infinity, ease: "easeInOut" }}
          className="absolute top-[-10%] right-[-10%] h-[80vw] w-[80vw] rounded-full bg-orange-100/50 mix-blend-multiply blur-[80px]"
        />
        <motion.div
          animate={{
            scale: [1, 1.3, 1],
            opacity: [0.3, 0.5, 0.3],
            x: [0, -30, 0],
            y: [0, 30, 0],
          }}
          transition={{ duration: 25, repeat: Infinity, ease: "easeInOut", delay: 2 }}
          className="absolute bottom-[-10%] left-[-20%] h-[90vw] w-[90vw] rounded-full bg-amber-100/40 mix-blend-multiply blur-[80px]"
        />
        <motion.div
          animate={{
            scale: [1, 1.1, 1],
            opacity: [0.2, 0.4, 0.2],
          }}
          transition={{ duration: 18, repeat: Infinity, ease: "easeInOut", delay: 5 }}
          className="absolute top-[40%] left-[20%] h-[60vw] w-[60vw] rounded-full bg-rose-100/30 mix-blend-multiply blur-[60px]"
        />
      </div>

      {/* Top Brand Area */}
      <motion.div
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 1, delay: 0.2 }}
        className="z-10 flex w-full justify-center pt-6"
      >
        <h1 className="font-sans text-[0.7rem] font-bold tracking-[0.4em] text-stone-400/80 uppercase select-none">
          Meow-AI
        </h1>
      </motion.div>

      {/* Intro Text Area */}
      <div className="z-10 flex w-full max-w-md flex-1 flex-col items-center justify-end pb-24">
        <AnimatePresence mode="wait">
          {!isRunning && (
            <motion.div
              key="intro"
              initial={{ opacity: 0, y: 20, filter: "blur(10px)" }}
              animate={{ opacity: 1, y: 0, filter: "blur(0px)" }}
              exit={{ opacity: 0, y: -20, filter: "blur(8px)", scale: 0.95 }}
              transition={{ duration: 0.6, ease: "easeOut" }}
              className="flex w-full flex-col items-center text-center"
            >
              {/* 装饰性元素 */}
              <div className="mb-8 flex gap-2 opacity-60">
                <motion.div
                  animate={{ opacity: [0.3, 1, 0.3] }}
                  transition={{ duration: 2, repeat: Infinity, delay: 0 }}
                  className="h-1 w-1 rounded-full bg-stone-400"
                />
                <motion.div
                  animate={{ opacity: [0.3, 1, 0.3] }}
                  transition={{ duration: 2, repeat: Infinity, delay: 0.6 }}
                  className="h-1 w-1 rounded-full bg-stone-400"
                />
                <motion.div
                  animate={{ opacity: [0.3, 1, 0.3] }}
                  transition={{ duration: 2, repeat: Infinity, delay: 1.2 }}
                  className="h-1 w-1 rounded-full bg-stone-400"
                />
              </div>

              <h2 className="mb-3 font-serif text-[2.5rem] font-light tracking-wider text-stone-700">连连</h2>

              <p className="mb-10 text-[0.7rem] font-medium tracking-[0.2em] text-stone-400 uppercase opacity-80">
                Your Soul Companion
              </p>

              {/* 诗意文案 - 衬线体增强文学感 */}
              <div className="relative px-8 py-6">
                <span className="absolute top-0 left-4 font-serif text-5xl leading-none font-normal text-stone-200/80">
                  “
                </span>
                <p className="min-w-[220px] font-serif text-sm leading-8 font-normal tracking-wide text-stone-600/90 italic">
                  在这个喧嚣的世界里
                  <br />
                  我想给你留一个
                  <br />
                  <span className="font-medium text-stone-700 not-italic">温柔的角落</span>
                </p>
                <span className="absolute right-4 bottom-[-10px] rotate-180 transform font-serif text-5xl leading-none font-normal text-stone-200/80">
                  “
                </span>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      {/* Main Button Area */}
      <div
        className={`z-10 flex w-full flex-none items-center justify-center transition-all duration-1000 ease-in-out ${isRunning ? "flex-1 pb-0" : "pb-20"}`}
      >
        <div className="group relative flex items-center justify-center">
          {/* Ambient Glow - Active State */}
          <AnimatePresence>
            {isRunning && (
              <>
                {/* 扩散的光晕 */}
                <motion.div
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0 }}
                  transition={{ duration: 1 }}
                  className="pointer-events-none absolute h-80 w-80 rounded-full bg-orange-200/20 blur-3xl"
                />
                {/* 细腻的涟漪圈 */}
                {[0, 1, 2].map((i) => (
                  <motion.div
                    key={i}
                    initial={{ width: "6rem", height: "6rem", opacity: 0.4 }}
                    animate={{
                      width: ["6rem", "18rem"],
                      height: ["6rem", "18rem"],
                      opacity: [0.3, 0],
                    }}
                    transition={{
                      duration: 3,
                      repeat: Infinity,
                      delay: i * 1,
                      ease: [0.22, 1, 0.36, 1],
                    }}
                    className="pointer-events-none absolute rounded-full border border-white/20 bg-amber-100/30"
                  />
                ))}
              </>
            )}
          </AnimatePresence>

          {/* The Button Container */}
          <motion.div
            animate={{
              scale: isRunning ? 1.1 : 1,
            }}
            transition={{ type: "spring", stiffness: 300, damping: 20 }}
          >
            <motion.button
              layout
              onClick={isRunning ? stop : start}
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.96 }}
              animate={{
                // 待机时轻微呼吸
                boxShadow: isRunning
                  ? "0 20px 50px -12px rgba(251, 146, 60, 0.3)"
                  : "0 20px 40px -12px rgba(168, 162, 158, 0.2)",
              }}
              className={`relative z-20 flex h-24 w-24 items-center justify-center rounded-full border transition-colors duration-700 ease-out ${
                isRunning
                  ? "border-orange-200/50 bg-linear-to-b from-orange-300 to-amber-400 text-white"
                  : "border-white/80 bg-white text-stone-500"
              } `}
            >
              {/* 待机呼吸动画 (仅在非运行状态显示) */}
              {!isRunning && (
                <motion.div
                  animate={{ opacity: [0.5, 1, 0.5] }}
                  transition={{ duration: 3, repeat: Infinity, ease: "easeInOut" }}
                  className="pointer-events-none absolute inset-0 rounded-full bg-stone-50"
                />
              )}

              {/* Inner Shine */}
              <div
                className={`pointer-events-none absolute inset-0 rounded-full bg-linear-to-tr from-white/50 to-transparent ${isRunning ? "opacity-20" : "opacity-60"}`}
              ></div>

              {/* Icon Swapping */}
              <div className="relative z-10 flex h-full w-full items-center justify-center">
                <AnimatePresence mode="wait">
                  {isRunning ? (
                    <motion.div
                      key="wave"
                      initial={{ opacity: 0, scale: 0.5, rotate: -30 }}
                      animate={{ opacity: 1, scale: 1, rotate: 0 }}
                      exit={{ opacity: 0, scale: 0.5, rotate: 30 }}
                      transition={{ duration: 0.3 }}
                      className="flex h-6 items-center gap-[3px]"
                    >
                      {/* 更加自然的声波动画 */}
                      {[0.4, 1.0, 0.6, 0.8, 0.5].map((height, i) => (
                        <motion.span
                          key={i}
                          animate={{
                            height: [height * 10, height * 24, height * 10],
                            backgroundColor: ["rgba(255,255,255,0.9)", "rgba(255,255,255,1)", "rgba(255,255,255,0.9)"],
                          }}
                          transition={{
                            duration: [0.8, 1.2, 1.0, 0.9, 1.1][i], // 固定随机值
                            repeat: Infinity,
                            ease: "easeInOut",
                            delay: i * 0.1,
                          }}
                          className="block w-1 rounded-full"
                          style={{ height: height * 12 }}
                        />
                      ))}
                    </motion.div>
                  ) : (
                    <motion.div
                      key="mic"
                      initial={{ opacity: 0, scale: 0.5 }}
                      animate={{ opacity: 1, scale: 1 }}
                      exit={{ opacity: 0, scale: 0.5 }}
                      transition={{ duration: 0.3 }}
                    >
                      <svg
                        xmlns="http://www.w3.org/2000/svg"
                        width="28"
                        height="28"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        className="opacity-60"
                      >
                        <path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3z" />
                        <path d="M19 10v2a7 7 0 0 1-14 0v-2" />
                        <line x1="12" y1="19" x2="12" y2="22" />
                      </svg>
                    </motion.div>
                  )}
                </AnimatePresence>
              </div>
            </motion.button>
          </motion.div>
        </div>
      </div>

      {/* Footer Spacer */}
      <motion.div layout className={`flex-1 transition-all duration-1000 ${isRunning ? "hidden" : "block"}`} />
    </div>
  )
}

export default App
