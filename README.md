### Bearded Wookie configuration management.
secure, fast, reliable, and self contained configuration management.

#### security
bearded-wookie uses SSL/TLS to encrypt all data transfered between agents and clients.

#### project goals - how features are decided on and accepted. (no particular order)
- be a secure, resource efficient, and reliable configuration mamagement tool.
- no required infrastructure. any additional infrastructure should be optional.
- support different deployment strategies. %age, batch, one at a time.
- no centralized server.
- ease of use. mainly around deployment and getting started.

#### workspace
workspace are the top level namespace that describes a deployment.
generally projects only have a single workspace. But multiple workspace may be
desirable for example: a separate workspace for local development configuration
from deployed environments, or for different distros.

#### environments
environments are used to namespace the different clusters
you deploy into. such as production vs staging. environments have there own configuration
that tell the agent how to connect to the cluster.

#### directives
directives are the final piece of a deployment, they represent the steps
and the order of the steps to take to deploy an application. directives
come in three types: package installation (.bwpkg), commands (.bwcmd), file archive installations (.bwfs).

example commands:  
 - `bw credentials create {common-name} {hosts...}` generates ssl/tls certificates for use with bw.  
 - `bw workspace create` creates a deployment workspace. this is a directory + skeleton.  
 - `bw environment create {address}`  
 - `bw agent` runs a bearded-wookie agent.  
 - `bw deploy {environment}` deploy to the production environment  
 - `bw deploy --workspace=".bw" {environment}`  
 - `bw deploy filtered --name='node1' --name='us-east.*' production` to the servers that match the given filters, in this case agents have the name `node1` or match the regex `us-east.*`.  
 - `bw info {environment}` display information and receive events about the environment.  
