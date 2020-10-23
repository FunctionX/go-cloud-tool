package aws

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewAwsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "aws",
	}
	cmd.AddCommand(NewListStackCmd())
	cmd.AddCommand(NewDeleteStackCmd())
	cmd.AddCommand(NewDeleteAllStackCmd())
	cmd.AddCommand(NewDescribeStacksCmd())
	cmd.AddCommand(NewGetCostAndUsageCmd())
	cmd.AddCommand(NewWatchCmd())
	return cmd
}

func NewListStackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Example: "fx aws list",
		RunE: func(*cobra.Command, []string) (err error) {
			client, err := NewDefAWSClient()
			if err != nil {
				return err
			}

			var total uint64

			var listInstance func(nextToken string) error

			listInstance = func(nextToken string) error {
				input := &cloudformation.ListStacksInput{
					StackStatusFilter: aws.StringSlice([]string{"CREATE_COMPLETE"}),
				}
				if nextToken != "" {
					input.NextToken = aws.String(nextToken)
				}
				stacks, err := cloudformation.New(client.Sess).ListStacks(input)
				if err != nil {
					return err
				}
				for _, summaries := range stacks.StackSummaries {
					if !strings.Contains(*summaries.StackName, viper.GetString("grep")) {
						continue
					}
					publicIP, privateIP, instanceId, err := GetCfStackIP(client, *summaries.StackName)
					if err != nil {
						return err
					}
					total++
					fmt.Printf("time: %s, instanceId: %s, stackName: %s, privateIP: %s, publicIP: %s\n", summaries.CreationTime.Format("2006-01-02T15:04:05Z"), instanceId, *summaries.StackName, privateIP, publicIP)
				}
				if stacks.NextToken != nil {
					return listInstance(*stacks.NextToken)
				}
				return nil
			}

			if err = listInstance(""); err != nil {
				return err
			}
			fmt.Printf("Total: %d\n", total)
			return
		},
	}
	cmd.Flags().String("grep", "", "search about")
	//cmd.Flags().BoolP("invert-match", "v", false, "select non-matching lines")
	return cmd
}

func NewDeleteStackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete",
		Example: "fx aws delete --name fx-chain-test-test-1589526808523378",
		RunE: func(*cobra.Command, []string) (err error) {
			client, err := NewDefAWSClient()
			if err != nil {
				return err
			}
			stackNameList := viper.GetStringSlice("name")
			if len(stackNameList) <= 0 {
				for {
					reader := bufio.NewReader(os.Stdin)
					name, _ := reader.ReadString('\n')
					if strings.Trim(name, "\n") == "" {
						break
					}
					stackNameList = append(stackNameList, strings.Trim(name, "\n"))
				}
			}
			for _, stackName := range stackNameList {
				if err := DeleteCfStackByName(client, stackName); err != nil {
					return err
				} else {
				}
			}
			return nil
		},
	}
	cmd.Flags().StringSlice("name", []string{""}, "")
	return cmd
}

func NewDeleteAllStackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delall",
		Example: "fx aws delall",
		RunE: func(*cobra.Command, []string) (err error) {
			client, err := NewDefAWSClient()
			if err != nil {
				return err
			}
			stacks, err := cloudformation.New(client.Sess).ListStacks(&cloudformation.ListStacksInput{
				StackStatusFilter: aws.StringSlice([]string{"CREATE_COMPLETE"}),
			})
			if err != nil {
				return err
			}
			var stackNameList []string
			for _, summaries := range stacks.StackSummaries {
				if strings.HasPrefix(*summaries.StackName, fmt.Sprintf("fx-chain-%s", os.Getenv("USER"))) {
					stackNameList = append(stackNameList, *summaries.StackName)
				}
			}
			reader := bufio.NewReader(os.Stdin)
			name, _ := reader.ReadString('\n')
			if strings.Trim(name, "\n") != "yes" {
				return
			}
			for _, stackName := range stackNameList {
				if err := DeleteCfStackByName(client, stackName); err != nil {
					return err
				} else {
				}
			}
			return nil
		},
	}
	return cmd
}

func NewDescribeStacksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "desc",
		Example: "fx aws desc fx-chain-ec2-tx-1589446839554265",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, arg []string) (err error) {
			client, err := NewDefAWSClient()
			if err != nil {
				return err
			}
			stacks, err := cloudformation.New(client.Sess).DescribeStacks(
				&cloudformation.DescribeStacksInput{StackName: aws.String(arg[0])})
			if err != nil {
				return err
			}
			fmt.Println(stacks.String())
			return nil
		},
	}
	return cmd
}

func NewGetCostAndUsageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cast",
		Example: "fx aws cast fx-chain-ec2-tx-1589446839554265",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, arg []string) (err error) {
			client, err := NewDefAWSClient()
			if err != nil {
				return err
			}
			start := time.Now().Add(viper.GetDuration("start"))
			end := time.Now().Add(viper.GetDuration("end"))
			usage, err := GetCostAndUsage(client, arg[0], start, end)
			if err != nil {
				return err
			}
			fmt.Println(usage.String())
			return nil
		},
	}
	cmd.Flags().Duration("start", -48*time.Hour, "")
	cmd.Flags().Duration("end", -24*time.Hour, "")
	return cmd
}

func NewWatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "watch",
		Example: "fx aws watch fx-chain-ec2-tx-1589446839554265",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, arg []string) (err error) {
			client, err := NewDefAWSClient()
			if err != nil {
				return err
			}
			start := time.Now().Add(viper.GetDuration("start"))
			end := time.Now().Add(viper.GetDuration("end"))
			statistics, err := GetInstanceStatistics(client, arg[0], start, end)
			if err != nil {
				return err
			}
			for _, statistic := range statistics {
				fmt.Println(statistic.String())
			}
			return nil
		},
	}
	cmd.Flags().Duration("start", -48*time.Hour, "")
	cmd.Flags().Duration("end", -24*time.Hour, "")
	return cmd
}
