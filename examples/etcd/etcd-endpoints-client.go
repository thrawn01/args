package main

import "github.com/thrawn01/args"

func main() {
	parser := args.NewParser(args.EtcdPath("etcd-endpoints"),
		args.Desc("Example endpoint service"))

}
