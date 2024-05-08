package main

import (
	_ "github.com/Synternet/data-layer-sdk/pkg/dotenv"

	"github.com/Synternet/osmosis-publisher/cmd"
)

func main() {
	cmd.Execute()
}
