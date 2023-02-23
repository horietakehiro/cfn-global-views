package subcommands

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/gocarina/gocsv"
	"github.com/google/subcommands"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/exp/slog"

	"github.com/horietakehiro/cfn-global-views/config"
)

type CfnOutput struct {
	Name        string
	Description string
	Value       string
	ExportName  string
}

type CfnOutputsView struct {
	AccountId   string
	AccountName string
	Region      string
	StackName   string
	Outputs     []CfnOutput
	Error       error
}

type CfnOutputsCsvView struct {
	AccountId         string
	AccountName       string
	Region            string
	StackName         string
	OutputName        string
	OutputValue       string
	OutputDescription string
	OutputExportName  string
	Error             string
}

type OutputsCmd struct {
	subcommands.Command
	configFilePath string
	outFilePath    string
	format         string
	verbose        bool
	logger         *slog.Logger
	config         *config.CfnGlobalViewsConfig
}

func (*OutputsCmd) Name() string {
	return "outputs"
}
func (*OutputsCmd) Synopsis() string {
	return "list cfn outputs"
}
func (*OutputsCmd) Usage() string {
	return "outputs -c path/to/config.yaml"
}
func (c *OutputsCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.configFilePath, "c", "", "path to config yaml file")
	f.StringVar(&c.outFilePath, "o", "", "path to output file path. if you dont't set, just stdout result")
	f.StringVar(&c.format, "f", "csv", "output data format [csv, json] (default is csv)")
	f.BoolVar(&c.verbose, "v", false, "if set, stdout debug log messages")
}

func (c *OutputsCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	var err error

	if c.configFilePath == "" {
		fmt.Println("arg '-c path/to/config.yaml' is required")
		return subcommands.ExitFailure
	}
	if c.format != "csv" && c.format != "json" {
		fmt.Println("allowed values for arg '-f' are [csv, json]")
	}

	if c.verbose {
		c.logger = slog.New(slog.NewJSONHandler(os.Stdout))
	} else {
		c.logger = slog.New(slog.NewJSONHandler(io.Discard))
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
	if c.format == "json" {
		err = c.DumpJson(globalViews)
		if err != nil {
			return subcommands.ExitFailure
		}
	}

	return subcommands.ExitSuccess
}

func (c *OutputsCmd) DumpCsv(views []*CfnOutputsView) error {
	csvViews := []CfnOutputsCsvView{}

	for _, view := range views {
		var errorString string
		if view.Error == nil {
			errorString = ""
		} else {
			errorString = view.Error.Error()
		}
		if len(view.Outputs) == 0 {
			csvViews = append(csvViews, CfnOutputsCsvView{
				AccountId:         view.AccountId,
				AccountName:       view.AccountName,
				Region:            view.Region,
				StackName:         view.StackName,
				OutputName:        "",
				OutputValue:       "",
				OutputDescription: "",
				OutputExportName:  "",
				Error:             errorString,
			})
		}
		for _, output := range view.Outputs {
			csvViews = append(csvViews, CfnOutputsCsvView{
				AccountId:         view.AccountId,
				AccountName:       view.AccountName,
				Region:            view.Region,
				StackName:         view.StackName,
				OutputName:        output.Name,
				OutputValue:       output.Value,
				OutputDescription: output.Description,
				OutputExportName:  output.ExportName,
				Error:             errorString,
			})
		}
	}

	var writer *os.File
	var err error
	if c.outFilePath != "" {
		writer, err = os.Create(c.outFilePath)
		if err != nil {
			return err
		}
		defer writer.Close()
	} else {
		writer = os.Stdout
	}

	err = gocsv.Marshal(csvViews, writer)
	if err != nil {
		return err
	}

	return nil

}

func (c *OutputsCmd) DumpJson(views []*CfnOutputsView) error {

	jsonViews, err := json.Marshal(views)
	if err != nil {
		return err
	}

	var writer *os.File
	if c.outFilePath != "" {
		writer, err = os.Create(c.outFilePath)
		if err != nil {
			return err
		}
		defer writer.Close()
	} else {
		writer = os.Stdout
	}

	_, err = writer.Write(jsonViews)
	if err != nil {
		return err
	}

	return nil

}

func (c *OutputsCmd) calcTotalViews() int {
	total := 0
	for _, accountConfig := range c.config.AccountConfigs {
		total += len(accountConfig.Filters.Regions)
	}
	return total
}

func (c *OutputsCmd) GetGlobalViews() []*CfnOutputsView {
	globalViews := []*CfnOutputsView{}

	totalViews := c.calcTotalViews()
	progress := 0
	channel := make(chan []*CfnOutputsView, totalViews)

	var bar *progressbar.ProgressBar
	if !c.verbose {
		bar = progressbar.Default(int64(totalViews))
	}
	for ai := range c.config.AccountConfigs {
		for ri := range c.config.AccountConfigs[ai].Filters.Regions {

			go func(ch chan []*CfnOutputsView, ai, ri int) {
				views := []*CfnOutputsView{}
				c.logger.Info(
					"get cfn views", "accountId", c.config.AccountConfigs[ai].Id, "region", c.config.AccountConfigs[ai].Filters.Regions[ri],
				)
				// setup cloudformation client
				var sess *session.Session
				if c.config.AccountConfigs[ai].Credential.Type == "CLI" {
					sess = session.Must(session.NewSessionWithOptions(session.Options{
						Profile: c.config.AccountConfigs[ai].Credential.ProfileName,
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
						views = append(views, &CfnOutputsView{
							AccountId:   c.config.AccountConfigs[ai].Id,
							AccountName: c.config.AccountConfigs[ai].Name,
							Region:      c.config.AccountConfigs[ai].Filters.Regions[ri],
							Error:       err,
						})

						break
					}
					for _, stack := range describeStacksOutpus.Stacks {

						matched, _ := regexp.MatchString(c.config.AccountConfigs[ai].Filters.StackNameRegex, *stack.StackName)
						if matched && c.hasAllTags(stack.Tags, c.config.AccountConfigs[ai].Filters.StackTags) {
							c.logger.Info(fmt.Sprintf("matched cfn stack: %s", *stack.StackName), "accountId", c.config.AccountConfigs[ai].Id, "region", c.config.AccountConfigs[ai].Filters.Regions[ri])
							matchedStacks = append(matchedStacks, *stack)
						}
					}

					if describeStacksOutpus.NextToken == nil {
						break
					}
				}

				// describe matched stacks' output definitions
				for _, matchedStack := range matchedStacks {
					var outputs []CfnOutput
					for _, output := range matchedStack.Outputs {
						description := ""
						exportName := ""
						if d := output.Description; d != nil {
							description = *d
						}
						if d := output.ExportName; d != nil {
							exportName = *d
						}
						outputs = append(outputs, CfnOutput{
							Name:        *output.OutputKey,
							Value:       *output.OutputValue,
							Description: description,
							ExportName:  exportName,
						})
					}
					views = append(views, &CfnOutputsView{
						AccountId:   c.config.AccountConfigs[ai].Id,
						AccountName: c.config.AccountConfigs[ai].Name,
						Region:      c.config.AccountConfigs[ai].Filters.Regions[ri],
						StackName:   *matchedStack.StackName,
						Outputs:     outputs,
						Error:       nil,
					})
				}

				ch <- views

				if !c.verbose {
					bar.Add(1)
				}
				progress += 1
				if progress == totalViews {
					close(ch)
				}
			}(channel, ai, ri)
		}
	}

	for views := range channel {
		globalViews = append(globalViews, views...)
	}

	sort.Slice(globalViews, func(i, j int) bool { return globalViews[i].AccountId < globalViews[j].AccountId })
	sort.Slice(globalViews, func(i, j int) bool { return globalViews[i].Region < globalViews[j].Region })
	sort.Slice(globalViews, func(i, j int) bool { return globalViews[i].StackName < globalViews[j].StackName })

	return globalViews

}

func (c *OutputsCmd) hasAllTags(stackTags []*cloudformation.Tag, filterTags []config.Tag) bool {
	hasAllTags := []bool{}

	if len(filterTags) == 0 {
		return true
	}
	for _, filterTag := range filterTags {
		for _, stackTag := range stackTags {
			if filterTag.Key == *stackTag.Key && filterTag.Value == *stackTag.Value {
				hasAllTags = append(hasAllTags, true)
			}
		}
	}
	if len(hasAllTags) == len(filterTags) {
		return true
	} else {
		return false
	}
}
