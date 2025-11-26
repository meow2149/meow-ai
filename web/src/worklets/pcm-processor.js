class PCMProcessor extends AudioWorkletProcessor {
  process(inputs) {
    if (!inputs.length || !inputs[0].length) {
      return true
    }
    const channel = inputs[0][0]
    if (!channel) {
      return true
    }
    const copy = new Float32Array(channel.length)
    copy.set(channel)
    this.port.postMessage(copy, [copy.buffer])
    return true
  }
}

registerProcessor("pcm-processor", PCMProcessor)
