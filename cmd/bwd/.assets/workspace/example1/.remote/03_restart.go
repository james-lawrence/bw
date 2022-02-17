package main

import (
	"context"
	"log"
	"time"

	"bw/interp/aws/elb"
	"bw/interp/shell"
	"bw/interp/systemd"
)

func main() {
	err := elb.Restart(context.Background(), func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(30 * time.Second):
			// wait 30 seconds to allow connections requests to finish
		}

		if err := shell.Run(ctx, "echo hello world"); err != nil {
			return err
		}

		// restart a systemd service
		if err := systemd.RestartUnit(ctx, "foo.service"); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Fatalln(err)
	}
}
