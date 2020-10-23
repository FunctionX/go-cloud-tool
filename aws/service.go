package aws

import (
	"errors"
	"fmt"
	"time"

	"hub/logger"
)

func NewAwsEC2Instance(stackName, instanceType, diskSize string) (ip, privateIp string, err error) {
	client, err := NewDefAWSClient()
	if err != nil {
		logger.L.Errorf("new aws client error: %s", err.Error())
		return
	}

	ip, privateIp, _, err = RunCFStack(client, stackName, instanceType, diskSize)
	if err != nil {
		logger.L.Errorf("run cf Stack error: %s", err.Error())
		return
	}
	return
}

func RunCFStack(client *Client, stackName, instanceType, diskSize string) (publicIP, privateIP, priKey string, err error) {
	priKey, pubKey, err := GenSSHKey()
	if err != nil {
		return
	}

	err = CreateCfStack(client, stackName, instanceType, diskSize, pubKey)
	if err != nil {
		return
	}

	var instanceId string
	du := time.Duration(70)
	for i := 0; i < 6; i++ {
		time.Sleep(du * time.Second)
		publicIP, privateIP, instanceId, err = GetCfStackIP(client, stackName)
		if err != nil {
			logger.L.Debugf("fetch instance publicIP: %s", err.Error())
			du = 5
			continue
		}
		break
	}
	if publicIP == "" || instanceId == "" {
		err = errors.New("failed to get ec2 instance publicIP")
		return
	}
	if err = InstanceVolumeSetName(client, instanceId, stackName); err != nil {
		return
	}
	return
}

func NewStackName(tag string) string {
	return fmt.Sprintf("fx-chain-%s-%s", tag, UUID(10))
}
