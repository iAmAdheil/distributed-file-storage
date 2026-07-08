package p2p

const (
	IncomingMessageByte = 0x1
	// this is used to block the running decoder and
	// handle incoming data from the peer directly inside the func
	IncomingStreamByte = 0x2
)

type RPC struct {
	From    string
	Payload []byte
	Stream  bool
}
