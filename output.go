package main

import (
	"fmt"
	"time"
)

// Output type
const (
	PRINTRAWBLOCK = iota
	PRINTDETAILEDBLOCK
	PRINTDETAILEDFRAME
)

type outputObj struct {
	outputType int
	acarsFrame acarsFrameObj
}

func controlOutput(output chan outputObj) {

	for {
		outputValue := <-output

		if outputValue.outputType == PRINTRAWBLOCK {
			printAcarsCleanBlock(outputValue.acarsFrame.block.raw)
		}
		if outputValue.outputType == PRINTDETAILEDBLOCK {
			printAcarsDetailedBlock(outputValue.acarsFrame.block)
		}
		if outputValue.outputType == PRINTDETAILEDFRAME {
			printAcarsDetailedFrame(outputValue.acarsFrame)
		}
	}
}

func printAcarsCleanBlock(byteStream []byte) string {

	acarsBlockBytes := make([]byte, len(byteStream))

	i := 0
	for i = 0; i < len(byteStream); i++ {

		if byteStream[i] >= 0x20 && byteStream[i] < DEL {
			acarsBlockBytes[i] = byteStream[i]

		} else if byteStream[i] == DEL {
			acarsBlockBytes[i] = 'd'

		} else if byteStream[i] == ETX || byteStream[i] == ETB {
			break

		} else if i > 25 && byteStream[i] == NUL {
			break

		} else {
			acarsBlockBytes[i] = '.'
		}
	}

	return string(acarsBlockBytes[:i])
}

func printAcarsDetailedBlock(acarsBlock acarsBlockObj) string {

	return string(acarsBlock.text)
}

func printAcarsDetailedFrame(acarsFrame acarsFrameObj) {

	label := acarsFrame.block.label
	if label[1] == DEL {
		label[1] = 'd'
	}

	fmt.Printf(
		"Time: %s\nFreq: %3.3f MHz\n Lvl: %d\n Blk: [%s]\nMode: %c\n Add: %s\n Ack: 0x%x\n Lbl: %s\n Blk: 0x%x\n Slb: %s\n Msn: %s\n Flt: %s\n Txt: %s\n Sfx: 0x%x\nUplk: %t\nMult: %t\n\n",
		acarsFrame.receptionTime.Format(time.RFC1123),
		acarsFrame.frequency,
		acarsFrame.level,
		printAcarsCleanBlock(acarsFrame.block.raw),
		acarsFrame.block.mode,
		string(acarsFrame.block.address),
		acarsFrame.block.ackID,
		label,
		acarsFrame.block.blkID,
		string(acarsFrame.block.sublabel),
		string(acarsFrame.block.msn),
		string(acarsFrame.block.flightID),
		string(acarsFrame.block.text),
		acarsFrame.block.suffix,
		acarsFrame.block.isUplink,
		acarsFrame.block.isMultiblock,
	)
}
