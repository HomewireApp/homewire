package messages

import (
	"github.com/HomewireApp/homewire/internal/proto/pb"
	"github.com/HomewireApp/homewire/peer"
	"github.com/HomewireApp/homewire/wire"
	"github.com/ztrue/tracerr"
)

func Introduction(s *peer.Self) (*pb.Envelope, error) {
	pubKeyBytes, err := s.GetPublicKey().Raw()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	return &pb.Envelope{
		Payload: &pb.Envelope_Introduction{
			Introduction: &pb.Introduction{
				Id:            s.Id,
				DisplayedName: s.DisplayedName,
				PublicKey:     pubKeyBytes,
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
