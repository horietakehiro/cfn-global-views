package subcommands

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/google/subcommands"
	"github.com/horietakehiro/cfn-global-views/config"
	"golang.org/x/exp/slog"
)

type AllCmd struct {
	subcommands.Command
	configFilePath string
	outFilePath    string
	format         string
	verbose        bool
	logger         *slog.Logger
	config         *config.CfnGlobalViewsConfig
}

func (*AllCmd) Name() string {
	return "all"
}
func (*AllCmd) Synopsis() string {
	return "list cfn parameters, resources, outputs"
}
func (*AllCmd) Usage() string {
	return "all -c path/to/config.yaml -o outfile.xlsx"
}
func (c *AllCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.configFilePath, "c", "", "path to config yaml file")
	f.StringVar(&c.outFilePath, "o", "", "path to output file path. if you dont't set, just stdout result")
	f.StringVar(&c.format, "f", "excel", "output data format [excel] (default is excel)")
	f.BoolVar(&c.verbose, "v", false, "if set, stdout debug log messages")
}

func (c *AllCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	var err error
	result := subcommands.ExitFailure
	if c.configFilePath == "" {
		fmt.Println("arg '-c path/to/config.yaml' is required")
		return result
	}
	if c.format != "excel" {
		fmt.Println("allowed values for arg '-f' is [excel]")
		return result
	}
	if c.format == "excel" && c.outFilePath == "" {
		fmt.Println("if format is excel, must specify output file path arg '-o'")
		return result
	}

	if c.verbose {
		c.logger = slog.New(slog.NewJSONHandler(os.Stdout))
	} else {
		c.logger = slog.New(slog.NewJSONHandler(io.Discard))
	}

	c.config, err = config.GetConfig(c.configFilePath)
	if err != nil {
		fmt.Println(err.Error())
		return result
	}

	parametersCmd := ParametersCmd{
		configFilePath: c.configFilePath,
		outFilePath:    c.outFilePath,
		format:         c.format,
		verbose:        c.verbose,
		logger:         c.logger,
		config:         c.config,
	}
	resourcesCmd := ResourcesCmd{
		configFilePath: c.configFilePath,
		outFilePath:    c.outFilePath,
		format:         c.format,
		verbose:        c.verbose,
		logger:         c.logger,
		config:         c.config,
	}
	outputsCmd := OutputsCmd{
		configFilePath: c.configFilePath,
		outFilePath:    c.outFilePath,
		format:         c.format,
		verbose:        c.verbose,
		logger:         c.logger,
		config:         c.config,
	}

	if _, err := os.Stat(c.outFilePath); err == nil {
		os.Rename(c.outFilePath, c.outFilePath+".bak")
	}
	defer func() {
		if result == subcommands.ExitFailure {
			os.Rename(c.outFilePath+".bak", c.outFilePath)
		} else {
			os.Remove(c.outFilePath + ".bak")
		}
	}()

	result = parametersCmd.Execute(context.TODO(), f, nil)
	if result == subcommands.ExitFailure {
		return result
	}
	resourcesCmd.Execute(context.TODO(), f, nil)
	if result == subcommands.ExitFailure {
		return result
	}
	outputsCmd.Execute(context.TODO(), f, nil)
	if result == subcommands.ExitFailure {
		return result
	}
	// var parametersViews []*CfnParametersView
	// var resourcesViews []*CfnResourcesView
	// var outputsViews []*CfnOutputsView

	// parametersChannel := make(chan []*CfnParametersView)
	// resourcesChannel := make(chan []*CfnResourcesView)
	// outputsChannel := make(chan []*CfnOutputsView)
	// done := make(chan interface{})

	// 	go func(ch chan []*CfnParametersView) {
	// 		views := parametersCmd.GetGlobalViews()
	// 		ch <- views
	// 		// close(ch)
	// 	}(parametersChannel)
	// 	go func(ch chan []*CfnResourcesView) {
	// 		views := resourcesCmd.GetGlobalViews()
	// 		ch <- views
	// 		// close(ch)
	// 	}(resourcesChannel)
	// 	go func(ch chan []*CfnOutputsView) {
	// 		views := outputsCmd.GetGlobalViews()
	// 		ch <- views
	// 		// close(ch)
	// 	}(outputsChannel)

	// breakPoint:
	// 	for {
	// 		select {
	// 		case parametersViews = <-parametersChannel:
	// 			c.logger.Info(fmt.Sprintf("parameters %v", len(parametersViews)))
	// 			err := parametersCmd.DumpExcel(parametersViews)
	// 			if err != nil {
	// 				c.logger.Error(err.Error(), err)
	// 			}
	// 		case resourcesViews = <-resourcesChannel:
	// 			c.logger.Info(fmt.Sprintf("resources %v", len(resourcesViews)))
	// 			err := resourcesCmd.DumpExcel(resourcesViews)
	// 			if err != nil {
	// 				c.logger.Error(err.Error(), err)
	// 			}
	// 		case outputsViews = <-outputsChannel:
	// 			c.logger.Info(fmt.Sprintf("outputs %v", len(outputsViews)))
	// 			err := outputsCmd.DumpExcel(outputsViews)
	// 			if err != nil {
	// 				c.logger.Error(err.Error(), err)
	// 			}
	// 		case <-done:
	// 			break breakPoint

	// 		default:
	// 			if len(parametersViews) != 0 && len(resourcesViews) != 0 && len(outputsViews) != 0 {
	// 				close(done)
	// 			}
	// 		}
	// 	}

	return result

}
