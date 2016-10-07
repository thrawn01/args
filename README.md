[![Coverage Status](https://img.shields.io/coveralls/thrawn01/args.svg)](https://coveralls.io/github/thrawn01/args)
[![Build Status](https://img.shields.io/travis/thrawn01/args/master.svg)](https://travis-ci.org/thrawn01/args)
[![Code Climate](https://codeclimate.com/github/thrawn01/args/badges/gpa.svg)](https://codeclimate.com/github/thrawn01/args)

**NOTE: This is alpha software, the api will continue to evolve until the 1.0 release**

## Introduction
A cloud native app configuration and argument parser designed for
 use in micro services with support for CLI clients and live server
 configuration reloading

## Configuration Philosophy
Configuration for modern cloud native applications is divided into two separate
phases. The first phase is initialization, This is the minimum configuration
the service needs to start up. At a minimum this may include configuration for:
Service Interface, Port Number, TLS Certs, and the source location of the
second phase configuration.

The second phase includes everything else the service needs to operate.
Including: Monitoring, Logging, Databases, etc.... Everything in the First
Phase should be mostly static items; things that can not change unless the
service is restarted. Everything in the second phase can and should change at
anytime during the operation of the service without the need for the service to
restart.

Args as a project has the following goals

1. Config option access should be thread safe. This allows new requests to a
   service to retrieve the newest version of the config on a per request basis.
2. Backend config watchers. Args should provide an interface to plugin key/value stores
   such as etcd and zookeeper with helper functions designed to make updating
   your local config super simple.
3. Built in JSON-RPC http handler. Some applications can't utilize external
   config systems like kubernetes or etcd for second phase configuration. Args
   should provide a JSON-RPC interface you can expose on a http route of your
   choosing which will allow admins to modify the configuration while the
   service is running. This makes your application cloud native without a hard
   requirement on external systems.
4. Easy to use subcommand support. I have years of experience using and writing
   subcommand like tools. Take my lessions learned and apply them here. [See
   subcommand.org](http://subcommand.org)
5. The relationship between, Environment, Commandline, and Config should NOT
   result in impedance mismatch. If a feature would result in an imedance
   mismatch the feature will be rejected. This results in args being somewhat
   opinionated. However, for the operators sake, cloud native applications
   should be opinionated and intuitive in how they are configured and operated.


***NOTE: Some of these features are still Work In Progress, and the API will
evolve until we have a full 1.0 release***

## Installation
```
go get github.com/thrawn01/args
```

## Development Guide
Args uses can use glide to ensure the proper dependencies are installed, but args
should compile without it.

Fetch the source
```
go get -d github.com/thrawn01/args
cd $GOPATH/src/github.com/thrawn01/args
```
Install glide and fetch the dependencies via glide
```
make get-deps
```
Run make to build the example and run the tests
```
make
```
## Thread safe option access
Options can be retrieved in a thread safe manner by calling ```GetOpts()```.
Using ```GetOpts()``` and ```SetOpts()``` allows the user to control when new
configs are applied to the service.

```go
parser := args.NewParser()
parser.AddConfig("some-key").Alias("-k").Default("default-key").
    Help("A fake api-key")

// Parses the commandline, calls os.Exit(1) if there is an error
opts := parser.ParseArgsSimple(nil)

// Simple handler that returns some-key
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    // GetOpts is a thread safe way to get the current options
    conf := parser.GetOpts()

    // Marshal the response to json
    payload, err := json.Marshal(map[string]interface{}{
        "some-key": conf.String("some-key"),
    })
    // Write the response to the user
    w.Header().Set("Content-Type", "application/json")
    w.Write(payload)
})
```

## Watch key store backends for config changes
Args supports additional backend configuration via any backend that implements the
 ```Backend``` interface. Currently only etcd is supported and is provided by the
  [args-backend](http://github.com/thrawn01/args-backends) repo.

```go
// -- snip ---
client, err := etcdv3.New(etcd.Config{
    Endpoints:   opts.StringSlice("etcd-endpoints"),
    DialTimeout: 5 * time.Second,
})
if err != nil {
    log.Fatal(err)
}

etcdBackend := backends.NewEtcdBackend(client, "/etcd-endpoints-service")

// Read all the available config values from etcd
opts, err = parser.FromBackend(etcdBackend)
if err != nil {
    fmt.Printf("Etcd error - %s\n", err.Error())
}

// Watch etcd for any configuration changes (This starts a go routine)
cancelWatch := etcdBackend.Watch(client, func(event args.ChangeEvent, err error) {
    if err != nil {
        fmt.Println(err.Error())
        return
    }
    fmt.Printf("Change Event - %+v\n", event)
    // This takes a ChangeEvent and updates the opts with the latest changes
    parser.Apply(opts.FromChangeEvent(event))
})
// -- snip ---
```

## JSON-RPC Handler
The JSON-RPC handler will provide access to the config via http using JSON-RPC (Feature is still a WIP)
```go
parser := args.NewParser(args.Desc("Demo Service showcasing args JSON-RPC interface"))
parser.AddConfig("simple").Help("Demo of simple config item")
opt := parser.ParseArgsSimple(nil)

// Simple Application that just displays our current config
http.HandleFunc("/my-app", func(w http.ResponseWriter, r *http.Request) {
    conf := parser.GetOpts()

    payload, err := json.Marshal(map[string]string{
        "simple": conf.String("simple"),
    })
    w.Header().Set("Content-Type", "application/json")
    w.Write(payload)
})

// Allow admin users to change our config remotely via the JSON-RPC handler
// This path should be protected by authentication middleware.
http.HandleFunc("/config", parser.JsonRPCHandler)

fmt.Printf("Listening on %s...\n", opt.String("bind"))
log.Fatal(http.ListenAndServe(opt.String("bind"), nil))
```

## Command Support
The following code creates a command, and sub-command such that usage works like this
```
$ my-cli show
Show things here

$ my-cli volume create my-new-volume
Volume 'my-new-volume' created

```

```go
func main() {
    parser := args.NewParser(args.Name("subcommand"),
        args.Desc("Example subcommand CLI"))

    // Add a subcommand
    parser.AddCommand("show", show)

    // Add a sub-sub-command
    parser.AddCommand("volume", func(subParser *args.ArgParser, data interface{}) int {
        subParser.AddCommand("create", createVolume)

        // Run the sub-commands
        retCode, err := subParser.ParseAndRun(nil, data)
        if err != nil {
            fmt.Println(err.Error())
            return 1
        }
        return retCode
    })

    // Run the command chosen by the user
    retCode, err := parser.ParseAndRun(nil, nil)
    if err != nil {
        fmt.Fprintln(os.Stderr, err.Error())
        os.Exit(1)
    }
    os.Exit(retCode)
}
func show(subParser *args.ArgParser, data interface{}) int {
    fmt.Printf("Show things here\n")
    return 0
}

func createVolume(subParser *args.ArgParser, data interface{}) int {
    subParser.AddArgument("name").Required().Help("The name of the volume to create")
    opts, err := subParser.ParseArgs(nil)
    if err != nil {
        fmt.Println(err.Error())
        return 1
    }

    // Create our volume
    if err := volume.Create(opts.String("name")); err != nil {
        fmt.Fprintln(os.Stderr, err)
        return 1
    }
    fmt.Printf("Volume '%s' created\n")
    return 0
}
```

## Watch Config with hot reload
Args can reload your config when modifications are made to a watched config file. **This works well
with Kubernetes ConfigMap**
```go
    parser := args.NewParser()
    parser.AddOption("--config").Alias("-c").Help("Read options from a config file")
    opt := parser.ParseArgsSimple(nil)
    configFile := opt.String("config")
    // Initial load of our config file
    opt, err = parser.FromIniFile(configFile)

    // check our config file for changes every second
    cancelWatch, err := args.WatchFile(configFile, time.Second, func(err error) {
        if err != nil {
            fmt.Printf("Error Watching %s - %s", configFile, err.Error())
            return
        }

        fmt.Println("Config file changed, Reloading...")
        opt, err = parser.FromIniFile(configFile)
        if err != nil {
            fmt.Printf("Failed to load config - %s\n", err.Error())
            return
        }
    })
    if err != nil {
        fmt.Printf("Failed to watch '%s' -  %s", configFile, err.Error())
    }
    // Shut down the watcher when done
    defer cancelWatch()
```

## Demo Code Examples
See more code examples in the ```examples/``` directory

```go
package main

import (
    "fmt"
    "os"

    "github.com/thrawn01/args"
)

type Config struct {
    PowerLevel  int
    Message     string
    StringSlice []string
    Verbose     int
    DbHost      string
    TheQuestion string
    TheAnswer   int
}

func main() {
    var conf Config

    // Create the parser with program name 'example'
    // and environment variables prefixed with APP_
    parser := args.NewParser(args.Name("demo"), args.EnvPrefix("APP_"),
        args.Desc("This is a demo app to showcase some features of args"))

    // Store Integers directly into a struct with a default value
    parser.AddOption("--power-level").Alias("-p").StoreInt(&conf.PowerLevel).
        Env("POWER_LEVEL").Default("10000").Help("set our power level")

    // Command line options can begin with -name, --name or even ++name
    // Most non word characters are supported
    parser.AddOption("++config-file").Alias("+c").IsString().
        Default("/path/to/config").Help("path to config file")

    // Use the args.Env() function to define an environment variable
    // NOTE: Since the parser was passed args.EnvPrefix("APP_") the actual
    // environment variable name is 'APP_MESSAGE'
    parser.AddOption("--message").Alias("-m").StoreStr(&conf.Message).
        Env("MESSAGE").Default("over-ten-thousand").Help("send a message")

    // Pass a comma separated list of strings and get a []string slice
    parser.AddOption("--slice").Alias("-s").StoreStringSlice(&conf.StringSlice).Env("LIST").
        Default("one,two,three").Help("list of messages")

    // Count the number of times an option is seen
    parser.AddOption("--verbose").Alias("-v").Count().StoreInt(&conf.Verbose).Help("be verbose")

    // Set bool to true if the option is present on the command line
    parser.AddOption("--debug").Alias("-d").IsTrue().Help("turn on Debug")

    // Specify the type of the arg with IsInt(), IsString(), IsBool() or IsTrue()
    parser.AddOption("--help").Alias("-h").IsTrue().Help("show this help message")

    // Add Required argument
    parser.AddArgument("the-question").Required().
        StoreStr(&conf.TheQuestion).Help("Before you have an answer")

    // Add Optional arguments
    parser.AddArgument("the-answer").IsInt().Default("42").
        StoreInt(&conf.TheAnswer).Help("It must be 42")

    // 'Conf' options are not set via the command line but can be set
    // via a config file or an environment variable
    parser.AddConfig("twelve-factor").Env("TWELVE_FACTOR").Help("Demo of config options")

    // Define a 'database' subgroup
    db := parser.InGroup("database")

    // Add command line options to the subgroup
    db.AddOption("--host").Alias("-dH").StoreStr(&conf.DbHost).
        Default("localhost").Help("database hostname")

    // Add subgroup specific config. 'Conf' options are not set via the
    // command line but can be set via a config file or anything that calls parser.Apply()
    db.AddConfig("debug").IsTrue().Help("enable database debug")

    // 'Conf' option names are not allowed to start with a non word character like
    // '--' or '++' so they can not be confused with command line options
    db.AddConfig("database").IsString().Default("myDatabase").Help("name of database to use")

    // If no type is specified, defaults to 'IsString'
    db.AddConfig("user").Help("database user")
    db.AddConfig("pass").Help("database password")

    // Pass our own argument list, or nil to parse os.Args[]
    opt := parser.ParseArgsSimple(nil)

    // NOTE: ParseArgsSimple() is just a convenience, you can call
    // parser.ParseArgs(nil) directly and handle the errors
    // yourself if you have more complicated use case

    // Demo default variables in a struct
    fmt.Printf("Power        '%d'\n", conf.PowerLevel)
    fmt.Printf("Message      '%s'\n", conf.Message)
    fmt.Printf("String Slice '%s'\n", conf.StringSlice)
    fmt.Printf("DbHost       '%s'\n", conf.DbHost)
    fmt.Printf("TheAnswer    '%d'\n", conf.TheAnswer)
    fmt.Println("")

    // If user asked for --help or there were no options passed
    if opt.NoArgs() || opt.Bool("help") {
        parser.PrintHelp()
        os.Exit(-1)
    }

    fmt.Println("")
    fmt.Println("==================")
    fmt.Println(" Direct Cast")
    fmt.Println("==================")

    // Fetch values by using the Cast functions
    fmt.Printf("Power               '%d'\n", opt.Int("power-level"))
    fmt.Printf("Message             '%s'\n", opt.String("message"))
    fmt.Printf("String Slice        '%s'\n", opt.StringSlice("slice"))
    fmt.Printf("Verbose             '%d'\n", opt.Int("verbose"))
    fmt.Printf("Debug               '%t'\n", opt.Bool("debug"))
    fmt.Printf("TheAnswer           '%d'\n", opt.Int("the-answer"))
    fmt.Printf("TheAnswer as String '%s'\n", opt.String("the-answer"))

    fmt.Println("")
    fmt.Println("==================")
    fmt.Println(" Database Group")
    fmt.Println("==================")

    // Fetch Group values
    dbAddOption := opt.Group("database")
    fmt.Printf("CAST DB Host   '%s'\n", dbAddOption.String("host"))
    fmt.Printf("CAST DB Debug  '%t'\n", dbAddOption.Bool("debug"))
    fmt.Printf("CAST DB User   '%s'\n", dbAddOption.String("user"))
    fmt.Printf("CAST DB Pass   '%s'\n", dbAddOption.String("pass"))

    fmt.Println("")

    iniFile := []byte(`
        power-level=20000
        message=OVER-TEN-THOUSAND!
        slice=three,four,five,six
        verbose=5
        debug=true

        [database]
        debug=false
        host=mysql.thrawn01.org
        user=my-username
        pass=my-password
    `)

    // Make configuration simple by reading arguments from an INI file
    opt, err := parser.FromIni(iniFile)
    if err != nil {
        fmt.Println(err.Error())
        os.Exit(-1)
    }

    fmt.Println("")
    fmt.Println("==================")
    fmt.Println("From INI file")
    fmt.Println("==================")

    // Values from the config file are used only if the argument is not present
    // on the commandline
    fmt.Printf("INI Power      '%d'\n", conf.PowerLevel)
    fmt.Printf("INI Message    '%s'\n", conf.Message)
    fmt.Printf("INI Slice      '%s'\n", conf.StringSlice)
    fmt.Printf("INI Verbose    '%d'\n", conf.Verbose)
    fmt.Printf("INI Debug      '%t'\n", opt.Bool("debug"))
    fmt.Println("")
}
```

Running this program produces this output

```
$ bin/demo why 50
Power        '10000'
Message      'over-ten-thousand'
String Slice '[one two three]'
DbHost       'localhost'
TheAnswer    '50'


==================
 Direct Cast
==================
Power               '10000'
Message             'over-ten-thousand'
String Slice        '[one two three]'
Verbose             '0'
Debug               'false'
TheAnswer           '50'
TheAnswer as String '50'

==================
 Database Group
==================
CAST DB Host   'localhost'
CAST DB Debug  'false'
CAST DB User   ''
CAST DB Pass   ''


==================
From INI file
==================
INI Power      '20000'
INI Message    'OVER-TEN-THOUSAND!'
INI Slice      '[three four five six]'
INI Verbose    '5'
INI Debug      'true'
```

Here is the help message
```
$ bin/demo -h
Usage: demo [OPTIONS] the-question [the-answer]

This is a demo app to showcase some features of args

Arguments:
  the-question   Before you have an answer
  the-answer     It must be 42

Options:
  +c, ++config-file   path to config file (Default=/path/to/config)
  -m, --message       send a message (Default=over-ten-thousand Env=MESSAGE)
  -s, --slice         list of messages (Default=one,two,three Env=LIST)
  -v, --verbose       be verbose
  -d, --debug         turn on Debug
  -h, --help          show this help message
  -p, --power-level   set our power level (Default=10000 Env=POWER_LEVEL)
  -dH, --host         database hostname (Default=localhost)

```

## Stuff that works
* Support list of strings '--list my,list,of,things'
* Support Counting the number of times an arg has been seen
* Support for Storing Strings,Ints,Booleans in a struct
* Support Default Arguments
* Support Reading arguments from an ini file
* Support different types of optional prefixes (--, -, ++, +, etc..)
* Support for Config only options
* Support for Groups
* Support for Etcd v3 (See: https://github.com/thrawn01/args-backends)
* Support for Watching Etcd v3 for changes and hot reload (See: https://github.com/thrawn01/args-backends)
* Support Positional Arguments
* Support for adhoc configuration groups using AddConfigGroups()
* Generate Help Message
* Support SubCommands
* Support Nested SubCommands
* Automatically adds a --help message if none defined
* Support for escaping arguments (IE: --help and \\-\\-help are different)
* Automatically generates help for SubCommands
* Tests for args.WatchFile()
* Support for Kubernetes ConfigMap file watching

## TODO
* Custom Help and Usage
* Support counting arguments in this format -vvvv
* Support list of ints,floats,etc.. '--list my,list,of,things'
* Support map type '--map={1:"thing", 2:"thing"}'
* Support float type '--float=3.14'
* Support '-arg=value'
* Support Parent Parsing
* Support Greedy Arguments ```[<files>….]```
* Write better intro document
* Write godoc
* Ability to include Config() options in help message
* if AddOption() is called with a name that doesn’t begin with a prefix, apply some default rules to match - or — prefix
* Add support for updating etcd values from the Option{} object. (shouldn't be hard)
