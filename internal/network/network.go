package network

import (
	"context"
	"io"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	msmux "github.com/multiformats/go-multistream"
	"github.com/ztrue/tracerr"
)

type ListenerPhase string

const (
	SetupTimeout  ListenerPhase = "setup-timeout"
	SetService    ListenerPhase = "set-service"
	ReserveMemory ListenerPhase = "reserve-memory"
	ReadBytes     ListenerPhase = "read-bytes"
)

func GetFirstConnectionToPeer(host host.Host, pi peer.ID) *network.Conn {
	conns := host.Network().ConnsToPeer(pi)
	if len(conns) < 1 {
		return nil
	}
	return &conns[0]
}

func SendMessageToPeer(conn *network.Conn, msg *PlainOutboundMessage) error {
	str, err := prepareMessageStream(conn, msg)
	if err != nil {
		return tracerr.Wrap(err)
	}

	bytes, err := msg.Marshal()
	if err != nil {
		str.Reset()
		return nil
	}

	defer str.Close()

	str.Write(bytes)
	return nil
}

func AttachListener(h host.Host, listener MessageReceiver) {
	h.SetStreamHandler(listener.GetProtocolID(), func(str network.Stream) {
		timeout := listener.GetReadTimeout()
		if timeout != nil {
			if err := str.SetDeadline(time.Now().Add(60 * time.Second)); err != nil {
				listener.HandleError(SetupTimeout, err)
				str.Reset()
				return
			}
		}

		if err := str.Scope().SetService(listener.GetServiceName()); err != nil {
			listener.HandleError(SetService, err)
			str.Reset()
			return
		}

		if err := str.Scope().ReserveMemory(listener.GetBufferSize(), network.ReservationPriorityAlways); err != nil {
			listener.HandleError(ReserveMemory, err)
			str.Reset()
			return
		}
		defer str.Scope().ReleaseMemory(listener.GetBufferSize())

		bytes, err := io.ReadAll(str)
		if err != nil {
			listener.HandleError(ReadBytes, err)
			str.Reset()
			return
		}

		listener.HandleMessage(bytes, str.Conn().RemotePeer())
	})
}

func prepareMessageStream(conn *network.Conn, msg OutboundMessage) (network.Stream, error) {
	str, err := (*conn).NewStream(context.Background())
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	timeout := msg.GetTimeout()
	if timeout != nil {
		if err := str.SetDeadline(time.Now().Add(*timeout)); err != nil {
			str.Reset()
			return nil, tracerr.Wrap(err)
		}
	}

	pid := msg.GetProtocolID()
	if err := str.SetProtocol(pid); err != nil {
		str.Reset()
		return nil, tracerr.Wrap(err)
	}

	if err := msmux.SelectProtoOrFail(pid, str); err != nil {
		str.Reset()
		return nil, tracerr.Wrap(err)
	}

	if err := str.Scope().SetService(msg.GetServiceName()); err != nil {
		str.Reset()
		return nil, tracerr.Wrap(err)
	}

	return str, nil
}
