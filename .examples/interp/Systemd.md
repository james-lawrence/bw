

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
  // restart a user service
  if err := systemdu.RestartUnit(context.Background(), "user-service.service"); err != nil {
    log.Fatalln(err)
  }

  // restart a system service
  if err := systemd.RestartUnit(context.Background(), "system-service.service"); err != nil {
    log.Fatalln(err)
  }
}
```
