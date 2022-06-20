### AWS ELB v2 API Examples

#### shell to restart a systemd service

```golang
package main

import (
  "context"
  "log"
  "time"

  "bw/interp/aws/elb"
  "bw/interp/shell"
)

// do something while instance is detached from load balancer
// this example causes a failure due to timeout.
func main() {
  ctx, done := context.WithTimeout(context.Background(), time.Minute)
  defer done()

  failed := elb.Restart(ctx, func(ctx context.Context) error {
    return shell.Run(
      ctx, // context is used to control the executed process, cancelling a context kills the process.
      "systemd restart foo.service", // the command to be executed.
      shell.Environ("Foo=Bar", "Biz=Baz"), // supply additional environment variables.
      shell.Lenient, // allow the command to fail, this means run will always return nil
      shell.Timeout(time.Second), // if the ctx's timeout is far longer than you want
      // the command to execute for use shell.Timeout to specify maximum time.
    )
  })

  if failed != nil {
    log.Fatalln(err)
  }
}
```

#### timeout example

```golang
package main

import (
  "context"
  "bw/interp/aws/elb"
)

// do something while instance is detached from load balancer
// this example causes a failure due to timeout.
func main() {
  ctx, done := context.WithTimeout(context.Background(), time.Second)
  defer done()

  elb.Restart(ctx, func(ctx context.Context) error {
    <-ctx.Done()
    return ctx.Err()
  })
}
```
