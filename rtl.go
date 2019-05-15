package main

import (
	"log"
	"time"

	rtl "github.com/jpoirier/gortlsdr"
)

type rtlObj struct {
	dev              *rtl.Context
	sampleRate       int
	sampleSize       int
	centralFrequency float64
	channels         []channelObj
	channelsRtl      []chan []complex128
}

func controlRtl(deviceIndex int, frequencies []float64, channelOutput chan outputObj) {

	sampleRate := int(2.0e6) // 2 MHz
	intermediateRate := 62500
	// intermediateRate := 80000
	sampleSize := int(sampleRate / intermediateRate)

	for {
		var device rtlObj
		err := device.initialize(deviceIndex, frequencies, sampleRate, sampleSize, channelOutput)
		if err == nil {
			device.process()
		}

		log.Printf("Restarting RTL-SDR device %d in 5 secs...\n", deviceIndex)
		time.Sleep(5 * time.Second)
	}
}

func (device *rtlObj) initialize(deviceIndex int, frequencies []float64,
	sampleRate int, sampleSize int,
	channelOutput chan outputObj) (err error) {

	device.sampleRate = sampleRate
	device.sampleSize = sampleSize
	averageFrequency := 0.0
	deltaFrequency := 0.0

	for i := range frequencies {
		averageFrequency += frequencies[i]
	}
	averageFrequency /= float64(len(frequencies))
	if len(frequencies) == 1 {
		deltaFrequency = 0.150 // 250 KHz
	}

	device.dev, err = rtl.Open(deviceIndex)

	device.dev.SetCenterFreq(int((averageFrequency - deltaFrequency) * 1e6))
	device.dev.SetTunerGainMode(false)
	device.dev.SetSampleRate(sampleRate)
	device.dev.ResetBuffer()

	device.sampleRate = device.dev.GetSampleRate()
	device.centralFrequency = float64(device.dev.GetCenterFreq()) / 1e6
	log.Printf("RTL-SDR device %d tunned at %.3f MHz\n", deviceIndex, device.centralFrequency)

	device.channels = make([]channelObj, len(frequencies))
	device.channelsRtl = make([]chan []complex128, len(frequencies))
	for i := range frequencies {
		device.channelsRtl[i] = make(chan []complex128, 256)
		device.channels[i].initialize(frequencies[i], device.centralFrequency, device.sampleRate, device.sampleSize, channelOutput)
		go device.channels[i].process(device.channelsRtl[i])
	}

	return
}

func (device rtlObj) process() (err error) {

	bufferSize := device.sampleSize * 2 * 1024
	byteBuffer := make([]byte, bufferSize)
	complexBuffer := make([]complex128, bufferSize/2)

	for {
		_, err = device.dev.ReadSync(byteBuffer, bufferSize)
		if err != nil {
			break
		}

		for n, m := 0, 0; n < len(byteBuffer); n, m = n+2, m+1 {
			i := byteBuffer[n]
			q := byteBuffer[n+1]
			complexBuffer[m] = complex(float64(i)/(255.0/2.0)-1.0,
				float64(q)/(255.0/2.0)-1.0)
		}

		for i := range device.channels {
			device.channelsRtl[i] <- complexBuffer
		}
	}

	return
}
