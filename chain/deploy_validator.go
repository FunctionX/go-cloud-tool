package chain

import (
	"fmt"
	"os"
	"sync"
	"time"

	"hub/common"

	"hub/app"
	"hub/logger"

	"fx-tools/aws"
	"fx-tools/docker"

	"github.com/spf13/cobra"
)

func NewDeployValidatorNodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "validator",
		Example: "fx deploy validator --seed  --node_number 4",
		RunE: func(*cobra.Command, []string) (err error) {
			return DeployMultiValidatorNode()
		},
	}
	return cmd
}

func DeployMultiValidatorNode() (err error) {

	cdc := app.MakeCodec()
	cfg := GetConfig()

	if err = (&cfg.ChainConfig).AddValidators(cdc, cfg.NodeNumber, fmt.Sprintf("%s%s", cfg.Delegate, cfg.ChainConfig.Token)); err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	maxParallelChan := make(chan struct{}, 20)
	for i, acc := range cfg.PresetAccounts {
		wg.Add(1)
		maxParallelChan <- struct{}{}
		stackName := fmt.Sprintf("fx-chain-%s-validator-%d-%d", os.ExpandEnv("$USER"), i, time.Now().UnixNano()/1000)

		go func(cfgStr string, acc common.Account) {
			defer wg.Done()
			defer func() { <-maxParallelChan }()

			var cfg Config
			(&cfg).JsonUnmarshal(cfgStr)

			publicIP, privateIP, err := aws.NewAwsEC2Instance(stackName, cfg.InstanceType, cfg.DiskSize)
			if err != nil {
				logger.L.Errorf("new aws ec2 instance error: %s", err.Error())
				return
			}

			cfg.P2P.ExternalAddress = fmt.Sprintf("tcp://%s:26656", privateIP)
			cfg.ValidatorPriKey = acc.NodeKey
			chainCfg, err := cfg.ChainConfig.GenDockerInitCfg(cdc, app.ModuleBasics)
			if err != nil {
				logger.L.Errorf("generate chain init config error: %s", err.Error())
				return
			}
			if err := docker.StartChain(publicIP, append([]string{"init"}, chainCfg)); err != nil {
				logger.L.Errorf("docker start chain error: %s", err.Error())
				return
			}
			fmt.Printf("node: http://%s:26657, name: %s, publicIP: %s, privateIP: %s, instanceType: %s, diskSize: %s\n", publicIP, stackName, publicIP, privateIP, cfg.InstanceType, cfg.DiskSize)
			fmt.Printf("nohup fx batch --ip %s --root %s --parallel 200 --times 15000 --debug > ~/node2/%s.log 2>&1 &\n", privateIP, acc.Key, privateIP)
		}(cfg.JsonMarshal(), acc)
	}
	wg.Wait()
	return nil
}
