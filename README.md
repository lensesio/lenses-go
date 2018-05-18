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

### Documentation

Detailed documentation can be found at [godocs](https://godoc.org/github.com/landoop/lenses-go).

## Versioning

Current: **v2.0.7**

Read more about Semantic Versioning 2.0.0

 - http://semver.org/
 - https://en.wikipedia.org/wiki/Software_versioning
 - https://wiki.debian.org/UpstreamGuide#Releases_and_Versions

## License

Distributed under Apache Version 2.0 License, click [here](LICENSE) for more details.