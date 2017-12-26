ordering of tasks in priority order. (minus cleanup those should be done as convient)

#### cleanup/bugfixes
- revisit TLS configuration, see if there are any improvements to be made around the TLS server name.

#### custom plugins
- allow for custom plugins to be executed.
- plugins must be registered with 2 pieces of information: /executable/path and the extension to register.
```
1) look at file extension, see that it is a registered plugin.
2) look at registry for executable to run.
3) read the file contents and make them available via stdin to the plugin.
```

#### improve event stream to have a historical record.
- have the quorum nodes store the last n events and be able to scan/seek them.
