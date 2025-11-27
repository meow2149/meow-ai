package voice

import (
	"encoding/binary"
	"fmt"
	"math"
)

const (
	targetSampleRate = 16000
	targetChannels   = 1
)

type Encoding string

const (
	EncodingF32 Encoding = "f32le"
	EncodingS16 Encoding = "s16le"
)

type InputFormat struct {
	SampleRate int
	Encoding   Encoding
}

type PCMProcessor struct {
	format    InputFormat
	resampler *linearResampler
}

func NewPCMProcessor(format InputFormat) (*PCMProcessor, error) {
	if format.SampleRate <= 0 {
		return nil, fmt.Errorf("invalid sample rate")
	}
	if format.Encoding == "" {
		format.Encoding = EncodingF32
	}
	var res *linearResampler
	if format.SampleRate != targetSampleRate {
		res = newLinearResampler(format.SampleRate, targetSampleRate)
	}
	return &PCMProcessor{
		format:    format,
		resampler: res,
	}, nil
}

func (p *PCMProcessor) Process(frame []byte) ([]byte, error) {
	samples, err := decodeSamples(frame, p.format.Encoding)
	if err != nil {
		return nil, err
	}
	if len(samples) == 0 {
		return nil, nil
	}
	if p.resampler != nil {
		samples = p.resampler.Process(samples)
	}
	if len(samples) == 0 {
		return nil, nil
	}
	return float32ToS16Bytes(samples), nil
}

func decodeSamples(data []byte, encoding Encoding) ([]float32, error) {
	switch encoding {
	case EncodingF32:
		if len(data)%4 != 0 {
			return nil, fmt.Errorf("unaligned f32 frame")
		}
		count := len(data) / 4
		samples := make([]float32, count)
		for i := 0; i < count; i++ {
			bits := binary.LittleEndian.Uint32(data[i*4 : (i+1)*4])
			samples[i] = math.Float32frombits(bits)
		}
		return samples, nil
	case EncodingS16:
		if len(data)%2 != 0 {
			return nil, fmt.Errorf("unaligned s16 frame")
		}
		count := len(data) / 2
		samples := make([]float32, count)
		for i := 0; i < count; i++ {
			v := int16(binary.LittleEndian.Uint16(data[i*2 : (i+1)*2]))
			samples[i] = float32(v) / 32768.0
		}
		return samples, nil
	default:
		return nil, fmt.Errorf("unsupported encoding %s", encoding)
	}
}

func float32ToS16Bytes(samples []float32) []byte {
	buf := make([]byte, len(samples)*2)
	for i, sample := range samples {
		bufSample := float32ToS16(sample)
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(bufSample))
	}
	return buf
}

func float32ToS16(v float32) int16 {
	if v > 1 {
		v = 1
	} else if v < -1 {
		v = -1
	}
	return int16(math.Round(float64(v) * 32767))
}

type linearResampler struct {
	srcRate int
	dstRate int
	step    float64
	pos     float64

	lastSample float32
	hasLast    bool
	work       []float32
}

func newLinearResampler(src, dst int) *linearResampler {
	return &linearResampler{
		srcRate: src,
		dstRate: dst,
		step:    float64(src) / float64(dst),
	}
}

func (r *linearResampler) Process(samples []float32) []float32 {
	if len(samples) == 0 {
		return nil
	}
	data := samples
	if r.hasLast {
		if cap(r.work) < len(samples)+1 {
			r.work = make([]float32, len(samples)+1)
		} else {
			r.work = r.work[:len(samples)+1]
		}
		r.work[0] = r.lastSample
		copy(r.work[1:], samples)
		data = r.work
	}
	lastIdx := len(data) - 1
	if lastIdx <= 0 {
		r.lastSample = data[lastIdx]
		r.hasLast = true
		return nil
	}
	outCap := int(float64(len(samples))*float64(r.dstRate)/float64(r.srcRate)) + 4
	out := make([]float32, 0, outCap)
	pos := r.pos
	for {
		idx := int(pos)
		next := idx + 1
		if next > lastIdx {
			break
		}
		frac := pos - float64(idx)
		a := data[idx]
		b := data[next]
		value := a*(1-float32(frac)) + b*float32(frac)
		out = append(out, value)
		pos += r.step
	}
	r.pos = pos - float64(lastIdx)
	if r.pos < 0 {
		r.pos = 0
	}
	r.lastSample = data[lastIdx]
	r.hasLast = true
	return out
}
