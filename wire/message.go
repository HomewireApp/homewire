package wire

type Message interface {
	Marshal() ([]byte, error)
}
