package main

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/net/context"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/pkg/errors"
	"github.com/thrawn01/args"
)

func etcdClientFactory(opts *args.Options) (*etcd.Client, error) {
	client, err := etcd.New(etcd.Config{
		Endpoints:   opts.StringSlice("endpoints"),
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, errors.Wrap(err, "etcdApiFactory()")
	}
	return client, nil
}

func main() {
	parser := args.NewParser(args.Name("etcd-endpoints-client"),
		args.Desc("A client to update etcd endpoint for the endpoint-service"))

	parser.AddOption("--endpoints").Default("dockerhost:2379").IsStringSlice().Env("ETCD_ENDPOINTS").
		Help("A comma seperated list of etcd endpoints")

	parser.AddCommand("add", func(parent *args.ArgParser, data interface{}) int {
		parent.AddPositional("name").Help("The name of the new endpoint")
		parent.AddPositional("url").Help("The url of the new endpoint")

		opts, err := parent.ParseArgs(nil)
		if err != nil {
			fmt.Println(err.Error())
			return 0
		}
		// TODO: Implement this as a condition of a positional argument
		if !opts.Required([]string{"name", "url"}) {
			fmt.Println("You must provide both 'name' and 'url' options")
			parent.PrintHelp()
			return 0
		}

		// Create our Client
		client, err := etcdClientFactory(opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "-- %s\n", err.Error())
			return 1
		}
		defer client.Close()

		// Create our context
		key := fmt.Sprintf("/etcd-endpoints/nginx-endpoints/%s", opts.String("name"))
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		// Put the key
		fmt.Printf("Adding New Endpoint '%s' - '%s'\n", key, opts.String("url"))
		_, err = client.Put(ctx, key, opts.String("url"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "-- %s\n", err.Error())
			return 1
		}
		return 0
	})

	// Parse the --endpoint argument, and run our command if any
	_, err := parser.ParseAndRun(nil, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "-- %s\n", err.Error())
		os.Exit(1)
	}
}
