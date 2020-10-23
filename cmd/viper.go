package cmd

import (
	"fmt"
	"os"

	"hub/app"
	"hub/client"
	"hub/common"
	"hub/logger"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func init() {
	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	logX, _ := config.Build()
	logger.L = logX.Sugar()

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if os.Getenv("FX_ENV") == "dev" {
		if err != nil {
			logger.L.Debugf("read config file error: %s\n", err.Error())
		} else {
			logger.L.Debugf("config file found and successfully parsed, %s\n", viper.ConfigFileUsed())
		}
	}
}

func BindFlagsToViper(cmd *cobra.Command, _ []string) error {
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return err
	}

	config := zap.NewDevelopmentConfig()
	if !viper.GetBool("debug") {
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
		logX, _ := config.Build()
		logger.L = logX.Sugar()
	}

	if viper.GetString("ip") == "" || viper.GetInt64("port") <= 0 {
		return nil
	}

	url := fmt.Sprintf("http://%s:%d", viper.GetString("ip"), viper.GetUint("port"))
	cli := client.NewFastClient(app.MakeCodec(), url)
	prefix, err := cli.AddressPrefix()
	if err != nil {
		return err
	}
	common.SetGlobalBech32Prefix(prefix)
	return nil
}

func SilenceMsg(cmd *cobra.Command) {
	cmd.SilenceUsage = true
	//cmd.SilenceErrors = true
	for _, c := range cmd.Commands() {
		c.SilenceUsage = true
		//c.SilenceErrors = true
	}
}

var LineBreak = &cobra.Command{Run: func(*cobra.Command, []string) {}}
