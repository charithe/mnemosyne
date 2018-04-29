package internal

// magic codes
const (
	requestMagic  byte = 0x80
	responseMagic byte = 0x81
)

// status codes
const (
	StatusNoError                       uint16 = 0x0000
	StatusKeyNotFound                   uint16 = 0x0001
	StatusKeyExists                     uint16 = 0x0002
	StatusValueTooLarge                 uint16 = 0x0003
	StatusInvalidArguments              uint16 = 0x0004
	StatusItemNotStored                 uint16 = 0x0005
	StatusIncrOnNonNumericValue         uint16 = 0x0006
	StatusVBucketBelongsToAnotherServer uint16 = 0x0007
	StatusAuthenticationError           uint16 = 0x0008
	StatusUnknownCommand                uint16 = 0x0081
	StatusOutOfMemory                   uint16 = 0x0082
	StatusNotSupported                  uint16 = 0x0083
	StatusInternalError                 uint16 = 0x0084
	StatusBusy                          uint16 = 0x0085
	StatusTemporaryFailure              uint16 = 0x0086
)

// op codes
const (
	OpGet           uint8 = 0x00
	OpSet           uint8 = 0x01
	OpAdd           uint8 = 0x02
	OpReplace       uint8 = 0x03
	OpDelete        uint8 = 0x04
	OpIncrement     uint8 = 0x05
	OpDecrement     uint8 = 0x06
	OpQuit          uint8 = 0x07
	OpFlush         uint8 = 0x08
	OpGetQ          uint8 = 0x09
	OpNoOp          uint8 = 0x0a
	OpVersion       uint8 = 0x0b
	OpGetK          uint8 = 0x0c
	OpGetKQ         uint8 = 0x0d
	OpAppend        uint8 = 0x0e
	OpPrepend       uint8 = 0x0f
	OpStat          uint8 = 0x10
	OpSetQ          uint8 = 0x11
	OpAddQ          uint8 = 0x12
	OpReplaceQ      uint8 = 0x13
	OpDeleteQ       uint8 = 0x14
	OpIncrementQ    uint8 = 0x15
	OpDecrementQ    uint8 = 0x16
	OpQuitQ         uint8 = 0x17
	OpFlushQ        uint8 = 0x18
	OpAppendQ       uint8 = 0x19
	OpPrependQ      uint8 = 0x1a
	OpVerbosity     uint8 = 0x1b
	OpTouch         uint8 = 0x1c
	OpGAT           uint8 = 0x1d
	OpGATQ          uint8 = 0x1e
	OpSASLListMechs uint8 = 0x20
	OpSASLAuth      uint8 = 0x21
	OpSASLStep      uint8 = 0x22
	OpRGet          uint8 = 0x30
	OpRSet          uint8 = 0x31
	OpRSetQ         uint8 = 0x32
	OpRAppend       uint8 = 0x33
	OpRAppendQ      uint8 = 0x34
	OpRPrepend      uint8 = 0x35
	OpRPrependQ     uint8 = 0x36
	OpRDelete       uint8 = 0x37
	OpRDeleteQ      uint8 = 0x38
	OpRIncr         uint8 = 0x39
	OpRIncrQ        uint8 = 0x3a
	OpRDecr         uint8 = 0x3b
	OpRDecrQ        uint8 = 0x3c
)

var quietCommands = map[uint8]bool{
	OpGetQ:       true,
	OpGetKQ:      true,
	OpSetQ:       true,
	OpAddQ:       true,
	OpReplaceQ:   true,
	OpDeleteQ:    true,
	OpIncrementQ: true,
	OpDecrementQ: true,
	OpQuitQ:      true,
	OpFlushQ:     true,
	OpAppendQ:    true,
	OpPrependQ:   true,
	OpGATQ:       true,
	OpRSetQ:      true,
	OpRAppendQ:   true,
	OpRPrependQ:  true,
	OpRDeleteQ:   true,
	OpRIncrQ:     true,
	OpRDecrQ:     true,
}

func IsQuietCommand(opCode uint8) bool {
	_, ok := quietCommands[opCode]
	return ok
}
