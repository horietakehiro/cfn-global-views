package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
)

var (
	TEST_LOGGER     = slog.New(slog.NewJSONHandler(os.Stdout))
	TMP_CONFIG_PATH = "tmp_config.yaml"
	TMP_OUT_PATH    = "tmp_out.csv"
)

func TestMain_Parameters_valid_fileout(t *testing.T) {
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

func TestMain_Parameters_valid_stdout_csv(t *testing.T) {
	defer func() { os.Remove(TMP_OUT_PATH) }()
	cmd := exec.Command("go", "run", "./main.go", "parameters", "-c", "../config/test_config.yaml")
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	assert.Nil(t, err)

	assert.Contains(t, string(out), "AccountId,AccountName,Region,StackName,ParameterName,ParameterType,ParameterDescription,ParameterDefaultValue,ParameterActualValue,Error")

}

func TestMain_Parameters_valid_stdout_json(t *testing.T) {
	defer func() { os.Remove(TMP_OUT_PATH) }()
	cmd := exec.Command("go", "run", "./main.go", "parameters", "-c", "../config/test_config.yaml", "-f", "json")
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	assert.Nil(t, err)

	assert.True(t, strings.HasPrefix(string(out), "["), string(out))
	assert.True(t, strings.HasSuffix(string(out), "]"), string(out))

}

func TestMain_Parameters_valid_fileout_excel(t *testing.T) {
	tmp_excel_path := "tmp.xlsx"
	defer func() { os.Remove(tmp_excel_path) }()
	cmd := exec.Command("go", "run", "main.go", "parameters", "-c", "../config/test_config.yaml", "-o", tmp_excel_path, "-f", "excel")
	_, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	assert.Nil(t, err)

	_, err = os.Stat(tmp_excel_path)
	assert.Nil(t, err)

}

func TestMain_Resources_valid_fileout_excel(t *testing.T) {
	tmp_excel_path := "tmp.xlsx"
	defer func() { os.Remove(tmp_excel_path) }()
	cmd := exec.Command("go", "run", "main.go", "resources", "-c", "../config/test_config.yaml", "-o", tmp_excel_path, "-f", "excel")
	_, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	assert.Nil(t, err)

	_, err = os.Stat(tmp_excel_path)
	assert.Nil(t, err)

}

func TestMain_Outputs_valid_fileout_excel(t *testing.T) {
	tmp_excel_path := "tmp.xlsx"
	defer func() { os.Remove(tmp_excel_path) }()
	cmd := exec.Command("go", "run", "main.go", "outputs", "-c", "../config/test_config.yaml", "-o", tmp_excel_path, "-f", "excel")
	_, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	assert.Nil(t, err)

	_, err = os.Stat(tmp_excel_path)
	assert.Nil(t, err)

}

func TestMain_Parameters_invalid_args(t *testing.T) {
	defer func() { os.Remove(TMP_OUT_PATH) }()
	cmd := exec.Command("go", "run", "./main.go", "parameters")
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	assert.Nil(t, err)
	assert.Contains(t, string(out), "required")

}

func TestMain_Resources_valid_fileout(t *testing.T) {
	defer func() { os.Remove(TMP_OUT_PATH) }()
	cmd := exec.Command("go", "run", "main.go", "resources", "-c", "../config/test_config.yaml", "-o", TMP_OUT_PATH)
	_, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	assert.Nil(t, err)

	_, err = os.Stat(TMP_OUT_PATH)
	assert.Nil(t, err)

}

func TestMain_Resources_valid_stdout_csv(t *testing.T) {
	defer func() { os.Remove(TMP_OUT_PATH) }()
	cmd := exec.Command("go", "run", "./main.go", "resources", "-c", "../config/test_config.yaml")
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	assert.Nil(t, err)

	assert.Contains(t, string(out), "AccountId,AccountName,Region,StackName,ResourcePhysicalId,ResourceLogicalId,ResourceType,ResourceDescription,ResourceStatus,ResourceDriftStatus,Error")

}

func TestMain_Resources_valid_stdout_json(t *testing.T) {
	defer func() { os.Remove(TMP_OUT_PATH) }()
	cmd := exec.Command("go", "run", "./main.go", "resources", "-c", "../config/test_config.yaml", "-f", "json")
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	assert.Nil(t, err)

	assert.True(t, strings.HasPrefix(string(out), "["), string(out))
	assert.True(t, strings.HasSuffix(string(out), "]"), string(out))

}

func TestMain_Resources_invalid_args(t *testing.T) {
	defer func() { os.Remove(TMP_OUT_PATH) }()
	cmd := exec.Command("go", "run", "./main.go", "resources")
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	assert.Nil(t, err)
	assert.Contains(t, string(out), "required")

}

func TestMain_Outputs_valid_fileout(t *testing.T) {
	defer func() { os.Remove(TMP_OUT_PATH) }()
	cmd := exec.Command("go", "run", "main.go", "outputs", "-c", "../config/test_config.yaml", "-o", TMP_OUT_PATH)
	_, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	assert.Nil(t, err)

	_, err = os.Stat(TMP_OUT_PATH)
	assert.Nil(t, err)

}

func TestMain_Outputs_valid_stdout_csv(t *testing.T) {
	defer func() { os.Remove(TMP_OUT_PATH) }()
	cmd := exec.Command("go", "run", "./main.go", "outputs", "-c", "../config/test_config.yaml")
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	assert.Nil(t, err)

	assert.Contains(t, string(out), "AccountId,AccountName,Region,StackName,OutputName,OutputValue,OutputDescription,OutputExportName,Error")

}

func TestMain_Outputs_valid_stdout_json(t *testing.T) {
	defer func() { os.Remove(TMP_OUT_PATH) }()
	cmd := exec.Command("go", "run", "./main.go", "outputs", "-c", "../config/test_config.yaml", "-f", "json")
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	assert.Nil(t, err)

	assert.True(t, strings.HasPrefix(string(out), "["), string(out))
	assert.True(t, strings.HasSuffix(string(out), "]"), string(out))

}

func TestMain_Outputs_invalid_args(t *testing.T) {
	defer func() { os.Remove(TMP_OUT_PATH) }()
	cmd := exec.Command("go", "run", "./main.go", "outputs")
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	assert.Nil(t, err)
	assert.Contains(t, string(out), "required")

}
