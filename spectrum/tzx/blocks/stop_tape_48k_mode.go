package blocks

import (
	"fmt"

	"retroio/spectrum/tap"
	"retroio/storage"
)

// StopTapeWhen48kMode
// ID: 2Ah (42d)
// When this block is encountered, the tape will stop ONLY if the machine is an 48K Spectrum.
// This block is to be used for multi-loading games that load one level at a time in 48K mode,
// but load the entire tape at once if in 128K mode.
// This block has no body of its own, but follows the extension rule.
type StopTapeWhen48kMode struct {
	Length uint32 // DWORD Length of the block without these four bytes (0)
}

// Read the tape and extract the data.
// It is expected that the tape pointer is at the correct position for reading.
func (s *StopTapeWhen48kMode) Read(reader *storage.Reader) {
	s.Length = reader.ReadLong()
}

// Id of the block as given in the TZX specification, written as a hexadecimal number.
func (s StopTapeWhen48kMode) Id() uint8 {
	return 0x2a
}

// Name of the block as given in the TZX specification.
func (s StopTapeWhen48kMode) Name() string {
	return "Stop Tape when in 48k Mode"
}

func (s StopTapeWhen48kMode) BlockData() tap.BlockI {
	return nil
}

// String returns a human readable string of the block data
func (s StopTapeWhen48kMode) String() string {
	return fmt.Sprintf("%s", s.Name())
}
