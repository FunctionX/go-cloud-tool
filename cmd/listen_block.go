package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hub/client"
	"hub/logger"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/go-amino"
	coreTypes "github.com/tendermint/tendermint/rpc/core/types"
	tmTypes "github.com/tendermint/tendermint/types"
)

func NewListenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "listen",
		Short:   "",
		Example: "fx listen --ip 127.0.0.1 --num 10 --times 5s",
		RunE: func(c *cobra.Command, args []string) error {
			logger.L.Infof(": [ip：%s, num：%d， times：%v]", viper.GetString("ip"), viper.GetInt("num"), viper.GetDuration("times"))

			url := fmt.Sprintf("http://%s:%d", viper.GetString("ip"), viper.GetUint("port"))
			return listenNewBlockV2(url, viper.GetDuration("times"), viper.GetInt("num"))
		},
	}
	cmd.Flags().Uint("port", 26657, "RPC")
	cmd.Flags().String("ip", "127.0.0.1", "IP")
	cmd.Flags().Uint("num", 10, "")
	cmd.Flags().Duration("times", 5*time.Second, "")
	return cmd
}

func listenNewBlockV2(url string, times time.Duration, num int) error {
	logger.L.Infof("=====> : [%s] ... ...", url)

	var sigs = make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	cdc := amino.NewCodec()
	tmTypes.RegisterEventDatas(cdc)
	cdc.Seal()

	ctx, cancel := context.WithCancel(context.Background())
	ws, err := client.NewWsClient(cdc, fmt.Sprintf("%s/websocket", url))
	if err != nil {
		return err
	}

	var responsesCh = make(chan client.RPCResponse, 1024)
	query := tmTypes.EventQueryNewBlock.String()
	_, err = ws.Subscribe(ctx, query, responsesCh)
	if err != nil {
		return err
	}

	var blockTxs = make([]int64, 0)
	var blockTimes = make([]time.Time, 0)

	tick := time.Tick(times)
	for {
		select {
		case resp := <-responsesCh:
			if resp.Error != nil {
				logger.L.Errorf("response code: %d, data: %s, msg: %s", resp.Error.Code, resp.Error.Data, resp.Error.Message)
				continue
			}
			var resultEvent coreTypes.ResultEvent
			err := cdc.UnmarshalJSON(resp.Result, &resultEvent)
			if err != nil {
				logger.L.Errorf("failed to unmarshal response err: %s", err)
				continue
			}
			eventBlock, ok := resultEvent.Data.(tmTypes.EventDataNewBlock)
			if !ok {
				continue
			}

			logger.L.Infof("%s ，: %d，: %d \n",
				eventBlock.Block.Time.Format("2006-01-02 15:04:05"),
				eventBlock.Block.Height, len(eventBlock.Block.Txs))

			blockTxs = append(blockTxs, int64(len(eventBlock.Block.Txs)))
			//blockTimes = append(blockTimes, header.Header.Time)
			blockTimes = append(blockTimes, time.Now())

		case <-tick:
			if len(blockTxs) <= 1 || len(blockTimes) <= 1 {
				continue
			}
			var aveNum = len(blockTxs)
			if aveNum >= num {
				aveNum = num
			}

			aveTxs := AverageInt(blockTxs, num)
			aveTime := AverageTime(blockTimes, num)
			logger.L.Infof("%s ===> %d: %fs，%d: %d, TPS: %f\n",
				time.Now().Format("2006-01-02 15:04:05"),
				aveNum, aveTime/float64(time.Second), aveNum, aveTxs,
				float64(aveTxs)/(aveTime/float64(time.Millisecond*1000)))

		case <-ws.ExitCh():
			logger.L.Errorf("，")
			return nil

		case <-sigs:
			logger.L.Infof("Closing ... ")
			ws.Close()
			cancel()
			time.Sleep(300 * time.Millisecond)
			return nil
		}
	}
}

func AverageInt(data []int64, late int) (res int64) {
	if len(data) <= 1 {
		return 0
	}
	if len(data) < late {
		late = len(data)
	}
	for _, number := range data[len(data)-late:] {
		res += number
	}
	return res / int64(late)
}

func AverageTime(data []time.Time, late int) float64 {
	if len(data) <= 1 {
		return 0
	}
	if len(data) < late {
		late = len(data)
	}
	var res time.Duration
	data = data[len(data)-late:]
	for i := 1; i < late; i++ {
		res += data[i].Sub(data[i-1])
	}
	return float64(res / time.Duration(late-1))
}
