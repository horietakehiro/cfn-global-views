build:
	go build -o ./bin/cfn-global-views cmd/main.go

test:
	go test ./...

update-stack-set:
	aws --profile default cloudformation  update-stack-set \
		--stack-set-name CfnGlobalViewsTestStackSet \
		--template-body file://deployments/common_stack.yaml \
		--parameters ParameterKey=StackType,ParameterValue=common ParameterKey=Env,ParameterValue=test \
		--tags Key=ENV,Value=test Key=APP,Value=cfn-global-views
update-main-stack:	
	aws --profile default cloudformation  update-stack \
		--stack-name CfnGlobalViewsMainStack \
		--template-body file://deployments/common_stack.yaml \
		--parameters ParameterKey=StackType,ParameterValue=main ParameterKey=Env,ParameterValue=test \
		--tags Key=ENV,Value=test Key=APP,Value=cfn-global-views
update-sub-stack:
	aws --profile sub cloudformation  update-stack \
		--stack-name CfnGlobalViewsSubStack \
		--template-body file://deployments/common_stack.yaml \
		--parameters ParameterKey=StackType,ParameterValue=sub ParameterKey=Env,ParameterValue=nottest \
		--tags Key=ENV,Value=nottest Key=APP,Value=cfn-global-views

cicd-deploy:
	cd deployments/ && cdk deploy --stack CiCdStack --require-approval=never --no-rollback