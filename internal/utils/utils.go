package utils

import (
	"crypto/rand"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/ztrue/tracerr"
)

const DefaultOtpTtl uint = 60

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

func getOtpOptions(name string, secret []byte) totp.GenerateOpts {
	return totp.GenerateOpts{
		Issuer:      "homewire.app",
		AccountName: name,
		Secret:      secret,
		Algorithm:   otp.AlgorithmSHA512,
		Period:      DefaultOtpTtl,
	}
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
	return totp.Generate(getOtpOptions(name, secret))
}

func GenerateOtpCode(secret []byte) (string, uint, error) {
	key, err := GenerateOtpFromExistingSecret("homewire", secret)
	if err != nil {
		return "", 0, err
	}

	code, err := totp.GenerateCodeCustom(key.Secret(), time.Now(), totp.ValidateOpts{
		Period:    DefaultOtpTtl,
		Algorithm: otp.AlgorithmSHA512,
	})

	if err != nil {
		return "", 0, err
	}

	return code, DefaultOtpTtl, nil
}

type RetryAttemptsExhaustedError struct {
	Attempts int
	Errors   []error
}

func (r *RetryAttemptsExhaustedError) Error() string {
	return "retry attempts have exhausted"
}

func Retry[T any](fn func() (T, error), maxAttempts int, delay time.Duration) (T, error) {
	result, err := fn()

	if err == nil {
		return result, err
	}

	timer := time.NewTimer(delay)
	errors := make([]error, 0, maxAttempts)

	currentAttempt := 1
	for {
		<-timer.C
		result, err := fn()

		if err == nil {
			timer.Stop()
			return result, nil
		}

		errors = append(errors, err)

		if currentAttempt >= maxAttempts {
			return *new(T), tracerr.Wrap(&RetryAttemptsExhaustedError{Attempts: currentAttempt, Errors: errors})
		}

		timer.Reset(delay)
	}
}
