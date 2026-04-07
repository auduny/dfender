package game

import (
	"math"
	"math/rand"
)

// sfxr-style procedural sound effect generator.
// Generates 32-bit float stereo PCM at the given sample rate.

const sfxrSampleRate = 44100

// SFXParams controls the shape of a generated sound.
type SFXParams struct {
	// Wave type: 0=square, 1=sawtooth, 2=sine, 3=noise
	WaveType int

	// Envelope (seconds)
	AttackTime  float64
	SustainTime float64
	DecayTime   float64

	// Frequency
	BaseFreq  float64 // Hz
	FreqSlide float64 // Hz per second (negative = downward sweep)
	FreqLimit float64 // Stop sliding below this

	// Vibrato
	VibratoDepth float64 // Hz
	VibratoSpeed float64 // Hz

	// Duty cycle (square wave only)
	DutyCycle      float64
	DutyCycleSweep float64

	// Filters
	LPFCutoff    float64 // 0.0-1.0, 1.0 = no filter
	LPFResonance float64 // 0.0-1.0
	HPFCutoff    float64 // 0.0-1.0, 0.0 = no filter

	// Volume
	Volume float64
}

// GenerateSFX produces a stereo 32-bit float PCM buffer from the given params.
func GenerateSFX(p SFXParams) []byte {
	duration := p.AttackTime + p.SustainTime + p.DecayTime
	numSamples := int(duration * sfxrSampleRate)
	if numSamples == 0 {
		return nil
	}

	// Output buffer: stereo float32, 4 bytes per sample per channel, 2 channels.
	buf := make([]byte, numSamples*2*4)

	var phase float64
	freq := p.BaseFreq
	duty := p.DutyCycle
	if duty == 0 {
		duty = 0.5
	}

	// Low-pass filter state
	var lpfPrev float64
	lpfCutoff := p.LPFCutoff
	if lpfCutoff == 0 {
		lpfCutoff = 1.0
	}

	// High-pass filter state
	var hpfPrev, hpfOut float64

	// Noise buffer
	noiseBuf := make([]float64, 32)
	for i := range noiseBuf {
		noiseBuf[i] = rand.Float64()*2 - 1
	}

	vol := p.Volume
	if vol == 0 {
		vol = 1.0
	}

	for i := 0; i < numSamples; i++ {
		t := float64(i) / sfxrSampleRate

		// Envelope
		var env float64
		if t < p.AttackTime {
			env = t / p.AttackTime
		} else if t < p.AttackTime+p.SustainTime {
			env = 1.0
		} else {
			elapsed := t - p.AttackTime - p.SustainTime
			if p.DecayTime > 0 {
				env = 1.0 - elapsed/p.DecayTime
			}
		}
		if env < 0 {
			env = 0
		}

		// Frequency with slide and vibrato
		freq += p.FreqSlide / sfxrSampleRate
		if p.FreqLimit > 0 && freq < p.FreqLimit {
			freq = p.FreqLimit
		}
		if freq < 0 {
			freq = 0
		}
		actualFreq := freq
		if p.VibratoDepth > 0 {
			actualFreq += p.VibratoDepth * math.Sin(2*math.Pi*p.VibratoSpeed*t)
		}

		// Duty cycle sweep
		duty += p.DutyCycleSweep / sfxrSampleRate
		if duty < 0.01 {
			duty = 0.01
		}
		if duty > 0.99 {
			duty = 0.99
		}

		// Phase advance
		phase += actualFreq / sfxrSampleRate
		for phase >= 1.0 {
			phase -= 1.0
		}

		// Oscillator
		var sample float64
		switch p.WaveType {
		case 0: // Square
			if phase < duty {
				sample = 1.0
			} else {
				sample = -1.0
			}
		case 1: // Sawtooth
			sample = 2.0*phase - 1.0
		case 2: // Sine
			sample = math.Sin(2 * math.Pi * phase)
		case 3: // Noise
			idx := int(phase * float64(len(noiseBuf)))
			if idx >= len(noiseBuf) {
				idx = len(noiseBuf) - 1
			}
			sample = noiseBuf[idx]
			// Refresh noise periodically
			if i%int(sfxrSampleRate/actualFreq+1) == 0 {
				for j := range noiseBuf {
					noiseBuf[j] = rand.Float64()*2 - 1
				}
			}
		}

		// Low-pass filter
		if lpfCutoff < 1.0 {
			cutoff := lpfCutoff * lpfCutoff * 0.1 // Map to useful range
			lpfPrev += cutoff * (sample - lpfPrev)
			sample = lpfPrev
		}

		// High-pass filter
		if p.HPFCutoff > 0 {
			cutoff := p.HPFCutoff * p.HPFCutoff * 0.1
			hpfOut += sample - hpfPrev
			hpfPrev = sample
			hpfOut *= 1.0 - cutoff
			sample = hpfOut
		}

		// Apply envelope and volume
		sample *= env * vol

		// Clamp
		if sample > 1.0 {
			sample = 1.0
		}
		if sample < -1.0 {
			sample = -1.0
		}

		// Write stereo float32
		f := float32(sample)
		bits := math.Float32bits(f)
		off := i * 8
		buf[off+0] = byte(bits)
		buf[off+1] = byte(bits >> 8)
		buf[off+2] = byte(bits >> 16)
		buf[off+3] = byte(bits >> 24)
		// Right channel = same
		buf[off+4] = byte(bits)
		buf[off+5] = byte(bits >> 8)
		buf[off+6] = byte(bits >> 16)
		buf[off+7] = byte(bits >> 24)
	}

	return buf
}

// Preset sound effects

// sfxLaserPitches generates 20 laser sound variants with pitch rising from
// 900 Hz (cool) to 1500 Hz (hot), giving an audible heat warning.
func sfxLaserPitches() [20][]byte {
	var bufs [20][]byte
	for i := range bufs {
		t := float64(i) / 19.0
		bufs[i] = GenerateSFX(SFXParams{
			WaveType:    2,
			AttackTime:  0.0,
			SustainTime: 0.005,
			DecayTime:   0.04,
			BaseFreq:    900 + t*600,
			FreqSlide:   -4000,
			FreqLimit:   150,
			Volume:      0.12,
		})
	}
	return bufs
}

func sfxExplosion() []byte {
	return GenerateSFX(SFXParams{
		WaveType:    3, // Noise
		AttackTime:  0.0,
		SustainTime: 0.05,
		DecayTime:   0.18,
		BaseFreq:    120,
		FreqSlide:   -80,
		LPFCutoff:   0.3,
		Volume:      0.35,
	})
}

func sfxSmallExplosion() []byte {
	return GenerateSFX(SFXParams{
		WaveType:    3, // Noise
		AttackTime:  0.0,
		SustainTime: 0.005,
		DecayTime:   0.03,
		BaseFreq:    180,
		FreqSlide:   -100,
		LPFCutoff:   0.25,
		Volume:      0.08,
	})
}

func sfxBounce() []byte {
	return GenerateSFX(SFXParams{
		WaveType:    2, // Sine
		AttackTime:  0.0,
		SustainTime: 0.01,
		DecayTime:   0.1,
		BaseFreq:    250,
		FreqSlide:   300,
		LPFCutoff:   0.4,
		Volume:      0.2,
	})
}

func sfxOverheat() []byte {
	return GenerateSFX(SFXParams{
		WaveType:    3, // Noise
		AttackTime:  0.01,
		SustainTime: 0.15,
		DecayTime:   0.25,
		BaseFreq:    300,
		FreqSlide:   -150,
		LPFCutoff:   0.3,
		Volume:      0.3,
	})
}

func sfxOverheatWarning() []byte {
	return GenerateSFX(SFXParams{
		WaveType:    0, // Square — harsh, alarm-like
		AttackTime:  0.0,
		SustainTime: 0.04,
		DecayTime:   0.04,
		BaseFreq:    800,
		FreqSlide:   -200,
		LPFCutoff:   0.5,
		Volume:      0.15,
	})
}

func sfxWaveComplete() []byte {
	return GenerateSFX(SFXParams{
		WaveType:    2, // Sine — cleaner fanfare
		AttackTime:  0.01,
		SustainTime: 0.12,
		DecayTime:   0.3,
		BaseFreq:    400,
		FreqSlide:   500,
		LPFCutoff:   0.5,
		Volume:      0.25,
	})
}

func sfxPlayerDeath() []byte {
	return GenerateSFX(SFXParams{
		WaveType:    3, // Noise
		AttackTime:  0.0,
		SustainTime: 0.15,
		DecayTime:   0.6,
		BaseFreq:    150,
		FreqSlide:   -60,
		LPFCutoff:   0.25,
		Volume:      0.4,
	})
}

func sfxWallHit() []byte {
	return GenerateSFX(SFXParams{
		WaveType:    3, // Noise
		AttackTime:  0.0,
		SustainTime: 0.002,
		DecayTime:   0.012,
		BaseFreq:    200,
		FreqSlide:   -300,
		LPFCutoff:   0.2,
		Volume:      0.06,
	})
}

func sfxPickup() []byte {
	return GenerateSFX(SFXParams{
		WaveType:    2, // Sine — clean rising chime
		AttackTime:  0.0,
		SustainTime: 0.05,
		DecayTime:   0.15,
		BaseFreq:    500,
		FreqSlide:   1200,
		Volume:      0.25,
	})
}

func sfxShieldAbsorb() []byte {
	return GenerateSFX(SFXParams{
		WaveType:    0, // Square — metallic ping
		AttackTime:  0.0,
		SustainTime: 0.03,
		DecayTime:   0.2,
		BaseFreq:    800,
		FreqSlide:   -400,
		LPFCutoff:   0.5,
		Volume:      0.3,
	})
}

func sfxMissileLaunch() []byte {
	return GenerateSFX(SFXParams{
		WaveType:    3, // Noise — whoosh
		AttackTime:  0.01,
		SustainTime: 0.08,
		DecayTime:   0.15,
		BaseFreq:    400,
		FreqSlide:   600,
		LPFCutoff:   0.35,
		Volume:      0.25,
	})
}

func sfxMissileExplode() []byte {
	return GenerateSFX(SFXParams{
		WaveType:    3, // Noise — big boom
		AttackTime:  0.0,
		SustainTime: 0.12,
		DecayTime:   0.35,
		BaseFreq:    80,
		FreqSlide:   -50,
		LPFCutoff:   0.4,
		Volume:      0.45,
	})
}

