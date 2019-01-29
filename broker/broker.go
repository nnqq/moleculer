package broker

import (
	"errors"
	"fmt"
	"strings"

	. "github.com/moleculer-go/goemitter"
	. "github.com/moleculer-go/moleculer/cacher"
	. "github.com/moleculer-go/moleculer/common"
	. "github.com/moleculer-go/moleculer/context"
	. "github.com/moleculer-go/moleculer/middleware"
	. "github.com/moleculer-go/moleculer/registry"
	. "github.com/moleculer-go/moleculer/serializer"
	. "github.com/moleculer-go/moleculer/service"
	. "github.com/moleculer-go/moleculer/strategy"
	. "github.com/moleculer-go/moleculer/transit"
	log "github.com/sirupsen/logrus"
)

type BrokerConfig struct {
	LogLevel       string
	LogFormat      string
	DiscoverNodeID func() string
}

var defaultConfig = BrokerConfig{
	LogLevel:       "INFO",
	LogFormat:      "TEXT",
	DiscoverNodeID: DiscoverNodeID,
}

// DiscoverNodeID - should return the node id for this machine
func DiscoverNodeID() string {
	// TODO: Check moleculer JS algo for this..
	return "fixed-node-value"
}

func mergeConfigs(baseConfig BrokerConfig, userConfig []*BrokerConfig) BrokerConfig {

	if len(userConfig) > 0 {

		config := userConfig[0]
		if config.LogLevel != "" {
			baseConfig.LogLevel = config.LogLevel
		}
		if config.LogFormat != "" {
			baseConfig.LogFormat = config.LogFormat
		}
		if config.DiscoverNodeID != nil {
			baseConfig.DiscoverNodeID = config.DiscoverNodeID
		}
	}

	return baseConfig
}

type ServiceBroker struct {
	context *Context

	namespace string
	nodeID    string

	logger *log.Entry

	localBus *Emitter

	registry *ServiceRegistry

	middlewares *MiddlewareHandler

	cacher *Cacher

	serializer *Serializer

	transit *Transit

	services []*Service

	started bool

	rootContext Context

	config BrokerConfig

	strategy Strategy

	info *BrokerInfo

	localNode Node

	registryMessageHandler RegistryMessageHandlerFunction
}

// GetLocalBus : return the service broker local bus (Event Emitter)
func (broker *ServiceBroker) GetLocalBus() *Emitter {
	return broker.localBus
}

// startService start a service within the provided broker
func startService(broker *ServiceBroker, service *Service) {

	broker.middlewares.CallHandlers("serviceStarting", service)

	waitForDependencies(service)

	service.Start()

	notifyServiceStarted(service)

	broker.registry.AddLocalService(service)

	broker.middlewares.CallHandlers("serviceStarted", service)
}

// wait for all service dependencies to load
func waitForDependencies(service *Service) {
	//TODO
}

// notify a service when it is started
func notifyServiceStarted(service *Service) {
	// if service.Started != nil {
	// 	service.Started()
	// }
	//TODO: notify mixins also.. that might have the started method
}

func (broker *ServiceBroker) broadcastLocal(eventName string, params ...interface{}) {
	//TODO
}

func (broker *ServiceBroker) createBrokerLogger() *log.Entry {

	if strings.ToUpper(broker.config.LogFormat) == "JSON" {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		log.SetFormatter(&log.TextFormatter{})
	}

	if strings.ToUpper(broker.config.LogLevel) == "WARN" {
		log.SetLevel(log.WarnLevel)
	} else if strings.ToUpper(broker.config.LogLevel) == "DEBUG" {
		log.SetLevel(log.DebugLevel)
	} else if strings.ToUpper(broker.config.LogLevel) == "TRACE" {
		log.SetLevel(log.TraceLevel)
	} else if strings.ToUpper(broker.config.LogLevel) == "ERROR" {
		log.SetLevel(log.ErrorLevel)
	} else if strings.ToUpper(broker.config.LogLevel) == "FATAL" {
		log.SetLevel(log.FatalLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	nodeID := broker.config.DiscoverNodeID()
	brokerLogger := log.WithFields(log.Fields{
		"broker": nodeID,
	})

	return brokerLogger
}

// AddService : for each service schema it will validate and create
// a service instance in the broker.
func (broker *ServiceBroker) AddService(schemas ...ServiceSchema) {
	for _, schema := range schemas {
		service := CreateService(schema)
		broker.services = append(broker.services, service)
	}
}

func (broker *ServiceBroker) Start() {
	broker.logger.Info("Broker -> starting ...")

	broker.started = false

	broker.middlewares.CallHandlers("starting", broker)

	<-(*broker.transit).Connect()

	broker.logger.Debug("Broker -> transit connected !")

	for _, service := range broker.services {
		startService(broker, service)
	}

	broker.logger.Debug("Broker -> services started !")

	broker.registry.Start()

	broker.logger.Debug("Broker -> registry started !")

	broker.started = true
	broker.broadcastLocal("$broker.started")

	//*broker.transit).Ready()

	broker.logger.Debug("Broker -> transit is ready !")

	broker.middlewares.CallHandlers("started", broker)

	broker.logger.Info("Broker -> started !")
}

type contextKey int

const (
	ContextBroker contextKey = iota
	ContextAction
)

type contextAction struct {
	actionName string
	params     interface{}
}

type contextBroker struct {
	//TODO add relevante broker info here
}

func (broker *ServiceBroker) emitWithContext(context *Context, groups ...string) {
}

func (broker *ServiceBroker) broadcastWithContext(context *Context, groups ...string) {
}

// callWithContext :  invoke a service action and return a channel which will eventualy deliver the results ;)
func (broker *ServiceBroker) callWithContext(context *Context, opts ...OptionsFunc) chan interface{} {
	actionName := (*context).GetActionName()
	params := (*context).GetParams()
	broker.logger.Debug("Broker - callWithContext() actionName: ", actionName, " params: ", params, " opts: ", opts)

	endpoint := broker.registry.NextActionEndpoint(actionName, broker.strategy, WrapOptions(opts))
	if endpoint == nil {
		msg := fmt.Sprintf("Broker - endpoint not found for actionName: %s", actionName)
		broker.logger.Error(msg)
		panic(errors.New(msg))
	}

	broker.logger.Debug("Broker - calling actionName: ", actionName)
	node := CreateNode(endpoint.GetNodeID())
	(*context).SetNode(&node)
	return endpoint.InvokeAction(context)
}

// Call :  invoke a service action and return a channel which will eventualy deliver the results ;)
func (broker *ServiceBroker) Call(actionName string, params interface{}, opts ...OptionsFunc) chan interface{} {
	broker.logger.Info("Broker - Call() actionName: ", actionName, " params: ", params, " opts: ", opts)

	actionContext := broker.rootContext.NewActionContext(actionName, params, WrapOptions(opts))
	broker.logger.Info("Broker - Call() actionContext created!  ")

	return actionContext.InvokeAction(WrapOptions(opts))
}

func (broker *ServiceBroker) Emit(event string, params interface{}) {
	broker.logger.Debug("Broker - emit !")
}

func (broker *ServiceBroker) GetInfo() *BrokerInfo {
	return broker.info
}

func (broker *ServiceBroker) IsStarted() bool {
	return broker.started
}

func (broker *ServiceBroker) GetLogger(name string, value string) *log.Entry {
	return broker.logger.WithField(name, value)
}

func (broker *ServiceBroker) GetLocalNode() *Node {
	return &broker.localNode
}

func (broker *ServiceBroker) init() {
	var serializer Serializer = CreateJSONSerializer(func() *BrokerInfo {
		return broker.info
	})
	broker.logger = broker.createBrokerLogger()
	broker.strategy = RoundRobinStrategy{}
	broker.setupLocalBus()
	broker.localNode = CreateNode(broker.config.DiscoverNodeID())
	broker.registryMessageHandler = func(command string, message *TransitMessage) {
		broker.registry.HandleTransitMessage(command, message)
	}
	broker.info = &BrokerInfo{
		broker.GetLocalNode,
		broker.GetLogger,
		broker.GetLocalBus,
		broker.GetTransit,
		broker.IsStarted,
		func() *Serializer {
			return &serializer
		},
		broker.registryMessageHandler,
		func() (ActionDelegateFunc, EventDelegateFunc, EventDelegateFunc) {
			return broker.callWithContext, broker.emitWithContext, broker.broadcastWithContext
		},
	}

	broker.registry = CreateRegistry(broker.GetInfo())

	broker.transit = CreateTransit(broker.GetInfo())
	broker.rootContext = CreateBrokerContext(
		broker.callWithContext,
		broker.emitWithContext,
		broker.broadcastWithContext,
		broker.GetLogger,
		&broker.localNode)
}

func (broker *ServiceBroker) GetTransit() *Transit {
	return broker.transit
}

func (broker *ServiceBroker) setupLocalBus() {
	broker.localBus = CreateEmitter()

	broker.localBus.On("$registry.service.added", func(args ...interface{}) {
		//TODO check code from -> this.broker.servicesChanged(true)
	})
}

// FromConfig : returns a valid broker based on environment configuration
// this is usually called when creating a broker to starting the service(s)
func FromConfig(userConfig []*BrokerConfig) *ServiceBroker {

	config := mergeConfigs(defaultConfig, userConfig)
	broker := ServiceBroker{config: config}
	broker.init()

	broker.logger.Info("Broker - FromConfig() ")
	return &broker
}
