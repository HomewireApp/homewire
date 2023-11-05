package network

import (
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

type MessageReceiver interface {
	GetServiceName() string
	GetProtocolID() protocol.ID
	GetReadTimeout() *time.Duration
	GetBufferSize() int
	HandleMessage(bytes []byte, pi peer.ID)
	HandleError(phase ListenerPhase, err error)
}
