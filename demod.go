package main

import (
	"math"
	"math/cmplx"
)

type demodMskObj struct {
	mskDf           float64
	MskPhi          float64
	mskClk          float64
	msklvl          float64
	mskS            int
	idx             int
	h               [1024]float64
	inb             [1024]complex128
	chanDemodDecode chan demodulatedObj
}

type demodulatedObj struct {
	bitFloat float64
	level    float64
}

func demodulateMsk(demodulationBuffer []float64, sampleRate int, mskDemodulation *demodMskObj) {
	idx := mskDemodulation.idx
	FLEN := ((sampleRate / 1200) + 1)
	j := 0
	o := 0
	PLLC := 3.8e-3

	for i := 0; i < len(demodulationBuffer); i++ {

		// VCO
		s := float64(1800.0/float64(sampleRate)*2*math.Pi + mskDemodulation.mskDf)
		mskDemodulation.MskPhi += s
		if mskDemodulation.MskPhi >= 2*math.Pi {
			mskDemodulation.MskPhi -= 2 * math.Pi
		}
		if mskDemodulation.MskPhi < 0.0 {
			mskDemodulation.MskPhi += 2 * math.Pi
		}

		// Mixer
		in := demodulationBuffer[i]
		mskDemodulation.inb[idx] = complex(
			in*math.Cos(mskDemodulation.MskPhi),
			in*-math.Sin(mskDemodulation.MskPhi),
		)
		idx = (idx + 1) % FLEN

		// Bit clock
		mskDemodulation.mskClk += s
		if mskDemodulation.mskClk >= 3*math.Pi/2.0 {
			var vo, lvl, dphi float64
			mskDemodulation.mskClk -= 3 * math.Pi / 2.0

			// Matched Filter
			o = FLEN - idx
			v := complex(0, 0)
			for j = 0; j < FLEN; j, o = j+1, o+1 {
				v += complex(
					mskDemodulation.h[o]*real(mskDemodulation.inb[j]),
					mskDemodulation.h[o]*imag(mskDemodulation.inb[j]))
			}

			// Normalize
			lvl = cmplx.Abs(v)
			v = complex(
				real(v)/(lvl+1e-6),
				imag(v)/(lvl+1e-6),
			)

			mskDemodulation.msklvl = 0.99*mskDemodulation.msklvl + 0.01*lvl/5.2

			if mskDemodulation.mskS&3 == 0 {
				vo = real(v)
				mskDemodulation.chanDemodDecode <- demodulatedObj{vo, mskDemodulation.msklvl}

				if vo >= 0 {
					dphi = imag(v)
				} else {
					dphi = -imag(v)
				}

			} else if mskDemodulation.mskS&3 == 1 {
				vo = imag(v)
				mskDemodulation.chanDemodDecode <- demodulatedObj{vo, mskDemodulation.msklvl}

				if vo >= 0 {
					dphi = -real(v)
				} else {
					dphi = real(v)
				}

			} else if mskDemodulation.mskS&3 == 2 {
				vo = real(v)
				mskDemodulation.chanDemodDecode <- demodulatedObj{-vo, mskDemodulation.msklvl}

				if vo >= 0 {
					dphi = imag(v)
				} else {
					dphi = -imag(v)
				}

			} else if mskDemodulation.mskS&3 == 3 {
				vo = imag(v)
				mskDemodulation.chanDemodDecode <- demodulatedObj{-vo, mskDemodulation.msklvl}

				if vo >= 0 {
					dphi = -real(v)
				} else {
					dphi = real(v)
				}
			}
			mskDemodulation.mskS++
			mskDemodulation.mskDf = PLLC * dphi
		}
	}
	mskDemodulation.idx = idx
}
