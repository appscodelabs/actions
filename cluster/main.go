package main

import (
	"github.com/appscode/go/log"
	logs "github.com/appscode/go/log/golog"
	"github.com/appscodelabs/actions/cluster/pkg/cmds"
	"os"
	"runtime"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	if err := cmds.NewRootCmd().Execute(); err != nil {
		log.Fatalln("Failed to execute root command:", err)
	}
	log.Infoln("Backup Successful")
	os.Exit(0)
}
