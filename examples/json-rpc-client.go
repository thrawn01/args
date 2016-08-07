package main

import (
	"fmt"

	"net/rpc/jsonrpc"
	"net/url"
	"os"

	"net"

	"github.com/pkg/errors"
	"github.com/thrawn01/args"
)

func getEndpoint(opts *args.Options, endpoint *string) error {
	*endpoint = opts.String("endpoint")
	_, err := url.Parse(*endpoint)
	if err != nil {
		return errors.Wrapf(err, "url endpoint '%s' is invalid", *endpoint, err.Error())
	}
	return nil
}

// TODO: Rewrite this to use args.JsonRPCClient()

func main() {
	parser := args.NewParser(args.Name("http-client"),
		args.Desc("Example http client client"))

	parser.AddOption("--verbose").Alias("-v").Count().Help("Be verbose")
	parser.AddOption("--endpoint").Default("http://localhost:1234/config").
		Help("The JSON-RPC endpoint our client will talk too")

	parser.AddCommand("list", list)
	parser.AddCommand("get", get)
	parser.AddCommand("set", set)

	// Run the command chosen by the user
	retCode, err := parser.ParseAndRun(nil, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	os.Exit(retCode)
}

func list(subParser *args.ArgParser, data interface{}) int {
	opts := subParser.GetOpts()
	var values []string
	var url string

	if err := getEndpoint(opts, &url); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	conn, err := net.Dial("tcp", url)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer conn.Close()

	client := jsonrpc.NewClient(conn)

	// List will return all keys that match the prefix passed
	err = client.Call("list", "root", &values)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("%v\n", values)
	return 0
}

func get(subParser *args.ArgParser, data interface{}) int {
	opts := subParser.GetOpts()
	var value string
	var url string

	if err := getEndpoint(opts, &url); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	conn, err := net.Dial("tcp", url)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer conn.Close()

	client := jsonrpc.NewClient(conn)

	// List will return all keys that match the prefix passed
	err = client.Call("get", "root", &value)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("%v\n", value)
	return 0
}

func set(subParser *args.ArgParser, data interface{}) int {
	opts := subParser.GetOpts()
	var reply int
	var url string

	if err := getEndpoint(opts, &url); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	conn, err := net.Dial("tcp", url)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer conn.Close()

	client := jsonrpc.NewClient(conn)

	// List will return all keys that match the prefix passed
	err = client.Call("set", []string{"key", "value"}, &reply)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if reply != 0 {
		fmt.Fprintf(os.Stderr, "'%s'='%s' failed\n", "key", "value")
		return 1
	}
	fmt.Printf("'%s'='%s' set successfully\n", "key", "value")
	return 0
}
