package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/thrawn01/args"
)

func main() {
	parser := args.NewParser(args.Desc("Demo Service showcasing args JSON-RPC interface"))
	parser.AddOption("--bind").Alias("-b").Env("BIND").
		Default("0.0.0.0:1234").Help("The interface to bind too")

	parser.AddOption("--power-level").Alias("-p").Default("10000").IsInt().
		Help("set our power level")

	parser.AddConfig("simple").Help("Demo of simple config item")

	// Since args does not natively support deeply nested structured data, we can fake it by using nested keys
	// Much like etcdv3 uses prefix matching to support deeply nested structures
	parser.AddConfig("root/sub/item1").Help("nested key demo1")
	parser.AddConfig("root/sub/item2").Help("nested key demo2")
	parser.AddConfig("root/sub/item3").Help("nested key demo3")

	// Config groups are accessed via th JSON-RPC interface just like nest keys
	parser.AddConfigGroup("endpoints").Help("Demo of a config group")

	opt := parser.ParseArgsSimple(nil)

	// Simple Application that just displays our current config
	http.HandleFunc("/my-app", func(w http.ResponseWriter, r *http.Request) {
		conf := parser.GetOpts()

		payload, err := json.Marshal(map[string]string{
			"bind":           conf.String("bind"),
			"power-level":    conf.Int("power-level"),
			"simple":         conf.String("simple"),
			"root/sub/item1": conf.String("root/sub/item1"),
			"root/sub/item2": conf.String("root/sub/item2"),
			"root/sub/item3": conf.String("root/sub/item3"),
			"endpoints":      conf.StringSlice("endpoints"),
		})
		if err != nil {
			fmt.Println("error:", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(payload)
	})

	// Allow external users to change our config remotely via the JSON-RPC handler
	http.HandleFunc("/config", parser.JsonRPCHandler)

	fmt.Printf("Listening on %s...\n", opt.String("bind"))
	log.Fatal(http.ListenAndServe(opt.String("bind"), nil))

}
