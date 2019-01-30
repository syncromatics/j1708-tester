package common

type J1587Message struct {
	Mid  int
	Pid  int
	Data []byte
	Raw  []byte
}
