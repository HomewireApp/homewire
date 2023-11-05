package network

import (
	"time"

	"github.com/HomewireApp/homewire/internal/proto"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	protobuf "google.golang.org/protobuf/proto"
)

type OutboundMessage interface {
	GetRecipient() peer.ID
	GetServiceName() string
	GetProtocolID() protocol.ID
	GetTimeout() *time.Duration
	Marshal() ([]byte, error)
}

type PlainOutboundMessage struct {
	Recipient   peer.ID
	ServiceName string
	ProtocolID  protocol.ID
	Payload     protobuf.Message
	Timeout     *time.Duration
}

type EncryptedOutboundMessage struct {
	Recipient   peer.ID
	PrivKey     crypto.PrivKey
	ServiceName string
	ProtocolID  protocol.ID
	Payload     protobuf.Message
	Timeout     *time.Duration
}

func (m *PlainOutboundMessage) GetRecipient() peer.ID {
	return m.Recipient
}

func (m *PlainOutboundMessage) GetServiceName() string {
	return m.ServiceName
}

func (m *PlainOutboundMessage) GetProtocolID() protocol.ID {
	return m.ProtocolID
}

func (m *PlainOutboundMessage) GetTimeout() *time.Duration {
	return m.Timeout
}

func (m *PlainOutboundMessage) Marshal() ([]byte, error) {
	return proto.MarshalPlain(m.Payload)
}

func (m *EncryptedOutboundMessage) GetRecipient() peer.ID {
	return m.Recipient
}

func (m *EncryptedOutboundMessage) GetServiceName() string {
	return m.ServiceName
}

func (m *EncryptedOutboundMessage) GetProtocolID() protocol.ID {
	return m.ProtocolID
}

func (m *EncryptedOutboundMessage) GetTimeout() *time.Duration {
	return m.Timeout
}

func (m *EncryptedOutboundMessage) Marshal() ([]byte, error) {
	return proto.MarshalAndSign(m.Payload, m.PrivKey)
}
