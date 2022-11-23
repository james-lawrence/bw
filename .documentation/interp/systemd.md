

### Managing Systemd services

```golang
package main

import (
  "log"
  "context"

  "bw/interp/systemdu"
  "bw/interp/systemd"
)

func main() {
  // start a system service
  if err := systemd.StartUnit(context.Background(), "system-service.service"); err != nil {
    log.Fatalln(err)
  }

  // restart a system service
  if err := systemd.RestartUnit(context.Background(), "system-service.service"); err != nil {
    log.Fatalln(err)
  }

  // ensure the service remains in the active state for at least 30 seconds.
  sctx, scancel := context.WithTimeout(context.Background(), 30*time.Second)
  defer scancel()
  if err := systemd.RemainActive(sctx, "system-service.service"); err != nil {
    log.Fatalln(err)
  }

  // restart a user service
  if err := systemdu.RestartUnit(context.Background(), "user-service.service"); err != nil {
    log.Fatalln(err)
  }
}
```
