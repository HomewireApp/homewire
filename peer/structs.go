package peer

import (
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Self struct {
	PeerId        peer.ID
	Id            string
	DisplayedName string
	PrivKey       crypto.PrivKey
	KnownWires    []*PeerWire
}

type PeerWire struct {
	Id   string
	Name string
}

type Peer struct {
	PeerId        peer.ID
	Id            string
	DisplayedName string
	PubKey        crypto.PubKey
	KnownWires    []*PeerWire
}

func (s *Self) GetPublicKey() crypto.PubKey {
	return s.PrivKey.GetPublic()
}
