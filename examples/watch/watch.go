package watch

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/thrawn01/args"
)

func main() {

	parser := args.NewParser("watch-example")
	parser.AddOption("--bind").Alias("-b").Default("localhost:8080").
		Help("Interface to bind the server too")
	parser.AddOption("--complex-example").Alias("-ce").IsBool().
		Help("Run the more complex example")
	parser.AddOption("--config-file").Alias("-c").
		Help("The Config file to load and watch our config from")

	// Add a connection string to the database group
	parser.AddOption("--connection-string").InGroup("database").Alias("-cS").
		Default("mysql://username@hostname:MyDB").
		Help("Connection string used to connect to the database")

	// Store the password in the config and not passed via the command line
	parser.AddConfig("password").InGroup("database").Help("database password")

	appConf, err := parser.ParseArgs(nil)
	if err != nil {
		log.Fatal(err.Error())
		os.Exit(-1)
	}

	// Simple handler that prints out our config information
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conf := appConf.ThreadSafe()

		db := conf.Group("database")
		payload, err := json.Marshal(map[string]string{
			"bind":     conf.String("bind"),
			"mysql":    db.String("connection-string"),
			"password": conf.Group("database").String("password"),
		})
		if err != nil {
			fmt.Println("error:", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(payload)
	})

	if appConf.Bool("complex-example") {
		// Complex Example
		args.WatchFile(opt.String("config-file"), func() {
			rawValues, err := parser.ParseIni(appConf.String("config-file"))
			if err != nil {
				fmt.Printf("Failed to update config - %s\n", err.Error())
				return
			}
			// Apply these raw values to the parser rules
			appConf, err = parser.Apply(rawValues)
			if err != nil {
				fmt.Printf("Probably a type cast error - %s\n", err.Error())
				return
			}
			// Compare the parsed file with our current values
			changedKeys := appConf.Compare(rawValues)
			// Iterate through all the keys that changed
			for key := range changedKeys {
				// If the database host changed
				if key == "database:connection-string" {
					// Re-init the database connection
					//initDb(appConf)
				}
			}
		})
	} else {
		// Simple example
		args.WatchFile(appConf.String("config-file"), func() {
			// You can safely ignore the returned Options{} object here.
			// the next call to ThreadSafe() from within the handler will
			// pick up the newly parsed config.
			appConf, err = parser.FromIni(appConf.String("config-file"))
			if err != nil {
				fmt.Printf("Failed to update config - %s\n", err.Error())
				return
			}
		})
	}

	// Listen and serve requests
	log.Printf("Listening for requests on %s", appConf.String("bind"))
	err = http.ListenAndServe(appConf.String("bind"), nil)
	if err != nil {
		log.Fatal(err)
	}

}
