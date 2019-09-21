// Package tzx implements reading of ZX Spectrum TZX formatted files,
// as specified in the TZX specification.
// https://www.worldofspectrum.org/TZXformat.html
//
// Rules and Definitions
//
//  * Any value requiring more than one byte is stored in little endian format (i.e. LSB first).
//  * Unused bits should be set to zero.
//  * Timings are given in Z80 clock ticks (T states) unless otherwise stated.
//      1 T state = (1/3500000)s
//  * Block IDs are given in hex.
//  * All ASCII texts use the ISO 8859-1 (Latin 1) encoding; some of them can have several lines, which
//    should be separated by ASCII code 13 decimal (0D hex).
//  * You might interpret 'full-period' as ----____ or ____----, and 'half-period' as ---- or ____.
//    One 'half-period' will also be referred to as a 'pulse'.
//  * Values in curly brackets {} are the default values that are used in the Spectrum ROM saving
//    routines. These values are in decimal.
//  * If there is no pause between two data blocks then the second one should follow immediately; not
//    even so much as one T state between them.
//  * This document refers to 'high' and 'low' pulse levels. Whether this is implemented as ear=1 and
//    ear=0 respectively or the other way around is not important, as long as it is done consistently.
//  * Zeros and ones in 'Direct recording' blocks mean low and high pulse levels respectively.
//    The 'current pulse level' after playing a Direct Recording block of CSW recording block
//    is the last level played.
//  * The 'current pulse level' after playing the blocks ID 10,11,12,13,14 or 19 is the opposite of
//    the last pulse level played, so that a subsequent pulse will produce an edge.
//  * A 'Pause' block consists of a 'low' pulse level of some duration. To ensure that the last edge
//    produced is properly finished there should be at least 1 ms. pause of the opposite level and only
//    after that the pulse should go to 'low'. At the end of a 'Pause' block the 'current pulse level'
//    is low (note that the first pulse will therefore not immediately produce an edge). A 'Pause' block
//    of zero duration is completely ignored, so the 'current pulse level' will NOT change in this case.
//    This also applies to 'Data' blocks that have some pause duration included in them.
//  * An emulator should put the 'current pulse level' to 'low' when starting to play a TZX file, either
//    from the start or from a certain position. The writer of a TZX file should ensure that the 'current
//    pulse level' is well-defined in every sequence of blocks where this is important, i.e. in any
//    sequence that includes a 'Direct recording' block, or that depends on edges generated by 'Pause'
//    blocks. The recommended way of doing this is to include a Pause after each sequence of blocks.
//  * When creating a 'Direct recording' block please stick to the standard sampling frequencies of 22050
//    or 44100 Hz. This will ensure correct playback when using PC's sound cards.
//  * The length of a block is given in the following format: numbers in square brackets [] mean that the
//    value must be read from the offset in the brackets. Other values are normal numbers.
//    Example: [02,03]+0A means: get number (a word) from offset 02 and add 0A. All numbers are in hex.
//  * General Extension Rule: ALL custom blocks that will be added after version 1.10 will have the length
//    of the block in first 4 bytes (long word) after the ID (this length does not include these 4 length
//    bytes). This should enable programs that can only handle older versions to skip that block.
//  * Just in case:
//      MSB = most significant byte
//      LSB = least significant byte
//      MSb = most significant bit
//      LSb = least significant bit
package tzx

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/pkg/errors"

	"github.com/mrcook/tzxit/tape"
	"github.com/mrcook/tzxit/tzx/blocks"
)

const (
	supportedMajorVersion = 1
	supportedMinorVersion = 20
)

// Reader wraps a bufio.Reader that can be used to read binary data from a tape
// file, but also provides addition functions for reading TZX files.
//
// TZX files store the header information at the start of the file, followed
// by zero or more data blocks. Some TZX files include an ArchiveInfo block,
// which is always stored as the first block, directly after the header.
type Reader struct {
	reader *bufio.Reader

	header  // valid after NewReader
	archive blocks.ArchiveInfo
	blocks  []tape.Block
}

// Header is the first block of data found in all TZX files.
// The file is identified with the first 7 bytes being `ZXTape!`, followed by the
// _end of file_ byte `26` (`1A` hex). This is followed by two bytes containing
// the major and minor version numbers of the TZX specification used.
type header struct {
	Signature    [7]byte // must be `ZXTape!`
	Terminator   uint8   // End of file marker
	MajorVersion uint8   // TZX major revision number
	MinorVersion uint8   // TZX minor revision number
}

// NewReader wraps the given buffered Reader and creates a new TZX Reader.
// NOTE: It's the caller's responsibility to call Close on the Reader when done.
//
// The Reader.header fields will be valid in the Reader returned.
func NewReader(r *bufio.Reader) (*Reader, error) {
	t := &Reader{reader: r}

	if err := t.readHeader(); err != nil {
		return nil, err
	}
	if err := t.header.valid(); err != nil {
		return nil, err
	}

	return t, nil
}

// ReadBlocks processes each TZX blocks in the tape file.
func (r *Reader) ReadBlocks() error {
	for {
		blockID, err := r.reader.ReadByte()
		if err != nil && err == io.EOF {
			break // no problems, we're done!
		} else if err != nil {
			return err
		}

		block, err := r.readDataBlock(blockID)
		if err != nil {
			// should never be an EOF error for valid tape files
			return errors.Wrap(err, "Unable to complete reading TZX blocks")
		}

		// TODO: improve how we handle this!
		if blockID == 0x32 {
			r.archive = blocks.ArchiveInfo{}
			r.archive.Read(r.reader)
		} else {
			r.blocks = append(r.blocks, block)
		}
	}
	return nil
}

// DisplayTapeMetadata outputs the metadata, archive info, data blocks, etc., to the terminal.
func (r Reader) DisplayTapeMetadata() {
	fmt.Println("Tzxit processing complete!")
	fmt.Println()
	fmt.Println("ARCHIVE INFORMATION:")
	fmt.Println(r.archive.ToString())

	fmt.Println("DATA BLOCKS:")
	for i, block := range r.blocks {
		fmt.Printf("#%d %s\n", i+1, block.ToString())
	}

	fmt.Println()
	fmt.Printf("TZX revision: v%d.%d\n", r.MajorVersion, r.MinorVersion)
}

// readHeader reads the tape header data and validates that the format is correct.
func (r *Reader) readHeader() error {
	r.header = header{}
	if err := binary.Read(r.reader, binary.LittleEndian, &r.header); err != nil {
		return fmt.Errorf("binary.Read failed: %v", err)
	}

	if string(r.header.Signature[:]) != "ZXTape!" {
		return fmt.Errorf("TZX file is not in correct format")
	}

	return nil
}

// readDataBlock reads the TZX data for the given block ID.
func (r Reader) readDataBlock(id byte) (tape.Block, error) {
	var block tape.Block

	switch id {
	case 0x10:
		block = &blocks.StandardSpeedData{}
	case 0x11:
		block = &blocks.TurboSpeedData{}
	case 0x12:
		block = &blocks.PureTone{}
	case 0x13:
		block = &blocks.SequenceOfPulses{}
	case 0x14:
		block = &blocks.PureData{}
	case 0x15:
		block = &blocks.DirectRecording{}
	case 0x18:
		block = &blocks.CswRecording{}
	case 0x19:
		block = &blocks.GeneralizedData{}
	case 0x20:
		block = &blocks.PauseTapeCommand{}
	case 0x21:
		block = &blocks.GroupStart{}
	case 0x22:
		block = &blocks.GroupEnd{}
	case 0x23:
		block = &blocks.JumpTo{}
	case 0x24:
		block = &blocks.LoopStart{}
	case 0x25:
		block = &blocks.LoopEnd{}
	case 0x26:
		block = &blocks.CallSequence{}
	case 0x27:
		block = &blocks.ReturnFromSequence{}
	case 0x28:
		block = &blocks.Select{}
	case 0x2a:
		block = &blocks.StopTapeWhen48kMode{}
	case 0x2b:
		block = &blocks.SetSignalLevel{}
	case 0x30:
		block = &blocks.TextDescription{}
	case 0x31:
		block = &blocks.Message{}
	case 0x32:
		// should never reach here, handle separately!
		return nil, nil
	case 0x33:
		block = &blocks.HardwareType{}
	case 0x35:
		block = &blocks.CustomInfo{}
	case 0x5a: // (90 dec, ASCII Letter 'Z')
		block = &blocks.GlueBlock{}
	default:
		// probably ID's 16,17,34,35,40 (HEX)
		return nil, fmt.Errorf("TZX block ID 0x%02X is deprecated/not supported", id)
	}
	block.Read(r.reader)

	return block, nil
}

// Validates the TZX header data.
func (h header) valid() error {
	var validationError error

	sig := [7]byte{}
	copy(sig[:], "ZXTape!")
	if h.Signature != sig {
		validationError = errors.Wrapf(validationError, "Incorrect signature, got '%s'", h.Signature)
	}

	if h.Terminator != 0x1a {
		validationError = errors.Wrapf(validationError, "Incorrect terminator, got '%b'", h.Terminator)
	}

	if h.MajorVersion != supportedMajorVersion {
		validationError = errors.Wrapf(validationError, "Invalid version, got v%d.%d", h.MajorVersion, h.MinorVersion)
	} else if h.MinorVersion < supportedMinorVersion {
		fmt.Printf(
			"WARNING! Expected TZX v%d.%d but got v%d.%d. This may lead to unexpected data or errors.\n",
			supportedMajorVersion,
			supportedMinorVersion,
			h.MajorVersion,
			h.MinorVersion,
		)
		fmt.Println()
	}

	return validationError
}
