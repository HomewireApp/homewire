package peer_discovery

import (
	"time"

	"github.com/HomewireApp/homewire/internal/context"
	"github.com/HomewireApp/homewire/internal/network"
	"github.com/HomewireApp/homewire/internal/proto/messages"
	"github.com/HomewireApp/homewire/peer"
	"github.com/libp2p/go-libp2p/core/host"
	p2pnet "github.com/libp2p/go-libp2p/core/network"
	p2ppeer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/ztrue/tracerr"
)

const mdnsServiceTag = "homewire:mdns:discovery"
const discoveryProtocolId = protocol.ID("/homewire/id/1.0.0")

const connectednessLoopInterval = 5 * time.Second

type PeerDiscovery struct {
	PeersFound chan *peer.Peer
	PeersLost  chan *peer.Peer

	self *peer.Self
	ctx  *context.Context
	host host.Host

	pendingIntroductions map[p2ppeer.ID]bool
	knownPeers           map[p2ppeer.ID]*peer.Peer
	connectednessTimer   *time.Timer

	identityListener *introductionMessageListener

	// destroySignal  chan bool
	// svcCloseSignal chan bool
}

type discoveryNotifee struct {
	host host.Host
	pd   *PeerDiscovery
}

func (dn *discoveryNotifee) HandlePeerFound(pi p2ppeer.AddrInfo) {
	if err := dn.pd.host.Connect(dn.pd.ctx.Context, pi); err != nil {
		dn.pd.ctx.Logger.Debug("[discoveryNotifee:HandlePeerFound] Failed to dial peer %v due to %v", pi, err)
		return
	}

	dn.pd.ctx.Logger.Debug("[discoveryNotifee:HandlePeerFound] Successfully connected to peer %v", pi.ID)
	dn.pd.pendingIntroductions[pi.ID] = false
}

func Init(ctx *context.Context, h host.Host, self *peer.Self) (*PeerDiscovery, error) {
	disc := &PeerDiscovery{
		PeersFound:           make(chan *peer.Peer, 16),
		PeersLost:            make(chan *peer.Peer, 16),
		self:                 self,
		ctx:                  ctx,
		host:                 h,
		pendingIntroductions: make(map[p2ppeer.ID]bool),
		knownPeers:           make(map[p2ppeer.ID]*peer.Peer),
		connectednessTimer:   time.NewTimer(connectednessLoopInterval),
		// destroySignal:        make(chan bool, 1),
		// svcCloseSignal:       make(chan bool, 1),
	}

	dn := &discoveryNotifee{pd: disc, host: h}
	svc := mdns.NewMdnsService(h, mdnsServiceTag, dn)
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

	go disc.connectednessLoop()

	return disc, nil
}

func (pd *PeerDiscovery) GetKnownPeer(id p2ppeer.ID) *peer.Peer {
	return pd.knownPeers[id]
}

func (pd *PeerDiscovery) Destroy() {
	pd.connectednessTimer.Stop()
	// d.destroySignal <- true
	// <-d.svcCloseSignal
}

func (pd *PeerDiscovery) connectednessLoop() {
	for {
		<-pd.connectednessTimer.C

		for _, p := range pd.host.Peerstore().Peers() {
			if p == pd.self.PeerId {
				continue
			}

			peer := pd.knownPeers[p]
			if peer == nil {
				continue
			}

			state := pd.host.Network().Connectedness(p)

			if state == p2pnet.Connected || state == p2pnet.CanConnect {
				continue
			}

			pd.ctx.Logger.Debug("[peer_discovery:connectednessLoop] Lost connection to peer %v", p)
			delete(pd.knownPeers, p)
			go func() { pd.PeersLost <- peer }()
		}

		for pi, isPending := range pd.pendingIntroductions {
			if isPending {
				continue
			}

			go pd.introduceSelfToPeer(pi)
		}

		pd.connectednessTimer.Reset(connectednessLoopInterval)
	}
}

func (pd *PeerDiscovery) introduceSelfToPeer(pi p2ppeer.ID) {
	conn := network.GetFirstConnectionToPeer(pd.host, pi)
	if conn == nil {
		return
	}

	pd.pendingIntroductions[pi] = true

	wires, err := pd.ctx.DB.FindAllWires()
	if err != nil {
		pd.ctx.Logger.Warn("[peer_discovery:introduceSelfToPeer] Failed to load list of wires from database %v", err)
		pd.pendingIntroductions[pi] = false
		return
	}

	peerWires := make([]*peer.PeerWire, 0, len(wires))
	for _, w := range wires {
		peerWires = append(peerWires, &peer.PeerWire{
			Id:   w.Id,
			Name: w.Name,
		})
	}

	envelope, err := messages.Introduction(pd.self, peerWires)
	if err != nil {
		pd.ctx.Logger.Warn("[peer_discovery:introduceSelfToPeer] Failed to marshal identity message due to %v", err)
		pd.pendingIntroductions[pi] = false
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
		pd.ctx.Logger.Warn("[peer_discovery:introduceSelfToPeer] Failed to send identity message to peer %v due to %v", pi, err)
		pd.pendingIntroductions[pi] = false
		return
	}

	pd.ctx.Logger.Debug("[peer_discovery:introduceSelfToPeer] Introduced self to %v", pi)
	delete(pd.pendingIntroductions, pi)
}
