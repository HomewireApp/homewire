package peer_discovery

import (
	"context"
	"time"

	"github.com/HomewireApp/homewire/internal/logger"
	"github.com/HomewireApp/homewire/internal/network"
	"github.com/HomewireApp/homewire/internal/proto/messages"
	"github.com/HomewireApp/homewire/peer"
	"github.com/libp2p/go-libp2p/core/host"
	p2ppeer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/ztrue/tracerr"
)

const mdnsServiceTag = "homewire:mdns:discovery"
const discoveryProtocolId = protocol.ID("/homewire/id/1.0.0")

type PeerDiscovery struct {
	PeersFound chan *peer.Peer

	self *peer.Self
	ctx  context.Context
	host host.Host

	pendingIntroductions map[p2ppeer.ID]bool
	knownPeers           map[p2ppeer.ID]*peer.Peer
	introductionTicker   *time.Ticker

	identityListener *introductionMessageListener

	// destroySignal  chan bool
	// svcCloseSignal chan bool
}

func Init(h host.Host, self *peer.Self) (*PeerDiscovery, error) {
	disc := &PeerDiscovery{
		PeersFound:           make(chan *peer.Peer, 16),
		self:                 self,
		ctx:                  context.Background(),
		host:                 h,
		pendingIntroductions: make(map[p2ppeer.ID]bool),
		knownPeers:           make(map[p2ppeer.ID]*peer.Peer),
		introductionTicker:   time.NewTicker(5 * time.Second),
		// destroySignal:        make(chan bool, 1),
		// svcCloseSignal:       make(chan bool, 1),
	}

	svc := mdns.NewMdnsService(h, mdnsServiceTag, disc)
	if err := svc.Start(); err != nil {
		return nil, tracerr.Wrap(err)
	}

	disc.identityListener = newIdentityMessageListener(disc)
	network.AttachListener(h, disc.identityListener)

	// go func() {
	// 	<-result.destroySignal
	// 	svc.Close()
	// 	result.svcCloseSignal <- true
	// }()

	go disc.introductionLoop()

	return disc, nil
}

func (n *PeerDiscovery) HandlePeerFound(pi p2ppeer.AddrInfo) {
	if err := n.host.Connect(n.ctx, pi); err != nil {
		logger.Debug("Failed to dial peer %v due to %v", pi, err)
		return
	}

	logger.Debug("Successfully connected to peer %v", pi.ID)
	n.pendingIntroductions[pi.ID] = false
}

func (d *PeerDiscovery) GetKnownPeer(id p2ppeer.ID) *peer.Peer {
	return d.knownPeers[id]
}

func (d *PeerDiscovery) Destroy() {
	d.introductionTicker.Stop()
	// d.destroySignal <- true
	// <-d.svcCloseSignal
}

func (d *PeerDiscovery) introductionLoop() {
	for {
		<-d.introductionTicker.C

		for pi, isPending := range d.pendingIntroductions {
			if isPending {
				continue
			}

			go d.introduceSelfToPeer(pi)
		}
	}
}

func (d *PeerDiscovery) introduceSelfToPeer(pi p2ppeer.ID) {
	conn := network.GetFirstConnectionToPeer(d.host, pi)

	if conn == nil {
		return
	}

	d.pendingIntroductions[pi] = true

	envelope, err := messages.Introduction(d.self)
	if err != nil {
		logger.Warn("Failed to marshal identity message due to %v", err)
		d.pendingIntroductions[pi] = true
		return
	}

	msg := &network.PlainOutboundMessage{
		Recipient:   pi,
		ProtocolID:  discoveryProtocolId,
		ServiceName: mdnsServiceTag,
		Payload:     envelope,
	}

	err = network.SendMessageToPeer(conn, msg)
	if err != nil {
		logger.Warn("Failed to send identity message to peer %v due to %v", pi, err)
		d.pendingIntroductions[pi] = false
		return
	}

	logger.Info("Introduced self to %v", pi)
	delete(d.pendingIntroductions, pi)
}
