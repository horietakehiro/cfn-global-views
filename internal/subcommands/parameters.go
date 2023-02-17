package subcommands

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/gocarina/gocsv"
	"github.com/google/subcommands"
	"golang.org/x/exp/slog"

	"github.com/horietakehiro/cfn-global-views/config"
)

type CfnParameter struct {
	Name         string
	Type         string
	Description  string
	DefaultValue string
	ActualValue  string
}

type CfnParametersView struct {
	AccountId   string
	AccountName string
	Region      string
	StackName   string
	Parameters  []CfnParameter
	Error       error
}

type CfnParametersCsvView struct {
	AccountId             string
	AccountName           string
	Region                string
	StackName             string
	ParameterName         string
	ParameterType         string
	ParameterDescription  string
	ParameterDefaultValue string
	ParameterActualValue  string
	Error                 string
}

type ParametersCmd struct {
	subcommands.Command
	configFilePath string
	outFilePath    string
	format         string
	verbose        bool
	logger         *slog.Logger
	config         *config.CfnGlobalViewsConfig
}

func (*ParametersCmd) Name() string {
	return "parameters"
}
func (*ParametersCmd) Synopsis() string {
	return "list cfn parameters and write given file"
}
func (*ParametersCmd) Usage() string {
	return "parameters -c path/to/config.yaml"
}
func (c *ParametersCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.configFilePath, "c", "", "path to config yaml file")
	f.StringVar(&c.outFilePath, "o", "", "path to output file path")
	f.StringVar(&c.format, "f", "csv", "output file format (default is csv)")
	f.BoolVar(&c.verbose, "v", false, "if set, stdout debug log messages")
}

func (c *ParametersCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	var err error

	if c.configFilePath == "" {
		fmt.Println("arg '-c path/to/config.yaml' is required")
		return subcommands.ExitFailure
	}

	if c.verbose {
		c.logger = slog.New(slog.NewJSONHandler(os.Stdout))
	} else {
		c.logger = slog.New(slog.NewJSONHandler(os.Stdout))
	}

	c.config, err = config.GetConfig(c.configFilePath)
	if err != nil {
		fmt.Println(err.Error())
		return subcommands.ExitFailure
	}

	globalViews := c.GetGlobalViews()

	if c.format == "csv" {
		err = c.DumpCsv(globalViews)
		if err != nil {
			return subcommands.ExitFailure
		}
	}

	return subcommands.ExitSuccess
}

func (c *ParametersCmd) DumpCsv(views []*CfnParametersView) error {
	csvViews := []CfnParametersCsvView{}

	for _, view := range views {
		var errorString string
		if view.Error == nil {
			errorString = ""
		} else {
			errorString = view.Error.Error()
		}
		for _, parameter := range view.Parameters {
			csvViews = append(csvViews, CfnParametersCsvView{
				AccountId:             view.AccountId,
				AccountName:           view.AccountName,
				Region:                view.Region,
				StackName:             view.StackName,
				ParameterName:         parameter.Name,
				ParameterType:         parameter.Type,
				ParameterDescription:  parameter.Description,
				ParameterDefaultValue: parameter.DefaultValue,
				ParameterActualValue:  parameter.ActualValue,
				Error:                 errorString,
			})
		}
	}

	f, err := os.Create(c.outFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	err = gocsv.Marshal(csvViews, f)
	if err != nil {
		return err
	}

	return nil

}

func (c *ParametersCmd) GetGlobalViews() []*CfnParametersView {
	views := []*CfnParametersView{}

	for ai := range c.config.AccountConfigs {
		for ri := range c.config.AccountConfigs[ai].Filters.Regions {

			c.logger.Info(
				fmt.Sprintf("get views from %s/%s", c.config.AccountConfigs[ai].Id, c.config.AccountConfigs[ai].Filters.Regions[ri]),
			)
			// setup cloudformation client
			var sess *session.Session
			if c.config.AccountConfigs[ai].Credential.Type == "CLI" {
				sess = session.Must(session.NewSessionWithOptions(session.Options{
					Profile: c.config.RootConfig.Credential.ProfileName,
					Config:  *aws.NewConfig().WithRegion(c.config.AccountConfigs[ai].Filters.Regions[ri]),
				}))
			} else {
				sess = session.Must(session.NewSessionWithOptions(session.Options{
					Config: *aws.NewConfig().WithRegion(c.config.AccountConfigs[ai].Filters.Regions[ri]),
				}))
			}

			cfn := cloudformation.New(sess)

			// describe all stacks at the account and region and filter them
			var matchedStacks []cloudformation.Stack
			for {
				describeStacksOutpus, err := cfn.DescribeStacks(&cloudformation.DescribeStacksInput{})
				if err != nil {
					views = append(views, &CfnParametersView{
						AccountId:   c.config.AccountConfigs[ai].Id,
						AccountName: c.config.AccountConfigs[ai].Name,
						Region:      c.config.AccountConfigs[ai].Filters.Regions[ri],
						Error:       err,
					})
					break
				}
				for _, stack := range describeStacksOutpus.Stacks {

					if strings.HasPrefix(*stack.StackName, c.config.AccountConfigs[ai].Filters.StackNamePrefix) && c.hasTag(stack.Tags, c.config.AccountConfigs[ai].Filters.StackTags) {
						c.logger.Info(fmt.Sprintf("matched: %s", *stack.StackName))
						matchedStacks = append(matchedStacks, *stack)
					}
				}

				if describeStacksOutpus.NextToken == nil {
					break
				}
			}

			// describe matched stacks' parameters definitions
			for _, matchedStack := range matchedStacks {
				templateSummary, err := cfn.GetTemplateSummary(&cloudformation.GetTemplateSummaryInput{
					StackName: matchedStack.StackName,
				})
				if err != nil {
					views = append(views, &CfnParametersView{
						AccountId:   c.config.AccountConfigs[ai].Id,
						AccountName: c.config.AccountConfigs[ai].Name,
						Region:      c.config.AccountConfigs[ai].Filters.Regions[ri],
						StackName:   *matchedStack.StackName,
						Error:       err,
					})
					break
				}
				var parameters []CfnParameter
				for _, parameter := range templateSummary.Parameters {
					parameters = append(parameters, CfnParameter{
						Name:         *parameter.ParameterKey,
						Type:         *parameter.ParameterType,
						Description:  *parameter.Description,
						DefaultValue: *parameter.DefaultValue,
						ActualValue:  c.getActulaParameterValue(parameter, matchedStack.Parameters),
					})
				}
				views = append(views, &CfnParametersView{
					AccountId:   c.config.AccountConfigs[ai].Id,
					AccountName: c.config.AccountConfigs[ai].Name,
					Region:      c.config.AccountConfigs[ai].Filters.Regions[ri],
					StackName:   *matchedStack.StackName,
					Parameters:  parameters,
					Error:       nil,
				})
			}
		}
	}

	return views
}

func (c *ParametersCmd) getActulaParameterValue(parameterDeclaration *cloudformation.ParameterDeclaration, parameters []*cloudformation.Parameter) string {
	for _, parameter := range parameters {
		if *parameterDeclaration.ParameterKey == *parameter.ParameterKey {
			return *parameter.ParameterValue
		}
	}
	return ""
}

func (c *ParametersCmd) hasTag(stackTags []*cloudformation.Tag, filterTags []config.Tag) bool {
	if len(filterTags) == 0 {
		return true
	}
	for _, stackTag := range stackTags {
		for _, filterTag := range filterTags {
			if filterTag.Key == *stackTag.Key && filterTag.Value == *stackTag.Value {
				return true
			}
		}
	}
	return false
}
