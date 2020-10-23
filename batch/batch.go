package batch

import (
	"github.com/spf13/cobra"
)

func NewBatchSendTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "batch",
		Example: "fx batch --help",
	}
	cmd.PersistentFlags().Uint("port", 26657, "")
	cmd.PersistentFlags().StringSlice("ip", []string{"127.0.0.1"}, "")
	cmd.PersistentFlags().String("root", "", "")
	cmd.PersistentFlags().Uint("parallel", 1000, "")
	cmd.PersistentFlags().String("fee", "", "")
	cmd.PersistentFlags().Uint64("gas", 100000, " price")
	cmd.PersistentFlags().Uint64("times", 50, "")

	cmd.AddCommand(
		NewBatchCommitTxCmd(),
		NewBatchPushSyncTxCmd(),
		NewBatchSyncTxCmd(),
		NewBatchRandomTxCmd(),
	)
	return cmd
}
