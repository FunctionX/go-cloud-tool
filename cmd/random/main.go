package main

import (
	"fmt"

	"fx-tools/batch"
	"fx-tools/cmd"
)

func main() {
	rootCmd := batch.NewBatchRandomTxCmd()
	rootCmd.PersistentPreRunE = cmd.BindFlagsToViper
	rootCmd.Example = "random --ip 127.0.0.1 --root  --fee 1000000000000000000fxc"
	rootCmd.Flags().Uint("port", 26657, "RPC")
	rootCmd.Flags().String("ip", "127.0.0.1", "IP")
	rootCmd.Flags().String("root", "", "")
	rootCmd.Flags().Uint("parallel", 100, "，，")
	rootCmd.Flags().String("fee", "1000000000000000000fxcoin", "")
	rootCmd.Flags().Uint64("gas", 100000, "gas price")
	rootCmd.Flags().Bool("debug", false, "")
	rootCmd.AddCommand(cmd.NewVersionCmd())
	cmd.SilenceMsg(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("\033[1;31m%s\033[0m", fmt.Sprintf("Failed to command execute: %s\n", err.Error()))
	}
}
