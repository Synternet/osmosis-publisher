package main

import (
	_ "github.com/syntropynet/data-layer-sdk/pkg/dotenv"

	"github.com/SyntropyNet/osmosis-publisher/cmd"
)

func main() {
	cmd.Execute()
}
