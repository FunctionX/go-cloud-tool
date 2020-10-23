package cmd

import (
	"fmt"
	"os"
	"time"

	"hub/logger"

	"fx-tools/aws"
	"fx-tools/docker"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewStartPromServer() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "prom",
		Short:   "",
		Example: "fx prom --ip 127.0.0.1 --node <chain ip>",
		RunE: func(*cobra.Command, []string) (err error) {
			ip := viper.GetString("ip")
			if err := docker.StartPrometheus(ip, viper.GetStringSlice("node")); err != nil {
				return err
			}
			fmt.Printf(": http://%s:9090\n", ip)
			return nil
		},
	}
	cmd.Flags().String("ip", "127.0.0.1", "IP")
	cmd.PersistentFlags().StringSlice("node", []string{"127.0.0.1"}, "IP")
	cmd.AddCommand(NewDeployPromCmd())
	return cmd
}

func NewDeployPromCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "new",
		Short:   "",
		Example: "fx prom new --node <chain ip>",
		RunE: func(*cobra.Command, []string) (err error) {

			stackName := fmt.Sprintf("fx-prom-%s-%d", os.ExpandEnv("$USER"), time.Now().UnixNano()/1000)
			publicIP, _, err := aws.NewAwsEC2Instance(stackName, "c5.large", "40")
			if err != nil {
				logger.L.Errorf("new aws ec2 instance error: %s", err.Error())
				return
			}
			if err = docker.StartPrometheus(publicIP, viper.GetStringSlice("node")); err != nil {
				logger.L.Errorf("docker start prometheus error: %s", err.Error())
				return
			}
			logger.L.Infof(": http://%s:9090", publicIP)
			return
		},
	}
	return cmd
}
