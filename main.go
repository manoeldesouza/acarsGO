package main

import (
	"sync"
)

func main() {
	var wg sync.WaitGroup

	frequencies := []float64{
		// 129.125,
		131.550,
		131.725,
		// 120.425,
	}
	channelOutput := make(chan outputObj, 256)

	go controlRtl(0, frequencies, channelOutput)
	wg.Add(1)

	go controlOutput(channelOutput)
	wg.Add(1)

	wg.Wait()
}
