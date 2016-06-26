package main

import "github.com/thrawn01/args"

// etcd-endpoints-client set endpoint1 http://thrawn01.org:8080

func main() {
	parser := args.NewParser(args.Name("etcd-endpoints-client"),
		args.Desc("A client to update etcd endpoint for the endpoint-service"))

}
