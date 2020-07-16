package connmanager

import (
	"github.com/kaspanet/kaspad/netadapter"
)

type connectionSet map[string]*netadapter.NetConnection

func (cs connectionSet) add(connection *netadapter.NetConnection) {
	cs[connection.String()] = connection
}

func (cs connectionSet) remove(connection *netadapter.NetConnection) {
	delete(cs, connection.String())
}

func (cs connectionSet) get(address string) (*netadapter.NetConnection, bool) {
	connection, ok := cs[address]
	return connection, ok
}

func convertToSet(connections []*netadapter.NetConnection) connectionSet {
	connSet := make(connectionSet, len(connections))

	for _, connection := range connections {
		connSet[connection.String()] = connection
	}

	return connSet
}
