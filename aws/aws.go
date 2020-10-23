package aws

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/route53"
)

func CreateCfStack(c *Client, name string, instanceType, diskSize, sshKey string) error {
	svc := cloudformation.New(c.Sess)
	_, err := svc.CreateStack(&cloudformation.CreateStackInput{
		DisableRollback: aws.Bool(false),
		Parameters: []*cloudformation.Parameter{{
			ParameterKey:   aws.String("InstanceType"),
			ParameterValue: aws.String(instanceType),
		}, {
			ParameterKey:   aws.String("SSHLocation"),
			ParameterValue: aws.String("0.0.0.0/0"),
		}, {
			ParameterKey:   aws.String("DiskSize"),
			ParameterValue: aws.String(diskSize),
		}, {
			ParameterKey:   aws.String("SSHKEY"),
			ParameterValue: aws.String(sshKey),
		}},
		StackName: aws.String(name),
		Tags: []*cloudformation.Tag{{
			Key:   aws.String("Name"),
			Value: aws.String(name),
		}},
		TemplateURL: aws.String(c.stackTemplateRUL),
	})
	if err != nil {
		return err
	}
	//logger.L.Debugf("cloud formation create stack success, name: %s", name)
	return nil
}

func GetCfStackIP(c *Client, stackName string) (publicIP, privateIP, instanceId string, err error) {
	svc := cloudformation.New(c.Sess)
	stacks, err := svc.DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String(stackName)})
	if err != nil {
		return
	}
	if len(stacks.Stacks) <= 0 {
		err = errors.New("stacks is empty")
		return
	}
	if status := *stacks.Stacks[0].StackStatus; status != "CREATE_COMPLETE" {
		err = fmt.Errorf("stack is being created: %s", status)
		return
	}

	var dns string
	for _, output := range stacks.Stacks[0].Outputs {
		switch *output.OutputKey {
		case "ServerIP":
			publicIP = *output.OutputValue
			continue
		case "ServerID":
			instanceId = *output.OutputValue
			continue
		case "PrivateIp":
			privateIP = *output.OutputValue
			continue
		case "ServerPubDns":
			dns = *output.OutputValue
			continue
		}
	}
	_ = dns
	return
}

func ListCreateCompleteStacks(c *Client) ([]*cloudformation.StackSummary, error) {
	stacks, err := cloudformation.New(c.Sess).ListStacks(&cloudformation.ListStacksInput{
		StackStatusFilter: aws.StringSlice([]string{"CREATE_COMPLETE"}),
	})
	if err != nil {
		return nil, err
	}
	return stacks.StackSummaries, nil
}

func DeleteCfStackByName(c *Client, stackName string) error {
	svc := cloudformation.New(c.Sess)
	_, err := svc.DeleteStack(&cloudformation.DeleteStackInput{StackName: aws.String(stackName)})
	if err != nil {
		return err
	}

	stacks, err := svc.DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String(stackName)})
	if err != nil {
		return err
	}

	for _, stack := range stacks.Stacks {
		if *stack.StackStatus == "DELETE_IN_PROGRESS" {
			//logger.L.Debugf("===========> success to delete stack: %s", *stack.StackName)
			return nil
		}
	}
	return fmt.Errorf("failed to delete stack")
}

func InstanceVolumeSetName(c *Client, instanceId string, name string) error {
	svc := ec2.New(c.Sess)
	output, err := svc.DescribeInstanceAttribute(&ec2.DescribeInstanceAttributeInput{
		Attribute:  aws.String("blockDeviceMapping"),
		InstanceId: aws.String(instanceId),
	})
	if err != nil {
		return err
	}
	var volumeId string
	for _, mapping := range output.BlockDeviceMappings {
		volumeId = *mapping.Ebs.VolumeId
	}
	if len(volumeId) == 0 {
		return fmt.Errorf("not found ec2 volume id")
	}
	_, err = svc.CreateTags(&ec2.CreateTagsInput{
		Resources: aws.StringSlice([]string{volumeId}),
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(name),
			},
		},
	})
	return err
}

func GetCostAndUsage(c *Client, stackName string, start, end time.Time) (*costexplorer.GetCostAndUsageOutput, error) {
	costExplorer := costexplorer.New(c.Sess)
	usage, err := costExplorer.GetCostAndUsage(&costexplorer.GetCostAndUsageInput{
		Filter: &costexplorer.Expression{
			Tags: &costexplorer.TagValues{
				Key:    aws.String("Name"),
				Values: aws.StringSlice([]string{stackName}),
			},
		},
		Granularity: aws.String("HOURLY"),
		GroupBy: []*costexplorer.GroupDefinition{
			{
				Key:  aws.String("USAGE_TYPE"),
				Type: aws.String("DIMENSION"),
			},
		},
		Metrics: aws.StringSlice([]string{"AmortizedCost", "UsageQuantity"}),
		TimePeriod: &costexplorer.DateInterval{
			Start: aws.String(start.Format("2006-01-02T15:04:05Z")),
			End:   aws.String(end.Format("2006-01-02T15:04:05Z")),
		},
	})
	if err != nil {
		return nil, err
	}
	return usage, nil
}

func ResourceRecordSet(client *Client, name, ip string) error {
	svc := route53.New(client.Sess)
	output, err := svc.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("CREATE"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						TTL:  aws.Int64(300),
						Type: aws.String("A"),
						Name: aws.String(name),
						ResourceRecords: []*route53.ResourceRecord{
							{Value: aws.String(ip)},
						},
					},
				},
			},
			Comment: aws.String("comment"),
		},
		HostedZoneId: aws.String(""),
	})
	if err != nil {
		return err
	}
	return nil
}

func NewTmpDomainName(parent string) string {
	return fmt.Sprintf("%s.%s", UUID(10), parent)
}

const letterBytes = "1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func UUID(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
