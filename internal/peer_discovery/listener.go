package peer_discovery

import (
	"time"

	"github.com/HomewireApp/homewire/internal/logger"
	"github.com/HomewireApp/homewire/internal/network"
	"github.com/HomewireApp/homewire/internal/proto"
	"github.com/HomewireApp/homewire/internal/utils"
	"github.com/HomewireApp/homewire/peer"
	p2ppeer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const identityServiceName = "homewire.identify"
const defaultBufferSize = 64 * 1024
const defaultReadTimeout = 60 * time.Second

type introductionMessageListener struct {
	discovery   *PeerDiscovery
	readTimeout time.Duration
	bufferSize  int
}

func newIdentityMessageListener(d *PeerDiscovery) *introductionMessageListener {
	result := &introductionMessageListener{
		discovery:   d,
		readTimeout: defaultReadTimeout,
		bufferSize:  defaultBufferSize,
	}

	return result
}

func (l *introductionMessageListener) GetServiceName() string {
	return identityServiceName
}

func (l *introductionMessageListener) GetProtocolID() protocol.ID {
	return discoveryProtocolId
}

func (l *introductionMessageListener) GetReadTimeout() *time.Duration {
	return &l.readTimeout
}

func (l *introductionMessageListener) GetBufferSize() int {
	return l.bufferSize
}

func (l *introductionMessageListener) HandleMessage(bytes []byte, pi p2ppeer.ID) {
	env, err := proto.UnmarshalPlainEnvelope(bytes)

	if err != nil {
		logger.Warn("Failed to read envelope from introduction message due to %v", err)
		return
	}

	id := env.GetIntroduction()
	if id == nil {
		logger.Debug("Received unexpected message from peer, ignoring...")
		return
	}

	pubKey, err := utils.UnmarshalPublicKey(id.PublicKey)
	if err != nil {
		logger.Debug("Failed to unmarshal public key sent by the peer due to %v", err)
		return
	}

	wirePeer := &peer.Peer{
		PeerId:        pi,
		Id:            id.Id,
		DisplayedName: id.DisplayedName,
		PubKey:        pubKey,
	}

	l.discovery.knownPeers[wirePeer.PeerId] = wirePeer
	logger.Info("Successfully identified peer: %v", wirePeer.Id)
	go func() { l.discovery.PeersFound <- wirePeer }()
}

func (l *introductionMessageListener) HandleError(phase network.ListenerPhase, err error) {
	logger.Warn("[discovery:identityMessageListener] An error has occurred during listener phase %v -- %v", phase, err)
}
