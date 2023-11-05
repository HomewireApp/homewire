package proto

import (
	"github.com/HomewireApp/homewire/internal/proto/pb"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/ztrue/tracerr"
	protobuf "google.golang.org/protobuf/proto"
)

func MarshalPlain(msg protobuf.Message) ([]byte, error) {
	return protobuf.Marshal(msg)
}

func UnmarshalPlainEnvelope(bytes []byte) (*pb.Envelope, error) {
	var env pb.Envelope

	if err := protobuf.Unmarshal(bytes, &env); err != nil {
		return nil, err
	}

	return &env, nil
}

func MarshalAndSign(msg protobuf.Message, privKey crypto.PrivKey) ([]byte, error) {
	plainBytes, err := MarshalPlain(msg)

	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	return privKey.Sign(plainBytes)
}
