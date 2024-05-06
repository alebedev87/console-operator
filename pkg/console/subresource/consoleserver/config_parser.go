package consoleserver

import (
	"gopkg.in/yaml.v2"
)

type ConsoleYAMLParser struct{}

func (b *ConsoleYAMLParser) Parse(configYAML []byte) (*Config, error) {
	config := &Config{}
	if err := yaml.Unmarshal(configYAML, config); err != nil {
		return nil, err
	}
	return config, nil
}
