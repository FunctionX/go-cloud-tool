package chain

import (
	"time"

	"github.com/spf13/cobra"
)

func NewDeployChainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deploy",
		Example: "fx deploy --help",
	}
	cmd.PersistentFlags().Uint("node_number", 4, "")
	cmd.PersistentFlags().String("instance_type", "c5.xlarge", "")
	cmd.PersistentFlags().String("disk_size", "40", "")
	cmd.PersistentFlags().String("delegate", "100000000000000000000000", "")
	cmd.PersistentFlags().String("node_name", "jack", "")
	cmd.PersistentFlags().String("chain_id", "hub", "")
	cmd.PersistentFlags().String("token", "", "")
	cmd.PersistentFlags().String("address_prefix", "", "")
	cmd.PersistentFlags().Duration("block_time", 5*time.Second, "")
	cmd.PersistentFlags().String("config.p2p.seeds", "", "")
	cmd.PersistentFlags().Uint("config.p2p.max_packet_msg_payload_size", 1, "")
	cmd.PersistentFlags().Uint64("p2p.send_rate", 6, "/")
	cmd.PersistentFlags().Uint64("p2p.recv_rate", 5, "/")
	cmd.MarkFlagRequired("seed")

	cmd.AddCommand(
		NewDeployValidatorNodeCmd(),
		NewDeployNormalNodeCmd(),
		NewDeployOneValidatorNodeCmd(),
	)
	return cmd
}
