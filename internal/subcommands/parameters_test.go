package subcommands

import (
	"os"
	"testing"

	"github.com/gookit/config/v2"
	cfnConfig "github.com/horietakehiro/cfn-global-views/config"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
)

var (
	TEST_LOGGER     = slog.New(slog.NewJSONHandler(os.Stdout))
	TMP_CONFIG_PATH = "tmp_config.yaml"
	TMP_OUT_PATH    = "tmp_out.csv"
)

func TestParameters_valid(t *testing.T) {
	defer config.ClearAll()

	configPath := "../../config/test_config.yaml"

	c, err := cfnConfig.GetConfig(configPath)
	assert.Nil(t, err)

	cmd := ParametersCmd{
		config: c,
		logger: TEST_LOGGER,
	}

	views := cmd.GetGlobalViews()
	assert.Equal(t, 4, len(views))
	numTokyo := 0
	numOsaka := 0
	for _, v := range views {
		assert.Equal(t, 7, len(v.Parameters))
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

func TestParameters_invalid(t *testing.T) {
	defer config.ClearAll()
	defer func() { os.Remove(TMP_CONFIG_PATH) }()

	tmpConfigYaml := `
RootConfig:
  Credential:
    Type: "CLI"
    ProfileName: not-exist-profile
  Filters:
    Regions:
      - "ap-northeast-1"
AccountConfigs:
  - Name: main-account
    Id: 123456789012
`
	writeTmpYaml(tmpConfigYaml)

	c, err := cfnConfig.GetConfig(TMP_CONFIG_PATH)
	assert.Nil(t, err)

	cmd := ParametersCmd{
		config: c,
		logger: TEST_LOGGER,
	}

	views := cmd.GetGlobalViews()
	assert.Equal(t, 1, len(views))
	assert.NotNil(t, views[0].Error, views[0])

}
