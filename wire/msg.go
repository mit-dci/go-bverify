package wire

type MessageType byte

const (
	MessageTypeCreateLog   MessageType = 0x01
	MessageTypeAppendLog   MessageType = 0x02
	MessageTypeAck         MessageType = 0x03
	MessageTypeError       MessageType = 0x04
	MessageTypeProofUpdate MessageType = 0x05
)
