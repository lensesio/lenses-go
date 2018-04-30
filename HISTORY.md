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
  "host": "http://localhost:9991",
  "user": "validName",
  "password": "validPassEncrypted",
  "token": ""
}
dev2test [invalid]
{
  "host": "http://localhost:9991",
  "user": "invalidName2",
  "password": "invalidPassEncrypted",
  "token": ""
}
devtest [invalid]
{
  "host": "http://localhost:9991",
  "user": "invalidName",
  "password": "invalidPassEncrypted",
  "token": ""
}
? Would you like to skip, edit or delete the 'dev2test' invalid configuration context? **skip**
? Would you like to skip, edit or delete the 'devtest' invalid configuration context? **edit**
? Host http://localhost:9991
? User kataras
? Password [? for help] **********
? Enable debug mode? No
'devtest' saved
```

> Other new commands except the `contexts` are the: `context delete $context_name` and `context update(or edit) $context_name` which is the same as `configure --context=$context_name --reset --no-banner --default-location`.

##### Compatibility:

If old configuration exists then it will be loaded and used as usual but at the end it will modify the configuration file to match the new style, users are not forced to touch the configuration file, this happens automatic.

### Client