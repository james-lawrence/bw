### Accessing Environment Variables

```golang
package main

import (
  "log"

  "bw/interp/env"
)

func main() {
  log.Println(env.FOO)
}
// export FOO=bar
// Output:
// bar
```
