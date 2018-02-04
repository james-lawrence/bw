ordering of tasks in priority order. (minus cleanup those should be done as convient)

#### cleanup/bugfixes
- make torrent storage timeout if it cannot successfully download an archive within a reasonable amount of time.
- make packagekit less sensitive to repository information.
- properly wait for instance to be reattached to ELBv2.

#### integrate with vault PKI to dump client/server credentials for a given environment and PKI mount.
#### integrate with aws KMS to dump client/server credentials for a given environment and key name.

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
