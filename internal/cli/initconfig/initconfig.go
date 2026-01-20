package initconfig

import (
	"github.com/BurntSushi/toml"
)

func ParseInitConfig(path string) (InitConfig, error) {

	// Parse toml file
	var config InitConfig

	_, err := toml.DecodeFile(path, &config)
	if err != nil {
		return InitConfig{}, err
	}

	// assign defaults
	config.PostProcess()

	return config, nil
}

// iterate over the parsed config and:
// - set missing default values, e.g., the default role of the user
func (*InitConfig) PostProcess() {
	// Todo check parsed config and apply defaults
}
