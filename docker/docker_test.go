package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"hub/app"
	"hub/common"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
)

func Test_Docker_ContainerList(t *testing.T) {
	envClient, err := client.NewEnvClient()
	assert.NoError(t, err)
	list, err := envClient.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	assert.NoError(t, err)
	indent, err := json.MarshalIndent(list, "", "\t")
	assert.NoError(t, err)
	t.Log(string(indent))
}

func Test_Docker_ExecAndRestart(t *testing.T) {
	envClient, err := client.NewEnvClient()
	assert.NoError(t, err)
	// sed -i 's#minimum-gas-prices = ""#minimum-gas-prices = "xxxx"#g' /root/.fx/config/app.toml
	cmd := []string{"sed", "-i", `s#minimum-gas-prices = ""#minimum-gas-prices = "zzzz"#g`, "/root/.fx/config/app.toml"}
	err = ExecAndRestart(envClient, "fx-chain", cmd, nil)
	assert.NoError(t, err)
}

func Test_Docker_Run_Validator_FxChain(t *testing.T) {
	cli, err := NewCli("tcp://127.0.0.1:2376")
	assert.NoError(t, err)

	cdc := app.MakeCodec()
	cfg := common.NewDefChainConfig()

	delegate := fmt.Sprintf("%s%s", "1000000000000000000", cfg.Token)
	validators, err := common.NewValidators(cdc, &cfg, 1, delegate)
	assert.NoError(t, err)
	initCfg, err := common.GenChainInitCfg(cdc, cfg, app.ModuleBasics, validators[0])
	assert.NoError(t, err)

	publicPorts := []string{"26656-26657:26656-26657/tcp", "26656-26657:26656-26657/udp"}
	dockerId, err := Run(cli, chainImage, "fx-chain", append([]string{"init"}, initCfg), nil, publicPorts)
	assert.NoError(t, err)
	t.Log("Container Id:", dockerId)
}
