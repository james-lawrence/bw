### Bearded Wookie is built around four concepts.
each concept builds on the other to handle configuration management.
these concepts are used by the bearded wookie toolset to configure
and deploy onto different clusters of machines. they are a way of thinking about deployments separate from the tools that manage the deployment.

#### goals:
- small, fast, robust, cluster configuration mamagement tool.
- support different deployment strategies.
- no centralized server.

#### what this project is not.
- bearded wookie is not about infrastructure management. its about configuration management.

#### credentials
bearded-wookie uses SSL/TLS to encrypt all data transfered between the agent and the cluster.

#### workspace
workspaces are the top level namespace that describes a deployment.
generally projects only have a single workspace. But multiple workspaces may be
desirable for example: a separate workspace for local development configuration
from deployed environments, or for different distros.

#### environments
generally environments are used to configure the different environments
you deploy into. such as production vs staging. Environments are represented
by .env files that contain environment variables used to configure the deployment.

#### filesystems
filesystems represent the different groups files you want deployed
onto a machine. Things like installing configuration files and creating
directories are filesystem operations.

#### directives
directives are the final piece of a deployment, they represent the steps
and the order of the steps to take to deploy an application. directives
come in three types package installation (.bwpkg), commands (.bwcmd), file installations (.bwfs).

example commands:
 - `bw init {common-name} {hosts...}` generates ssl/tls certificates for use with bw.
 - `bw workspace create {name}`
 - `bw agent` runs a bearded-wookie agent.
 - `bw deploy production` deploy to the production environment
 - `bw deploy --workspace=".bearded-wookie-deployment" production`
 - `bw filtered deploy --name='node1' --name='us-east.*' production` to the servers that match the given filters, in this case up to 5 servers whose agents have the name `node1` or match the regex `us-east-.*`.
 - `bw info production` display information about the production environment.

#### TODO
 - implement directory cleanup of previous deployments.
 - implement info.
 - implement members.
 - implement server configuration file.
