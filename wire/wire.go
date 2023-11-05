package wire

import (
	"errors"
	"sync"
	"time"

	ctx "github.com/HomewireApp/homewire/internal/context"
	"github.com/HomewireApp/homewire/internal/database"
	"github.com/HomewireApp/homewire/internal/logger"
	"github.com/HomewireApp/homewire/internal/utils"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/pquerna/otp"
	"github.com/ztrue/tracerr"
)

const BufferSize = 8

var ErrWireNotJoined = errors.New("not joined to the wire")
var ErrWireNotConnected = errors.New("not connected to the wire")

type JoinStatus string
type ConnectionStatus string

type OnJoinStatusChangedHandler func(new JoinStatus, old JoinStatus)
type OnConnectionStatusChangedHandler func(new ConnectionStatus, old ConnectionStatus)

const (
	NotJoined  JoinStatus = "not-joined"
	Joining    JoinStatus = "joining"
	Joined     JoinStatus = "joined"
	JoinFailed JoinStatus = "join-failed"
)

const (
	NotConnected     ConnectionStatus = "not-connected"
	Connecting       ConnectionStatus = "connecting"
	Connected        ConnectionStatus = "connected"
	ConnectionFailed ConnectionStatus = "connection-failed"
)

type Wire struct {
	ctx *ctx.Context

	Id               string
	Name             string
	JoinStatus       JoinStatus
	ConnectionStatus ConnectionStatus

	privKey crypto.PrivKey
	otpKey  *otp.Key

	pubsub  *pubsub.PubSub
	topic   *pubsub.Topic
	subs    *pubsub.Subscription
	mutConn *sync.Mutex

	joinStatusChangedHandlers       []OnJoinStatusChangedHandler
	connectionStatusChangedHandlers []OnConnectionStatusChangedHandler
	mutHooks                        *sync.Mutex
}

func (w *Wire) ensureConnected() {
	ticker := time.NewTicker(3 * time.Second)
	for {
		<-ticker.C

		w.mutConn.Lock()
		defer w.mutConn.Unlock()

		w.setConnectionStatus(Connecting)

		if w.pubsub != nil {
			ticker.Stop()
			return
		}

		pubsub, err := pubsub.NewGossipSub(w.ctx.Context, w.ctx.Host)
		if err != nil {
			logger.Warn("[wire:ensureConnected] [%v] Failed to set up pubsub, will retry later %v", w.Id, err)
			w.setConnectionStatus(ConnectionFailed)
			continue
		}

		topic, err := pubsub.Join(w.getTopicName())
		if err != nil {
			logger.Warn("[wire:ensureConnected] [%v] Failed to set up pubsub topic, will retry later %v", w.Id, err)
			w.setConnectionStatus(ConnectionFailed)
			continue
		}

		subs, err := topic.Subscribe()
		if err != nil {
			topic.Close()
			logger.Warn("[wire:ensureConnected] [%v] Failed to subscribe to topic, will retry later %v", w.Id, err)
			w.setConnectionStatus(ConnectionFailed)
			continue
		}

		w.pubsub = pubsub
		w.topic = topic
		w.subs = subs
		w.setConnectionStatus(Connected)
		ticker.Stop()
		logger.Debug("[wire:ensureConnected] [%v] Successfully connected to wire", w.Id)
	}
}

func (w *Wire) setJoinStatus(new JoinStatus) {
	if w.JoinStatus != new {
		old := w.JoinStatus
		w.JoinStatus = new
		go w.triggerJoinStatusChangedHandlers(new, old)
	}
}

func (w *Wire) setConnectionStatus(new ConnectionStatus) {
	if w.ConnectionStatus != new {
		old := w.ConnectionStatus
		w.ConnectionStatus = new
		go w.triggerConnectionStatusChangedHandlers(new, old)
	}
}

func (w *Wire) triggerJoinStatusChangedHandlers(new JoinStatus, old JoinStatus) {
	w.mutHooks.Lock()
	defer w.mutHooks.Unlock()
	for _, handler := range w.joinStatusChangedHandlers {
		go handler(new, old)
	}
}

func (w *Wire) triggerConnectionStatusChangedHandlers(new ConnectionStatus, old ConnectionStatus) {
	w.mutHooks.Lock()
	defer w.mutHooks.Unlock()
	for _, handler := range w.connectionStatusChangedHandlers {
		go handler(new, old)
	}
}

func (w *Wire) getTopicName() string {
	return "homewire:wires:" + w.Id
}

func CreateExistingKnownWire(ctx *ctx.Context, wireModel *database.WireModel) (*Wire, error) {
	key, err := utils.GenerateOtpFromExistingSecret(wireModel.Name, wireModel.OtpSecret)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	privKey, err := utils.UnmarshalPrivateKey(wireModel.PrivateKey)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	w := &Wire{
		Id:               wireModel.Id,
		Name:             wireModel.Name,
		JoinStatus:       Joined,
		ConnectionStatus: NotConnected,
		privKey:          privKey,
		otpKey:           key,
		ctx:              ctx,
		mutConn:          &sync.Mutex{},
		mutHooks:         &sync.Mutex{},
	}

	go w.ensureConnected()

	return w, nil
}

func CreateNewKnownWire(ctx *ctx.Context, name string) (*Wire, error) {
	privKey, err := utils.GeneratePrivateKey()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	prvBytes, err := utils.MarshalPrivateKey(privKey)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	key, secret, err := utils.GenerateNewOtp(name)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	wireModel := &database.WireModel{
		Id:         utils.NewUUID(),
		Name:       name,
		PrivateKey: prvBytes,
		OtpSecret:  secret,
	}

	if err := ctx.DB.Insert(wireModel); err != nil {
		return nil, err
	}

	w := &Wire{
		ctx:              ctx,
		Id:               wireModel.Id,
		Name:             name,
		JoinStatus:       Joining,
		ConnectionStatus: NotConnected,
		privKey:          privKey,
		otpKey:           key,
		mutConn:          &sync.Mutex{},
		mutHooks:         &sync.Mutex{},
	}

	go w.ensureConnected()

	return w, nil
}

func CreateUnknownWire(ctx *ctx.Context, id string, name string) *Wire {
	return &Wire{
		ctx:              ctx,
		Id:               id,
		Name:             name,
		JoinStatus:       NotJoined,
		ConnectionStatus: NotConnected,
		mutConn:          &sync.Mutex{},
		mutHooks:         &sync.Mutex{},
	}
}

func (w *Wire) SendPlainMessage(msg Message) error {
	if w.JoinStatus != Joined {
		return ErrWireNotJoined
	}

	if w.ConnectionStatus != Connected {
		return ErrWireNotConnected
	}

	bytes, err := msg.Marshal()
	if err != nil {
		return tracerr.Wrap(err)
	}

	return w.topic.Publish(w.ctx.Context, bytes)
}

func (w *Wire) Join(otp string) error {
	return errors.New("not yet implemented")
}

func (w *Wire) OnJoinStatusChanged(handler OnJoinStatusChangedHandler) {
	w.mutHooks.Lock()
	defer w.mutHooks.Unlock()
	w.joinStatusChangedHandlers = append(w.joinStatusChangedHandlers, handler)
}

func (w *Wire) OnConnectionStatusChanged(handler OnConnectionStatusChangedHandler) {
	w.mutHooks.Lock()
	defer w.mutHooks.Unlock()
	w.connectionStatusChangedHandlers = append(w.connectionStatusChangedHandlers, handler)
}

func (w *Wire) Destroy() {
	if w.subs != nil {
		w.subs.Cancel()
		w.subs = nil
	}

	if w.topic != nil {
		w.topic.Close()
		w.topic = nil
	}

	w.pubsub = nil
}
