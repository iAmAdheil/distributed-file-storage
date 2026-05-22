package p2p

const (
	IncomingMessageByte = 0x1
	IncomingStreamByte  = 0x2
)

type RPC struct {
	From    string
	Payload []byte
	Stream  bool
}
