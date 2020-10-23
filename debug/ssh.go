package debug

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

func NewClearLog() *cobra.Command {
	xcmd := &cobra.Command{
		Use: "clear-log",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := ioutil.ReadDir("./keys")
			if err != nil {
				return err
			}
			for _, file := range dir {
				fileData, err := ioutil.ReadFile(filepath.Join("./keys", file.Name()))
				if err != nil {
					return err
				}
				ip := file.Name()
				if strings.HasSuffix(ip, ".pem") {
					ip = ip[:len(ip)-4]
				}
				_, session, err := NewSSHClient(ip, fileData)
				if err != nil {
					return err
				}

				cmd := `echo "clear log success" | sudo tee $(docker inspect --format='{{.LogPath}}' fx-chain)`
				//if err = session.Run(cmd); err != nil {
				//	return err
				//}
				output, err := session.Output(cmd)
				if err != nil {
					return err
				}

				session.Close()
			}
			return nil
		}}
	return xcmd
}

func NewDFH() *cobra.Command {
	xcmd := &cobra.Command{
		Use: "df-all",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := ioutil.ReadDir("./keys")
			if err != nil {
				return err
			}
			for _, file := range dir {
				fileData, err := ioutil.ReadFile(filepath.Join("./keys", file.Name()))
				if err != nil {
					return err
				}
				ip := file.Name()
				if strings.HasSuffix(ip, ".pem") {
					ip = ip[:len(ip)-4]
				}
				_, session, err := NewSSHClient(ip, fileData)
				if err != nil {
					return err
				}

				output, err := session.Output("df -h | grep /dev/nvme")
				if err != nil {
					return err
				}

				session.Close()
			}
			return nil
		}}
	return xcmd
}

func NewDockerLog() *cobra.Command {
	xcmd := &cobra.Command{
		Use:   "docker-level",
		Short: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := ioutil.ReadDir("./keys")
			if err != nil {
				return err
			}
			for i, file := range dir {
				fileData, err := ioutil.ReadFile(filepath.Join("./keys", file.Name()))
				if err != nil {
					return err
				}
				ip := file.Name()
				if strings.HasSuffix(ip, ".pem") {
					ip = ip[:len(ip)-4]
				}
				_, session, err := NewSSHClient(ip, fileData)
				if err != nil {
					return err
				}

				output, err := session.Output("docker cp fx-chain:/root/.fx/config/config.toml . && cat config.toml | grep log_level")
				if err != nil {
					return err
				}

				session.Close()
			}
			return nil
		}}
	return xcmd
}

func NewAddPublicKey() *cobra.Command {
	xcmd := &cobra.Command{
		Use: "add-pub-key",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := ioutil.ReadDir("./keys")
			if err != nil {
				return err
			}
			for i, file := range dir {
				fileData, err := ioutil.ReadFile(filepath.Join("./keys", file.Name()))
				if err != nil {
					return err
				}
				ip := file.Name()
				if strings.HasSuffix(ip, ".pem") {
					ip = ip[:len(ip)-4]
				}
				_, session, err := NewSSHClient(ip, fileData)
				if err != nil {
					return err
				}

				cmd := `echo -e "`
				output, err := session.Output(cmd)
				if err != nil {
					return err
				}

				session.Close()
			}
			return nil
		}}
	return xcmd
}

func NewSSHClient(IP string, privKey []byte) (*ssh.Client, *ssh.Session, error) {
	auth, err := GetSSHAuth(privKey)
	if err != nil {
		return nil, nil, err
	}

	cfg := &ssh.ClientConfig{
		Config:          ssh.Config{},
		User:            "ubuntu",
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		BannerCallback:  ssh.BannerDisplayStderr(),
		Timeout:         5 * time.Second,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", IP, 22), cfg)
	if err != nil {
		return nil, nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, nil, err
	}
	return client, session, nil
}

func GetSSHAuth(pemKey []byte) (ssh.AuthMethod, error) {
	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(pemKey)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(signer), nil
}
