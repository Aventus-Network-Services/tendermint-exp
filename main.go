package main

import (
	"github.com/ArtosSystems/tendermint-exp/oddeven"
	"github.com/tendermint/tendermint/abci/server"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"
	"os"
)

func main() {
	app := oddeven.NewOddEvenApplication()
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	// Start the listener
	srv, err := server.NewServer("tcp://0.0.0.0:26658", "socket", app)
	if err != nil {
		panic(err)
	}
	srv.SetLogger(logger.With("module", "abci-server"))
	if err := srv.Start(); err != nil {
		panic(err)
	}
	// Stop upon receiving SIGTERM or CTRL-C.
	cmn.TrapSignal(logger, func() {
		// Cleanup
		_ = srv.Stop()
	})

	// Run forever.
	select {}
}
