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
	parser := args.NewParser(args.Name("etcd-config-service"),
		args.EtcdPath("etcd-config"),
		args.Desc("Example versioned config service"))

	// A Comma Separated list of etcd endpoints
	parser.AddOption("--etcd-endpoints").Alias("-e").Default("dockerhost:2379").
		Help("A Comma Separated list of etcd server endpoints")

	// A Command line only option
	parser.AddOption("--bind").Alias("-b").Default("localhost:1234").
		Help("Interface to bind the server too")

	// Just to demonstrate a single key/value in etcd
	parser.AddConfig("name").Help("The name of our user")
	parser.AddConfig("age").IsInt().Help("The age of our user")
	parser.AddConfig("sex").Help("The sex of our user")
	parser.AddConfig("config-version").IsInt().Default("0").
		Help("When version is changed, the service will update the config")

	// Parse the command line arguments
	opts, err := parser.ParseArgs(nil)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Simple handler that returns a list of endpoints to the caller
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// GetOpts is a thread safe way to get the current options
		conf := parser.GetOpts()

		// Marshal the endpoints and our api-key to json
		payload, err := json.Marshal(map[string]interface{}{
			"name":           conf.String("name"),
			"age":            conf.Int("age"),
			"sex":            conf.String("sex"),
			"config-version": conf.Int("config-version"),
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

	// Read the config values from etcd
	opts, err = parser.FromEtcd(client)
	if err != nil {
		fmt.Printf("Etcd error - %s\n", err.Error())
	}

	// Watch etcd for any configuration changes
	stagedOpts := parser.NewOptions()
	cancelWatch := parser.WatchEtcd(client, func(event *args.ChangeEvent, err error) {
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		fmt.Printf("Change Event - %+v\n", event)
		// This takes a ChangeEvent and updates the stagedOpts with the latest changes
		stagedOpts.FromChangeEvent(event)

		// Only apply our config change once the config values
		// have all be collected and our version number changes
		if event.Key == "config-version" {
			// Apply the new config to the parser
			opts, err := parser.Apply(stagedOpts)
			if err != nil {
				fmt.Print(err.Error())
				return
			}
			// Clear the staged config values
			stagedOpts = parser.NewOptions()
			fmt.Printf("Config updated to version %d\n", opts.Int("config-version"))
		}
	})

	// Listen and serve requests
	log.Printf("Listening for requests on %s", opts.String("bind"))
	err = http.ListenAndServe(opts.String("bind"), nil)
	if err != nil {
		log.Fatal(err)
	}
	cancelWatch()
}
