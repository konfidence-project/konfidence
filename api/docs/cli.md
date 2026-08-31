## kden

Kden CLI tool for working with Konfidence

### Synopsis

Kden is an extensible command-line interface tool with Go & Cobra.

Kden CLI supports developers, DevOps engineers, and release managers working with Konfidence -
a cloud-native continuous delivery and application lifecycle management
framework built on Kubernetes.

Example usage:
  kden version
  kden help

### Options

```
      --api-endpoint string      Address of the Konfidence API gateway. Env: KDEN_API_ENDPOINT (default: http://localhost:8090)
  -h, --help                     help for kden
      --log-format string        Defines the output format of the application's logs . Supported values are: 'json', 'text' and 'pretty'
      --log-level string         Defines the base log level for the application. Supported values are: 'info', 'debug' and 'error'
      --login-timeout string     Maximum time to wait for browser login. Env: KDEN_LOGIN_TIMEOUT (default: 2m)
      --output string            Defines the output format for the application. Supported values are: 'json', 'yaml' and 'pretty'
      --request-timeout string   Maximum duration for an API request. Env: KDEN_REQUEST_TIMEOUT (default: 30s)
```

### SEE ALSO

* [kden artifact](#kden-artifact)  - Manage artifacts
* [kden completion](#kden-completion)  - Generate shell completion scripts
* [kden config](#kden-config)  - Manage the CLI's configuration
* [kden login](#kden-login)  - Kden API Login
* [kden logout](#kden-logout)  - Kden API Logout
* [kden project](#kden-project)  - Manage projects
* [kden vector](#kden-vector)  - Manage vectors
* [kden version](#kden-version)  - Print the kden CLI version


## kden artifact

Manage artifacts

```
kden artifact [flags]
```

### Options

```
  -h, --help   help for artifact
```

### Options inherited from parent commands

```
      --api-endpoint string      Address of the Konfidence API gateway. Env: KDEN_API_ENDPOINT (default: http://localhost:8090)
      --log-format string        Defines the output format of the application's logs . Supported values are: 'json', 'text' and 'pretty'
      --log-level string         Defines the base log level for the application. Supported values are: 'info', 'debug' and 'error'
      --login-timeout string     Maximum time to wait for browser login. Env: KDEN_LOGIN_TIMEOUT (default: 2m)
      --output string            Defines the output format for the application. Supported values are: 'json', 'yaml' and 'pretty'
      --request-timeout string   Maximum duration for an API request. Env: KDEN_REQUEST_TIMEOUT (default: 30s)
```

### SEE ALSO

* [kden](#kden)  - Kden CLI tool for working with Konfidence
* [kden artifact alias](#kden-artifact-alias)  - Create or update a mutable alias tag for a component version
* [kden artifact push](#kden-artifact-push)  - Push a component version to a registry
* [kden artifact sign](#kden-artifact-sign)  - Sign artifact
* [kden artifact validate](#kden-artifact-validate)  - Validate artifacts against predefined JSON schema.


## kden artifact alias

Create or update a mutable alias tag for a component version

### Synopsis

Create or update a mutable alias tag pointing to an existing component version.

    kden artifact alias registry.example.com//konfidence.io/payment-hub:1.0.0 edge

```
kden artifact alias <source-ref> <alias> [flags]
```

### Options

```
  -h, --help   help for alias
```

### Options inherited from parent commands

```
      --api-endpoint string      Address of the Konfidence API gateway. Env: KDEN_API_ENDPOINT (default: http://localhost:8090)
      --log-format string        Defines the output format of the application's logs . Supported values are: 'json', 'text' and 'pretty'
      --log-level string         Defines the base log level for the application. Supported values are: 'info', 'debug' and 'error'
      --login-timeout string     Maximum time to wait for browser login. Env: KDEN_LOGIN_TIMEOUT (default: 2m)
      --output string            Defines the output format for the application. Supported values are: 'json', 'yaml' and 'pretty'
      --request-timeout string   Maximum duration for an API request. Env: KDEN_REQUEST_TIMEOUT (default: 30s)
```

### SEE ALSO

* [kden artifact](#kden-artifact)  - Manage artifacts


## kden artifact push

Push a component version to a registry

```
kden artifact push [flags]
```

### Options

```
  -f, --file string       --file=<path>
  -h, --help              help for push
  -r, --registry string   --registry=docker.io/<subpath>
```

### Options inherited from parent commands

```
      --api-endpoint string      Address of the Konfidence API gateway. Env: KDEN_API_ENDPOINT (default: http://localhost:8090)
      --log-format string        Defines the output format of the application's logs . Supported values are: 'json', 'text' and 'pretty'
      --log-level string         Defines the base log level for the application. Supported values are: 'info', 'debug' and 'error'
      --login-timeout string     Maximum time to wait for browser login. Env: KDEN_LOGIN_TIMEOUT (default: 2m)
      --output string            Defines the output format for the application. Supported values are: 'json', 'yaml' and 'pretty'
      --request-timeout string   Maximum duration for an API request. Env: KDEN_REQUEST_TIMEOUT (default: 30s)
```

### SEE ALSO

* [kden artifact](#kden-artifact)  - Manage artifacts


## kden artifact sign

Sign artifact

### Synopsis

Sign an artifact with a signer specification.

```
kden artifact sign [flags]
```

### Options

```
      --dry-run                          If enabled, the signature will not be persisted
      --hash-algorithm string            Hash algorithm to use. Supported values are: SHA-512, SHA-256 (default "SHA-256")
  -h, --help                             help for sign
      --normalization-algorithm string   Normalization algorithm to use (default "jsonNormalisation/v4alpha1")
      --overwrite-signatures             Overwrite if a signature with the same name exists
      --signature-name string            Name of the signature to use (default "default")
      --signer-spec string               Path to signer specification file
```

### Options inherited from parent commands

```
      --api-endpoint string      Address of the Konfidence API gateway. Env: KDEN_API_ENDPOINT (default: http://localhost:8090)
      --log-format string        Defines the output format of the application's logs . Supported values are: 'json', 'text' and 'pretty'
      --log-level string         Defines the base log level for the application. Supported values are: 'info', 'debug' and 'error'
      --login-timeout string     Maximum time to wait for browser login. Env: KDEN_LOGIN_TIMEOUT (default: 2m)
      --output string            Defines the output format for the application. Supported values are: 'json', 'yaml' and 'pretty'
      --request-timeout string   Maximum duration for an API request. Env: KDEN_REQUEST_TIMEOUT (default: 30s)
```

### SEE ALSO

* [kden artifact](#kden-artifact)  - Manage artifacts


## kden artifact validate

Validate artifacts against predefined JSON schema.

```
kden artifact validate [flags]
```

### Options

```
      --files strings   comma-separated list of artifact path files to validate
  -h, --help            help for validate
```

### Options inherited from parent commands

```
      --api-endpoint string      Address of the Konfidence API gateway. Env: KDEN_API_ENDPOINT (default: http://localhost:8090)
      --log-format string        Defines the output format of the application's logs . Supported values are: 'json', 'text' and 'pretty'
      --log-level string         Defines the base log level for the application. Supported values are: 'info', 'debug' and 'error'
      --login-timeout string     Maximum time to wait for browser login. Env: KDEN_LOGIN_TIMEOUT (default: 2m)
      --output string            Defines the output format for the application. Supported values are: 'json', 'yaml' and 'pretty'
      --request-timeout string   Maximum duration for an API request. Env: KDEN_REQUEST_TIMEOUT (default: 30s)
```

### SEE ALSO

* [kden artifact](#kden-artifact)  - Manage artifacts


## kden completion

Generate shell completion scripts

### Synopsis

Generate shell completion scripts for kden.

To load completions:

Bash:

    source <(kden completion bash)

    # To load completions for each session, execute once:
    # Linux:
    kden completion bash > /etc/bash_completion.d/kden
    # macOS:
    kden completion bash > $(brew --prefix)/etc/bash_completion.d/kden

Zsh:

    # If shell completion is not already enabled, enable it once:
    echo "autoload -U compinit; compinit" >> ~/.zshrc

    # To load completions for each session, execute once:
    kden completion zsh > "${fpath[1]}/_kden"

    # Start a new shell for this setup to take effect.

Fish:

    kden completion fish | source

    # To load completions for each session, execute once:
    kden completion fish > ~/.config/fish/completions/kden.fish

PowerShell:

    kden completion powershell | Out-String | Invoke-Expression

    # To load completions for every new session, run:
    kden completion powershell > kden.ps1
    # and source this file from your PowerShell profile.


```
kden completion [bash|zsh|fish|powershell]
```

### Options

```
  -h, --help   help for completion
```

### Options inherited from parent commands

```
      --api-endpoint string      Address of the Konfidence API gateway. Env: KDEN_API_ENDPOINT (default: http://localhost:8090)
      --log-format string        Defines the output format of the application's logs . Supported values are: 'json', 'text' and 'pretty'
      --log-level string         Defines the base log level for the application. Supported values are: 'info', 'debug' and 'error'
      --login-timeout string     Maximum time to wait for browser login. Env: KDEN_LOGIN_TIMEOUT (default: 2m)
      --output string            Defines the output format for the application. Supported values are: 'json', 'yaml' and 'pretty'
      --request-timeout string   Maximum duration for an API request. Env: KDEN_REQUEST_TIMEOUT (default: 30s)
```

### SEE ALSO

* [kden](#kden)  - Kden CLI tool for working with Konfidence


## kden config

Manage the CLI's configuration

### Synopsis


The CLI uses a configuration file to store settings and preferences. 

The file is stored in the $XDG_CONFIG_HOME/kden/config.json directory. 
The value of $XDG_CONFIG_HOME depends on the operating system. 
For more information, check the XDG Base Directory documentation: https://specifications.freedesktop.org/basedir/latest/.

```
kden config [flags]
```

### Options

```
  -h, --help   help for config
```

### Options inherited from parent commands

```
      --api-endpoint string      Address of the Konfidence API gateway. Env: KDEN_API_ENDPOINT (default: http://localhost:8090)
      --log-format string        Defines the output format of the application's logs . Supported values are: 'json', 'text' and 'pretty'
      --log-level string         Defines the base log level for the application. Supported values are: 'info', 'debug' and 'error'
      --login-timeout string     Maximum time to wait for browser login. Env: KDEN_LOGIN_TIMEOUT (default: 2m)
      --output string            Defines the output format for the application. Supported values are: 'json', 'yaml' and 'pretty'
      --request-timeout string   Maximum duration for an API request. Env: KDEN_REQUEST_TIMEOUT (default: 30s)
```

### SEE ALSO

* [kden](#kden)  - Kden CLI tool for working with Konfidence
* [kden config set](#kden-config-set)  - Set a value for a configuration property.
* [kden config unset](#kden-config-unset)  - Unset a configuration property.


## kden config set

Set a value for a configuration property.

### Synopsis

This command sets a value for a configuration property inside the CLI's configuration file.
The accepted values are:

    'api-endpoint' - URL (e.g. https://api.example.com)
    'log-format' - [json text pretty]
    'log-level' - [debug info error]
    'login-timeout' - duration (e.g. 2m, 30s)
    'output' - [json yaml pretty]
    'request-timeout' - duration (e.g. 30s, 1m)

Additional information:

The CLI uses a configuration file to store settings and preferences. 

The file is stored in the $XDG_CONFIG_HOME/kden/config.json directory. 
The value of $XDG_CONFIG_HOME depends on the operating system. 
For more information, check the XDG Base Directory documentation: https://specifications.freedesktop.org/basedir/latest/.

```
kden config set <configuration_property> <value> [flags]
```

### Options

```
  -h, --help   help for set
```

### Options inherited from parent commands

```
      --api-endpoint string      Address of the Konfidence API gateway. Env: KDEN_API_ENDPOINT (default: http://localhost:8090)
      --log-format string        Defines the output format of the application's logs . Supported values are: 'json', 'text' and 'pretty'
      --log-level string         Defines the base log level for the application. Supported values are: 'info', 'debug' and 'error'
      --login-timeout string     Maximum time to wait for browser login. Env: KDEN_LOGIN_TIMEOUT (default: 2m)
      --output string            Defines the output format for the application. Supported values are: 'json', 'yaml' and 'pretty'
      --request-timeout string   Maximum duration for an API request. Env: KDEN_REQUEST_TIMEOUT (default: 30s)
```

### SEE ALSO

* [kden config](#kden-config)  - Manage the CLI's configuration


## kden config unset

Unset a configuration property.

### Synopsis

This command unsets a configuration property inside the CLI's configuration file.
The accepted values are: [api-endpoint log-format log-level login-timeout output request-timeout]

Additional information:

The CLI uses a configuration file to store settings and preferences. 

The file is stored in the $XDG_CONFIG_HOME/kden/config.json directory. 
The value of $XDG_CONFIG_HOME depends on the operating system. 
For more information, check the XDG Base Directory documentation: https://specifications.freedesktop.org/basedir/latest/.

```
kden config unset <configuration_property> [flags]
```

### Options

```
  -h, --help   help for unset
```

### Options inherited from parent commands

```
      --api-endpoint string      Address of the Konfidence API gateway. Env: KDEN_API_ENDPOINT (default: http://localhost:8090)
      --log-format string        Defines the output format of the application's logs . Supported values are: 'json', 'text' and 'pretty'
      --log-level string         Defines the base log level for the application. Supported values are: 'info', 'debug' and 'error'
      --login-timeout string     Maximum time to wait for browser login. Env: KDEN_LOGIN_TIMEOUT (default: 2m)
      --output string            Defines the output format for the application. Supported values are: 'json', 'yaml' and 'pretty'
      --request-timeout string   Maximum duration for an API request. Env: KDEN_REQUEST_TIMEOUT (default: 30s)
```

### SEE ALSO

* [kden config](#kden-config)  - Manage the CLI's configuration


## kden login

Kden API Login

### Synopsis

Start the login process for the Kden API.

```
kden login [flags]
```

### Options

```
  -h, --help   help for login
```

### Options inherited from parent commands

```
      --api-endpoint string      Address of the Konfidence API gateway. Env: KDEN_API_ENDPOINT (default: http://localhost:8090)
      --log-format string        Defines the output format of the application's logs . Supported values are: 'json', 'text' and 'pretty'
      --log-level string         Defines the base log level for the application. Supported values are: 'info', 'debug' and 'error'
      --login-timeout string     Maximum time to wait for browser login. Env: KDEN_LOGIN_TIMEOUT (default: 2m)
      --output string            Defines the output format for the application. Supported values are: 'json', 'yaml' and 'pretty'
      --request-timeout string   Maximum duration for an API request. Env: KDEN_REQUEST_TIMEOUT (default: 30s)
```

### SEE ALSO

* [kden](#kden)  - Kden CLI tool for working with Konfidence


## kden logout

Kden API Logout

### Synopsis

Logs the current user out of the Kden API.

```
kden logout [flags]
```

### Options

```
  -h, --help   help for logout
```

### Options inherited from parent commands

```
      --api-endpoint string      Address of the Konfidence API gateway. Env: KDEN_API_ENDPOINT (default: http://localhost:8090)
      --log-format string        Defines the output format of the application's logs . Supported values are: 'json', 'text' and 'pretty'
      --log-level string         Defines the base log level for the application. Supported values are: 'info', 'debug' and 'error'
      --login-timeout string     Maximum time to wait for browser login. Env: KDEN_LOGIN_TIMEOUT (default: 2m)
      --output string            Defines the output format for the application. Supported values are: 'json', 'yaml' and 'pretty'
      --request-timeout string   Maximum duration for an API request. Env: KDEN_REQUEST_TIMEOUT (default: 30s)
```

### SEE ALSO

* [kden](#kden)  - Kden CLI tool for working with Konfidence


## kden project

Manage projects

```
kden project [flags]
```

### Options

```
  -h, --help   help for project
```

### Options inherited from parent commands

```
      --api-endpoint string      Address of the Konfidence API gateway. Env: KDEN_API_ENDPOINT (default: http://localhost:8090)
      --log-format string        Defines the output format of the application's logs . Supported values are: 'json', 'text' and 'pretty'
      --log-level string         Defines the base log level for the application. Supported values are: 'info', 'debug' and 'error'
      --login-timeout string     Maximum time to wait for browser login. Env: KDEN_LOGIN_TIMEOUT (default: 2m)
      --output string            Defines the output format for the application. Supported values are: 'json', 'yaml' and 'pretty'
      --request-timeout string   Maximum duration for an API request. Env: KDEN_REQUEST_TIMEOUT (default: 30s)
```

### SEE ALSO

* [kden](#kden)  - Kden CLI tool for working with Konfidence
* [kden project list](#kden-project-list)  - List all projects


## kden project list

List all projects

### Synopsis

Retrieve and display a complete list of all projects.

```
kden project list [flags]
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
      --api-endpoint string      Address of the Konfidence API gateway. Env: KDEN_API_ENDPOINT (default: http://localhost:8090)
      --log-format string        Defines the output format of the application's logs . Supported values are: 'json', 'text' and 'pretty'
      --log-level string         Defines the base log level for the application. Supported values are: 'info', 'debug' and 'error'
      --login-timeout string     Maximum time to wait for browser login. Env: KDEN_LOGIN_TIMEOUT (default: 2m)
      --output string            Defines the output format for the application. Supported values are: 'json', 'yaml' and 'pretty'
      --request-timeout string   Maximum duration for an API request. Env: KDEN_REQUEST_TIMEOUT (default: 30s)
```

### SEE ALSO

* [kden project](#kden-project)  - Manage projects


## kden vector

Manage vectors

```
kden vector [flags]
```

### Options

```
  -h, --help   help for vector
```

### Options inherited from parent commands

```
      --api-endpoint string      Address of the Konfidence API gateway. Env: KDEN_API_ENDPOINT (default: http://localhost:8090)
      --log-format string        Defines the output format of the application's logs . Supported values are: 'json', 'text' and 'pretty'
      --log-level string         Defines the base log level for the application. Supported values are: 'info', 'debug' and 'error'
      --login-timeout string     Maximum time to wait for browser login. Env: KDEN_LOGIN_TIMEOUT (default: 2m)
      --output string            Defines the output format for the application. Supported values are: 'json', 'yaml' and 'pretty'
      --request-timeout string   Maximum duration for an API request. Env: KDEN_REQUEST_TIMEOUT (default: 30s)
```

### SEE ALSO

* [kden](#kden)  - Kden CLI tool for working with Konfidence
* [kden vector push](#kden-vector-push)  - 
* [kden vector sign](#kden-vector-sign)  - Sign vector
* [kden vector validate](#kden-vector-validate)  - Validate a vector against a predefined JSON schema.


## kden vector push



```
kden vector push [flags]
```

### Options

```
  -a, --alias string      --alias=<alias-name>
  -f, --file string       --file=<path>
  -h, --help              help for push
  -r, --registry string   --registry=docker.io/<subpath>
```

### Options inherited from parent commands

```
      --api-endpoint string      Address of the Konfidence API gateway. Env: KDEN_API_ENDPOINT (default: http://localhost:8090)
      --log-format string        Defines the output format of the application's logs . Supported values are: 'json', 'text' and 'pretty'
      --log-level string         Defines the base log level for the application. Supported values are: 'info', 'debug' and 'error'
      --login-timeout string     Maximum time to wait for browser login. Env: KDEN_LOGIN_TIMEOUT (default: 2m)
      --output string            Defines the output format for the application. Supported values are: 'json', 'yaml' and 'pretty'
      --request-timeout string   Maximum duration for an API request. Env: KDEN_REQUEST_TIMEOUT (default: 30s)
```

### SEE ALSO

* [kden vector](#kden-vector)  - Manage vectors


## kden vector sign

Sign vector

### Synopsis

Sign a vector with a signer specification.

```
kden vector sign [flags]
```

### Options

```
      --dry-run                          If enabled, the signature will not be persisted
      --hash-algorithm string            Hash algorithm to use. Supported values are: SHA-512, SHA-256 (default "SHA-256")
  -h, --help                             help for sign
      --normalization-algorithm string   Normalization algorithm to use (default "jsonNormalisation/v4alpha1")
      --overwrite-signatures             Overwrite if a signature with the same name exists
      --signature-name string            Name of the signature to use (default "default")
      --signer-spec string               Path to signer specification file
```

### Options inherited from parent commands

```
      --api-endpoint string      Address of the Konfidence API gateway. Env: KDEN_API_ENDPOINT (default: http://localhost:8090)
      --log-format string        Defines the output format of the application's logs . Supported values are: 'json', 'text' and 'pretty'
      --log-level string         Defines the base log level for the application. Supported values are: 'info', 'debug' and 'error'
      --login-timeout string     Maximum time to wait for browser login. Env: KDEN_LOGIN_TIMEOUT (default: 2m)
      --output string            Defines the output format for the application. Supported values are: 'json', 'yaml' and 'pretty'
      --request-timeout string   Maximum duration for an API request. Env: KDEN_REQUEST_TIMEOUT (default: 30s)
```

### SEE ALSO

* [kden vector](#kden-vector)  - Manage vectors


## kden vector validate

Validate a vector against a predefined JSON schema.

```
kden vector validate [flags]
```

### Options

```
      --files strings   comma-separated list of vector path files to validate
  -h, --help            help for validate
```

### Options inherited from parent commands

```
      --api-endpoint string      Address of the Konfidence API gateway. Env: KDEN_API_ENDPOINT (default: http://localhost:8090)
      --log-format string        Defines the output format of the application's logs . Supported values are: 'json', 'text' and 'pretty'
      --log-level string         Defines the base log level for the application. Supported values are: 'info', 'debug' and 'error'
      --login-timeout string     Maximum time to wait for browser login. Env: KDEN_LOGIN_TIMEOUT (default: 2m)
      --output string            Defines the output format for the application. Supported values are: 'json', 'yaml' and 'pretty'
      --request-timeout string   Maximum duration for an API request. Env: KDEN_REQUEST_TIMEOUT (default: 30s)
```

### SEE ALSO

* [kden vector](#kden-vector)  - Manage vectors


## kden version

Print the kden CLI version

```
kden version [flags]
```

### Options

```
  -h, --help   help for version
```

### Options inherited from parent commands

```
      --api-endpoint string      Address of the Konfidence API gateway. Env: KDEN_API_ENDPOINT (default: http://localhost:8090)
      --log-format string        Defines the output format of the application's logs . Supported values are: 'json', 'text' and 'pretty'
      --log-level string         Defines the base log level for the application. Supported values are: 'info', 'debug' and 'error'
      --login-timeout string     Maximum time to wait for browser login. Env: KDEN_LOGIN_TIMEOUT (default: 2m)
      --output string            Defines the output format for the application. Supported values are: 'json', 'yaml' and 'pretty'
      --request-timeout string   Maximum duration for an API request. Env: KDEN_REQUEST_TIMEOUT (default: 30s)
```

### SEE ALSO

* [kden](#kden)  - Kden CLI tool for working with Konfidence


