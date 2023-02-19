package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
)

var (
	TEST_LOGGER     = slog.New(slog.NewJSONHandler(os.Stdout))
	TMP_CONFIG_PATH = "tmp_config.yaml"
	TMP_OUT_PATH    = "tmp_out.csv"
)

func TestMain_valid_fileout(t *testing.T) {
	defer func() { os.Remove(TMP_OUT_PATH) }()
	cmd := exec.Command("go", "run", "main.go", "parameters", "-c", "../config/test_config.yaml", "-o", TMP_OUT_PATH)
	_, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	assert.Nil(t, err)

	_, err = os.Stat(TMP_OUT_PATH)
	assert.Nil(t, err)

}

func TestMain_valid_stdout(t *testing.T) {
	defer func() { os.Remove(TMP_OUT_PATH) }()
	cmd := exec.Command("go", "run", "./main.go", "parameters", "-c", "../config/test_config.yaml")
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	assert.Nil(t, err)

	assert.Contains(t, string(out), "AccountId,AccountName,Region,StackName,ParameterName,ParameterType,ParameterDescription,ParameterDefaultValue,ParameterActualValue,Error")

}

func TestMain_invalid_args(t *testing.T) {
	defer func() { os.Remove(TMP_OUT_PATH) }()
	cmd := exec.Command("go", "run", "./main.go", "parameters")
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	assert.Nil(t, err)
	assert.Contains(t, string(out), "required")

}
