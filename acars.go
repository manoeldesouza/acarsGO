package main

import (
	"bytes"
	"log"
	"math"
	"time"
)

// ASCII table
const (
	NUL = 0x00
	SOH = 0x01
	STX = 0x02
	ETX = 0x03
	EOT = 0x04
	ENQ = 0x05
	ACK = 0x06
	EL  = 0x07
	BS  = 0x08
	HT  = 0x09
	LF  = 0x0a
	VT  = 0x0b
	FF  = 0x0c
	CR  = 0x0d
	SO  = 0x0e
	SI  = 0x0f

	DLE = 0x10
	DC1 = 0x11
	DC2 = 0x12
	DC3 = 0x13
	DC4 = 0x14
	NAK = 0x15
	SYN = 0x16
	ETB = 0x17
	CAN = 0x18
	EM  = 0x19
	SUB = 0x1a
	ESC = 0x1b
	FS  = 0x1c
	GS  = 0x1d
	RS  = 0x1e
	US  = 0x1f

	DEL = 0x7f
)

// ACARS Frame stages
const (
	WSYN1 = iota
	WSYN2
	WSOH
	BLK
	CRC1
	CRC2
	END
)

type acarsFrameObj struct {
	receptionTime time.Time
	frequency     float64
	level         int
	block         acarsBlockObj
	crc1          byte
	crc2          byte
}

// Go routine to build the acars frame from SYN1 to CRC
func acarsFrameDecoder(frequency float64, chanDemodDecode chan demodulatedObj, channelOutput chan outputObj) {
	const maxHeaderSize = 23
	const maxPayload = 220
	const maxBlockSize = maxHeaderSize + maxPayload

	log.Printf("ACARS decoder at %.3f MHz\n", frequency)

	frameStage := WSYN1
	inByte := byte(0x00)
	nbits := 1
	blockLen := 0
	outputBlock := make([]byte, maxBlockSize+2)

	var acarsFrame acarsFrameObj
	for {
		demodulatedObj := <-chanDemodDecode

		inByte >>= 1
		if demodulatedObj.bitFloat > 0 {
			inByte |= 0x80
		}
		nbits--

		if nbits <= 0 {
			switch frameStage {
			case WSYN1:
				if inByte == SYN {
					frameStage, blockLen, nbits = WSYN2, 0, 8
				} else {
					frameStage, blockLen, nbits = WSYN1, 0, 0
				}

			case WSYN2:
				if inByte == SYN {
					frameStage, blockLen, nbits = WSOH, 0, 8
				} else {
					frameStage, blockLen, nbits = WSYN1, 0, 0
				}

			case WSOH:
				if inByte == SOH {
					acarsFrame.receptionTime = time.Now()
					acarsFrame.level = int(10 * math.Log10(demodulatedObj.level))
					acarsFrame.frequency = frequency

					frameStage, blockLen, nbits = BLK, 0, 8
				} else {
					frameStage, blockLen, nbits = WSYN1, 0, 0
				}

			case BLK:

				// CHECK PARITY REQUIRED

				nbits = 8
				inByte &= 0x7f

				if inByte == ETX || inByte == ETB || blockLen >= maxBlockSize {
					frameStage, nbits = CRC1, 8
				}

				outputBlock[blockLen] = inByte
				outputBlock[blockLen+1] = 0x00
				blockLen++

			case CRC1:
				acarsFrame.crc1 = inByte
				frameStage, nbits = CRC2, 8

			case CRC2:
				acarsFrame.crc2 = inByte
				frameStage, nbits = END, 8

			case END:
				frameStage, nbits = WSYN1, 0
				acarsFrame.block = acarsBlockBuilder(outputBlock, blockLen)

				channelOutput <- outputObj{PRINTDETAILEDFRAME, acarsFrame}
			}
		}
	}
}

type acarsBlockObj struct {
	raw           []byte
	mode          byte
	address       []byte
	ackID         byte
	label         []byte
	blkID         byte
	endOfPreamble byte
	text          []byte
	suffix        byte

	isUplink     bool
	isMultiblock bool

	msn      []byte
	flightID []byte
	sublabel []byte
}

func acarsBlockBuilder(acarsBlockBytes []byte, blockLen int) (acarsBlock acarsBlockObj) {
	/*
	   BLOCK STRUCTURE:
	   =======================================
	   2.C-FMWP0_dC
	   B.PR-MAQJ_d0.S14AJJ3242
	   2........SQ..02XSYULCYUL04527N07344WV136975/
	   2.C-FGKJ.SA6.M06AW806190EV235956V
	   B.PR-MAQ.H10.D11AJJ3242#DFBA47/A31947,1,1/AMDAR,REPORT/C1100,PR-MAQ,2337S,04639W/C203,102229,02370,0
	   ---------------------------------------
	   0123456789012345678901234567890123456789
	   ||      || |||   |Flight Identifier (Only mandatory in Downlinks)
	   ||      || |||Message Sequence Number (Only mandatory in Downlinks)
	   ||      || |||Text (Optional)
	   ||      || ||Start of Text (Optional)
	   ||      || |Block Identifier
	   ||      ||Label
	   ||      |Technical Ack
	   ||Aircraft
	   |Mode
	   ----------------------------------------
	*/
	acarsBlock.address = make([]byte, 7)
	acarsBlock.label = make([]byte, 2)

	acarsBlock.raw = acarsBlockBytes
	acarsBlock.mode = acarsBlockBytes[0]
	acarsBlock.address = acarsBlockBytes[1:8]
	acarsBlock.ackID = acarsBlockBytes[8]
	acarsBlock.label = acarsBlockBytes[9:11]
	acarsBlock.blkID = acarsBlockBytes[11]
	acarsBlock.endOfPreamble = acarsBlockBytes[12]
	acarsBlock.suffix = acarsBlockBytes[blockLen-1]

	if acarsBlock.blkID >= 0x40 || acarsBlock.blkID == NUL {
		acarsBlock.isUplink = true
	}

	if acarsBlock.suffix == ETB {
		acarsBlock.isMultiblock = true
	}

	if acarsBlock.endOfPreamble == STX {
		if acarsBlock.isUplink == false {
			acarsBlock.msn = acarsBlockBytes[13:17]
			acarsBlock.flightID = acarsBlockBytes[17:23]
			if blockLen > 25 {
				acarsBlock.text = make([]byte, blockLen-2)
				acarsBlock.text = acarsBlockBytes[23 : blockLen-1]
			}
		} else if blockLen > 15 {
			acarsBlock.text = make([]byte, blockLen-2)
			acarsBlock.text = acarsBlockBytes[13 : blockLen-1]
		}
	}

	if bytes.Equal(acarsBlock.label, []byte("H1")) {
		acarsBlock.sublabel = acarsBlockBytes[24:26]
	}

	return
}
