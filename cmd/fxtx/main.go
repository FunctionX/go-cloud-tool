package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"hub/app"
	"hub/client"
	"hub/common"

	"fx-tools/cmd"

	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/exported"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	cobra.EnableCommandSorting = false
	rootCmd := &cobra.Command{
		Use:               "fxtx",
		Short:             "fx chain transaction validation",
		Example:           "fxtx --ip 3.23.227.132 --key 0000000000000000000000000000000000000000000000000000000000000000 --tx iAMoKBapCv...yjUiAA==",
		PersistentPreRunE: cmd.BindFlagsToViper,
		RunE: func(_ *cobra.Command, _ []string) (err error) {
			cdc := app.MakeCodec()
			var fastClient *client.FastClient
			var isNet = false
			if viper.GetString("ip") != "" {
				isNet = true
				fastClient = client.NewFastClient(cdc, fmt.Sprintf("http://%s:26657", viper.GetString("ip")))
			}

			chainId := viper.GetString("chain-id")
			if chainId == "" && isNet {
				chainId, err = fastClient.ChainId()
				if err != nil {
					return err
				}
			}
			fmt.Println("Chain Id:", chainId)
			prefix := viper.GetString("prefix")
			if prefix == "" && isNet {
				prefix, err = fastClient.AddressPrefix()
				if err != nil {
					return err
				}
			}

			fmt.Println(":", prefix)
			common.SetGlobalBech32Prefix(prefix)
			key := common.PrivKeySecp256k1FromHex(viper.GetString("key"))
			var account exported.Account
			if isNet {
				account, err = fastClient.Account(key.PubKey().Address().Bytes())
				if err != nil {
					return err
				}
			} else {
				account = &auth.BaseAccount{
					Address:       key.PubKey().Address().Bytes(),
					AccountNumber: 0,
					Sequence:      0,
				}
			}

			fmt.Println(":", account.GetAddress().String(),
				"\n\tAccountNumber:", account.GetAccountNumber(), "Sequence:", account.GetSequence(),
				"\n\t:", account.GetCoins().String())

			var stdTx auth.StdTx
			txStr := viper.GetString("tx")
			if account.GetSequence() == 0 && account.GetAccountNumber() == 0 {
				if err := cdc.UnmarshalJSON([]byte(txStr), &stdTx); err != nil {
					return err
				}
			} else {
				if err := cdc.UnmarshalBinaryLengthPrefixed(common.Base64Decode(txStr), &stdTx); err != nil {
					return err
				}
			}

			stdTxData, err := cdc.MarshalJSONIndent(stdTx, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(":\n", string(stdTxData))
			if err = stdTx.ValidateBasic(); err != nil {
				return err
			}
			stdSignMsg := auth.StdSignMsg{
				ChainID:       chainId,
				AccountNumber: account.GetAccountNumber(),
				Sequence:      account.GetSequence(),
				Fee:           stdTx.Fee,
				Msgs:          stdTx.Msgs,
				Memo:          stdTx.Memo,
			}
			var buf = new(bytes.Buffer)
			if err = json.Indent(buf, stdSignMsg.Bytes(), "", "  "); err != nil {
				return err
			}
			fmt.Println("（）:\n", buf.String())
			sign, err := key.Sign(stdSignMsg.Bytes())
			if err != nil {
				return err
			}
			fmt.Println(":", common.Base64EncodToStr(sign))
			return nil
		},
	}
	rootCmd.Flags().String("ip", "", "IP")
	rootCmd.Flags().String("tx", "", "Base64")
	rootCmd.Flags().String("key", "", "Hex")
	rootCmd.Flags().String("chain-id", "", "chain id")
	rootCmd.Flags().String("prefix", "", "")
	rootCmd.AddCommand(cmd.NewVersionCmd())
	cmd.SilenceMsg(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("\033[1;31m%s\033[0m", fmt.Sprintf("Failed to command execute: %s\n", err.Error()))
	}
}
