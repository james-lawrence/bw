package main

import (
	"log"
	"time"

	"bw/interp/env"
	"bw/interp/envx"
)

func main() {
	if envx.Boolean(false, env.DEPLOY_IGNORE_RESTRICTIONS) {
		return
	}

	if envx.Boolean(false, env.DEPLOY_FROZEN) {
		log.Fatalln("deployments are currently frozen")
		return
	}

	// 4PM
	deadline := time.Now().Local().Truncate(24 * time.Hour).Add(16 * time.Hour)

	if time.Now().After(deadline) {
		log.Fatalln("deploy not allowed after", deadline.Format(time.Kitchen))
		return
	}
}
