package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/thrawn01/args"
)

func main() {
	parser := args.NewParser(args.Name("etcd-endpoints-service"),
		args.EtcdPath("etcd-endpoints"),
		args.Desc("Example endpoint service"))

	// A Comma Separated list of etcd endpoints
	parser.AddOption("--etcd-endpoints").Alias("-e").Default("dockerhost:2379").
		Help("A Comma Separated list of etcd server endpoints")

	// A Command line only option
	parser.AddOption("--bind").Alias("-b").Default("localhost:1234").
		Help("Interface to bind the server too")

	// Just to demonstrate a single key/value in etcd
	parser.AddConfig("api-key").Alias("-k").Default("default-key").
		Help("A fake api-key")

	// Print Help message
	parser.AddOption("--help").Alias("-h").IsTrue().Help("show this help message")

	// This represents an etcd prefix of /etcd-endpoints/nginx-endpoints any key/value
	// stored under this prefix in etcd will be in the 'nginx-endpoints' group
	parser.AddConfigGroup("nginx-endpoints").
		Help("a list of nginx endpoints")

	// Parse the command line arguments
	opts, err := parser.ParseArgs(nil)
	if err != nil {
		log.Fatal(err.Error())
		os.Exit(-1)
	}

	if opts.Bool("help") {
		parser.PrintHelp()
		os.Exit(-1)
	}

	// Simple handler that prints out a list of endpoints
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// GetOpts is a thread safe get of the current options
		conf := parser.GetOpts()

		// Convert the nginx-endpoints group to a map
		endpoints := conf.Group("nginx-endpoints").ToMap()

		// Marshal the endpoints and our api-key to json
		payload, err := json.Marshal(map[string]interface{}{
			"endpoints": endpoints,
			"api-key":   conf.String("api-key"),
		})
		if err != nil {
			fmt.Println("error:", err)
		}
		// Write the response to the user
		w.Header().Set("Content-Type", "application/json")
		w.Write(payload)
	})

	client, err := etcd.New(etcd.Config{
		Endpoints:   opts.StringSlice("etcd-endpoints"),
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Read the config values like 'api-key' or 'nginx-endpoints' from etcd
	opts, err = parser.FromEtcd(client)
	if err != nil {
		fmt.Printf("Etcd error - %s\n", err.Error())
	}

	// Watch etcd for any configuration changes
	cancelWatch := parser.WatchEtcd(client, func(event *args.ChangeEvent) {
		// This takes a ChangeEvent and updates the opts with the latest changes
		parser.Apply(opts.FromChangeEvent(event))
	})

	// Listen and serve requests
	log.Printf("Listening for requests on %s", opts.String("bind"))
	err = http.ListenAndServe(opts.String("bind"), nil)
	if err != nil {
		log.Fatal(err)
	}
	cancelWatch()

}
