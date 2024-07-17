package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/synternet/data-layer-sdk/pkg/service"
	"github.com/synternet/osmosis-publisher/internal/osmosis"
)

var (
	flagPublisherName *string
	flagTendermintAPI *string
	flagRPCAPI        *string
	flagGRPCAPI       *string
	flagPricesSubject *string
	flagSocketAddr    *string
	flagBlocks        *uint64
	metricsUrl        *string
)

// startCmd represents the nft command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		poolIdsSigned, _ := cmd.Flags().GetInt64Slice("pool-ids")
		poolIds := make([]uint64, len(poolIdsSigned))
		for i, id := range poolIdsSigned {
			if id < 0 {
				panic(fmt.Errorf("Pool ID cannot be negative: %d at %d", id, i))
			}
			poolIds[i] = uint64(id)
		}

		publisher, err := osmosis.New(
			database,
			service.WithContext(ctx),
			service.WithTelemetryPeriod(*flagTelemetryPeriod),
			service.WithName(*flagPublisherName),
			service.WithPrefix(*flagPrefixName),
			service.WithPubNats(natsPubConnection),
			service.WithSubNats(natsSubConnection),
			service.WithUserCreds(*flagUserPubCreds),
			service.WithNKeySeed(*flagNkeyPub),
			service.WithPemPrivateKey(*flagPemFile),
			service.WithVerbose(*flagVerbose),
			osmosis.WithTendermintAPI(*flagTendermintAPI),
			osmosis.WithRPCAPI(*flagRPCAPI),
			osmosis.WithGRPCAPI(*flagGRPCAPI),
			osmosis.WithPoolIds(poolIds),
			osmosis.WithBlocksToIndex(*flagBlocks),
			osmosis.WithPriceSubject(*flagPricesSubject),
			osmosis.WithMetrics(*metricsUrl),
			osmosis.WithSocketAddr(*flagSocketAddr),
		)
		if publisher == nil {
			return
		}
		if err != nil {
			slog.Error("publisher failed", "err", err)
			return
		}

		pubCtx := publisher.Start()
		defer publisher.Close()

		select {
		case <-ctx.Done():
			slog.Info("Shutdown")
		case <-pubCtx.Done():
			slog.Info("Publisher stopped", "cause", context.Cause(pubCtx).Error())
			stop()
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	const (
		METRICS_URL        = "PROMETHEUS_EXPORT"
		OSMOSIS_TENDERMINT = "TENDERMINT_API"
		OSMOSIS_RPC        = "APP_API"
		OSMOSIS_GRPC       = "GRPC_API"
		OSMOSIS_NAME       = "PUBLISHER_NAME"
		OSMOSIS_POOLS      = "POOL_IDS"
		OSMOSIS_BLOCKS     = "BLOCKS_TO_INDEX"
		PRICES_SUBJECT     = "PRICES_SUBJECT"
		SOCKET_ADDR        = "SOCKET"
	)

	setDefault(OSMOSIS_TENDERMINT, "tcp://localhost:26657")
	setDefault(OSMOSIS_RPC, "http://localhost:1317")
	setDefault(OSMOSIS_GRPC, "localhost:9090")
	setDefault(OSMOSIS_NAME, "osmosis")
	setDefault(OSMOSIS_POOLS, "1,1077,1223,678,1251,1265,1133,1220,1247,1135,1221,1248")
	setDefault(OSMOSIS_BLOCKS, "20000")
	setDefault(PRICES_SUBJECT, "syntropy_defi.price.single.OSMO")

	metricsUrl = startCmd.Flags().String("prometheus-export", os.Getenv(METRICS_URL), "Interface address and port for Prometheus export (e.g. 0.0.0.0:2112)")

	flagPublisherName = startCmd.Flags().String("publisher-name", os.Getenv(OSMOSIS_NAME), "NATS publisher name as in {prefix}.{name}.>")
	flagTendermintAPI = startCmd.Flags().String("tendermint-api", os.Getenv(OSMOSIS_TENDERMINT), "Full address to the Tendermint RPC")
	flagRPCAPI = startCmd.Flags().String("app-api", os.Getenv(OSMOSIS_RPC), "Full address to the Applications RPC")
	flagGRPCAPI = startCmd.Flags().String("grpc-api", os.Getenv(OSMOSIS_GRPC), "Full address to the Applications gRPC")

	flagPricesSubject = startCmd.Flags().String("prices-subject", os.Getenv(PRICES_SUBJECT), "Subject for prices feed to subscribe to")

	flagSocketAddr = startCmd.Flags().String("socket", os.Getenv(SOCKET_ADDR), "Socket addr to publish data")

	pools := SplitAndTrimEmpty(os.Getenv(OSMOSIS_POOLS), ",", " \t\r\n\b")
	dp := make([]int64, len(pools))
	for i, p := range pools {
		val, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			panic(err)
		}
		dp[i] = val
	}
	startCmd.Flags().Int64Slice("pool-ids", dp, "A list of Osmosis pools to stream volume and liquidity each block")

	envBlocks := os.Getenv(OSMOSIS_BLOCKS)
	blocks, err := strconv.ParseUint(envBlocks, 10, 64)
	if err != nil {
		blocks = 17000
		slog.Warn("Bad number of blocks format", "err", err, "default", blocks)
	}
	flagBlocks = startCmd.Flags().Uint64("blocks-to-index", blocks, "Number of previous blocks to keep track of")
}
