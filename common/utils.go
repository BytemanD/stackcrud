package common

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/BytemanD/stackcrud/openstack/common"
	"gopkg.in/yaml.v3"

	"github.com/BytemanD/easygo/pkg/global/logging"
)

func GetIndentJson(v interface{}) (string, error) {
	jsonBytes, _ := json.Marshal(v)
	var buffer bytes.Buffer
	json.Indent(&buffer, jsonBytes, "", "    ")
	return buffer.String(), nil
}
func GetYaml(v interface{}) (string, error) {
	jsonString, err := GetIndentJson(v)
	if err != nil {
		return "", nil
	}
	bytes := []byte(jsonString)
	var out interface{}
	yaml.Unmarshal(bytes, &out)
	yamlBytes, err := yaml.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(yamlBytes), nil
}
func LogError(err error, message string, exit bool) {
	if httpError, ok := err.(*common.HttpError); ok {
		logging.Error("%s, %s, %s", message, httpError.Reason, httpError.Message)
	} else {
		logging.Error("%s, %v", message, err)
	}
	if exit {
		os.Exit(1)
	}
}