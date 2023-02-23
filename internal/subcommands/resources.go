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
	"github.com/horietakehiro/cfn-global-views/config"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/exp/slog"
)

type CfnResource struct {
	PhysicalId  string
	LogicalId   string
	Type        string
	Description string
	Status      string
	DriftStatus string
}

type CfnResourcesView struct {
	AccountId   string
	AccountName string
	Region      string
	StackName   string
	Resources   []CfnResource
	Error       error
}

type CfnResourcesCsvView struct {
	AccountId           string
	AccountName         string
	Region              string
	StackName           string
	ResourcePhysicalId  string
	ResourceLogicalId   string
	ResourceType        string
	ResourceDescription string
	ResourceStatus      string
	ResourceDriftStatus string
	Error               string
}

type ResourcesCmd struct {
	subcommands.Command
	configFilePath string
	outFilePath    string
	format         string
	verbose        bool
	logger         *slog.Logger
	config         *config.CfnGlobalViewsConfig
}

func (*ResourcesCmd) Name() string {
	return "resources"
}
func (*ResourcesCmd) Synopsis() string {
	return "list cfn resources"
}
func (*ResourcesCmd) Usage() string {
	return "resources -c path/to/config.yaml"
}
func (c *ResourcesCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.configFilePath, "c", "", "path to config yaml file")
	f.StringVar(&c.outFilePath, "o", "", "path to output file path. if you dont't set, just stdout result")
	f.StringVar(&c.format, "f", "csv", "output data format [csv, json] (default is csv)")
	f.BoolVar(&c.verbose, "v", false, "if set, stdout debug log messages")
}

func (c *ResourcesCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
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

func (c *ResourcesCmd) DumpCsv(views []*CfnResourcesView) error {
	csvViews := []CfnResourcesCsvView{}

	for _, view := range views {
		var errorString string
		if view.Error == nil {
			errorString = ""
		} else {
			errorString = view.Error.Error()
		}
		if len(view.Resources) == 0 {
			csvViews = append(csvViews, CfnResourcesCsvView{
				AccountId:           view.AccountId,
				AccountName:         view.AccountName,
				Region:              view.Region,
				StackName:           view.StackName,
				ResourcePhysicalId:  "",
				ResourceLogicalId:   "",
				ResourceType:        "",
				ResourceDescription: "",
				ResourceStatus:      "",
				ResourceDriftStatus: "",
				Error:               errorString,
			})
		}
		for _, resource := range view.Resources {
			csvViews = append(csvViews, CfnResourcesCsvView{
				AccountId:           view.AccountId,
				AccountName:         view.AccountName,
				Region:              view.Region,
				StackName:           view.StackName,
				ResourcePhysicalId:  resource.PhysicalId,
				ResourceLogicalId:   resource.LogicalId,
				ResourceType:        resource.Type,
				ResourceDescription: resource.Description,
				ResourceStatus:      resource.Status,
				ResourceDriftStatus: resource.DriftStatus,
				Error:               errorString,
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

func (c *ResourcesCmd) DumpJson(views []*CfnResourcesView) error {

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

func (c *ResourcesCmd) calcTotalViews() int {
	total := 0
	for _, accountConfig := range c.config.AccountConfigs {
		total += len(accountConfig.Filters.Regions)
	}
	return total
}

func (c *ResourcesCmd) GetGlobalViews() []*CfnResourcesView {
	globalViews := []*CfnResourcesView{}

	totalViews := c.calcTotalViews()
	progress := 0
	channel := make(chan []*CfnResourcesView, totalViews)

	var bar *progressbar.ProgressBar
	if !c.verbose {
		bar = progressbar.Default(int64(totalViews))
	}
	for ai := range c.config.AccountConfigs {
		for ri := range c.config.AccountConfigs[ai].Filters.Regions {

			go func(ch chan []*CfnResourcesView, ai, ri int) {
				views := []*CfnResourcesView{}
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
						views = append(views, &CfnResourcesView{
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

				// describe matched stacks' parameters definitions
				for _, matchedStack := range matchedStacks {
					stackResources, err := cfn.DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
						StackName: matchedStack.StackName,
					})
					if err != nil {
						views = append(views, &CfnResourcesView{
							AccountId:   c.config.AccountConfigs[ai].Id,
							AccountName: c.config.AccountConfigs[ai].Name,
							Region:      c.config.AccountConfigs[ai].Filters.Regions[ri],
							StackName:   *matchedStack.StackName,
							Error:       err,
						})
						break
					}
					var resources []CfnResource
					for _, resource := range stackResources.StackResources {
						description := ""
						driftStatus := ""
						if d := resource.Description; d != nil {
							description = *d
						}
						if d := resource.DriftInformation.StackResourceDriftStatus; d != nil {
							driftStatus = *d
						}
						resources = append(resources, CfnResource{
							PhysicalId:  *resource.PhysicalResourceId,
							LogicalId:   *resource.LogicalResourceId,
							Type:        *resource.ResourceType,
							Status:      *resource.ResourceStatus,
							Description: description,
							DriftStatus: driftStatus,
						})
					}
					views = append(views, &CfnResourcesView{
						AccountId:   c.config.AccountConfigs[ai].Id,
						AccountName: c.config.AccountConfigs[ai].Name,
						Region:      c.config.AccountConfigs[ai].Filters.Regions[ri],
						StackName:   *matchedStack.StackName,
						Resources:   resources,
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

func (c *ResourcesCmd) hasAllTags(stackTags []*cloudformation.Tag, filterTags []config.Tag) bool {
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
