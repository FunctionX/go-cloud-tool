package chain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"fx-tools/aws"

	"github.com/aws/aws-sdk-go/aws/credentials"

	"hub/app"
	"hub/logger"

	"fx-tools/docker"

	"github.com/docker/docker/api/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewChainCmd() *cobra.Command {
	chainCmd := &cobra.Command{
		Use:     "chain",
		Example: "fx chain --help",
	}
	chainCmd.PersistentFlags().String("ip", "127.0.0.1", "")
	chainCmd.AddCommand(
		NewChainStartCmd(),
		NewChainStopCmd(),
		NewChainStopAllCmd(),
		NewChainStatusAllCmd(),
		NewChainStartAllCmd(),
		NewChainResetCmd(),
		NewChainDeployCmd(),
		NewChainGasPriceUpdateCmd(),
	)
	return chainCmd
}

func NewChainStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start",
		Example: "fx chain start --ip 127.0.0.1",
		RunE: func(*cobra.Command, []string) (err error) {
			ip := viper.GetString("ip")
			cli, err := docker.NewCli(fmt.Sprintf("tcp://%s:2376", ip))
			if err != nil {
				return err
			}
			inspect, err := cli.ContainerInspect(context.Background(), "fx-chain")
			if err != nil {
				return err
			}
			if inspect.State.Restarting {
				return
			}
			if inspect.State.Running {
				return
			}
			if err := cli.ContainerStart(context.Background(), inspect.ID, types.ContainerStartOptions{}); err != nil {
				return err
			}
			return
		},
	}
	return cmd
}

func NewChainStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "stop",
		Example: "fx chain stop --ip 127.0.0.1",
		RunE: func(*cobra.Command, []string) (err error) {
			ip := viper.GetString("ip")
			cli, err := docker.NewCli(fmt.Sprintf("tcp://%s:2376", ip))
			if err != nil {
				return err
			}
			inspect, err := cli.ContainerInspect(context.Background(), "fx-chain")
			if err != nil {
				return err
			}
			if inspect.State.Paused || inspect.State.Dead {
				return
			}
			if err := cli.ContainerStop(context.Background(), inspect.ID, nil); err != nil {
				return err
			}
			return
		},
	}
	return cmd
}

func NewChainStopAllCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "stop-all",
		Example: "fx chain stop-all",
		RunE: func(*cobra.Command, []string) (err error) {
			stackTemplateRUL := ""
			client, err := aws.NewAWSClient("us-east-1", stackTemplateRUL, credentials.StaticProvider{
				Value: credentials.Value{
					AccessKeyID:     "",
					SecretAccessKey: "",
				},
			})
			if err != nil {
				return err
			}

			stacks, err := aws.ListCreateCompleteStacks(client)
			if err != nil {
				return err
			}

			var list []string
			for _, stack := range stacks {
				publicIP, privateIP, instanceId, err := aws.GetCfStackIP(client, *stack.StackName)
				if err != nil {
					return err
				}
				_, _, _ = publicIP, privateIP, instanceId
				list = append(list, privateIP)
			}
			for _, ip := range list {
				cli, err := docker.NewCli(fmt.Sprintf("tcp://%s:2376", ip))
				if err != nil {
					return err
				}
				inspect, err := cli.ContainerInspect(context.Background(), "fx-chain")
				if err != nil {
					return err
				}
				if inspect.State.Paused || inspect.State.Dead {
					continue
				}
				if err := cli.ContainerStop(context.Background(), inspect.ID, nil); err != nil {
					return err
				}
			}
			return nil
		},
	}
	return cmd
}

func NewChainStartAllCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start-all",
		Example: "fx chain start-all",
		RunE: func(*cobra.Command, []string) (err error) {
			stackTemplateRUL := ""
			client, err := aws.NewAWSClient("us-east-1", stackTemplateRUL, credentials.StaticProvider{
				Value: credentials.Value{
					AccessKeyID:     "",
					SecretAccessKey: "",
				},
			})
			if err != nil {
				return err
			}

			stacks, err := aws.ListCreateCompleteStacks(client)
			if err != nil {
				return err
			}

			var list []string
			for _, stack := range stacks {
				publicIP, privateIP, instanceId, err := aws.GetCfStackIP(client, *stack.StackName)
				if err != nil {
					return err
				}
				_, _, _ = publicIP, privateIP, instanceId
				list = append(list, privateIP)
			}
			wg := sync.WaitGroup{}
			for _, ip := range list {
				wg.Add(1)
				go func(ip string) {
					defer wg.Done()
					cli, err := docker.NewCli(fmt.Sprintf("tcp://%s:2376", ip))
					if err != nil {
						logger.L.Error(ip, err.Error())
						return
					}
					inspect, err := cli.ContainerInspect(context.Background(), "fx-chain")
					if err != nil {
						logger.L.Error(ip, err.Error())
						return
					}
					if inspect.State.Status == "running" {
						return
					}
					if err := cli.ContainerStart(context.Background(), inspect.ID, types.ContainerStartOptions{}); err != nil {
						logger.L.Error(ip, err.Error())
					}
					inspect, err = cli.ContainerInspect(context.Background(), "fx-chain")
					if err != nil {
						logger.L.Error(ip, err.Error())
						return
					}

				}(ip)
			}
			wg.Wait()
			return nil
		},
	}
	return cmd
}

func NewChainStatusAllCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status",
		Example: "fx chain status",
		RunE: func(*cobra.Command, []string) (err error) {
			stackTemplateRUL := ""
			client, err := aws.NewAWSClient("us-east-1", stackTemplateRUL, credentials.StaticProvider{
				Value: credentials.Value{
					AccessKeyID:     "",
					SecretAccessKey: "",
				},
			})
			if err != nil {
				return err
			}

			stacks, err := aws.ListCreateCompleteStacks(client)
			if err != nil {
				return err
			}

			for _, stack := range stacks {
				publicIP, privateIP, instanceId, err := aws.GetCfStackIP(client, *stack.StackName)
				if err != nil {
					return err
				}
				_, _, _ = publicIP, privateIP, instanceId
				cli, err := docker.NewCli(fmt.Sprintf("tcp://%s:2376", privateIP))
				if err != nil {
					return err
				}
				inspect, err := cli.ContainerInspect(context.Background(), "fx-chain")
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	return cmd
}

func NewChainResetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reset",
		Example: "fx reset --ip 127.0.0.1",
		RunE: func(*cobra.Command, []string) (err error) {
			ip := viper.GetString("ip")
			cli, err := docker.NewCli(fmt.Sprintf("tcp://%s:2376", ip))
			if err != nil {
				return err
			}
			inspect, err := cli.ContainerInspect(context.Background(), "fx-chain")
			if err != nil {
				return err
			}
			if inspect.State.Restarting {
				return
			}
			if !inspect.State.Running {
				if err := cli.ContainerStart(context.Background(), inspect.ID, types.ContainerStartOptions{}); err != nil {
					return err
				}
				time.Sleep(100 * time.Millisecond)
			}
			cmd := []string{"rm", "-rf", "/root/.fx"}
			if err = docker.ExecAndRestart(cli, inspect.ID, cmd, nil); err != nil {
				return err
			}
			logger.L.Infof("http://%s:26657/status", ip)
			return
		},
	}
	cmd.Flags().String("ip", "127.0.0.1", "")
	return cmd
}

func NewChainGasPriceUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "set-gas-price",
		Example: "fx chain set-gas-price --ip 127.0.0.1 --gas-price 0.1atom",
		RunE: func(*cobra.Command, []string) (err error) {

			cli, err := docker.NewCli(fmt.Sprintf("tcp://%s:2376", viper.GetString("ip")))
			if err != nil {
				return err
			}
			gasprice := viper.GetString("gas-price")
			if len(gasprice) == 0 {
				return errors.New("gas-price can not be empty")
			}
			gasprice = strings.Replace(gasprice, "/", `\/`, 5)
			setGasPriceStr := fmt.Sprintf(`s/.*minimum-gas-prices =.*/minimum-gas-prices = "%s"/`, gasprice)
			// sed -i 's#minimum-gas-prices = ""#minimum-gas-prices = "xxxx"#g' /root/.fx/config/app.toml
			cmd := []string{"sed", "-i", setGasPriceStr, "/root/.fx/config/app.toml"}
			return docker.ExecAndRestart(cli, "fx-chain", cmd, nil)
		},
	}
	cmd.Flags().String("gas-price", "", "gas-price")
	return cmd
}

func NewChainDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deploy",
		Example: "fx chain deploy --public_ip 127.0.0.1 --private_ip 127.0.0.1",
		RunE: func(*cobra.Command, []string) (err error) {

			privateIP := viper.GetString("private_ip")
			publicIP := viper.GetString("public_ip")

			cdc := app.MakeCodec()
			cfg := GetConfig()
			cfg.ChainConfig.Instrumentation.Prometheus = true

			if err := (&cfg.ChainConfig).AddValidators(cdc, 1, fmt.Sprintf("%s%s", cfg.Delegate, cfg.ChainConfig.Token)); err != nil {
				return err
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
			logger.L.Infof("publicIP: %s, privateIP: %s, node: http://%s:26657", publicIP, privateIP, publicIP)
			logger.L.Infof("nohup fx batch commit --ip %s --root %s --power 1500 --times 100 --debug > /tmp/%s.log 2>&1 &", privateIP, cfg.PresetAccounts[0].Key, privateIP)

			if viper.GetBool("prom") {
				if err = docker.StartPrometheus(publicIP, []string{privateIP}); err != nil {
					logger.L.Errorf("docker start prometheus error: %s", err.Error())
					return
				}
				logger.L.Infof("deploy success to prometheus: http://%s:9090", publicIP)
			}
			return nil
		},
	}
	cmd.Flags().String("public_ip", "", "")
	cmd.Flags().String("private_ip", "", "")
	cmd.Flags().Bool("prom", false, "")
	return cmd
}
