package cmd

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"hub/client"
	"hub/logger"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/go-amino"
	coreTypes "github.com/tendermint/tendermint/rpc/core/types"
	tmTypes "github.com/tendermint/tendermint/types"
)

func NewPromCollectorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "collector",
		Short:   "",
		Example: "fx collector --ip 127.0.0.1",
		RunE: func(c *cobra.Command, args []string) error {
			startPrometheusServer()
			url := fmt.Sprintf("http://%s:%d", viper.GetString("ip"), viper.GetUint("port"))
			return listenChainNewBlock(url)
		},
	}
	cmd.Flags().Uint("port", 26657, "RPC")
	cmd.Flags().String("ip", "127.0.0.1", "IP")
	return cmd
}

func listenChainNewBlock(url string) error {
	logger.L.Infof("=====> : [%s] ... ...", url)

	cdc := amino.NewCodec()
	tmTypes.RegisterEventDatas(cdc)
	cdc.Seal()

	ws, err := client.NewWsClient(cdc, fmt.Sprintf("%s/websocket", url))
	if err != nil {
		return err
	}

	var responsesCh = make(chan client.RPCResponse, 1024)
	query := tmTypes.EventQueryNewBlock.String()
	_, err = ws.Subscribe(context.Background(), query, responsesCh)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(1 * time.Second)
	var lastBlockTime time.Time
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

			blockNumTxs.Set(float64(len(eventBlock.Block.Txs)))
			realityBlockTime.Set(time.Now().Sub(lastBlockTime).Seconds())

			lastBlockTime = time.Now()
		case <-ticker.C:
			res, err := ws.NumUnconfirmedTxs()
			if err != nil {
				logger.L.Errorf("GetNumUnconfirmedTxs err: %s", err)
				continue
			}
			unconfirmedNumTxs.Set(float64(res.Total))

		case <-ws.ExitCh():
			logger.L.Errorf("，")
			return nil
		}
	}
}

var (
	realityBlockTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "fx",
		Subsystem: "tools",
		Name:      "block_interval_seconds",
		Help:      "",
	})

	blockNumTxs = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "fx",
		Subsystem: "tools",
		Name:      "block_num_txs",
		Help:      "",
	})

	unconfirmedNumTxs = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "fx",
		Subsystem: "tools",
		Name:      "unconfirmed_num_txs",
		Help:      "",
	})
)

func startPrometheusServer() {
	registerer := prometheus.DefaultRegisterer
	registerer.MustRegister(realityBlockTime)
	registerer.MustRegister(blockNumTxs)
	registerer.MustRegister(unconfirmedNumTxs)

	srv := &http.Server{
		Addr: ":8080",
		Handler: promhttp.InstrumentMetricHandler(
			registerer, promhttp.HandlerFor(
				prometheus.DefaultGatherer,
				promhttp.HandlerOpts{MaxRequestsInFlight: 3},
			),
		),
	}
	go func() {
		logger.L.Infof("=====> prometheus gatherer running ... ...")
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.L.Errorf("Prometheus HTTP server ListenAndServe err: %s", err)
		}
	}()
}
