package main

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscodebuild"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscodepipeline"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscodepipelineactions"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscodestarnotifications"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssns"

	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type CdkStackProps struct {
	awscdk.StackProps
}

const (
	APP_NAME = "cfn-global-views"
)

func CicdStack(scope constructs.Construct, id string, props *CdkStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	pipelinRole := awsiam.NewRole(stack, jsii.String("PipelineRole"), &awsiam.RoleProps{
		AssumedBy: awsiam.NewServicePrincipal(jsii.String("codepipeline.amazonaws.com"), nil),
		RoleName:  jsii.String(fmt.Sprintf("%s-cicd-pipeline-role", APP_NAME)),
	})
	pipelinRole.AddManagedPolicy(awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AdministratorAccess")))
	buildRole := awsiam.NewRole(stack, jsii.String("BuildRole"), &awsiam.RoleProps{
		AssumedBy: awsiam.NewServicePrincipal(jsii.String("codebuild.amazonaws.com"), nil),
		RoleName:  jsii.String(fmt.Sprintf("%s-cicd-build-role", APP_NAME)),
	})
	buildRole.AddManagedPolicy(awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AdministratorAccess")))

	pipeline := awscodepipeline.NewPipeline(stack, jsii.String("Pipeline"), &awscodepipeline.PipelineProps{
		ArtifactBucket: awss3.Bucket_FromBucketName(
			stack, jsii.String("PrivateartifacatBucket"), jsii.String("private-artifact-bucket-382098889955-ap-northeast-1"),
		),
		PipelineName:             jsii.String(fmt.Sprintf("%s-cicd-pipeline", APP_NAME)),
		RestartExecutionOnUpdate: jsii.Bool(false),
		Role:                     pipelinRole,
	})

	sourceArtifact := awscodepipeline.NewArtifact(jsii.String("SourceArtifact"))
	batchBuildArtifact := awscodepipeline.NewArtifact(jsii.String("BatchBuildArtifact"))
	gitBuildArtifact := awscodepipeline.NewArtifact(jsii.String("GitBuildArtifact"))

	githubSourceAction := awscodepipelineactions.NewCodeStarConnectionsSourceAction(
		&awscodepipelineactions.CodeStarConnectionsSourceActionProps{
			ActionName:           jsii.String("Source"),
			RunOrder:             jsii.Number(1),
			VariablesNamespace:   jsii.String("SourceVariables"),
			Role:                 pipelinRole,
			ConnectionArn:        jsii.String("arn:aws:codestar-connections:ap-northeast-1:382098889955:connection/26404591-2de4-4d56-acd0-93232fcdfb27"),
			Repo:                 jsii.String(APP_NAME),
			Branch:               jsii.String("dev"),
			TriggerOnPush:        jsii.Bool(true),
			Output:               sourceArtifact,
			Owner:                jsii.String("horietakehiro"),
			CodeBuildCloneOutput: jsii.Bool(true),
		},
	)
	pipeline.AddStage(&awscodepipeline.StageOptions{
		StageName:           jsii.String("Source"),
		TransitionToEnabled: jsii.Bool(true),
		Actions: &[]awscodepipeline.IAction{
			githubSourceAction,
		},
	})

	batchBuildProject := awscodebuild.NewPipelineProject(stack, jsii.String("BatchBuildProject"), &awscodebuild.PipelineProjectProps{
		BuildSpec: awscodebuild.BuildSpec_FromSourceFilename(jsii.String("deployments/buildspec_batch.yaml")),
		Environment: &awscodebuild.BuildEnvironment{
			BuildImage:           awscodebuild.LinuxBuildImage_AMAZON_LINUX_2_4(),
			ComputeType:          awscodebuild.ComputeType_SMALL,
			Privileged:           jsii.Bool(true),
			EnvironmentVariables: &map[string]*awscodebuild.BuildEnvironmentVariable{},
		},
		GrantReportGroupPermissions: jsii.Bool(true),
		ProjectName:                 jsii.String(fmt.Sprintf("%s-cicd-batch-build-project", APP_NAME)),
		Logging: &awscodebuild.LoggingOptions{
			CloudWatch: &awscodebuild.CloudWatchLoggingOptions{
				Enabled: jsii.Bool(true),
				LogGroup: awslogs.NewLogGroup(stack, jsii.String("BatchBuildLogGroup"), &awslogs.LogGroupProps{
					Retention:     awslogs.RetentionDays_FIVE_DAYS,
					LogGroupName:  jsii.String(fmt.Sprintf("/aws/codebuild/%s-cicd-batch-build-project", APP_NAME)),
					RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
				}),
			},
		},
		Role: buildRole,
	})
	batchBuildAction := awscodepipelineactions.NewCodeBuildAction(
		&awscodepipelineactions.CodeBuildActionProps{
			ActionName:                          jsii.String("Build"),
			RunOrder:                            jsii.Number(1),
			VariablesNamespace:                  jsii.String("BatchBuildVariables"),
			Role:                                pipelinRole,
			Input:                               sourceArtifact,
			CheckSecretsInPlainTextEnvVariables: jsii.Bool(false),
			// EnvironmentVariables:                map[string]awscodebuild.BuildEnvironmentVariable{},
			ExecuteBatchBuild: jsii.Bool(true),
			Project:           batchBuildProject,
			Outputs: &[]awscodepipeline.Artifact{
				batchBuildArtifact,
			},
			CombineBatchBuildArtifacts: jsii.Bool(true),
		},
	)
	pipeline.AddStage(&awscodepipeline.StageOptions{
		StageName:           jsii.String("BatchBuild"),
		TransitionToEnabled: jsii.Bool(true),
		Actions: &[]awscodepipeline.IAction{
			batchBuildAction,
		},
	})

	gitBuildProject := awscodebuild.NewPipelineProject(stack, jsii.String("GitBuildProject"), &awscodebuild.PipelineProjectProps{
		BuildSpec: awscodebuild.BuildSpec_FromSourceFilename(jsii.String("deployments/buildspec_git.yaml")),
		Environment: &awscodebuild.BuildEnvironment{
			BuildImage:           awscodebuild.LinuxBuildImage_AMAZON_LINUX_2_4(),
			ComputeType:          awscodebuild.ComputeType_SMALL,
			Privileged:           jsii.Bool(false),
			EnvironmentVariables: &map[string]*awscodebuild.BuildEnvironmentVariable{},
		},
		GrantReportGroupPermissions: jsii.Bool(true),
		ProjectName:                 jsii.String(fmt.Sprintf("%s-cicd-git-build-project", APP_NAME)),
		Logging: &awscodebuild.LoggingOptions{
			CloudWatch: &awscodebuild.CloudWatchLoggingOptions{
				Enabled: jsii.Bool(true),
				LogGroup: awslogs.NewLogGroup(stack, jsii.String("GitBuildLogGroup"), &awslogs.LogGroupProps{
					Retention:     awslogs.RetentionDays_FIVE_DAYS,
					LogGroupName:  jsii.String(fmt.Sprintf("/aws/codebuild/%s-cicd-git-build-project", APP_NAME)),
					RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
				}),
			},
		},
		Role: buildRole,
	})
	gitBuildAction := awscodepipelineactions.NewCodeBuildAction(
		&awscodepipelineactions.CodeBuildActionProps{
			ActionName:                          jsii.String("Build"),
			RunOrder:                            jsii.Number(1),
			VariablesNamespace:                  jsii.String("GitBuildVariables"),
			Role:                                pipelinRole,
			Input:                               sourceArtifact,
			CheckSecretsInPlainTextEnvVariables: jsii.Bool(false),
			// EnvironmentVariables:                map[string]awscodebuild.BuildEnvironmentVariable{},
			ExecuteBatchBuild: jsii.Bool(false),
			Project:           gitBuildProject,
			Outputs: &[]awscodepipeline.Artifact{
				gitBuildArtifact,
			},
		},
	)
	pipeline.AddStage(&awscodepipeline.StageOptions{
		StageName:           jsii.String("GitBuild"),
		TransitionToEnabled: jsii.Bool(true),
		Actions: &[]awscodepipeline.IAction{
			gitBuildAction,
		},
	})

	// cfnDeployAction := awscodepipelineactions.NewCloudFormationCreateUpdateStackAction(
	// 	&awscodepipelineactions.CloudFormationCreateUpdateStackActionProps{
	// 		ActionName: jsii.String("E2EStackDeploy"),
	// 		RunOrder: jsii.Number(1),
	// 		VariablesNamespace: jsii.String("E2EVariables"),
	// 		Role: ,
	// 	}
	// )

	notifyTopic := awssns.NewTopic(stack, jsii.String("NotifyTopic"), &awssns.TopicProps{
		DisplayName: jsii.String(fmt.Sprintf("%s-cicd-topic", APP_NAME)),
		TopicName:   jsii.String(fmt.Sprintf("%s-cicd-topic", APP_NAME)),
		Fifo:        jsii.Bool(false),
	})
	notifyRule := awscodestarnotifications.NewNotificationRule(
		stack, jsii.String("NotifyCation"), &awscodestarnotifications.NotificationRuleProps{
			DetailType:           awscodestarnotifications.DetailType_BASIC,
			Enabled:              jsii.Bool(true),
			NotificationRuleName: jsii.String(fmt.Sprintf("%s-cicd-notification-rule", APP_NAME)),
			Events: &[]*string{
				jsii.String("codepipeline-pipeline-pipeline-execution-failed"),
				jsii.String("codepipeline-pipeline-pipeline-execution-succeeded"),
			},
			Source: pipeline,
		},
	)
	notifyRule.AddTarget(notifyTopic)

	return stack
}

// func Ec2E2ETestStack(scope constructs.Construct, id string, props *CdkStackProps) awscdk.Stack {
// 	var sprops awscdk.StackProps
// 	if props != nil {
// 		sprops = props.StackProps
// 	}
// 	stack := awscdk.NewStack(scope, &id, &sprops)

// }

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	CicdStack(app, "CicdStack", &CdkStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {
	// If unspecified, this stack will be "environment-agnostic".
	// Account/Region-dependent features and context lookups will not work, but a
	// single synthesized template can be deployed anywhere.
	//---------------------------------------------------------------------------
	return nil

	// Uncomment if you know exactly what account and region you want to deploy
	// the stack to. This is the recommendation for production stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String("123456789012"),
	//  Region:  jsii.String("us-east-1"),
	// }

	// Uncomment to specialize this stack for the AWS Account and Region that are
	// implied by the current CLI configuration. This is recommended for dev
	// stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
	//  Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	// }
}
