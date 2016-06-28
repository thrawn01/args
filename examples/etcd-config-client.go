package main

import (
	"fmt"
	"os"
	"time"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/thrawn01/args"
	"golang.org/x/net/context"
)

func main() {
	parser := args.NewParser(args.Name("etcd-config-client"),
		args.Desc("A client to update etcd configs for the config-service"))

	parser.AddOption("--endpoints").Default("dockerhost:2379").IsStringSlice().Env("ETCD_ENDPOINTS").
		Help("A comma seperated list of etcd endpoints")

	parser.AddPositional("key").Required().Help("The key to set")
	parser.AddPositional("value").Required().Help("The value to set")

	opts, err := parser.ParseArgs(nil)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Create our Client
	client, err := etcd.New(etcd.Config{
		Endpoints:   opts.StringSlice("endpoints"),
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "-- %s\n", err.Error())
		os.Exit(1)
	}
	defer client.Close()

	// Create our context
	key := fmt.Sprintf("/etcd-config/DEFAULT/%s", opts.String("key"))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Put the key
	fmt.Printf("Adding New Endpoint '%s' - '%s'\n", key, opts.String("value"))
	_, err = client.Put(ctx, key, opts.String("value"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "-- %s\n", err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}
