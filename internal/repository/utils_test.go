package repository_test

import (
	"log/slog"
	"time"

	"github.com/Synternet/osmosis-publisher/internal/repository"
	"github.com/Synternet/osmosis-publisher/internal/repository/sqlite"
	repotypes "github.com/Synternet/osmosis-publisher/pkg/repository"
	_ "github.com/lib/pq"

	sdk "github.com/cosmos/cosmos-sdk/types"
	IBCTypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
)

func makeDB() *repository.Repository {
	db, err := sqlite.New("file::memory:?cache=shared")
	if err != nil {
		panic(err)
	}
	repo, err := repository.New(db, slog.Default())
	if err != nil {
		panic(err)
	}

	return repo
}

func addIBCDenoms(repo *repository.Repository) {
	err := repo.SaveIBCDenom(IBCTypes.DenomTrace{Path: "transfer/channel-229", BaseDenom: "A"})
	if err != nil {
		panic(err)
	}
	err = repo.SaveIBCDenom(IBCTypes.DenomTrace{Path: "transfer/channel-229", BaseDenom: "B"})
	if err != nil {
		panic(err)
	}
	err = repo.SaveIBCDenom(IBCTypes.DenomTrace{Path: "transfer/channel-229", BaseDenom: "C"})
	if err != nil {
		panic(err)
	}
}

const TimestampBaseOsmo = 1706716320

func addTokenPrices(repo *repository.Repository) {
	err := repo.SaveTokenPrice(repotypes.TokenPrice{
		LastUpdated: time.Unix(TimestampBaseOsmo, 0),
		Value:       100,
		Name:        "OSMO",
		Base:        "USD",
	})
	if err != nil {
		panic(err)
	}
	err = repo.SaveTokenPrice(repotypes.TokenPrice{
		LastUpdated: time.Unix(TimestampBaseOsmo, 0),
		Value:       100,
		Name:        "ATOM",
		Base:        "USD",
	})
	if err != nil {
		panic(err)
	}
	err = repo.SaveTokenPrice(repotypes.TokenPrice{
		LastUpdated: time.Unix(TimestampBaseOsmo, 0).Add(time.Second),
		Value:       200,
		Name:        "OSMO",
		Base:        "USD",
	})
	if err != nil {
		panic(err)
	}
	err = repo.SaveTokenPrice(repotypes.TokenPrice{
		LastUpdated: time.Unix(TimestampBaseOsmo, 0).Add(time.Second),
		Value:       200,
		Name:        "ATOM",
		Base:        "USD",
	})
	if err != nil {
		panic(err)
	}
	err = repo.SaveTokenPrice(repotypes.TokenPrice{
		LastUpdated: time.Unix(TimestampBaseOsmo, 0).Add(time.Second * 2),
		Value:       300,
		Name:        "OSMO",
		Base:        "USD",
	})
	if err != nil {
		panic(err)
	}
	err = repo.SaveTokenPrice(repotypes.TokenPrice{
		LastUpdated: time.Unix(TimestampBaseOsmo, 0).Add(time.Second * 2),
		Value:       300,
		Name:        "ATOM",
		Base:        "USD",
	})
	if err != nil {
		panic(err)
	}
	err = repo.SaveTokenPrice(repotypes.TokenPrice{
		LastUpdated: time.Unix(TimestampBaseOsmo, 0).Add(time.Second * 6),
		Value:       600,
		Name:        "OSMO",
		Base:        "USD",
	})
	if err != nil {
		panic(err)
	}
	err = repo.SaveTokenPrice(repotypes.TokenPrice{
		LastUpdated: time.Unix(TimestampBaseOsmo, 0).Add(time.Second * 6),
		Value:       600,
		Name:        "ATOM",
		Base:        "USD",
	})
	if err != nil {
		panic(err)
	}
}

func addPools(repo *repository.Repository) {
	repo.SavePool(
		repotypes.Pool{
			Height:    1,
			PoolId:    1,
			Liquidity: must(sdk.ParseCoinsNormalized("10stake")),
			Volume:    must(sdk.ParseCoinsNormalized("100500uosmo")),
		},
	)
	repo.SavePool(
		repotypes.Pool{
			Height:    2,
			PoolId:    1,
			Liquidity: must(sdk.ParseCoinsNormalized("20stake")),
			Volume:    must(sdk.ParseCoinsNormalized("100700uosmo")),
		},
	)
	repo.SavePool(
		repotypes.Pool{
			Height:    3,
			PoolId:    1,
			Liquidity: must(sdk.ParseCoinsNormalized("30stake")),
			Volume:    must(sdk.ParseCoinsNormalized("123700uosmo")),
		},
	)
}

func must[T any](obj T, err error) T {
	if err != nil {
		panic(err)
	}
	return obj
}
