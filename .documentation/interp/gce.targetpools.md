### GCE target pool example

```golang
package main

import (
	"bw/interp/gcloud/targetpool"
	"bw/interp/systemd"
	"context"
	"log"
	"time"
)

func main() {
	err := targetpool.Restart(context.Background(), func(ctx context.Context) (err error) {
		// give time for connections to drain
		select {
		case <-time.After(15 * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}

		if err = systemd.RestartUnit(ctx, "bma-daemon.target"); err != nil {
			return err
		}

		// ensure the service remains in the active state for at least 5 seconds.
		sctx, scancel := context.WithTimeout(ctx, 5*time.Second)
		defer scancel()
		if err := systemd.RemainActive(sctx, "bma-daemon.target"); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Fatalln(err)
	}
}
```