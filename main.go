package main

import (
	_ "github.com/synternet/data-layer-sdk/pkg/dotenv"

	"github.com/synternet/osmosis-publisher/cmd"
)

func main() {
	cmd.Execute()
}
