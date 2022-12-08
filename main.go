package main

import (
	"context"

	log "github.com/sirupsen/logrus"
)

func main() {
	args, err := ParseArgs()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Fatal("failed to parse command line arguments")
	}

	ctx := context.Background()

	if err := Execute(ctx, args); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Fatal("failover unsuccessful")
	}
}
