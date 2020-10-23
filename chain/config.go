package chain

import (
	"encoding/json"
	"time"

	"hub/common"

	"github.com/spf13/viper"
)

type Config struct {
	NodeNumber         int    `mapstructure:"node_number"`
	InstanceType       string `mapstructure:"instance_type"`
	DiskSize           string `mapstructure:"disk_size"`
	Delegate           string `mapstructure:"delegate"`
	common.ChainConfig `mapstructure:",squash"`
}

func (c Config) JsonMarshal() string {
	data, err := json.Marshal(c)
	if err != nil {
		panic(err.Error())
	}
	return string(data)
}

func (c *Config) JsonUnmarshal(data string) {
	if err := json.Unmarshal([]byte(data), c); err != nil {
		panic(err.Error())
	}
}

func GetConfig() Config {
	var config = Config{ChainConfig: common.NewDefChainConfig()}
	if err := viper.Unmarshal(&config); err != nil {
		panic(err.Error())
	}
	switch config.ChainConfig.BlockTime {
	case 5 * time.Second:
		common.SetBlockTime5s(config.ChainConfig.Consensus)
	case 1 * time.Second:
		common.SetBlockTime1s(config.ChainConfig.Consensus)
	default:
		config.ChainConfig.Consensus.TimeoutCommit = config.ChainConfig.BlockTime
	}
	return config
}
