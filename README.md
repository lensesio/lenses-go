# Lenses Client (Go)

The [Lenses](http://www.lenses.io) [REST API](https:/api.lenses.io) client written in Go.

[![Build Status](https://travis-ci.org/lensesio/lenses-go.svg?branch=master)](https://travis-ci.org/lensesio/lenses-go) [![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=flat-square)](http://goreportcard.com/report/lensesio/lenses-go) [![chat](https://img.shields.io/badge/join-%20chat-00BCD4.svg?style=flat-square)](https://slackpass.io/lensesio)

## Installation

The only requirement is the [Go Programming Language](https://golang.org/dl) version **1.13+** and a [Lenses Box](https://lenses.io/box/) of version **2.0 at least**.

```sh
# If you have Go < 1.13 you may need to set GO111MODULE=on
$ go get -u github.com/lensesio/lenses-go/cmd/lenses-cli
```

> This command will install both the client library for development usage and the CLI in $PATH ([setup your $GOPATH/bin](https://github.com/golang/go/wiki/SettingGOPATH) if you didn't already).

## CLI

Lenses offers a powerful CLI (command-line tool) built in Go that utilizes the REST and WebSocket APIs of Lenses, to communicate with Apache Kafka and exposes a straight forward way to perform common data engineering and site reliability engineering tasks, such as:

- Automate your CI/CD (continuous-integration and continuous-delivery)
- Create topics/acls/quotas/schemas/connectors/processors
- Change or retrieve configurations to store in github

### Documentation

Please navigate to <https://docs.lenses.io/dev/lenses-cli/> to learn how to install and use the `lenses-cli`.

### Development

#### Build

`lenses-go` use [go modules](https://github.com/golang/go/wiki/Modules) as dependency management system. For daily development workflow you need to use `Makefile` with certain actions.

Builds a binary based on your current OS system
```
make build
```

Builds binaries for all OS systems
```
make cross-build
```

#### Lint

We use the Golang [golint](https://github.com/golang/lint) for linting using:

```
make lint
```

#### Tests

If you want to run the tests, use the following:

```
make test
```

#### Clean

Clean all binaries and coverage files:

```
make clean
```

## Client

### Getting started

```go
import "github.com/lensesio/lenses-go"
```

### Authentication

```go
// Prepare authentication using raw Username and Password.
//
// Use it when Lenses setup with "BASIC" or "LDAP" authentication.
auth := lenses.BasicAuthentication{Username: "user", Password: "pass"}
```

```go
auth := lenses.KerberosAuthentication{
    ConfFile: "/etc/krb5.conf",
    Method:   lenses.KerberosWithPassword{
        Realm: "my.realm or default if empty",
        Username: "user",
        Password: "pass",
    },
}
```

```go
auth := lenses.KerberosAuthentication{
    ConfFile: "/etc/krb5.conf",
    Method:   lenses.KerberosWithKeytab{KeytabFile: "/home/me/krb5_my_keytab.txt"},
}
```

```go
auth := lenses.KerberosAuthentication{
    ConfFile: "/etc/krb5.conf",
    Method:   lenses.KerberosFromCCache{CCacheFile: "/tmp/krb5_my_cache_file.conf"},
}
```

> Custom auth can be implement as well: `Authenticate(client *lenses.Client) error`, see [client_authentication.go](client_authentication.go) file for more.

### Config

```go
// Prepare the client's configuration based on the host and the authentication above.
currentConfig := lenses.ClientConfig{Host: "domain.com", Authentication: auth, Timeout: "15s", Debug: true}

// Creating the client using the configuration.
client, err := lenses.OpenConnection(currentConfig)
if err != nil {
    // handle error.
}
```

#### Read `Config` from any `io.Reader` or file

```go
// ReadConfig reads and decodes Config from an io.Reader based on a custom unmarshaler.
// This can be useful to read configuration via network or files (see `ReadConfigFromFile`).
// Sets the `outPtr`. Retruns a non-nil error on any unmarshaler's errors.
ReadConfig(r io.Reader, unmarshaler UnmarshalFunc, outPtr *Config) error

// ReadConfigFromFile reads and decodes Config from a file based on a custom unmarshaler,
// `ReadConfigFromJSON` and `ReadConfigFromYAML` are the internal users,
// but the end-developer can use any custom type of decoder to read a configuration file
// with ease using this function, but keep note that the default behavior of the fields
// depend on the existing unmarshalers, use these tag names to map your decoder's properties.
//
// Accepts the absolute or the relative path of the configuration file.
// Sets the `outPtr`. Retruns a non-nil error if parsing or decoding the file failed or file doesn't exist.
ReadConfigFromFile(filename string, unmarshaler UnmarshalFunc, outPtr *Config) error

// TryReadConfigFromFile will try to read a specific file and unmarshal to `Config`.
// It will try to read it with one of these built'n formats:
// 1. JSON
// 2. YAML
TryReadConfigFromFile(filename string, outPtr *Config) error
```

```go
// TryReadConfigFromHome will try to read the `Config`
// from the current user's home directory/.lenses, the lookup is based on
// the common configuration filename pattern:
// lenses-cli.json, lenses-cli.yml or lenses.json and lenses.yml.
TryReadConfigFromHome(outPtr *Config) bool

// TryReadConfigFromExecutable will try to read the `Config`
// from the (client's caller's) executable path that started the current process.
// The lookup is based on the common configuration filename pattern:
// lenses-cli.json, lenses-cli.yml or lenses.json and lenses.yml.
TryReadConfigFromExecutable(outPtr *Config) bool

// TryReadConfigFromCurrentWorkingDir will try to read the `Config`
// from the current working directory, note that it may differs from the executable path.
// The lookup is based on the common configuration filename pattern:
// lenses-cli.json, lenses-cli.yml or lenses.json and lenses.yml.
TryReadConfigFromCurrentWorkingDir(outPtr *Config) bool

// ReadConfigFromJSON reads and decodes Config from a json file, i.e `configuration.json`.
//
// Accepts the absolute or the relative path of the configuration file.
// Error may occur when the file doesn't exists or is not formatted correctly.
ReadConfigFromJSON(filename string, outPtr *Config) error

// ReadConfigFromYAML reads and decodes Config from a yaml file, i.e `configuration.yml`.
//
// Accepts the absolute or the relative path of the configuration file.
// Error may occur when the file doesn't exists or is not formatted correctly.
ReadConfigFromYAML(filename string, outPtr *Config) error
```

**Example Code:**

```yaml
# file: ./lenses.yml
CurrentContext: main
Contexts:
  main:
    Host: https://<your-lenses-host-url>
    Kerberos:
      ConfFile: /etc/krb5.conf
      WithPassword:
        Username: the_username
        Password: the_password
        Realm: empty_for_default
```

**Usage:**

```go
var config lenses.Config
err := lenses.ReadConfigFromYAML("./lenses.yml", &config)
if err != nil {
    // handle error.
}

client, err := lenses.OpenConnection(*config.GetCurrent())
```

> `Config` contains tons of capabilities and helpers, you can quickly check them by navigating to the [config.go](config.go) source file.

### API Calls

All `lenses-go#Client` methods return a typed value based on the call
and an error as second output to catch any errors coming from backend or client, forget panics.

**Go types are first class citizens here**, we will not confuse you or let you work based on luck!

```go
topics, err := client.GetTopics()
if err != nil {
    // handle error.
}

// Print the length of the topics we've just received from our Lenses Box.
print(len(topics))
```

Example on how deeply we make the difference here:
`Client#GetTopics` returns `[]lenses.Topic`, so you can work safely.

```go
topics[0].ConsumersGroup[0].Coordinator.Host
```

### Documentation

Detailed documentation can be found at [godocs](https://godoc.org/github.com/lensesio/lenses-go).

## Versioning

 - http://semver.org/
 - https://en.wikipedia.org/wiki/Software_versioning
 - https://wiki.debian.org/UpstreamGuide#Releases_and_Versions

## License

Distributed under Apache Version 2.0 License, click [here](LICENSE) for more details.
