package subcommands

import (
	"os"
	"testing"

	"github.com/gookit/config/v2"
	cfnConfig "github.com/horietakehiro/cfn-global-views/config"
	"github.com/stretchr/testify/assert"
)

func TestOutputs_valid(t *testing.T) {
	defer config.ClearAll()

	configPath := "../../config/test_config.yaml"

	c, err := cfnConfig.GetConfig(configPath)
	assert.Nil(t, err)

	cmd := OutputsCmd{
		config: c,
		logger: TEST_LOGGER,
	}

	views := cmd.GetGlobalViews()
	assert.Equal(t, 4, len(views))
	numTokyo := 0
	numOsaka := 0
	for _, v := range views {
		assert.Equal(t, 1, len(v.Outputs))
		if v.AccountName == "sub-account" {
			assert.NotEqual(t, "ap-northeast-3", v.Region)
			assert.Contains(t, v.StackName, "StackSet")
		}
		if v.Region == "ap-northeast-1" {
			numTokyo += 1
		} else if v.Region == "ap-northeast-3" {
			numOsaka += 1
		}

		assert.Nil(t, v.Error)
	}
	assert.Equal(t, 3, numTokyo)
	assert.Equal(t, 1, numOsaka)
}

func TestOutputs_invalid(t *testing.T) {
	defer config.ClearAll()
	defer func() { os.Remove(TMP_CONFIG_PATH) }()

	tmpConfigYaml := `
RootConfig:
  Credential:
    Type: "CLI"
    ProfileName: not-exist-profile
  Filters:
    Regions:
      - "not-exist-region"
AccountConfigs:
  - Name: main-account
    Id: 123456789012
`
	writeTmpYaml(tmpConfigYaml)

	c, err := cfnConfig.GetConfig(TMP_CONFIG_PATH)
	assert.Nil(t, err)

	cmd := OutputsCmd{
		config: c,
		logger: TEST_LOGGER,
	}

	views := cmd.GetGlobalViews()
	assert.Equal(t, 1, len(views))
	assert.NotNil(t, views[0].Error, views[0])

}
