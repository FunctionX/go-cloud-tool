package debug

import (
	"fmt"
	"io/ioutil"
	"strings"

	"fx-tools/aws"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/spf13/cobra"
)

func NewRestoreAuthorizedKeys() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "restore",
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

			dir, err := ioutil.ReadDir("./keys")
			if err != nil {
				return err
			}

			for _, stack := range stacks {
				publicIP, privateIP, instanceId, err := aws.GetCfStackIP(client, *stack.StackName)
				if err != nil {
					return err
				}
				_, _, _ = publicIP, privateIP, instanceId

				isExist := false
				for _, file := range dir {
					ip := file.Name()
					if strings.HasSuffix(ip, ".pem") {
						ip = ip[:len(ip)-4]
					}
					if publicIP == ip {
						isExist = true
					}
				}
				if isExist {
					fmt.Printf("docker --tls -H %s:2376 run --rm -v /home/ubuntu/.ssh:/root/ssh alpine sed -i '' \n", privateIP)
				}
			}
			return nil
		},
	}
	return cmd
}
