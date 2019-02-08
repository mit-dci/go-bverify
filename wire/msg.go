package wire

type MessageType byte

const (
	MessageTypeCreateLog MessageType = 0x01
	MessageTypeAppendLog MessageType = 0x02
	MessageTypeAck       MessageType = 0x03
)
