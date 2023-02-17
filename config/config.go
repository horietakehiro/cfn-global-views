package config

import (
	"fmt"
	"strings"

	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yamlv3"
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
	Regions         []string
	StackTags       []Tag
	StackNamePrefix string
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
		if config.AccountConfigs[i].Credential.ProfileName == "" {
			if config.RootConfig.Credential.ProfileName != "" {
				config.AccountConfigs[i].Credential.ProfileName = config.RootConfig.Credential.ProfileName
			} else {
				config.AccountConfigs[i].Credential.ProfileName = ""
			}
		}
		if len(config.AccountConfigs[i].Filters.Regions) == 0 {
			config.AccountConfigs[i].Filters.Regions = config.RootConfig.Filters.Regions
		}
		if config.AccountConfigs[i].Filters.StackNamePrefix == "" {
			if config.RootConfig.Filters.StackNamePrefix != "" {
				config.AccountConfigs[i].Filters.StackNamePrefix = config.RootConfig.Filters.StackNamePrefix
			} else {
				config.AccountConfigs[i].Filters.StackNamePrefix = ""
			}
		}
		if len(config.AccountConfigs[i].Filters.StackTags) == 0 {
			if len(config.RootConfig.Filters.StackTags) != 0 {
				config.AccountConfigs[i].Filters.StackTags = config.RootConfig.Filters.StackTags
			} else {
				config.AccountConfigs[i].Filters.StackTags = []Tag{}
			}
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
		if len(accountConfig.Filters.Regions) == 0 && len(config.RootConfig.Filters.Regions) == 0 {
			err = append(err, fmt.Sprintf("either AccountConfigs[%v].Filter.Regions or RootConfig.Filter.Regions are required", i))
		}
		if accountConfig.Id == "" {
			err = append(err, fmt.Sprintf("AccountConfigs[%v].Id is required", i))
		}
		if len(accountConfig.Filters.StackTags) == 0 && accountConfig.Filters.StackNamePrefix == "" {
			err = append(err, fmt.Sprintf("either AccountConfigs[%v].Filters.StackTags or AccountConfigs[%v].Filters.StackNamePrefix are required", i, i))
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
