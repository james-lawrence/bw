ordering of tasks in priority order. (minus cleanup those should be done as convient)

#### cleanup
- [ ] Fork raft protocol to add enhancements around observations.
- [ ] ux improvements

#### implement event stream.
- [ ] send noteworthy events from each agent to the leader during a deploy.
- [ ] store events in a file on the leader.
- [ ] allow clients to receive these events and display them.
- [ ] allow additional clients stream the events.

#### implement initial deploy on agent startup (when possible).
when an agent joins a cluster the cluster leader it should contact the leader and check if it has deployed the latest version of the software.

#### retrieve detailed logs of a deploy for a particular agent.
allow the client to specify the deploy id it wants logs for and the agents to pull from, then print those logs to the client.

#### initial setup work
- [ ] generate skeleton directories. ie) `.bw .bwconfig`
- [ ] populate skeleton directories with pre-populated configurations.

#### local directives
- used to build artifacts to place within the deployspace and then deployed.

#### modules
- modules will allow for organizing the directives in a hierarchical manner.
- will use folders to represent a module. all the directives within the folder (including any child modules)
are executed in order. once all directives are executed it will return
up the stack and continue running siblings.

#### custom plugins
- allow for custom plugins to be executed.
- plugins must be registered with 2 pieces of information: /executable/path and the extension to register.
```
1) look at file extension, see that it is a registered plugin.
2) look at registry for executable to run.
3) read the file contents and make them available via stdin to the plugin.
```
