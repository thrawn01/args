package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/thrawn01/args"
	"github.com/thrawn01/args/ini"
)

func main() {

	parser := args.NewParser().Name("watch")
	parser.AddFlag("--bind").Alias("-b").Default("localhost:8080").
		Help("Interface to bind the server too")
	parser.AddFlag("--complex-example").Alias("-ce").IsBool().
		Help("Run the more complex example")
	parser.AddFlag("--config-file").Alias("-c").
		Help("The Config file to load and watch our config from")

	// Add a connection string to the database group
	parser.AddFlag("--connection-string").InGroup("database").Alias("-cS").
		Default("mysql://username@hostname:MyDB").
		Help("Connection string used to connect to the database")

	// Store the password in the config and not passed via the command line
	parser.AddConfig("password").InGroup("database").Help("database password")
	// Specify a config file version, when this version number is updated, the user is signaling to the application
	// that all edits are complete and the application can reload the config
	parser.AddConfig("version").IsInt().Default("0").Help("config file version")

	appConf, err := parser.Parse(nil)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}

	// Simple handler that prints out our config information
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conf := appConf.GetOpts()

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

	var cancelWatch args.WatchCancelFunc
	if appConf.Bool("complex-example") {
		cancelWatch = complex(parser)
	} else {
		cancelWatch = simple(parser)
	}

	// Shut down the watcher when done
	defer cancelWatch()

	// Listen and serve requests
	log.Printf("Listening for requests on %s", appConf.String("bind"))
	err = http.ListenAndServe(appConf.String("bind"), nil)
	if err != nil {
		log.Println(err)
		os.Exit(-1)
	}
}

// Simple example always updates the config when file changes are detected.
func simple(parser *args.Parser) args.WatchCancelFunc {
	// Get our current config
	appConf := parser.GetOpts()
	configFile := appConf.String("config-file")

	// Parse the ini file
	backend, err := ini.NewBackendFromFile(configFile)

	// Load the entire config
	opts, err := parser.FromBackend(backend)
	if err != nil {
		fmt.Printf("Unable to start load '%s' -  %s", configFile, err.Error())
	}

	// Watch the file and report when fields have changed
	cancelWatch := parser.Watch(backend, func(event args.ChangeEvent, err error) {
		if err != nil {
			fmt.Printf("While loading updated config - %s\n", err.Error())
			return
		}
		fmt.Printf("Config: %s:%s changed to %s", event.Key.Group, event.Key.Name, event.Value)

		// Apply the updated config change
		parser.Apply(opts.FromChangeEvent(event))
	})

	if err != nil {
		fmt.Printf("Unable to start watch '%s' -  %s", configFile, err.Error())
	}
	return cancelWatch
}

// The complex example allows a user to write the config file multiple times, possibly applying edits incrementally.
// When the user is ready for the application to apply the config changes, modify the 'version' value and the
// new config is applied.
func complex(parser *args.Parser) args.WatchCancelFunc {
	// Get our current config
	appConf := parser.GetOpts()
	configFile := appConf.String("config-file")

	// Parse the ini file
	backend, err := ini.NewBackendFromFile(configFile)

	// Load the entire config
	local, err := parser.FromBackend(backend)
	if err != nil {
		fmt.Printf("Unable to start load '%s' -  %s", configFile, err.Error())
	}

	// Watch the file and report when fields have changed
	cancelWatch := parser.Watch(backend, func(event args.ChangeEvent, err error) {
		if err != nil {
			fmt.Printf("While loading updated config - %s\n", err.Error())
			return
		}
		// Update local copy of the config
		local = local.FromChangeEvent(event)

		// Only apply the config if the version changed
		if event.Key.Name == "version" {
			appConf, err = parser.Apply(local.FromChangeEvent(event))
		}
	})
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to watch '%s' -  %s", configFile, err.Error()))
	}
	return cancelWatch
}
