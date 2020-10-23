package docker

import (
	"fmt"

	"hub/logger"
)

func StartChain(ip string, cmd []string) (err error) {
	cli, err := NewCli(fmt.Sprintf("tcp://%s:2376", ip))
	if err != nil {
		logger.L.Errorf("docker new cli error: %s", err.Error())
		return
	}

	_, err = Run(cli, , "fx-chain", cmd, nil, nil)
	if err != nil {
		logger.L.Errorf("docker run error: %s", err.Error())
		return
	}
	return nil
}

func StartPrometheus(ip string, cmd []string) (err error) {
	cli, err := NewCli(fmt.Sprintf("tcp://%s:2376", ip))
	if err != nil {
		return
	}

	_, err = Run(cli, , "fx-prometheus", cmd, nil, nil)
	return
}
