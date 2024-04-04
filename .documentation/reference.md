# quick reference for bw

### variable substitutions (and their related environment variable)

```
%bw.deploy.id% BW_ENVIRONMENT_DEPLOY_ID - unique id for the deployment only available for remote directives
%bw.deploy.commit% BW_ENVIRONMENT_DEPLOY_COMMIT - the git commit of the repository.
%bw.work.directory% BW_ENVIRONMENT_WORK_DIRECTORY - working directory - usually the root directory of the git repository.
%bw.config.directory% BW_ENVIRONMENT_CONFIG_DIRECTORY - bw config directory - directory where the bw config yml file is defined.
%bw.archive.directory% BW_ENVIRONMENT_ARCHIVE_DIRECTORY - the root directory for the workspace `bw workspace create`.
%bw.cache.directory% BW_ENVIRONMENT_CACHE_DIRECTORY - long term cache directory data persists between deployments.
%bw.temp.directory% BW_ENVIRONMENT_TEMP_DIRECTORY - short term directory data persists for the duration of the deployment.
%bw.machine.hostname% BW_ENVIRONMENT_HOST - hostname of the current machine
%bw.machine.id% BW_ENVIRONMENT_MACHINE_ID - a unique id for the current machine
%bw.machine.domain% BW_ENVIRONMENT_DOMAIN - the domain of the current machine
%bw.machine.fqdn% BW_ENVIRONMENT_FQDN - the fully qualified domain name of the current machine
%bw.user.id% BW_ENVIRONMENT_USERID - id of the current user
%bw.user.name% BW_ENVIRONMENT_USERNAME - name of the current user
%bw.user.home.directory% BW_ENVIRONMENT_USERHOME - home directory of the current user.
```

