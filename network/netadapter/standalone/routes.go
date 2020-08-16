package standalone

import (
	"time"

	"github.com/kaspanet/kaspad/network/netadapter"

	"github.com/pkg/errors"

	"github.com/kaspanet/kaspad/network/domainmessage"
	"github.com/kaspanet/kaspad/network/netadapter/router"
)

// Routes holds the incoming and outgoing routes of a connection created by MinimalNetAdapter
type Routes struct {
	netConnection                *netadapter.NetConnection
	IncomingRoute, OutgoingRoute *router.Route
	handshakeRoute               *router.Route
	pingRoute                    *router.Route
}

// WaitForMessageOfType waits for a message of requested type up to `timeout`, skipping all messages of any other type
// received while waiting
func (r *Routes) WaitForMessageOfType(command domainmessage.MessageCommand, timeout time.Duration) (domainmessage.Message, error) {
	timeoutTime := time.Now().Add(timeout)
	for {
		message, err := r.IncomingRoute.DequeueWithTimeout(timeoutTime.Sub(time.Now()))
		if err != nil {
			return nil, errors.Wrapf(err, "error waiting for message of type %s", command)
		}
		if message.Command() == command {
			return message, nil
		}
	}
}

// WaitForDisconnect waits for a disconnect up to `timeout`, skipping all messages received while waiting
func (r *Routes) WaitForDisconnect(timeout time.Duration) error {
	timeoutTime := time.Now().Add(timeout)
	for {
		_, err := r.IncomingRoute.DequeueWithTimeout(timeoutTime.Sub(time.Now()))
		if errors.Is(err, router.ErrRouteClosed) {
			return nil
		}
		if err != nil {
			return errors.Wrap(err, "error waiting for disconnect")
		}
	}
}

// Disconnect closes the connection behind the routes, thus closing all routes
func (r *Routes) Disconnect() {
	r.netConnection.Disconnect()
}
