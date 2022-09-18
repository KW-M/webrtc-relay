package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

func StringToLogLevel(s string) (log.Level, error) {
	s = strings.ToLower(s)
	switch s {
	case "debug":
		return log.DebugLevel, nil
	case "info":
		return log.InfoLevel, nil
	case "warn":
		return log.WarnLevel, nil
	case "error":
		return log.ErrorLevel, nil
	case "fatal":
		return log.FatalLevel, nil
	case "panic":
		return log.PanicLevel, nil
	case "critical":
		return log.PanicLevel, nil
	default:
		return log.WarnLevel, errors.New("Invalid log level: " + s)
	}
}

func ReadConfigFile(configFilePath string) (WebrtcRelayConfig, error) {
	config := GetDefaultRelayConfig()

	// Read the config file
	configFile, err := os.Open(configFilePath)
	if err != nil {
		return config, err
	}
	defer configFile.Close()

	// read our opened json file as a byte array.
	jsonConfigBytes, err := ioutil.ReadAll(configFile)
	if err != nil {
		return config, err
	}

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'config' which we defined above
	err = json.Unmarshal(jsonConfigBytes, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}
