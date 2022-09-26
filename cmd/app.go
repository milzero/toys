package main

import (
	"flag"

	"github.com/milzero/toys/common"
	"github.com/milzero/toys/server"
)

var (
	addr = flag.String("addr", ":18080", "http service address")
)

func main() {
	log := common.NewLog().WithField("module", "main")
	log.Infof("service have starting")
	flag.Parse()
	if err := server.Start(*addr); err != nil {
		log.Logger.Panicf("service exit because : %s", err)
	}
}
