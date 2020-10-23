package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

const namespace = "AWS/EC2"
const period = 300

func GetInstanceStatistics(c *Client, instanceId string, startTime, endTime time.Time) ([]*cloudwatch.MetricDataResult, error) {
	svc := cloudwatch.New(c.Sess)
	data, err := svc.GetMetricData(&cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(startTime),
		EndTime:   aws.Time(endTime),
		MetricDataQueries: []*cloudwatch.MetricDataQuery{
			{
				Id: aws.String("cpu_utilization_01"),
				MetricStat: &cloudwatch.MetricStat{
					Metric: &cloudwatch.Metric{
						Namespace:  aws.String(namespace),
						MetricName: aws.String("CPUUtilization"),
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String("InstanceId"),
								Value: aws.String(instanceId),
							},
						},
					},
					Period: aws.Int64(period),
					Stat:   aws.String("Average"),
					Unit:   aws.String("Percent"),
				},
				ReturnData: aws.Bool(true),
			},
			{
				Id: aws.String("network_in_01"),
				MetricStat: &cloudwatch.MetricStat{
					Metric: &cloudwatch.Metric{
						Namespace:  aws.String(namespace),
						MetricName: aws.String("NetworkIn"),
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String("InstanceId"),
								Value: aws.String(instanceId),
							},
						},
					},
					Period: aws.Int64(period),
					Stat:   aws.String("Average"),
					Unit:   aws.String("Bytes"),
				},
				ReturnData: aws.Bool(true),
			},
			{
				Id: aws.String("network_out_01"),
				MetricStat: &cloudwatch.MetricStat{
					Metric: &cloudwatch.Metric{
						Namespace:  aws.String(namespace),
						MetricName: aws.String("NetworkOut"),
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String("InstanceId"),
								Value: aws.String(instanceId),
							},
						},
					},
					Period: aws.Int64(period),
					Stat:   aws.String("Average"),
					Unit:   aws.String("Bytes"),
				},
				ReturnData: aws.Bool(true),
			},
			{
				Id: aws.String("network_packets_in_01"),
				MetricStat: &cloudwatch.MetricStat{
					Metric: &cloudwatch.Metric{
						Namespace:  aws.String(namespace),
						MetricName: aws.String("NetworkPacketsIn"),
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String("InstanceId"),
								Value: aws.String(instanceId),
							},
						},
					},
					Period: aws.Int64(period),
					Stat:   aws.String("Average"),
					Unit:   aws.String("Count"),
				},
				ReturnData: aws.Bool(true),
			},
			{
				Id: aws.String("network_packets_out_01"),
				MetricStat: &cloudwatch.MetricStat{
					Metric: &cloudwatch.Metric{
						Namespace:  aws.String(namespace),
						MetricName: aws.String("NetworkPacketsOut"),
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String("InstanceId"),
								Value: aws.String(instanceId),
							},
						},
					},
					Period: aws.Int64(period),
					Stat:   aws.String("Average"),
					Unit:   aws.String("Count"),
				},
				ReturnData: aws.Bool(true),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return data.MetricDataResults, nil
}
