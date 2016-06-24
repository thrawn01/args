package etcdExample

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	etcd "github.com/coreos/etcd/client"
	"github.com/thrawn01/args"
)

func main() {

	parser := args.NewParser(args.EtcdPath("watch-example"))
	parser.AddOption("--bind").Alias("-b").Default("localhost:8080").
		Help("Interface to bind the server too")
	parser.AddOption("--complex-example").Alias("-ce").IsBool().
		Help("Run the more complex example")
	parser.AddOption("--config-file").Alias("-c").
		Help("The Config file to load and watch our config from")

	// Add a connection string to the database group
	parser.AddOption("--bar").InGroup("foo").Alias("-").
		Default("mysql://username@hostname:MyDB").
		Help("Connection string used to connect to the database")

	// Store the password in the config and not passed via the command line
	parser.AddConfig("password").InGroup("database").Help("database password")

	// Tells us the list of endpoints will be the keys of /watch-example/endpoints/nginx
	// We can get the keys by using opts.Group("nginx-endpoints").Keys()
	// Then get the values using opt.Group("nginx-endpoints").String("endpoint1")

	// Config groups define a parse-able group, Groups don't have types, and can't be type checked at parse time.
	// for ini files it's a section
	// for etcd it's a directory
	// for yaml it can be a list of maps
	parser.AddConfigGroup("nginx-endpoints").Help("a list of nginx endpoints")

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

	cfg := etcd.Config{
		Endpoints: appConf.StringSlice("etcd-endpoints"),
		Transport: etcd.DefaultTransport,
		// set timeout per request to fail fast when the target endpoint is unavailable
		HeaderTimeoutPerRequest: time.Second,
	}

	client, err := etcd.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// Grab values for config options defined with 'Etcd()'
	appConf, err = parser.FromEtcd(client)
	if err != nil {
		fmt.Printf("Etcd or value parse issue - %s\n", err.Error())
	}

	// etcd config changed don't get applied until 'config-version' is changed.
	stagedConf := parser.NewOptions()
	// Watch etcd for any configuration changes
	cancelWatch := parser.WatchEtcd(client, func(group, key, value string) {
		// Apply all the key value to staged config
		stagedConf.Group(group).Set(key, value, false)
		if group == "" && key == "config-version" {
			// NOTE: If you are using opt.ThreadSafe() you can safely
			// ignore the 'opt' returned by Apply(). This is because Apply()
			// will update parsers internal pointer to the new version of the
			// config and subsequent calls to opt.ThreadSafe() will always
			// safely return the new version of the config

			// Apply the new config to the parser
			appConf, err := parser.Apply(stagedConf)
			if err != nil {
				fmt.Printf("Probably a type cast error - %s\n", err.Error())
				return
			}
			// Clear the staged config values
			stagedConf = parser.NewOptions()
			fmt.Printf("Config has been updated to version %d\n", appConf.Int("config-version"))
		}
		if group == "nginx-endpoints" {
			// Immediately apply the new config when endpoints change
			parser.Apply(parser.NewOptions().Group(group).Set(key, value, false))
		}
	})

	// Listen and serve requests
	log.Printf("Listening for requests on %s", appConf.String("bind"))
	err = http.ListenAndServe(appConf.String("bind"), nil)
	if err != nil {
		log.Fatal(err)
	}
	cancelWatch()

}
