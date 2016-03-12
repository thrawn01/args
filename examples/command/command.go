package command

import (
	"fmt"
	"net/http"
	"os"

	"bytes"
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/thrawn01/args"
)

func curlString(req *http.Request, payload *[]byte) string {
	parts := []string{"curl", "-i", "-X", req.Method, req.URL.String()}
	for key, value := range req.Header {
		parts = append(parts, fmt.Sprintf("-H \"%s: %s\"", key, value[0]))
	}

	if payload != nil {
		parts = append(parts, fmt.Sprintf(" -d '%s'", string(*payload)))
	}

	return strings.Join(parts, " ")
}

func main() {
	parser := args.NewParser("command-example")
	parser.AddOption("--verbose").Alias("-v").Count().
		Help("Increase verbosity  to bind the server too")
	parser.AddOption("--endpoint").Alias("-e").
		Help("Our API REST Endpoint")

	parser.AddCommand("create").Action(func(parent *args.ArgParser) int {
		// CREATE Specific Options
		parser := parent.SubParser()
		parser.AddRequired("message").Positional().Help("The message to create")

		// Parse the additional arguments for 'create'
		opts, err := parser.ParseArgs(nil)
		if err != nil {
			fmt.Println(err.Error())
			return 1
		}

		// Create the payload
		payload, err := json.Marshal(map[string]string{
			"message": opts.String("message"),
		})
		if err != nil {
			fmt.Println("JSON Marshalling Error -", err)
			return 1
		}

		// Create the new Request
		req, err := http.NewRequest("POST", opts.String("endpoint"), bytes.NewBuffer(*payload))
		if err != nil {
			fmt.Println(err)
			return 1
		}
		req.Header.Set("Content-Type", "application/json")

		// Preform the Request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Client Error - %s on %s", err.Error(), curlString(req, payload))
			return 1
		}
		defer resp.Body.Close()

		// Read in the entire response
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
		}

		// Output the response
		fmt.Println(body)
		return 0
	})

	_, err := parser.ParseArgs(nil)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}
}
