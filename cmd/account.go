package cmd

import (
	"fmt"

	"hub/app"
	"hub/client"
	"hub/common"
	"hub/logger"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "account",
		Short:   "",
		Example: "fx account --ip 127.0.0.1 --address ",
		RunE: func(_ *cobra.Command, args []string) (err error) {
			url := fmt.Sprintf("http://%s:%d", viper.GetString("ip"), viper.GetUint("port"))

			var address types.AccAddress
			if privateKey := viper.GetString("key"); privateKey != "" {
				address = common.PrivKeySecp256k1FromHex(privateKey).PubKey().Address().Bytes()
			} else {
				address = common.MustAccAddressFromBech32(viper.GetString("address"))
				if len(args) > 0 {
					address = common.MustAccAddressFromBech32(args[0])
				}
			}
			cdc := app.MakeCodec()
			cli := client.NewFastClient(cdc, url)
			info, err := cli.Account(address)
			if err != nil {
				return err
			}

			bts, err := cdc.MarshalJSONIndent(info, "", "  ")
			if err != nil {
				return err
			}
			logger.L.Infof(string(bts))
			return
		},
	}
	cmd.Flags().Uint("port", 26657, "RPC")
	cmd.Flags().String("ip", "127.0.0.1", "IP")
	cmd.Flags().String("address", "", "")
	cmd.Flags().String("key", "", "")

	return cmd
}
