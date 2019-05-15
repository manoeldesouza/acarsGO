package main

import (
	"math"
	"math/cmplx"
)

type channelObj struct {
	channelFrequency       float64
	centralFrequency       float64
	localOscilator         []complex128
	sampleRate             int
	decimationFactor       int
	demodulationBufferReal []float64
	mskDemodulation        demodMskObj
}

func (channel *channelObj) initialize(channelFrequency float64, centralFrequency float64, sampleRate int, sampleSize int, channelOutput chan outputObj) {
	channel.sampleRate = 12500 // 12000 usado no acars-sdrplay (testar)
	channel.channelFrequency = channelFrequency
	channel.centralFrequency = centralFrequency
	channel.decimationFactor = sampleRate / channel.sampleRate
	channel.localOscilator = make([]complex128, sampleSize*1024)

	shiftFrequency := (channelFrequency - centralFrequency) * 1e6
	shiftFactor := shiftFrequency / float64(sampleRate)

	FLEN := ((channel.sampleRate / 1200) + 1)
	for i := 0; i < len(channel.localOscilator); i++ {
		channel.localOscilator[i] = cmplx.Rect(1, -shiftFactor*2.0*math.Pi*float64(i))
	}

	for i := 0; i < FLEN; i++ {
		channel.mskDemodulation.h[i] = math.Cos(2.0 * math.Pi * 600.0 / float64(channel.sampleRate) * float64(i-FLEN/2))
		channel.mskDemodulation.h[i+FLEN] = channel.mskDemodulation.h[i]
		channel.mskDemodulation.inb[i] = 0
	}

	channel.mskDemodulation.chanDemodDecode = make(chan demodulatedObj, 256)
	go acarsFrameDecoder(channelFrequency, channel.mskDemodulation.chanDemodDecode, channelOutput)
}

func (channel *channelObj) process(channelRtl chan []complex128) {

	for {
		complexBuffer := <-channelRtl
		i, ii := 0, 0
		local := complex(0, 0)

		mixedSignal := make([]complex128, len(complexBuffer))
		channel.demodulationBufferReal = make([]float64, len(complexBuffer)/channel.decimationFactor+1)

		for i = range complexBuffer {
			mixedSignal[i] = complexBuffer[i] * channel.localOscilator[i]
			if i%channel.decimationFactor == 0 {
				r := real(local) / float64(channel.decimationFactor)
				g := imag(local) / float64(channel.decimationFactor)
				channel.demodulationBufferReal[ii] = cmplx.Abs(complex(r, g))
				local = complex(0, 0)
				ii++
			} else {
				local += mixedSignal[i]
			}
		}
		demodulateMsk(channel.demodulationBufferReal, channel.sampleRate, &channel.mskDemodulation)
	}

}
