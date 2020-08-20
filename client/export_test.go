package client

import "github.com/hyperledger-labs/perun-node"

func NewClientForTest(pClient pClient, msgBus perun.WireBus, msgBusRegistry perun.Registerer) *client {
	return &client{
		pClient:        pClient,
		msgBus:         msgBus,
		msgBusRegistry: msgBusRegistry,
	}
}
