package config

import (
	"fmt"
	"strings"

	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yamlv3"
)

const (
	CRED_TYPE_CLI          = "CLI"
	CRED_TYPE_SERVICE_ROLE = "ServiceRole"
)

type Credential struct {
	Type        string
	ProfileName string
}

type Tag struct {
	Key   string
	Value string
}

type Filters struct {
	Regions        []string
	StackTags      []Tag
	StackNameRegex string
}

type RootConfig struct {
	Credential Credential
	Filters    Filters
}

type AccountConfig struct {
	Name       string
	Id         string
	Credential Credential
	Filters    Filters
}

type CfnGlobalViewsConfig struct {
	RootConfig     RootConfig
	AccountConfigs []AccountConfig
}

func init() {
	config.WithOptions()
	config.AddDriver(yamlv3.Driver)
}

func setDefaultConfig(config *CfnGlobalViewsConfig) error {
	err := []string{}
	for i := range config.AccountConfigs {
		// Credential.Type
		if config.AccountConfigs[i].Credential.Type == "" {
			config.AccountConfigs[i].Credential.Type = config.RootConfig.Credential.Type
		}
		// Credential.ProfileName
		if config.AccountConfigs[i].Credential.ProfileName == "" {
			config.AccountConfigs[i].Credential.ProfileName = config.RootConfig.Credential.ProfileName
		}

		// Filters.Regions
		if len(config.AccountConfigs[i].Filters.Regions) == 0 {
			config.AccountConfigs[i].Filters.Regions = config.RootConfig.Filters.Regions
		}
		// Filters.StackNameRegex
		if config.AccountConfigs[i].Filters.StackNameRegex == "" {
			config.AccountConfigs[i].Filters.StackNameRegex = config.RootConfig.Filters.StackNameRegex
		}

		// FIlters.StackTags
		if len(config.AccountConfigs[i].Filters.StackTags) == 0 {
			config.AccountConfigs[i].Filters.StackTags = config.RootConfig.Filters.StackTags
		}
	}
	if len(err) == 0 {
		return nil
	} else {
		return fmt.Errorf("%s", strings.Join(err, "; "))
	}
}

func validate(config *CfnGlobalViewsConfig) error {
	err := []string{}
	for i, accountConfig := range config.AccountConfigs {
		// Credentials
		if accountConfig.Credential.Type == CRED_TYPE_CLI && accountConfig.Credential.ProfileName == "" {
			err = append(err, fmt.Sprintf(
				"you must specify AccountConfigs[%v].Credential.ProfileName if you select AccountConfigs[%v].Credential.Type as %s", i, i, CRED_TYPE_CLI,
			))
		}
		// Filters
		if len(accountConfig.Filters.Regions) == 0 && len(config.RootConfig.Filters.Regions) == 0 {
			err = append(err, fmt.Sprintf("you must specify at least 1 region at either AccountConfigs[%v].Filter.Regions or RootConfig.Filter.Regions", i))
		}
		// Account
		if accountConfig.Id == "" {
			err = append(err, fmt.Sprintf("AccountConfigs[%v].Id is required", i))
		}
	}

	if len(err) == 0 {
		return nil
	} else {
		return fmt.Errorf("%s", strings.Join(err, "; "))
	}
}

func GetConfig(filePath string) (*CfnGlobalViewsConfig, error) {
	CfnGlobalViewsConfig := &CfnGlobalViewsConfig{}

	err := config.LoadFiles(filePath)
	if err != nil {
		return CfnGlobalViewsConfig, err
	}

	err = config.BindStruct("", &CfnGlobalViewsConfig)
	if err != nil {
		return CfnGlobalViewsConfig, err
	}

	setDefaultConfig(CfnGlobalViewsConfig)
	err = validate(CfnGlobalViewsConfig)
	if err != nil {
		return CfnGlobalViewsConfig, err
	}

	return CfnGlobalViewsConfig, nil

}
