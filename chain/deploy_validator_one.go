package chain

import (
	"fmt"
	"os"
	"time"

	"hub/app"
	"hub/logger"

	"fx-tools/aws"
	"fx-tools/docker"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewDeployOneValidatorNodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "one",
		Example: "fx deploy one",
		RunE: func(*cobra.Command, []string) (err error) {
			return DeployOneValidatorNode()
		},
	}
	cmd.Flags().Bool("prom", false, "")
	return cmd
}

func DeployOneValidatorNode() (err error) {

	cdc := app.MakeCodec()
	cfg := GetConfig()

	if err := (&cfg.ChainConfig).AddValidators(cdc, 1, fmt.Sprintf("%s%s", cfg.Delegate, cfg.ChainConfig.Token)); err != nil {
		return err
	}

	stackName := fmt.Sprintf("fx-chain-%s-%s-%d", os.ExpandEnv("$USER"), "one", time.Now().UnixNano()/1000)

	publicIP, privateIP, err := aws.NewAwsEC2Instance(stackName, cfg.InstanceType, cfg.DiskSize)
	if err != nil {
		logger.L.Errorf("new aws ec2 instance error: %s", err.Error())
		return
	}

	cfg.ChainConfig.P2P.ExternalAddress = fmt.Sprintf("tcp://%s:26656", privateIP)
	cfg.ValidatorPriKey = cfg.PresetAccounts[0].NodeKey
	chainCfg, err := cfg.ChainConfig.GenDockerInitCfg(cdc, app.ModuleBasics)
	if err != nil {
		logger.L.Errorf("generate chain init config error: %s", err.Error())
		return
	}
	if err = docker.StartChain(publicIP, append([]string{"init"}, chainCfg)); err != nil {
		logger.L.Errorf("docker start chain error: %s", err.Error())
		return
	}
	logger.L.Infof("name: %s, publicIP: %s, privateIP: %s, node: http://%s:26657", stackName, publicIP, privateIP, publicIP)
	logger.L.Infof("nohup fx push --ip %s --root %s --power 10 --times 50 --debug > /tmp/%s.log 2>&1 &", privateIP, cfg.PresetAccounts[0].Key, privateIP)

	if viper.GetBool("prom") {
		if err = docker.StartPrometheus(publicIP, []string{privateIP}); err != nil {
			logger.L.Errorf("docker start prometheus error: %s", err.Error())
			return
		}
		logger.L.Infof("deploy success to prometheus: http://%s:9090", publicIP)
	}
	return nil
}
