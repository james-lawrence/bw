package main

import (
	"bw/interp/shell"
	"context"
	"log"
)

func main() {
	ctx := context.Background()

	if err := shell.Run(ctx, "false", shell.Lenient); err != nil {
		log.Fatalln(err)
	}

	if err := shell.Run(ctx, "echo ${EXAMPLE1}", shell.Environ("EXAMPLE1=foobar")); err != nil {
		log.Fatalln(err)
	}

	// sctx, scancel := context.WithTimeout(ctx, 2*time.Second)
	// defer scancel()
	// if err := systemdu.StartUnit(sctx, "wds-ssh-agent.service"); err != nil {
	// 	log.Fatalln(err)
	// }

	// if err := systemdu.RemainActive(sctx, "wds-ssh-agent.service"); err != nil {
	// 	log.Fatalln(err)
	// }
	// if err := elb.Restart(ctx, restart); err != nil {
	// 	log.Fatalln(err)
	// }
}

func restart(ctx context.Context) error {
	log.Println("application restart initiated")
	defer log.Println("application restart completed")
	return shell.Run(ctx, "sleep 1")
}
