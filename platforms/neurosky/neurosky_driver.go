package neurosky

import (
	"bytes"

	"github.com/hybridgroup/gobot"
)

const BTSync byte = 0xAA
const CodeEx byte = 0x55            // Extended code
const CodeSignalQuality byte = 0x02 // POOR_SIGNAL quality 0-255
const CodeAttention byte = 0x04     // ATTENTION eSense 0-100
const CodeMeditation byte = 0x05    // MEDITATION eSense 0-100
const CodeBlink byte = 0x16         // BLINK strength 0-255
const CodeWave byte = 0x80          // RAW wave value: 2-byte big-endian 2s-complement
const CodeAsicEEG byte = 0x83       // ASIC EEG POWER 8 3-byte big-endian integers

type NeuroskyDriver struct {
	gobot.Driver
}

type EEG struct {
	Delta    int
	Theta    int
	LoAlpha  int
	HiAlpha  int
	LoBeta   int
	HiBeta   int
	LoGamma  int
	MidGamma int
}

func NewNeuroskyDriver(a *NeuroskyAdaptor, name string) *NeuroskyDriver {
	n := &NeuroskyDriver{
		Driver: *gobot.NewDriver(
			name,
			"NeuroskyDriver",
			a,
		),
	}

	n.AddEvent("extended")
	n.AddEvent("signal")
	n.AddEvent("attention")
	n.AddEvent("meditation")
	n.AddEvent("blink")
	n.AddEvent("wave")
	n.AddEvent("eeg")

	return n
}

func (n *NeuroskyDriver) adaptor() *NeuroskyAdaptor {
	return n.Adaptor().(*NeuroskyAdaptor)
}
func (n *NeuroskyDriver) Start() bool {
	go func() {
		for {
			var buff = make([]byte, int(2048))
			_, err := n.adaptor().sp.Read(buff[:])
			if err != nil {
				panic(err)
			} else {
				n.parse(bytes.NewBuffer(buff))
			}
		}
	}()
	return true
}
func (n *NeuroskyDriver) Halt() bool { return true }

func (n *NeuroskyDriver) parse(buf *bytes.Buffer) {
	for buf.Len() > 2 {
		b1, _ := buf.ReadByte()
		b2, _ := buf.ReadByte()
		if b1 == BTSync && b2 == BTSync {
			length, _ := buf.ReadByte()
			var payload = make([]byte, int(length))
			buf.Read(payload)
			//checksum, _ := buf.ReadByte()
			buf.Next(1)
			n.parsePacket(payload)
		}
	}
}

func (n *NeuroskyDriver) parsePacket(data []byte) {
	buf := bytes.NewBuffer(data)
	for buf.Len() > 0 {
		b, _ := buf.ReadByte()
		switch b {
		case CodeEx:
			gobot.Publish(n.Event("extended"), nil)
		case CodeSignalQuality:
			ret, _ := buf.ReadByte()
			gobot.Publish(n.Event("signal"), ret)
		case CodeAttention:
			ret, _ := buf.ReadByte()
			gobot.Publish(n.Event("attention"), ret)
		case CodeMeditation:
			ret, _ := buf.ReadByte()
			gobot.Publish(n.Event("meditation"), ret)
		case CodeBlink:
			ret, _ := buf.ReadByte()
			gobot.Publish(n.Event("blink"), ret)
		case CodeWave:
			buf.Next(1)
			var ret = make([]byte, 2)
			buf.Read(ret)
			gobot.Publish(n.Event("wave"), ret)
		case CodeAsicEEG:
			var ret = make([]byte, 25)
			i, _ := buf.Read(ret)
			if i == 25 {
				gobot.Publish(n.Event("eeg"), n.parseEEG(ret))
			}
		}
	}
}

func (n *NeuroskyDriver) parseEEG(data []byte) EEG {
	return EEG{
		Delta:    n.parse3ByteInteger(data[0:3]),
		Theta:    n.parse3ByteInteger(data[3:6]),
		LoAlpha:  n.parse3ByteInteger(data[6:9]),
		HiAlpha:  n.parse3ByteInteger(data[9:12]),
		LoBeta:   n.parse3ByteInteger(data[12:15]),
		HiBeta:   n.parse3ByteInteger(data[15:18]),
		LoGamma:  n.parse3ByteInteger(data[18:21]),
		MidGamma: n.parse3ByteInteger(data[21:25]),
	}
}

func (n *NeuroskyDriver) parse3ByteInteger(data []byte) int {
	return ((int(data[0]) << 16) | (((1 << 16) - 1) & (int(data[1]) << 8)) | (((1 << 8) - 1) & int(data[2])))
}
