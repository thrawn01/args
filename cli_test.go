package args_test


import (
	"net/http"
	"bytes"
	"log"
	"fmt"
	"github.com/thrawn01/args"
	"encoding/json"
	"strings"
)

func ExampleCurlString() {
	// Payload
	payload, err := json.Marshal(map[string]string{
		"stuff":     "junk",
	})

	// Create the new Request
	req, err := http.NewRequest("POST", "http://google.com/stuff", bytes.NewBuffer(payload))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s\n", args.CurlString(req, &payload))

	// Output:
	// curl -i -X POST http://google.com/stuff  -d '{"stuff":"junk"}'
}

func ExampleLoadFile() {
	// Loads the entire file into a byte slice
	contents, err := args.LoadFile("example.conf")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(contents))

	// Output:
	// # Comments are ignored
	// foo=bar
}

func ExampleDedent() {
	desc := args.Dedent(`Example is a fast and flexible thingy

	Complete documentation is available at http://thingy.com

	Example Usage:
	    $ example-cli some-argument
	    Hello World!`)

	fmt.Println(desc)
	// Output:
	// Example is a fast and flexible thingy
    //
	// Complete documentation is available at http://thingy.com
    //
	// Example Usage:
	//     $ example-cli some-argument
	//     Hello World!
}

func ExampleWordWrap() {
	msg := args.WordWrap(`No code is the best way to write secure and reliable applications.
		Write nothing; deploy nowhere. This is just an example application, but imagine it doing 
		anything you want.`,
		3, 80)
	fmt.Println(msg)
	// Output:
	// No code is the best way to write secure and reliable applications.Write
	//    nothing; deploy nowhere. This is just an example application, but imagine
    //    it doing anything you want.
}

func ExampleStringToSlice() {
	// Returns []string{"one"}
	fmt.Println(args.StringToSlice("one"))

	// Returns []string{"one", "two", "three"}
 	fmt.Println(args.StringToSlice("one, two, three", strings.TrimSpace))

	//  Returns []string{"ONE", "TWO", "THREE"}
 	fmt.Println(args.StringToSlice("one,two,three", strings.ToUpper, strings.TrimSpace))

	// Output:
	// [one]
	// [one two three]
	// [ONE TWO THREE]
}

func ExampleStringToMap() {
	// Returns map[string]string{"foo": "bar"}
	fmt.Println(args.StringToMap("foo=bar"))

	// Returns map[string]string{"foo": "bar", "kit": "kitty kat"}
	m, _ := args.StringToMap(`foo=bar,kit="kitty kat"`)
	fmt.Printf("foo: %s\n", m["foo"])
	fmt.Printf("kit: %s\n", m["kit"])

	// Returns map[string]string{"foo": "bar", "kit": "kitty kat"}
	m, _ = args.StringToMap(`{"foo":"bar","kit":"kitty kat"}`)
	fmt.Printf("foo: %s\n", m["foo"])
	fmt.Printf("kit: %s\n", m["kit"])

	// Output:
	// map[foo:bar] <nil>
	// foo: bar
	// kit: "kitty kat"
	// foo: bar
	// kit: kitty kat
}
