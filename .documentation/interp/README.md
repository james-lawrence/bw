### Interpreter functionality

bearded wookie has a golang interpreter available for fully scripting
deployments. it attempts to support fully the stdlib minus unsafe - see [yaegi](https://github.com/containous/yaegi) for issues/details.

### Current Functionality
- access to the BW environment variables.
- access to aws elbv2 attach/detach (see AWSELBv2.md).
- access to the BW shell, identical functionality to the bwcmd files.
- systemd unit monitoring and restart/reload/stopping.

### changes to the runtime
- `os.Getwd()` is overridden to be the root of unpacked archive.
- `os.Chdir(dir)` is overridden to always return an error.
- `context.Background()` is overridden to provide the context from the deploy.
- `log.Fatal*` functions are changed to not completely exit the program thereby killing the agent, instead they panic causing the interpreter to fail.
- `log.SetOutput/SetFlags/SetPrefix` are all disabled for the time being. they are noops.

### Planned Functionality
- [ ] gcloud target pools

### Basic Example - shell command

```golang
package main

import (
  "context"
  "log"
  "time"

  "bw/interp/aws/elb"
  "bw/interp/shell"
)

func main() {
  ctx, done := context.WithTimeout(context.Background(), time.Second)
  defer done()

  err := shell.Run(
    ctx, // context is used to control the executed process, cancelling a context kills the process.
    "systemd restart foo.service", // the command to be executed.
    shell.Environ("Foo=bar"), // supply additional environment variables.
    shell.Lenient, // allow the command to fail, this means run will always return nil
    shell.Timeout(time.Minute), // use shell.Timeout to specify maximum execution time allowed if ctx is too long.
  )

  if failed != nil {
    log.Fatalln(err)
  }
}
```
