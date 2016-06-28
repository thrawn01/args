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

func Add(subParser *args.ArgParser, data interface{}) int {
	subParser.AddPositional("name").Required().Help("The name of the new endpoint")
	subParser.AddPositional("url").Required().Help("The url of the new endpoint")

	opts, err := subParser.ParseArgs(nil)
	if err != nil {
		fmt.Println(err.Error())
		return 1
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
}

func Delete(subParser *args.ArgParser, data interface{}) int {
	subParser.AddPositional("name").Required().Help("The name of the endpoint to delete")

	opts, err := subParser.ParseArgs(nil)
	if err != nil {
		fmt.Println(err.Error())
		return 1
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

	fmt.Printf("Deleting Endpoint '%s'\n", key)
	_, err = client.Delete(ctx, key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "-- %s\n", err.Error())
		return 1
	}
	return 0
}

func main() {
	parser := args.NewParser(args.Name("etcd-endpoints-client"),
		args.Desc("A client to update etcd endpoint for the endpoint-service"))

	parser.AddOption("--endpoints").Default("dockerhost:2379").IsStringSlice().Env("ETCD_ENDPOINTS").
		Help("A comma seperated list of etcd endpoints")

	parser.AddCommand("add", Add).Help("Add an endpoint to the etcd store")
	parser.AddCommand("delete", Delete).Help("Deletes an endpoint from the etcd store")

	// Parse the --endpoint argument, and run our commands if provided
	_, err := parser.ParseAndRun(nil, nil)
	if err != nil {
		if !args.AskedForHelp(err) {
			fmt.Fprintf(os.Stderr, "-- %s\n", err.Error())
		}
		os.Exit(1)
	}
}
