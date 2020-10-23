package docker

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"

	"hub/logger"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/docker/pkg/term"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-connections/tlsconfig"
)

func NewCli(host string) (*client.Client, error) {
	if host == "tcp://127.0.0.1:2376" && runtime.GOOS == "darwin" {
		return client.NewClientWithOpts(client.FromEnv)
	}

	tlsCert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err != nil {
		return nil, err
	}

	tlsConfig := tlsconfig.ClientDefault()
	tlsConfig.InsecureSkipVerify = true
	tlsConfig.Certificates = []tls.Certificate{tlsCert}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return client.NewClientWithOpts(client.WithHost(host), client.WithVersion("1.40"), client.WithHTTPClient(httpClient))
}

func Pull(cli *client.Client, image string) error {
	logger.L.Debugf("docker pull image: %s", image)

	body, err := cli.ImagePull(context.Background(), image, types.ImagePullOptions{RegistryAuth: fxAuth()})
	if err != nil {
		return err
	}

	defer body.Close()
	inFd, is := term.GetFdInfo(body)

	var buf = &bytes.Buffer{}
	if err := jsonmessage.DisplayJSONMessagesStream(body, buf, inFd, is, nil); err != nil {
		logger.L.Warnf("docker pull image display result: %s", err.Error())
	}
	buf.Reset()
	return nil
}

func Run(cli *client.Client, image, name string, cmd, env, ports []string) (id string, err error) {
	logger.L.Debugf("docker run image: %s", image)

	if err = Pull(cli, image); err != nil {
		logger.L.Errorf("docker pull image %s error: %s", image, err.Error())
	}

	config := &container.Config{Image: image, Cmd: cmd, Env: env}
	hostConfig := &container.HostConfig{NetworkMode: container.NetworkMode("host"), RestartPolicy: container.RestartPolicy{Name: "no"}}
	networkConnect := &network.NetworkingConfig{}

	if len(ports) > 0 {
		hostConfig.NetworkMode = container.NetworkMode("default")
		config.ExposedPorts, hostConfig.PortBindings, err = nat.ParsePortSpecs(ports)
		if err != nil {
			return
		}
	}

	containerCreateBody, err := cli.ContainerCreate(context.Background(), config, hostConfig, networkConnect, name)
	if err != nil {
		return
	}

	for _, warning := range containerCreateBody.Warnings {
		logger.L.Warnf("docker create container: ", warning)
	}
	logger.L.Debugf("docker create container success, Id: %s", containerCreateBody.ID)

	err = cli.ContainerStart(context.Background(), containerCreateBody.ID, types.ContainerStartOptions{})
	if err != nil {
		return
	}

	time.Sleep(2 * time.Second)
	inspect, err := cli.ContainerInspect(context.Background(), containerCreateBody.ID)
	if err != nil {
		return
	}

	if !inspect.State.Running || inspect.State.ExitCode != 0 {
		var responseBody io.ReadCloser
		responseBody, err = cli.ContainerLogs(context.Background(), containerCreateBody.ID,
			types.ContainerLogsOptions{ShowStderr: true, ShowStdout: true})
		if err != nil {
			return
		}

		defer responseBody.Close()
		_, _ = stdcopy.StdCopy(os.Stdout, os.Stderr, responseBody)
		err = fmt.Errorf("failed to start %s, please check config", name)
	}
	return containerCreateBody.ID, err
}

func ExecAndRestart(cli *client.Client, container string, cmd, env []string) error {
	logger.L.Infof("docker exec %s %v, %v", container, cmd, env)

	execConfig := types.ExecConfig{AttachStderr: true, AttachStdout: true, Env: env, Cmd: cmd}
	resp, err := cli.ContainerExecCreate(context.Background(), container, execConfig)
	if err != nil {
		return err
	}
	if resp.ID == "" {
		return errors.New("docker exec id empty")
	}

	attach, err := cli.ContainerExecAttach(context.Background(), resp.ID, types.ExecStartCheck{})
	if err != nil {
		return err
	}

	var writerBuf = new(bytes.Buffer)
	_, _ = stdcopy.StdCopy(writerBuf, writerBuf, attach.Reader)
	attach.Close()

	inspect, err := cli.ContainerExecInspect(context.Background(), resp.ID)
	if err != nil {
		return err
	}
	if inspect.ExitCode != 0 {
		return errors.New(writerBuf.String())
	}

	logger.L.Infof("docker restart: %s", inspect.ContainerID)
	return cli.ContainerRestart(context.Background(), container, nil)
}

func ExecAndStop(cli *client.Client, container string, cmd, env []string) error {
	logger.L.Infof("docker exec %s %v, %v", container, cmd, env)

	execConfig := types.ExecConfig{AttachStderr: true, AttachStdout: true, Env: env, Cmd: cmd}
	resp, err := cli.ContainerExecCreate(context.Background(), container, execConfig)
	if err != nil {
		return err
	}
	if resp.ID == "" {
		return errors.New("docker exec id empty")
	}

	attach, err := cli.ContainerExecAttach(context.Background(), resp.ID, types.ExecStartCheck{})
	if err != nil {
		return err
	}

	var writerBuf = new(bytes.Buffer)
	_, _ = stdcopy.StdCopy(writerBuf, writerBuf, attach.Reader)
	attach.Close()

	inspect, err := cli.ContainerExecInspect(context.Background(), resp.ID)
	if err != nil {
		return err
	}
	if inspect.ExitCode != 0 {
		return errors.New(writerBuf.String())
	}

	logger.L.Infof("docker stop: %s", inspect.ContainerID)
	return cli.ContainerStop(context.Background(), container, nil)
}

func fxAuth() string {
	authConfig := types.AuthConfig{}
	encodedJSON, _ := json.Marshal(authConfig)
	return base64.URLEncoding.EncodeToString(encodedJSON)
}
