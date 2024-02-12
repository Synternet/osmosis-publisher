package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"

	"github.com/syntropynet/osmosis-publisher/internal/osmosis"
	"github.com/spf13/cobra"
	"github.com/syntropynet/data-layer-sdk/pkg/service"
)

var (
	flagPublisherName *string
	flagTendermintAPI *string
	flagRPCAPI        *string
	flagGRPCAPI       *string
	flagPricesSubject *string
	flagBlocks        *uint64
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
				log.Fatal("Pool ID cannot be negative: ", id, "at", i)
			}
			poolIds[i] = uint64(id)
		}

		publisher := osmosis.New(
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
		)

		if publisher == nil {
			return
		}

		pubCtx := publisher.Start()
		defer publisher.Close()

		select {
		case <-ctx.Done():
			log.Println("Shutdown")
		case <-pubCtx.Done():
			log.Println("Publisher stopped with cause: ", context.Cause(pubCtx).Error())
			stop()
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	const (
		OSMOSIS_TENDERMINT = "TENDERMINT_API"
		OSMOSIS_RPC        = "APP_API"
		OSMOSIS_GRPC       = "GRPC_API"
		OSMOSIS_NAME       = "PUBLISHER_NAME"
		OSMOSIS_POOLS      = "POOL_IDS"
		OSMOSIS_BLOCKS     = "BLOCKS_TO_INDEX"
		PRICES_SUBJECT     = "PRICES_SUBJECT"
	)

	setDefault(OSMOSIS_TENDERMINT, "tcp://localhost:26657")
	setDefault(OSMOSIS_RPC, "http://localhost:1317")
	setDefault(OSMOSIS_GRPC, "localhost:9090")
	setDefault(OSMOSIS_NAME, "osmosis")
	setDefault(OSMOSIS_POOLS, "1,1077,1223,678,1251,1265,1133,1220,1247,1135,1221,1248")
	setDefault(OSMOSIS_BLOCKS, "20000")
	setDefault(PRICES_SUBJECT, "syntropy_defi.price.single.OSMO")

	flagPublisherName = startCmd.Flags().String("publisher-name", os.Getenv(OSMOSIS_NAME), "NATS publisher name as in {prefix}.{name}.>")
	flagTendermintAPI = startCmd.Flags().String("tendermint-api", os.Getenv(OSMOSIS_TENDERMINT), "Full address to the Tendermint RPC")
	flagRPCAPI = startCmd.Flags().String("app-api", os.Getenv(OSMOSIS_RPC), "Full address to the Applications RPC")
	flagGRPCAPI = startCmd.Flags().String("grpc-api", os.Getenv(OSMOSIS_GRPC), "Full address to the Applications gRPC")

	flagPricesSubject = startCmd.Flags().String("prices-subject", os.Getenv(PRICES_SUBJECT), "Subject for prices feed to subscribe to")

	pools := SplitAndTrimEmpty(os.Getenv(OSMOSIS_POOLS), ",", " \t\r\n\b")
	dp := make([]int64, len(pools))
	for i, p := range pools {
		val, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		dp[i] = val
	}
	startCmd.Flags().Int64Slice("pool-ids", dp, "A list of Osmosis pools to stream volume and liquidity each block")

	envBlocks := os.Getenv(OSMOSIS_BLOCKS)
	blocks, err := strconv.ParseUint(envBlocks, 10, 64)
	if err != nil {
		blocks = 17000
		log.Println("Bad number of blocks format: ", err)
	}
	flagBlocks = startCmd.Flags().Uint64("blocks-to-index", blocks, "Number of previous blocks to keep track of")
}
