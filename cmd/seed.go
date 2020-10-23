package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"hub/app"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/p2p/pex"
	"github.com/tendermint/tendermint/version"
)

func NewSeedCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "seed",
		Short:   "",
		Long:    "nohup fx seed > /tmp/fx-seed.log 2>&1 &",
		Example: "fx seed --secret functionx", //
		RunE: func(cmd *cobra.Command, args []string) error {
			myLog := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
			chainID := viper.GetString("chain_id")
			listenAddress := viper.GetString("laddr")

			cfg := config.DefaultP2PConfig()

			// allow a lot of inbound peers since we disconnect from them quickly in seed mode
			cfg.MaxNumInboundPeers = viper.GetInt("max_num_inbound_peers")

			// keep trying to make outbound connections to exchange peering info
			cfg.MaxNumOutboundPeers = viper.GetInt("max_num_outbound_peers")

			addrBookStrict := viper.GetBool("addr_book_strict")

			nodeKey := GetNodeKey(viper.GetString("secret"))
			nodeKeyBts, err := app.MakeCodec().MarshalJSON(nodeKey)
			if err != nil {
				return err
			}
			addrBookFile := filepath.Join(os.Getenv("HOME"), ".fx-seed-addr-book.json")
			myLog.Info("addr book file", "path", addrBookFile)
			if err = ioutil.WriteFile(addrBookFile, nodeKeyBts, os.ModePerm); err != nil {
				return err
			}

			myLog.Info("seed running", "key", nodeKey.ID(), "listen", listenAddress, "chain", chainID,
				"strict-routing", addrBookStrict, "max-inbound", cfg.MaxNumInboundPeers, "max-outbound", cfg.MaxNumOutboundPeers,
			)

			// TODO(roman) expose per-module log levels in the config
			filteredLogger := log.NewFilter(myLog, log.AllowDebug())

			protocolVersion := p2p.NewProtocolVersion(version.P2PProtocol, version.BlockProtocol, 0)

			nodeInfo := p2p.DefaultNodeInfo{
				ProtocolVersion: protocolVersion,
				DefaultNodeID:   nodeKey.ID(),
				ListenAddr:      listenAddress,
				Network:         chainID,
				Version:         "0.0.1",
				Channels:        []byte{pex.PexChannel},
				Moniker:         fmt.Sprintf("%s-seed", chainID),
			}

			addr, err := p2p.NewNetAddressString(p2p.IDAddressString(nodeInfo.DefaultNodeID, nodeInfo.ListenAddr))
			if err != nil {
				return err
			}

			transport := p2p.NewMultiplexTransport(nodeInfo, *nodeKey, p2p.MConnConfig(cfg))
			if err := transport.Listen(*addr); err != nil {
				panic(err)
			}

			book := pex.NewAddrBook(addrBookFile, addrBookStrict)
			book.SetLogger(filteredLogger.With("module", "book"))

			pexReactor := pex.NewReactor(book, &pex.ReactorConfig{
				SeedMode: true,
				// Seeds:    args.SeedConfig.Seeds,
			})
			pexReactor.SetLogger(filteredLogger.With("module", "pex"))

			sw := p2p.NewSwitch(cfg, transport)
			sw.SetLogger(filteredLogger.With("module", "switch"))
			sw.SetNodeKey(nodeKey)
			sw.AddReactor("pex", pexReactor)

			// last
			sw.SetNodeInfo(nodeInfo)

			err = sw.Start()
			if err != nil {
				return err
			}
			sw.Wait()
			return nil
		}}
	if !rootCmd.HasParent() {
		rootCmd.PersistentPreRunE = BindFlagsToViper
	}
	rootCmd.Flags().String("secret", "functionx", "node key secret")
	rootCmd.Flags().String("chain_id", "hub", "network identifier")
	rootCmd.Flags().String("laddr", "tcp://0.0.0.0:26656", "Address to listen for incoming connections")
	rootCmd.Flags().Uint("max_num_inbound_peers", 1000, "maximum number of inbound connections")
	rootCmd.Flags().Uint("max_num_outbound_peers", 100, "maximum number of outbound connections")
	rootCmd.Flags().Bool("addr_book_strict", false, "")
	rootCmd.AddCommand(NewShowNodeId())
	return rootCmd
}

func NewShowNodeId() *cobra.Command {
	xcmd := &cobra.Command{
		Use:     "id",
		Short:   "ID",
		Example: "fx seed id --secret functionx", //
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeKey := GetNodeKey(viper.GetString("secret"))

			data, err := app.MakeCodec().MarshalJSON(nodeKey)
			if err != nil {
				return err
			}
			fmt.Printf("{\"node_id\":\"%s\"}\n", nodeKey.ID())
			fmt.Println(string(data))
			return nil
		}}
	xcmd.Flags().String("secret", "functionx", "node key secret")
	return xcmd
}

func GetNodeKey(secret string) *p2p.NodeKey {
	if secret != "" {
		return &p2p.NodeKey{PrivKey: ed25519.GenPrivKeyFromSecret([]byte(secret))}
	} else {
		return &p2p.NodeKey{PrivKey: ed25519.GenPrivKey()}
	}
}
