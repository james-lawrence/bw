### Bearded Wookie configuration management.
fast, secure, reliable, and self sufficient configuration management.

#### project goals - how features are decided on and accepted. (no particular order)
- be a small, fast, robust, cluster configuration mamagement tool.
- support different deployment strategies. %age, batch, rolling, one at a time.
- no centralized server.
- ease of use. mainly around deployment and getting started.
- no required infrastructure. any additional infrastructure should be optional.

#### what this project is not.
- bearded wookie is not about infrastructure management. use something like terraform.

#### security
bearded-wookie uses SSL/TLS to encrypt all data transfered between agents and clients.

#### deployspace
deployspace are the top level namespace that describes a deployment.
generally projects only have a single deployspace. But multiple deployspace may be
desirable for example: a separate deployspace for local development configuration
from deployed environments, or for different distros.

#### environments
generally environments are used to configure the different environments
you deploy into. such as production vs staging. environments have there own configuration
that tell the agent how to connect to the cluster.

#### directives
directives are the final piece of a deployment, they represent the steps
and the order of the steps to take to deploy an application. directives
come in three types: package installation (.bwpkg), commands (.bwcmd), file archive installations (.bwfs).

example commands:  
 - `bw init {common-name} {hosts...}` generates ssl/tls certificates for use with bw.  
 - `bw deployspace create {name}` creates a deployment workspace with the given name. this is a directory + skeleton.  
 - `bw environment create {address}`  
 - `bw agent` runs a bearded-wookie agent.  
 - `bw deploy production` deploy to the production environment  
 - `bw deploy --deployspace=".bw" production`  
 - `bw deploy filtered --name='node1' --name='us-east.*' production` to the servers that match the given filters, in this case up to 5 servers whose agents have the name `node1` or match the regex `us-east-.*`.  
 - `bw info {environment}` display information about the environment.  
