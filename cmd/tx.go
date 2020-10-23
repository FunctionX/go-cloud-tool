package cmd

import (
	"encoding/hex"
	"fmt"
	"strings"

	"hub/app"
	"hub/client"
	"hub/common"
	"hub/logger"
	tokenTypes "order/x/token/types"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	tm "github.com/tendermint/tendermint/types"
)

func NewTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tx",
		Short:   "",
		Example: "fx tx --help",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) <= 0 {
				return cmd.Help()
			}
			url := fmt.Sprintf("http://%s:%d", viper.GetString("ip"), viper.GetUint("port"))
			cdc := app.MakeCodec()
			cli := client.NewFastClient(cdc, url)
			tx, err := cli.TxByHash(args[0])
			if err != nil {
				return err
			}
			data, err := cdc.MarshalJSONIndent(tx, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		},
	}

	cmd.PersistentFlags().Uint("port", 26657, "RPC")
	cmd.PersistentFlags().String("ip", "127.0.0.1", "IP")
	cmd.AddCommand(NewUnjailValidatorTxCmd(), NewTransferTxCmd(), NewTokenIssueCmd())
	return cmd
}

func NewUnjailValidatorTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "unjail",
		Short:   "",
		Example: "fx tx unjail --ip 127.0.0.1 --from  --address ",
		RunE: func(_ *cobra.Command, args []string) (err error) {
			url := fmt.Sprintf("http://%s:%d", viper.GetString("ip"), viper.GetUint("port"))
			cdc := app.MakeCodec()
			cli := client.NewFastClient(cdc, url)
			sender := common.PrivKeySecp256k1FromHex(viper.GetString("from"))

			stdSignMsg := auth.StdSignMsg{
				Fee:  auth.StdFee{Gas: 100000, Amount: types.NewCoins(MustParseCoin())},
				Msgs: []types.Msg{slashing.MsgUnjail{ValidatorAddr: sender.PubKey().Address().Bytes()}},
			}
			stdTx, err := cli.GenStdTx(sender, stdSignMsg)
			if err != nil {
				return err
			}
			stdTxData, err := cdc.MarshalJSONIndent(stdTx, "", "  ")
			if err != nil {
				return
			}
			logger.L.Debugf("Std Tx：%s", string(stdTxData))
			res, err := cli.BroadcastStdTxCommit(stdTx)
			if err != nil {
				return err
			}
			indent, err := cdc.MarshalJSONIndent(res, "", "  ")
			if err != nil {
				return
			}
			logger.L.Infof("Tx Result：%s", string(indent))
			return
		},
	}
	cmd.Flags().String("from", "", "")
	cmd.Flags().String("fee", "0", "")
	return cmd
}

func NewTransferTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "transfer",
		Short:   "",
		Example: "fx tx transfer --ip 127.0.0.1 --from  --to  --amount 10000000000 --fee 200000",
		RunE: func(_ *cobra.Command, args []string) (err error) {
			url := fmt.Sprintf("http://%s:%d", viper.GetString("ip"), viper.GetUint("port"))
			sender := common.PrivKeySecp256k1FromHex(viper.GetString("from"))
			to := common.MustAccAddressFromBech32(viper.GetString("to"))
			cdc := app.MakeCodec()
			cli := client.NewFastClient(cdc, url)

			coin, err := types.ParseCoin(viper.GetString("amount"))
			if err != nil {
				return err
			}

			stdSignMsg := auth.StdSignMsg{
				Fee:  auth.StdFee{Gas: 100000, Amount: types.NewCoins(MustParseCoin())},
				Msgs: []types.Msg{bank.MsgSend{FromAddress: sender.PubKey().Address().Bytes(), ToAddress: to, Amount: types.NewCoins(coin)}},
			}
			res, err := cli.BroadcastMsgTxCommit(sender, stdSignMsg)
			if err != nil {
				return err
			}
			logger.L.Infof(", ：%s", res.Hash.String())
			indent, err := cdc.MarshalJSONIndent(res, "", "  ")
			if err != nil {
				return
			}
			logger.L.Infof("：%s", string(indent))
			return
		},
	}
	cmd.Flags().Uint("port", 26657, "RPC")
	cmd.Flags().String("ip", "127.0.0.1", "IP")
	cmd.Flags().String("from", "", "")
	cmd.Flags().String("to", "", "")
	cmd.Flags().String("amount", "10000000000", "")
	cmd.Flags().String("fee", "0", "")
	return cmd
}

func NewTokenIssueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "issue",
		Short:   "",
		Example: "fx token --ip 127.0.0.1 --from  --denom usdt --fee 16000fxt --simulation true",
		RunE: func(_ *cobra.Command, args []string) (err error) {
			url := fmt.Sprintf("http://%s:%d", viper.GetString("ip"), viper.GetUint("port"))
			sender := common.PrivKeySecp256k1FromHex(viper.GetString("from"))
			cdc := app.MakeCodec()
			cli := client.NewFastClient(cdc, url)
			denom := viper.GetString("denom")

			stdSignMsg := auth.StdSignMsg{
				Fee: auth.StdFee{Gas: 100000, Amount: types.NewCoins(MustParseCoin())},
				Msgs: []types.Msg{
					tokenTypes.MsgIssue{
						Denom:       denom,
						Precision:   18,
						TotalSupply: types.NewIntWithDecimal(10000000000, 18),
						Owner:       sender.PubKey().Address().Bytes(),
						Mintable:    true,
					},
				},
			}

			simulation := viper.GetBool("simulation")
			if simulation {
				tx, err := cli.GenStdTx(sender, stdSignMsg)
				if err != nil {
					return err
				}
				txBytes, err := cdc.MarshalBinaryLengthPrefixed(tx)
				if err != nil {
					return err
				}
				hash := hex.EncodeToString(tm.Tx(txBytes).Hash())
				logger.L.Infof("，hash:%s, :%s", strings.ToUpper(hash),
					strings.ToLower(denom+"/"+hash[:3]))
			} else {

				res, err := cli.BroadcastMsgTxCommit(sender, stdSignMsg)
				if err != nil {
					return err
				}
				logger.L.Infof(", ：%s", res.Hash.String())
				indent, err := cdc.MarshalJSONIndent(res, "", "  ")
				if err != nil {
					return err
				}
				logger.L.Infof("：%s", string(indent))
			}
			return
		},
	}
	cmd.Flags().String("from", "", "")
	cmd.Flags().String("fee", "0", "")
	cmd.Flags().String("denom", "", "")
	cmd.Flags().Bool("simulation", false, "hash")
	_ = cmd.MarkFlagRequired("denom")
	return cmd
}

func MustParseCoin() types.Coin {
	feeStr := viper.GetString("fee")
	var fee types.Coin
	if len(feeStr) > 0 {
		parseCoins, err := types.ParseCoin(feeStr)
		if err != nil {
			panic(err.Error())
		}
		fee = parseCoins
	}
	return fee
}
