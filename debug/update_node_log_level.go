package debug

import (
	"context"
	"fmt"

	"fx-tools/aws"
	"fx-tools/docker"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/docker/docker/api/types"
	"github.com/spf13/cobra"
)

func NewUpdateNodeLogLevel() *cobra.Command {
	cmd := &cobra.Command{
		Use: "up-log-level",
		RunE: func(_ *cobra.Command, _ []string) (err error) {

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
			for i, privateIP := range list {
				cli, err := docker.NewCli(fmt.Sprintf("tcp://%s:2376", privateIP))
				if err != nil {
					return err
				}

				err = cli.ContainerStart(context.Background(), "fx-chain", types.ContainerStartOptions{})
				if err != nil {
					return err
				}
				cmd := []string{"sed", "-i", fmt.Sprintf(`s/.*log_level =.*/log_level = "%s"/`, "info"), "/root/.fx/config/config.toml"}
				err = docker.ExecAndStop(cli, "fx-chain", cmd, nil)
				if err != nil {
					return err
				}
				//time.Sleep(1 * time.Minute)
			}
			return nil
		},
	}
	return cmd
}
