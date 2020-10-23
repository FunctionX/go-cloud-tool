package cmd

import (
	"fmt"
	"strings"

	"hub/app"
	"hub/client"

	"fx-tools/aws"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "doctor",
		Short:   "",
		Example: "fx doctor --help",
		RunE: func(cmd *cobra.Command, args []string) error {
			return CheckValidator()
		},
	}
	cmd.PersistentFlags().Uint("port", 26657, "RPC")
	cmd.PersistentFlags().String("grep", "", "")
	cmd.MarkFlagRequired("grep")
	cmd.AddCommand(NewCheckChainCmd())
	return cmd
}

func CheckValidator() (err error) {

	awsCli, err := aws.NewDefAWSClient()
	if err != nil {
		return err
	}

	stacksSummaries, err := aws.ListCreateCompleteStacks(awsCli)
	if err != nil {
		return err
	}

	var nodesPublicIP = make([]string, 0)
	for _, summaries := range stacksSummaries {
		if !strings.Contains(*summaries.StackName, viper.GetString("grep")) {
			continue
		}

		publicIP, _, _, err := aws.GetCfStackIP(awsCli, *summaries.StackName)
		if err != nil {
			return err
		}

		nodesPublicIP = append(nodesPublicIP, publicIP)
	}

	if len(nodesPublicIP) <= 0 {
		fmt.Printf("no found node by: %s\n", viper.GetString("grep"))
		return
	}

	cli := client.NewFastClient(app.MakeCodec(), fmt.Sprintf("http://%s:%d", nodesPublicIP[0], viper.GetInt("port")))

	initValidators, err := cli.Validators(1, 0, 0)
	if err != nil {
		return
	}

	height, err := cli.BlockHeight()
	if err != nil {
		return
	}

	nowValidators, err := cli.Validators(height, 0, 0)
	if err != nil {
		return
	}

	for i := 0; i < len(initValidators.Validators); i++ {
		for j := 0; j < len(nowValidators.Validators); j++ {
			if initValidators.Validators[i].Address.String() == nowValidators.Validators[j].Address.String() {
				initValidators.Validators = append(initValidators.Validators[:i], initValidators.Validators[i+1:]...)
				nowValidators.Validators = append(nowValidators.Validators[:j], nowValidators.Validators[j+1:]...)
				i--
				break
			}
		}
	}

	for i, validator := range initValidators.Validators {
		//cli.ABCIQuery(context.Background(), "/store/staking/key", types.GetValidatorKey(addr))
		fmt.Printf("%d validator address: %s, voting power: %d, proposer priority: %d\n", i, validator.Address, validator.VotingPower, validator.ProposerPriority)
	}
	if len(initValidators.Validators) <= 0 {
		return
	}

	for _, publicIP := range nodesPublicIP {
		cli.Remote = fmt.Sprintf("http://%s:26657", publicIP)
		status, err := cli.Status()
		if err != nil {
			return err
		}

		for i, validator := range initValidators.Validators {
			if validator.Address.String() == status.ValidatorInfo.Address.String() {
				fmt.Printf("%d validator address: %s, listen addr: %s, public IP: http://%s:26657\n",
					i, validator.Address, status.NodeInfo.ListenAddr, publicIP)
			}
		}
	}
	return nil
}

func NewCheckChainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "check",
		Short:   "",
		Example: "fx doctor check --help",
		RunE: func(cmd *cobra.Command, args []string) error {

			awsCli, err := aws.NewDefAWSClient()
			if err != nil {
				return err
			}

			stacksSummaries, err := aws.ListCreateCompleteStacks(awsCli)
			if err != nil {
				return err
			}

			cli := client.NewFastClient(app.MakeCodec(), "")

			for _, summaries := range stacksSummaries {
				if !strings.Contains(*summaries.StackName, viper.GetString("grep")) {
					continue
				}

				publicIP, privateIP, _, err := aws.GetCfStackIP(awsCli, *summaries.StackName)
				if err != nil {
					return err
				}

				cli.Remote = fmt.Sprintf("http://%s:%d", publicIP, viper.GetInt("port"))
				_, err = cli.Health()
				if err != nil {
					fmt.Printf(">> url: http://%s:26657, private IP: %s, name: %s, maybe have quit\n", publicIP, privateIP, *summaries.StackName)
				}
				fmt.Printf("url: http://%s:26657, private IP: %s is very health\n", publicIP, privateIP)
			}
			return nil
		},
	}
	return cmd
}
