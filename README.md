[![Coverage Status](https://img.shields.io/coveralls/thrawn01/args.svg)](https://coveralls.io/github/thrawn01/args)
[![Build Status](https://img.shields.io/travis/thrawn01/args/master.svg)](https://travis-ci.org/thrawn01/args)

**NOTE: This is alpha software, the api will continue to evolve**

## Introduction
A 12 factor app configuration and argument parser designed for
 use in distributed services with support for CLI clients and live server
 configuration reloading

### Configuration Philosophy
Configuration for modern cloud native applications is divided into two separate
phases. The first phase is initialization, This is the minimum configuration
the service needs to start up. At a minimum this may include configuration for:
Service Interface, Port Number, and the source location of the second phase configuration.

The second phase includes everything else the service needs to operate. Including:
Monitoring, Logging, Databases, etc.... Everything in the First Phase should be mostly static
items; things that can not change unless the service is restarted. Everything in
the second phase can and should change at anytime during the operation of the service
without the need for the service to restart.

Since the first phase remains mostly static, Simple narrowly nested configuration
 is suitable here. The Second phase should be saved for rich deeply nested configuration
 typically loaded from a json, yaml, or a distributed key store.

Args is designed to make shallowly nested first and second phase configuration easy
 and flexible. It include capabilities to assist in synchronization of second phase
 hot reloading. For items that require *deeply nested* configuration a json, yaml, or
 distributed key store is almost always better suited after first phase has completed.

## Key Store Backends
Args supports additional backend configuration via any backend that implements the
 ```Backend``` interface. Currently only etcd is supported and is provided by the
  [args-backend](http://github.com/thrawn01/args-backends) repo.

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

## Usage
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

	// Count the number of times an argument is seen
	parser.AddOption("--verbose").Alias("-v").Count().StoreInt(&conf.Verbose).Help("be verbose")

	// Set bool to true if the argument is present on the command line
	parser.AddOption("--debug").Alias("-d").IsTrue().Help("turn on Debug")

	// Specify the type of the arg with IsInt(), IsString(), IsBool() or IsTrue()
	parser.AddOption("--help").Alias("-h").IsTrue().Help("show this help message")

	// Add Required positional arguments
	parser.AddPositional("the-question").Required().
		StoreStr(&conf.TheQuestion).Help("Before you have an answer")

	// Add Optional positional arguments
	parser.AddPositional("the-answer").IsInt().Default("42").
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

Positionals:
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
* Support for Etcd v3 (See: https://github.com/thrawn01/args-etcd)
* Support for Watching Etcd v3 for changes and hot reload (See: https://github.com/thrawn01/args-etcd)
* Support Positional Arguments
* Support for adhoc configuration groups using AddConfigGroups()
* Generate Help Message
* Support SubCommands
* Support Nested SubCommands
* Automatically adds a --help message if none defined
* Support for escaping positionals (IE: --help and \\-\\-help are different)
* Automatically generates help for SubCommands


## TODO
* Custom Help and Usage
* Support counting arguments in this format -vvvv
* Support list of ints,floats,etc.. '--list my,list,of,things'
* Support map type '--map={1:"thing", 2:"thing"}'
* Support float type '--float=3.14'
* Support '-arg=value'
* Support Parent Parsing
* Support for ConfigMap
* Support Greedy Positional Arguments ```[<files>….]```
* Write better intro document
* Write godoc
* Test args.FileWatcher()
* Example for k8s configMap
* Ability to include Config() options in help message
* if AddOption() is called with a name that doesn’t begin with a prefix, apply some default rules to match - or — prefix
* Add support for updating etcd values from the Option{} object. (shouldn't be hard)
