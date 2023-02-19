package config

import (
	"os"
	"testing"

	"github.com/gookit/config/v2"
	"github.com/stretchr/testify/assert"
)

func TestConfig_valid(t *testing.T) {
	defer config.ClearAll()

	configPath := "./sample_config.yaml"

	c, err := GetConfig(configPath)
	assert.Nil(t, err)

	mainAccount := c.AccountConfigs[0]
	assert.Equal(t, "123456789012", mainAccount.Id)
	assert.Equal(t, "main-account", mainAccount.Name)
	assert.Equal(t, "root-profile", mainAccount.Credential.ProfileName)
	assert.Equal(t, 2, len(mainAccount.Filters.Regions))
	assert.Equal(t, "ENV", mainAccount.Filters.StackTags[0].Key)
	assert.Equal(t, "test", mainAccount.Filters.StackTags[0].Value)
	assert.Equal(t, "APP", mainAccount.Filters.StackTags[1].Key)
	assert.Equal(t, "cfn-global-views", mainAccount.Filters.StackTags[1].Value)
	assert.Equal(t, "^.*CfnGlobalViews.*$", mainAccount.Filters.StackNameRegex)

	subAccount := c.AccountConfigs[1]
	assert.Equal(t, "210987654321", subAccount.Id)
	assert.Equal(t, "sub-account", subAccount.Name)
	assert.Equal(t, "sub", subAccount.Credential.ProfileName)
	assert.Equal(t, 1, len(subAccount.Filters.Regions))
	assert.Equal(t, 1, len(subAccount.Filters.StackTags))
	assert.Equal(t, "ENV", subAccount.Filters.StackTags[0].Key)
	assert.Equal(t, "prod", subAccount.Filters.StackTags[0].Value)
	assert.Equal(t, "^CfnGlobalViews.*$", subAccount.Filters.StackNameRegex)

}

const (
	TMP_CONFIG_PATH = "tmp_config.yaml"
)

func writeTmpYaml(body string) {
	tmpFile, err := os.Create(TMP_CONFIG_PATH)
	if err != nil {
		panic(err)
	}
	defer tmpFile.Close()
	_, err = tmpFile.WriteString(body)
	if err != nil {
		panic(err)
	}

}

func TestConfig_invalid_credential(t *testing.T) {
	defer config.ClearAll()
	defer func() { os.Remove(TMP_CONFIG_PATH) }()

	tmpConfigYaml := `
RootConfig:
  Credential:
    Type: "CLI"
  Filters:
    Regions:
      - "ap-northeast-1"
AccountConfigs:
  - Name: main-account
    Id: 382098889955
`
	writeTmpYaml(tmpConfigYaml)

	_, err := GetConfig(TMP_CONFIG_PATH)
	assert.NotNil(t, err)

	assert.Contains(t, err.Error(), "you must specify AccountConfigs[0].Credential.ProfileName", err.Error())

}

func TestConfig_invalid_regions(t *testing.T) {
	defer config.ClearAll()
	defer func() { os.Remove(TMP_CONFIG_PATH) }()

	tmpConfigYaml := `
RootConfig:
  Credential:
    Type: "CLI"
    ProfileName: default
  Filters:
    StackNameRegex: "*"
AccountConfigs:
  - Name: main-account
    Id: 382098889955
`
	writeTmpYaml(tmpConfigYaml)

	_, err := GetConfig(TMP_CONFIG_PATH)
	assert.NotNil(t, err)

	assert.Contains(t, err.Error(), "you must specify at least 1 region", err.Error())

}

func TestConfig_invalid_account_id(t *testing.T) {
	defer config.ClearAll()
	defer func() { os.Remove(TMP_CONFIG_PATH) }()

	tmpConfigYaml := `
RootConfig:
  Credential:
    Type: "CLI"
    ProfileName: default
  Filters:
    StackNameRegex: "*"
	Regions:
	  - ap-northeast-1
AccountConfigs:
  - Name: main-account
`
	writeTmpYaml(tmpConfigYaml)

	_, err := GetConfig(TMP_CONFIG_PATH)
	assert.NotNil(t, err)

	assert.Contains(t, err.Error(), "AccountConfigs[0].Id is required", err.Error())

}
