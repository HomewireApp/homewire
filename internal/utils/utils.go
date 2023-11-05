package utils

import (
	"crypto/rand"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/ztrue/tracerr"
)

func NewUUID() string {
	return uuid.NewString()
}

func NewPetName() string {
	return petname.Generate(3, "-")
}

func GeneratePrivateKey() (crypto.PrivKey, error) {
	privKey, _, err := crypto.GenerateEd25519Key(rand.Reader)

	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	return privKey, nil
}

func GeneratePrivateKeyBytes() ([]byte, error) {
	privKey, err := GeneratePrivateKey()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	return MarshalPrivateKey(privKey)
}

func MarshalPrivateKey(privKey crypto.PrivKey) ([]byte, error) {
	return privKey.Raw()
}

func UnmarshalPrivateKey(bytes []byte) (crypto.PrivKey, error) {
	return crypto.UnmarshalEd25519PrivateKey(bytes)
}

func UnmarshalPublicKey(bytes []byte) (crypto.PubKey, error) {
	return crypto.UnmarshalEd25519PublicKey(bytes)
}

func GenerateNewOtp(name string) (*otp.Key, []byte, error) {
	secret := make([]byte, 64)
	_, err := rand.Reader.Read(secret)
	if err != nil {
		return nil, nil, tracerr.Wrap(err)
	}

	key, err := GenerateOtpFromExistingSecret(name, secret)
	if err != nil {
		return nil, nil, tracerr.Wrap(err)
	}

	return key, secret, nil
}

func GenerateOtpFromExistingSecret(name string, secret []byte) (*otp.Key, error) {
	return totp.Generate(totp.GenerateOpts{
		Issuer:      "homewire.app",
		AccountName: name,
		Secret:      secret,
		Algorithm:   otp.AlgorithmSHA512,
	})
}
