package args

import (
	"fmt"
	"net/http"
)

// This method exposes the args.RPC interface via JSON-RPC
func (self *ArgParser) JsonRPCHandler(resp http.ResponseWriter, req *http.Request) {

	// Decode the JSON Request

	// Execute the Method

	// Encode the response

	fmt.Fprintf(resp, `{ "message": "JSON RPC HERE"}`)
}
