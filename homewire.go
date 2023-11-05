package homewire

import (
	"context"
	"sync"
	"time"

	ctx "github.com/HomewireApp/homewire/internal/context"
	"github.com/HomewireApp/homewire/internal/database"
	"github.com/HomewireApp/homewire/internal/logger"
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
	mutWires *sync.Mutex

	pubsub *pubsub.PubSub
	topic  *pubsub.Topic
	subs   *pubsub.Subscription

	wireFoundHandlers []OnWireFoundHandler
	mutHooks          *sync.Mutex
}

// Starts a new Homewire with the specified options. If options is nil, uses the default options
func Open(opts *options.Options) (*Homewire, error) {
	ctx := &ctx.Context{
		Context: context.Background(),
	}

	if opts == nil {
		defaultOpts, err := options.Default()

		if err != nil {
			return nil, tracerr.Wrap(err)
		}

		opts = defaultOpts
	}

	db, err := database.Open(opts.DatabasePath)
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

	pd, err := peer_discovery.Init(host, self)
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

		options:       opts,
		db:            db,
		self:          self,
		peerDiscovery: pd,

		wires:    wires,
		mutWires: &sync.Mutex{},
		pubsub:   pubsub,
		topic:    topic,
		subs:     subs,

		wireFoundHandlers: make([]OnWireFoundHandler, 0),
		mutHooks:          &sync.Mutex{},
	}

	go hw.wireDiscoveryLoop()

	go func() {
		wireInfo := make([]*pb.BasicWireInfo, len(knownWireModels))
		for _, m := range knownWireModels {
			wireInfo = append(wireInfo, &pb.BasicWireInfo{
				Id:   m.Id,
				Name: m.Name,
			})
		}

		msg := &pb.Envelope{
			Payload: &pb.Envelope_WireList{
				WireList: &pb.WireList{
					Wires: wireInfo,
				},
			},
		}

		timer := time.NewTimer(3 * time.Second)
		go func() {
			for {
				<-timer.C

				bytes, err := proto.MarshalPlain(msg)
				if err != nil {
					logger.Warn("[homewire:open] Failed to marshal initial announcement message %v", err)
					timer.Reset(3 * time.Second)
					continue
				}

				if err := topic.Publish(hw.Context.Context, bytes); err != nil {
					logger.Warn("[homewire:open] Failed to publish initial announcement message %v", err)
					timer.Reset(3 * time.Second)
					continue
				}

				logger.Debug("[homewire:open] Successfully announced %v initial wires", len(wireInfo))
			}
		}()
	}()

	return hw, nil
}

func (hw *Homewire) wireDiscoveryLoop() {
	for {
		msgBytes, err := hw.subs.Next(hw.Context.Context)

		if err != nil {
			logger.Warn("[homewire:wireDiscoveryLoop] Failed to read next message from subscription")
			continue
		}

		env, err := proto.UnmarshalPlainEnvelope(msgBytes.Data)
		if err != nil {
			logger.Warn("[homewire:wireDiscoveryLoop] Failed to unmarshal incoming message")
			continue
		}

		ann := env.GetWireAnnouncement()
		if ann == nil {
			logger.Warn("[homewire:wireDiscoveryLoop] Unexpected message received from wire: %v", ann)
			continue
		}

		hw.mutWires.Lock()

		if hw.wires[ann.Id] == nil {
			w := wire.CreateUnknownWire(hw.Context, ann.Id, ann.Name)
			hw.wires[ann.Id] = w
			go hw.triggerOnWireFound(w)
		}

		hw.mutWires.Unlock()
	}
}

func (hw *Homewire) triggerOnWireFound(w *wire.Wire) {
	hw.mutHooks.Lock()
	defer hw.mutHooks.Unlock()
	for _, handler := range hw.wireFoundHandlers {
		go handler(w)
	}
}

func (hw *Homewire) CreateNewWire(name string) (*wire.Wire, error) {
	return wire.CreateNewKnownWire(hw.Context, name)
}

func (hw *Homewire) ListWires() []wire.Wire {
	hw.mutWires.Lock()
	defer hw.mutWires.Unlock()

	results := make([]wire.Wire, 0, len(hw.wires))
	for _, w := range hw.wires {
		results = append(results, *w)
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
