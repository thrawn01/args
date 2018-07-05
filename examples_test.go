package args_test

import (
	"github.com/thrawn01/args"
	"fmt"
	"log"
)

func ExampleParser_FromBackend() {
	// Simple backend, usually an INI, YAML, ETCD, or CONSOL backend
	backend := newHashMapBackend(map[string]string{
		"/root/foo": "bar",
		"/root/kit": "kat",
	}, "/root/")

	parser := args.NewParser()
	parser.AddFlag("foo")
	parser.AddFlag("kit")

	// Parse our command line args first
	_, err := parser.Parse([]string{"--foo", "bash"})
	if err != nil {
		log.Fatal(err)
	}

	// Now apply our backend values, any existing values from the
	// command line always take precedence
	opts, err := parser.FromBackend(backend)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("foo = %s\n", opts.String("foo"))
	fmt.Printf("kit = %s\n", opts.String("kit"))

	// Output:
	// foo = bash
	// kit = kat
}

func ExampleOptions_StringSlice() {
	parser := args.NewParser()
	parser.AddFlag("--list").IsStringSlice()

	// String slices will parse items separated by a comma
	opt, _ := parser.Parse([]string{"--list", "foo,bar,bit"})
	fmt.Println(opt.StringSlice("list"))

	// Output:
	// [foo bar bit]
}

func Example_options() {
	// This example shows how options and groups work
	parser := args.NewParser()

	// Create an Options object from a map
	opts := parser.NewOptionsFromMap(
		map[string]interface{}{
			"int":    1,
			"bool":   true,
			"string": "one",
			"endpoints": map[string]interface{}{
				"endpoint1": "host1",
				"endpoint2": "host2",
				"endpoint3": "host3",
			},
			"deeply": map[string]interface{}{
				"nested": map[string]interface{}{
					"thing": "foo",
				},
				"foo": "bar",
			},
		},
	)

	// Small demo of how Group() works with flags
	opts.String("string")                       // == "one"
	opts.Group("endpoints")                     // == *args.Options
	opts.Group("endpoints").String("endpoint1") // == "host1"
	opts.Group("endpoints").ToMap()             // map[string]interface{} {"endpoint1": "host1", ...}
	opts.StringMap("endpoints")                 // map[string]string {"endpoint1": "host1", ...}
	opts.KeySlice("endpoints")                  // [ "endpoint1", "endpoint2", ]
	opts.StringSlice("endpoints")               // [ "host1", "host2", "host3" ]
}


