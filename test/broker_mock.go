package test

import (
	bus "github.com/moleculer-go/goemitter"
	"github.com/nnqq/moleculer"
	log "github.com/sirupsen/logrus"
)

var logger = log.WithField("unit test", "<root>")

func Logger(name string, value string) *log.Entry {
	return logger.WithField(name, value)
}

func DelegatesWithId(nodeID string) *moleculer.BrokerDelegates {
	return DelegatesWithIdAndConfig(nodeID, moleculer.Config{})
}

func DelegatesWithIdAndConfig(nodeID string, config moleculer.Config) *moleculer.BrokerDelegates {
	localBus := bus.Construct()
	localNode := NodeMock{ID: nodeID}
	broker := &moleculer.BrokerDelegates{
		LocalNode: func() moleculer.Node {
			return &localNode
		},
		Logger: Logger,
		Bus: func() *bus.Emitter {
			return localBus
		},
		Config: config,
	}
	return broker
}

// func ContextAndDelegated(nodeID string, config moleculer.Config) (moleculer.BrokerContext, *moleculer.BrokerDelegates) {
// 	dl := DelegatesWithIdAndConfig(nodeID, config)
// 	ctx := context.BrokerContext(dl)
// 	return ctx, dl
// }
