package main

import (
	"context"
	"flag"
	"testing"

	"github.com/google/subcommands"

	cfnSubcommands "github.com/horietakehiro/cfn-global-views/internal/subcommands"
)

func init() {

	testing.Init()

	subcommands.Register(subcommands.HelpCommand(), "help")
	subcommands.Register(subcommands.FlagsCommand(), "help")
	subcommands.Register(subcommands.CommandsCommand(), "help")

	subcommands.Register(&cfnSubcommands.ParametersCmd{}, "")
	subcommands.Register(&cfnSubcommands.ResourcesCmd{}, "")
	subcommands.Register(&cfnSubcommands.OutputsCmd{}, "")
	subcommands.Register(&cfnSubcommands.AllCmd{}, "")

	flag.Parse()

}

func main() {
	ctx := context.Background()
	subcommands.Execute(ctx)
}
