package config

import (
	"io"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

// Config represents the configuration for the exporter
type Config struct {
	Devices  map[string]Device `yaml:"devices"`
	Features struct {
		HWMon bool `yaml:"hwmon,omitempty"`
		POE   bool `yaml:"poe,omitempty"`
	} `yaml:"features,omitempty"`
}

// Device represents a target device
type Device struct {
	Address  string `yaml:"address"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Schema   string `yaml:"schema"`
}

// Load reads YAML from reader and unmashals in Config
func Load(r io.Reader) (*Config, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	c := &Config{}
	err = yaml.Unmarshal(b, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}
