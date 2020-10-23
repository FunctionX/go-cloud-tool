package main

import (
	"fmt"

	"fx-tools/debug"

	"fx-tools/aws"
	"fx-tools/batch"
	"fx-tools/chain"
	"fx-tools/cmd"

	"github.com/spf13/cobra"
)

func main() {
	cobra.EnableCommandSorting = false
	rootCmd := &cobra.Command{
		Use:               "fx",
		Short:             "fx chain tools",
		PersistentPreRunE: cmd.BindFlagsToViper,
	}
	rootCmd.PersistentFlags().Bool("debug", false, "")

	rootCmd.AddCommand(
		aws.NewAwsCmd(),
		chain.NewChainCmd(),
		chain.NewDeployChainCmd(),
		cmd.NewListenCmd(),
		batch.NewBatchSendTxCmd(),
		cmd.NewSeedCmd(),
		cmd.NewStartPromServer(),
		cmd.NewPromCollectorCmd(),
		cmd.LineBreak,
		cmd.NewAccountCmd(),
		cmd.NewTxCmd(),
		cmd.NewDoctorCmd(),
		debug.NewUpdateNodeLogLevel(),
		debug.NewClearLog(),
		debug.NewDFH(),
		debug.NewDockerLog(),
		debug.NewAddPublicKey(),
		debug.NewRestoreAuthorizedKeys(),
	)
	rootCmd.AddCommand(cmd.NewVersionCmd())

	cmd.SilenceMsg(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("\033[1;31m%s\033[0m", fmt.Sprintf("Failed to command execute: %s\n", err.Error()))
	}
}
