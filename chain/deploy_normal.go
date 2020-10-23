package chain

import (
	"fmt"
	"os"
	"sync"
	"time"

	"hub/logger"

	"fx-tools/aws"
	"fx-tools/docker"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewDeployNormalNodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "normal",
		Example: "fx deploy normal --seed [] --ip <> --num 4",
		Run: func(*cobra.Command, []string) {
			DeployMultiNormalNode()
		},
	}
	cmd.Flags().String("ip", "", "")
	return cmd
}

func DeployMultiNormalNode() {
	cfg := GetConfig()
	valIp := viper.GetString("ip")

	wg := sync.WaitGroup{}
	maxParallelChan := make(chan struct{}, 20)
	for i := 0; i < cfg.NodeNumber; i++ {
		wg.Add(1)
		maxParallelChan <- struct{}{}

		stackName := fmt.Sprintf("fx-chain-%s-normal-%d-%d", os.ExpandEnv("$USER"), i, time.Now().UnixNano()/1000)
		cfg.NodeName = fmt.Sprintf("%s-normal-%d", cfg.NodeName, i)

		go func(cfgStr string, valIP, stackName string) {
			defer wg.Done()
			defer func() { <-maxParallelChan }()

			var cfg Config
			(&cfg).JsonUnmarshal(cfgStr)

			ip, privateIp, err := aws.NewAwsEC2Instance(stackName, cfg.InstanceType, cfg.DiskSize)
			if err != nil {
				logger.L.Errorf("new aws ec2 instance error: %s", err.Error())
				return
			}

			cfg.P2P.ExternalAddress = fmt.Sprintf("tcp://%s:26656", privateIp)
			if err := docker.StartChain(ip, append([]string{"normal"}, cfg.ChainConfig.String(), fmt.Sprintf("http://%s:26657", valIP))); err != nil {
				logger.L.Errorf("docker start chain error: %s", err.Error())
				return
			}
		}(cfg.JsonMarshal(), valIp, stackName)
	}
	wg.Wait()
}
