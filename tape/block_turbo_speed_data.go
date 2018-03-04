package tape

import "fmt"

// TurboSpeedData
// ID: 11h (17d)
// This block is very similar to the normal TAP block but with some additional info on the timings
// and other important differences. The same tape encoding is used as for the standard speed data
// block. If a block should use some non-standard sync or pilot tones (i.e. all sorts of protection
// schemes) then use the next three blocks to describe it.
type TurboSpeedData struct {
	PilotPulse      uint16  // WORD      Length of PILOT pulse {2168}
	SyncFirstPulse  uint16  // WORD      Length of SYNC first pulse {667}
	SyncSecondPulse uint16  // WORD      Length of SYNC second pulse {735}
	ZeroBitPulse    uint16  // WORD      Length of ZERO bit pulse {855}
	OneBitPulse     uint16  // WORD      Length of ONE bit pulse {1710}
	PilotTone       uint16  // WORD      Length of PILOT tone (number of pulses) {8063 header (flag<128), 3223 data (flag>=128)}
	UsedBits        uint8   // BYTE      Used bits in the last byte (other bits should be 0) {8} (e.g. if this is 6, then the bits used (x) in the last byte are: xxxxxx00, where MSb is the leftmost bit, LSb is the rightmost bit)
	Pause           uint16  // WORD      Pause after this block (ms.) {1000}
	Length          uint16  // N BYTE[3] Length of data that follow - NOTE: 3rd byte will always be 0 (correct?)
	LengthSpareByte uint8   // NOTE: `length` above uses only 2-bytes for value but specification says 3-bytes, so this is for the spare.
	Data            []uint8 // BYTE[N]   Data as in .TAP files
}

func (t *TurboSpeedData) Process(file *File) {
	t.PilotPulse = file.ReadShort()
	t.SyncFirstPulse = file.ReadShort()
	t.SyncSecondPulse = file.ReadShort()
	t.ZeroBitPulse = file.ReadShort()
	t.PilotTone = file.ReadShort()
	t.UsedBits, _ = file.ReadByte()
	t.Pause = file.ReadShort()
	t.Length = file.ReadShort()
	t.LengthSpareByte, _ = file.ReadByte()

	// Yep, we're discarding the data for the moment
	file.ReadBytes(int(t.Length))
}

func (t TurboSpeedData) Id() int {
	return 17
}

func (t TurboSpeedData) Name() string {
	return "Turbo Speed Data"
}

// Metadata returns a human readable string of the block data
func (t TurboSpeedData) Metadata() string {
	str := ""
	str += fmt.Sprintf("PilotPulse:      %d\n", t.PilotPulse)
	str += fmt.Sprintf("SyncFirstPulse:  %d\n", t.SyncFirstPulse)
	str += fmt.Sprintf("SyncSecondPulse: %d\n", t.SyncSecondPulse)
	str += fmt.Sprintf("ZeroBitPulse:    %d\n", t.ZeroBitPulse)
	str += fmt.Sprintf("PilotTone:       %d\n", t.PilotTone)
	str += fmt.Sprintf("UsedBits:        %d\n", t.UsedBits)
	str += fmt.Sprintf("Pause:           %d\n", t.Pause)
	str += fmt.Sprintf("Length:          %d\n", t.Length)

	return str
}
