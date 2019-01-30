package common

type Sender interface {
	Send(message []byte) error
}
