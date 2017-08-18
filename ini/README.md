[![Coverage Status](https://img.shields.io/coveralls/thrawn01/args-etcd.svg)](https://coveralls.io/github/thrawn01/args-etcd)
[![Build Status](https://img.shields.io/travis/thrawn01/args-etcd/master.svg)](https://travis-ci.org/thrawn01/args-etcd)

## Introduction
This repo provides an ini key=value storage backend for use with
 [args](http://github.com/thrawn01/args)

## Installation
```
$ go get github.com/thrawn01/argsini
```

## Usage
```go
    import (
    	"github.com/thrawn01/args"
        "github.com/thrawn01/argsini"
    )

	parser := args.NewParser()
	parser.AddOption("listen-address").Help("Specify local address to listen on")
	parser.AddOption("config").Short("c").Required().Default("/etc/app/config.ini").
	    Help("Specify configuration file")
	parser.AddOption("cache-size").IsInt().Default(150).
	    Help("Specify the size of the cache in entries")

	// Parse the command line args
	opt := parser.ParseOrExit(nil)

    // Create a new backend object
    backend := argsini.NewFromFile(opt.String("config"), "")

    // Load the config file specified on the command line
    opts, err := parser.FromBackend()
	if err != nil {
		fmt.Printf("config error - %s\n", err.Error())
	}

    // Print out the loaded config items
    fmt.Printf("listen-address: %s\n", opts.String("listen"))
    fmt.Printf("cache-size: %d\n", opts.Int("listen"))

    // Watch ini file for any configuration changes
    cancelWatch := parser.Watch(backend, func(event *args.ChangeEvent, err error) {
        if err != nil {
            fmt.Println(err.Error())
            return
        }
        fmt.Printf("Changed Event - %+v\n", event)
        // This takes a ChangeEvent and update the opts with the latest changes
        parser.Apply(opts.FromChangeEvent(event))
    })
```
