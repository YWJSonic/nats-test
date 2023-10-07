package mynats

import (
	"github.com/nats-io/nats.go"
)

func New(urls string) (*nats.Conn, error) {
	natsConn, err := nats.Connect(urls)
	if err != nil {
		return nil, err
	}
	return natsConn, nil
}
