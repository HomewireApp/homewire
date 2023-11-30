package homewire

import (
	"context"
	"sync"
	"time"

	ctx "github.com/HomewireApp/homewire/internal/context"
	"github.com/HomewireApp/homewire/internal/database"
	"github.com/HomewireApp/homewire/internal/peer_discovery"
	"github.com/HomewireApp/homewire/internal/proto"
	"github.com/HomewireApp/homewire/internal/proto/pb"
	"github.com/HomewireApp/homewire/internal/utils"
	"github.com/HomewireApp/homewire/options"
	"github.com/HomewireApp/homewire/peer"
	"github.com/HomewireApp/homewire/wire"
	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/ztrue/tracerr"
)

const discoveryTopicName = "homewire:discovery:wire"

type OnWireFoundHandler func(*wire.Wire)

type Homewire struct {
	Context *ctx.Context

	options       *options.Options
	db            *database.Database
	self          *peer.Self
	peerDiscovery *peer_discovery.PeerDiscovery

	wires    map[string]*wire.Wire
	mutWires *sync.RWMutex

	topic *pubsub.Topic
	subs  *pubsub.Subscription

	wireFoundHandlers []OnWireFoundHandler
	mutHooks          *sync.RWMutex
}

func InitWithOptions(opts options.Options) (*Homewire, error) {
	ctx := &ctx.Context{
		Context: context.Background(),
		Logger:  opts.Logger,
	}

	db, err := database.Open(opts.DatabasePath, ctx.Logger)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	ctx.DB = db

	if opts.MigrateDatabase {
		if err := db.Migrate(); err != nil {
			return nil, tracerr.Wrap(err)
		}
	}

	id, err := db.GetOrCreateIdentity(opts.ProfileName)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	privKey, err := utils.UnmarshalPrivateKey(id.PrivateKey)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	self := &peer.Self{
		Id:            id.Id,
		DisplayedName: id.DisplayedName,
		PrivKey:       privKey,
	}

	host, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/udp/0/quic-v1"))
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	ctx.Host = host

	pd, err := peer_discovery.Init(ctx, host, self)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	knownWireModels, err := db.FindAllWires()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	pubsub, err := pubsub.NewGossipSub(ctx.Context, host)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	ctx.Pubsub = pubsub

	topic, err := pubsub.Join(discoveryTopicName)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	subs, err := topic.Subscribe()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	wires := make(map[string]*wire.Wire)
	for _, model := range knownWireModels {
		wire, err := wire.CreateExistingKnownWire(ctx, model)

		if err != nil {
			return nil, tracerr.Wrap(err)
		}

		wires[wire.Id] = wire
	}

	hw := &Homewire{
		Context: ctx,

		options:       &opts,
		db:            db,
		self:          self,
		peerDiscovery: pd,

		wires:    wires,
		mutWires: &sync.RWMutex{},
		topic:    topic,
		subs:     subs,

		wireFoundHandlers: make([]OnWireFoundHandler, 0),
		mutHooks:          &sync.RWMutex{},
	}

	go hw.wireDiscoveryLoop()
	go hw.peerDiscoveryLoop()
	go hw.peerConnectednessLoop()

	return hw, nil
}

// Initializes homewire with the default options
func Init() (*Homewire, error) {
	opts, err := options.Default()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	return InitWithOptions(*opts)
}

func (hw *Homewire) peerDiscoveryLoop() {
	for {
		peer := <-hw.peerDiscovery.PeersFound

		hw.mutWires.Lock()

		for _, w := range peer.KnownWires {
			exists := hw.wires[w.Id] == nil

			if !exists {
				w := wire.CreateUnknownWire(hw.Context, w.Id, w.Name)
				hw.wires[w.Id] = w
			}

			hw.wires[w.Id].AddKnownProviderPeer(peer.PeerId)

			if !exists {
				go hw.afterWireFound(hw.wires[w.Id])
			}
		}

		hw.Context.Logger.Debug("[homewire:peerDiscoveryLoop] Got %d wires from new peer", len(peer.KnownWires))

		hw.mutWires.Unlock()
	}
}

func (hw *Homewire) peerConnectednessLoop() {
	for {
		peer := <-hw.peerDiscovery.PeersLost
		hw.mutWires.RLock()

		for _, w := range hw.wires {
			w.RemoveKnownProviderPeer(peer.PeerId)
		}
	}
}

func (hw *Homewire) wireDiscoveryLoop() {
	for {
		msg, err := hw.subs.Next(hw.Context.Context)

		if msg.ReceivedFrom == hw.Context.Host.Network().LocalPeer() {
			continue
		}

		if err != nil {
			hw.Context.Logger.Warn("[homewire:wireDiscoveryLoop] Failed to read next message from subscription")
			continue
		}

		env, err := proto.UnmarshalPlainEnvelope(msg.Data)
		if err != nil {
			hw.Context.Logger.Warn("[homewire:wireDiscoveryLoop] Failed to unmarshal incoming message")
			continue
		}

		if wireList := env.GetWireList(); wireList != nil {
			hw.mutWires.Lock()

			received := 0
			saved := 0

			for _, w := range wireList.Wires {
				received += 1

				if hw.wires[w.Id] == nil {
					w := wire.CreateUnknownWire(hw.Context, w.Id, w.Name)
					hw.wires[w.Id] = w
					go hw.afterWireFound(w)

					saved += 1
				}
			}

			hw.Context.Logger.Debug("[homewire:wireDiscoveryLoop] Received %v wires and saved %v of them", received, saved)

			hw.mutWires.Unlock()
		} else if ann := env.GetWireAnnouncement(); ann != nil {
			hw.mutWires.Lock()

			saved := false

			if hw.wires[ann.Id] == nil {
				w := wire.CreateUnknownWire(hw.Context, ann.Id, ann.Name)
				hw.wires[ann.Id] = w
				go hw.afterWireFound(w)

				w.AddKnownProviderPeer(msg.ReceivedFrom)

				saved = true
			}

			hw.Context.Logger.Debug("[homewire:wireDiscoveryLoop] Received announcement for new wire %v (saved? %v)", ann.Id, saved)

			hw.mutWires.Unlock()
		} else {
			hw.Context.Logger.Warn("[homewire:wireDiscoveryLoop] Unexpected message received from wire: %v", env)
			continue
		}
	}
}

func (hw *Homewire) afterWireFound(w *wire.Wire) {
	hw.mutHooks.RLock()
	defer hw.mutHooks.RUnlock()
	for _, handler := range hw.wireFoundHandlers {
		go handler(w)
	}
}

func (hw *Homewire) announceWireWhenConnected(w *wire.Wire) {
	timer := time.NewTimer(200 * time.Millisecond)
	retryInterval := 1 * time.Second

	for {
		<-timer.C

		if w.ConnectionStatus != wire.Connected {
			timer.Reset(retryInterval)
			continue
		}

		msg := &pb.Envelope{
			Payload: &pb.Envelope_WireAnnouncement{
				WireAnnouncement: &pb.BasicWireInfo{
					Id:   w.Id,
					Name: w.Name,
				},
			},
		}

		bytes, err := proto.MarshalPlain(msg)
		if err != nil {
			hw.Context.Logger.Debug("[homewire:announceWireWhenConnected] Failed to announce wire due to marshalling error, will retry later %v", err)
			timer.Reset(retryInterval)
			continue
		}

		if err := hw.topic.Publish(context.Background(), bytes); err != nil {
			hw.Context.Logger.Debug("[homewire:announceWireWhenConnected] Failed to announce wire due to message publishing error, will retry later %v", err)
			timer.Reset(retryInterval)
			continue
		}

		hw.Context.Logger.Debug("[homewire:announceWireWhenConnected] Successfully announced wire %v on topic %v", w.Id, hw.topic.String())
		timer.Stop()
	}
}

func (hw *Homewire) CreateNewWire(name string) (*wire.Wire, error) {
	w, err := wire.CreateNewKnownWire(hw.Context, name)

	if err != nil {
		return nil, err
	}

	hw.mutWires.Lock()
	defer hw.mutWires.Unlock()

	if hw.wires[w.Id] == nil {
		hw.wires[w.Id] = w
	}

	go hw.afterWireFound(w)

	go hw.announceWireWhenConnected(w)

	return w, nil
}

func (hw *Homewire) ListWires() []*wire.Wire {
	hw.mutWires.RLock()
	defer hw.mutWires.RUnlock()

	results := make([]*wire.Wire, 0, len(hw.wires))
	for _, w := range hw.wires {
		results = append(results, w)
	}

	return results
}

func (hw *Homewire) OnWireFound(handler OnWireFoundHandler) {
	hw.mutHooks.Lock()
	defer hw.mutHooks.Unlock()
	hw.wireFoundHandlers = append(hw.wireFoundHandlers, handler)
}

func (wm *Homewire) Destroy() {
	for _, w := range wm.wires {
		w.Destroy()
	}

	wm.peerDiscovery.Destroy()
	wm.Context.Destroy()
}
