package batch

import (
	"fmt"
	"os"
	"sync"
	"time"

	"hub/app"
	"hub/client"
	"hub/common"
	"hub/logger"

	"fx-tools/account"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewBatchCommitTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "commit",
		Example: "fx batch commit --ip 127.0.0.1 --root  --parallel 10",
		RunE: func(*cobra.Command, []string) (err error) {

			nodeIPs := viper.GetStringSlice("ip")

			cdc := app.MakeCodec()
			cli := client.NewFastClient(cdc, fmt.Sprintf("http://%s:%d", nodeIPs[0], viper.GetUint("port")))

			fee, err := types.ParseCoin(viper.GetString("fee"))
			if err != nil {
				return
			}

			privateKey := common.PrivKeySecp256k1FromHex(viper.GetString("root"))
			adminAcc, err := account.NewAccount(cli, privateKey, fee)
			if err != nil {
				return
			}
			logger.L.Infof("root account info \n%s", adminAcc.String())

			parallel := viper.GetInt64("parallel")
			adminAcc.Times = viper.GetInt64("times")
			adminAcc.CheckAmount(int64(len(nodeIPs)), parallel)

			accounts, err := adminAcc.GenAccounts(cli, int64(len(nodeIPs)))
			if err != nil {
				return
			}

			time.Sleep(1 * time.Second)
			wg := sync.WaitGroup{}
			for i, acc := range accounts {

				url := fmt.Sprintf("http://%s:%d", nodeIPs[i], viper.GetUint("port"))
				wg.Add(1)
				go func(acc *account.Account, url string, times, parallel int64) {
					defer wg.Done()

					cli := client.NewFastClient(cdc, url)
					newAccChan := acc.BatchDerivedNewAcc(cli, parallel)
					CommitTx(cli, newAccChan)

				}(acc, url, adminAcc.Times, parallel)
			}
			wg.Wait()
			return nil
		},
	}
	return cmd
}

func CommitTx(cli *client.FastClient, accounts chan *account.Account) {

	start := time.Now()

	accountsLen := int64(len(accounts))
	maxParallelChan := make(chan struct{}, accountsLen)

	tmpAcc := <-accounts
	times := tmpAcc.Times
	accounts <- tmpAcc
	iterations := accountsLen * times

	for i := int64(0); i < iterations; i++ {
		acc := <-accounts

		if i != 0 && i%accountsLen == 0 {
		}

		maxParallelChan <- struct{}{}
		go func(times int64, acc *account.Account) {
			defer func() { <-maxParallelChan }()

			if acc.Times <= 0 {
				return
			}

			// key -> receiver
			sendMsg := bank.MsgSend{
				FromAddress: acc.Key.PubKey().Address().Bytes(),
				ToAddress:   acc.Receiver,
				Amount:      types.NewCoins(types.NewCoin(acc.Coin.Denom, types.NewInt(1))),
			}
			if acc.Times == times {
				coin := types.NewCoin(acc.Coin.Denom, acc.Coin.Amount.Sub(acc.Fee.Amount.AddRaw(1).MulRaw(times)))
				sendMsg.Amount = types.NewCoins(coin)
			}
			acc.Times = acc.Times - 1

			stdTx := acc.GenTransferStdTx(types.NewCoins(acc.Fee), sendMsg)
			_, err := cli.BroadcastStdTxCommitIsOk(stdTx)
			if err != nil {
				logger.L.Errorf("batch commit send stdtx, err: %s", err.Error())
				os.Exit(1)
			}

			acc.Sequence = acc.Sequence + 1
			accounts <- acc
		}(times, acc)
	}

	for {
		if len(maxParallelChan) == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
}
