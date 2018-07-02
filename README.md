# Lenses Client (Go)

The [Landoop's](http://www.landoop.com) Lenses [REST API](https://lenses.stream/dev/lenses-apis/rest-api/index.html) client written in Go.

[![build status](https://img.shields.io/travis/Landoop/lenses-go/master.svg?style=flat-square)](https://travis-ci.org/Landoop/lenses-go) [![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=flat-square)](http://goreportcard.com/report/Landoop/lenses-go) [![chat](https://img.shields.io/badge/join-%20chat-00BCD4.svg?style=flat-square)](https://slackpass.io/landoop-community)

## Installation

The only requirement is the [Go Programming Language](https://golang.org/dl) version **1.10+** and a [Lenses Box](http://www.landoop.com/kafka-lenses/) of version **2.0 at least**.

```sh
$ go get -u github.com/landoop/lenses-go/cmd/lenses-cli
```

> This command will install both the client library for development usage and the CLI in $PATH ([setup your $GOPATH/bin](https://github.com/golang/go/wiki/SettingGOPATH) if you didn't already).

## CLI

Lenses offers a powerful CLI (command-line tool) built in Go that utilizes the REST and WebSocket APIs of Lenses, to communicate with Apache Kafka and exposes a straight forward way to perform common data engineering and site reliability engineering tasks, such as:

- Automate your CI/CD (continuous-integration and continuous-delivery)
- Create topics/acls/quotas/schemas/connectors/processors
- Change or retrieve configurations to store in github

### Documentation

Please navigate to <https://lenses.stream/dev/lenses-cli/> to learn how to install and use the `lenses-cli`.

## Client

The `lenses-go` package is made to be used by Go developers to communicate with Lenses by calling the REST and Websocket APIs. 

### Getting started

```go
import "github.com/landoop/lenses-go"
```

### Authentication

```go
// Prepare authentication using raw Username and Password.
auth := lenses.BasicAuthentication{Username: "user", Password: "pass"}
```

```go
auth := lenses.KerberosAuthentication{
    ConfFile: "/etc/krb5.conf",
    Method:   lenses.KerberosWithPassword{Realm: "my.realm or default if empty", Username: "user", Password: "pass"},
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

### Configuration

```go
// Prepare the client's configuration based on the host and the authentication above.
config := lenses.ClientConfiguration{Host: "domain.com", Authentication: auth, Timeout: "15s", Debug: true}

// Creating the client using the configuration.
client, err := lenses.OpenConnection(config) // or (config, lenses.UsingClient(customClient)/UsingToken(ready token string))
if err != nil {
    // handle error.
}
```

### API Call

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

Detailed documentation can be found at [godocs](https://godoc.org/github.com/landoop/lenses-go).

## Versioning

Current: **v2.1.0**

Read more about Semantic Versioning 2.0.0

 - http://semver.org/
 - https://en.wikipedia.org/wiki/Software_versioning
 - https://wiki.debian.org/UpstreamGuide#Releases_and_Versions

## License

Distributed under Apache Version 2.0 License, click [here](LICENSE) for more details.