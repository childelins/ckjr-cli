package workflow

import "gopkg.in/yaml.v3"

type Input struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Hint        string `yaml:"hint,omitempty"`
}

type Step struct {
	ID          string            `yaml:"id"`
	Description string            `yaml:"description"`
	Command     string            `yaml:"command"`
	Params      map[string]string `yaml:"params"`
	Output      map[string]string `yaml:"output,omitempty"`
}

type Workflow struct {
	Description string   `yaml:"description"`
	Triggers    []string `yaml:"triggers"`
	Inputs      []Input  `yaml:"inputs"`
	Steps       []Step   `yaml:"steps"`
	Summary     string   `yaml:"summary,omitempty"`
}

type Config struct {
	Name        string              `yaml:"name"`
	Description string              `yaml:"description"`
	Workflows   map[string]Workflow `yaml:"workflows"`
}

func Parse(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
