package cli

import (
	"github.com/nnqq/moleculer"
	"github.com/nnqq/moleculer/broker"
	"github.com/nnqq/moleculer/cli/cmd"
	"github.com/spf13/cobra"
)

// Start parse the config from the cli args. creates a service broker and pass down to the startHandler.
func Start(config *moleculer.Config, startHandler func(*broker.ServiceBroker, *cobra.Command)) {
	cmd.Execute(cmd.RunOpts{Config: config, StartHandler: startHandler})
}
