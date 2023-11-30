package messages

import (
	"github.com/HomewireApp/homewire/internal/proto/pb"
	"github.com/HomewireApp/homewire/peer"
	"github.com/HomewireApp/homewire/wire"
	"github.com/ztrue/tracerr"
)

func Introduction(s *peer.Self, peerWires []*peer.PeerWire) (*pb.Envelope, error) {
	pubKeyBytes, err := s.GetPublicKey().Raw()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	wires := make([]*pb.BasicWireInfo, 0, len(peerWires))
	for _, w := range peerWires {
		wires = append(wires, &pb.BasicWireInfo{
			Id:   w.Id,
			Name: w.Name,
		})
	}

	return &pb.Envelope{
		Payload: &pb.Envelope_Introduction{
			Introduction: &pb.Introduction{
				Id:            s.Id,
				DisplayedName: s.DisplayedName,
				PublicKey:     pubKeyBytes,
				Wires:         wires,
			},
		},
	}, nil
}

func WireAnnouncement(w *wire.Wire) *pb.Envelope {
	return &pb.Envelope{
		Payload: &pb.Envelope_WireAnnouncement{
			WireAnnouncement: &pb.BasicWireInfo{
				Id:   w.Id,
				Name: w.Name,
			},
		},
	}
}
