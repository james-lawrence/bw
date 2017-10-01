package agent

import "io"

// Client - client facade interface.
type Client interface {
	Upload(srcbytes uint64, src io.Reader) (Archive, error)
	Deploy(info Archive) error
	Connect() (ConnectInfo, error)
	Info() (Status, error)
	Watch(out chan<- Message) error
	Dispatch(messages ...Message) error
	Close() error
}
