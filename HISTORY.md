# Changelog

## v2.1

### CLI

- `--clusterAlias` renamed to `--clusterName` for *flags* and *yaml* files. Both `processor` and `connector` commands are using the term `clusterName` now.

#### Credentials & configuration

Configuration structure changed in order to accept more than one users/environments/contexts, read below.

##### First time, configuration does not exist:

`lenses-cli $command` will throw an error if configuration is missing and will prompt for credentials, the default environment name (`CurrentContext`) is `"master"`, same as standalone `configure` command for first time.

##### Next times, configuration and at least one valid context key exists:

`lenses-cli --context=doesnotexist` will throw `unknown context 'doesnotexist' given, please use the "configure --context=doesnotexist --reset"`. 

The command `configure --context=doesnotexist --reset` will prompt for credentials, add a new context key mapped with these credentials, change the `CurrentContext` to the new `--context` and save the configuration. 

`lenses-cli $command` will select the credentials from the configuration's `CurrentContext`, which defaults to `master` unless changed by a previous command using the `--context=value` flag.

`lenses-cli --context=dev $command` will select the credentials from the `"dev`" context key and change the configuration file's `CurrentContext` value to `"dev"` if differs.

The **new** command `lenses-cli contexts` will print the contexts and validate (through calls to the backend) all the available contexts that exist in the configuration file, example output:

```sh
master [valid]
{
  "host": "http://master.example.com:9991",
  "user": "kataras",
  "password": "kataras999",
  "token": ""
}
dev [invalid]
{
  "host": "http://dev.example.com:9991",
  "user": "invalid_username",
  "password": "e\ufffda\ufffd\ufffdY\ufffd\ufffd\u0000F",
  "token": ""
}
```

##### Compatibility:

If old configuration exists then it will be loaded and used as usual but at the end it will modify the configuration file to match the new style, users are not forced to touch the configuration file, this happens automatic.

### Client