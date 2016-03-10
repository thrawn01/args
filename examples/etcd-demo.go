package main

import (
	"fmt"
	"log"
	"os"
	"time"

	etcd "github.com/coreos/etcd/client"
	"github.com/thrawn01/args"
)

func main() {

	// Create the parser with program name 'example'
	// and etcd keys prefixed with exampleApp/
	parser := args.Parser(args.Name("example"), args.EtcdPath("exampleApp/"))

	// Since 'Etcd()' is not used, this option is not configurable via etcd
	parser.AddOption("--etcd-endpoints").Alias("-eP").IsSlice().
		Default("192.168.5.1,192.168.5.2").Help("List of etcd endpoints")

	// Define a name used by other services to discover this service
	parser.AddOption("--service-name").Alias("-sN").
		Default("frontend1").Help("Name used for service discovery")

	// if Etcd() is given etcd keys are crafted by using
	// the name of the option. This etcd key will be '/exampleApp/message'
	parser.AddOption("--message").Etcd().
		Default("over-ten-thousand").Help("send a message")

	// Defines --power-level command line option, but defines the
	// etcd key as '/exampleApp/powerLevel'
	parser.AddOption("--power-level").EtcdKey("powerLevel").
		Default("10000").Help("set our power level")

	// Config options can also be used
	parser.AddConfig("config-version").InGroup("database").IsInt().Etcd().
		Help("Indicates updates to etcd are complete and we should reload the service")

	// You can also use groups with etcd keys
	db = parser.InGroup("database")

	// etcd key will be '/exampleApp/database/host'
	db.AddConfig("database").Default("localhost").Etcd().Help("database hostname")

	// etcd key will be '/exampleApp/database/debug'
	db.AddConfig("debug").IsTrue().Etcd().Help("enable database debug")
	// etcd key will be '/exampleApp/database/database'
	db.AddConfig("database").IsString().Etcd().Default("myDatabase").Help("name of database to use")
	// etcd key will be '/exampleApp/database/user'
	db.AddConfig("user").Etcd().Help("database user")
	// etcd key will be '/exampleApp/database/pass'
	db.AddConfig("pass").Etcd().Help("database password")

	opts, err := parser.ParseArgs(nil)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}

	cfg := etcd.Config{
		Endpoints: opts.Slice("etcd-endpoints"),
		Transport: etcd.DefaultTransport,
		// set timeout per request to fail fast when the target endpoint is unavailable
		HeaderTimeoutPerRequest: time.Second,
	}

	client, err := etcd.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	keysAPI := etcd.NewKeysAPI(client)

	// Grab values for config options defined with 'Etcd()'
	opt, err = parser.FromEtcd(keysAPI, log)
	if err != nil {
		fmt.Printf("Etcd or value parse issue - %s\n", err.Error())
	}

	// Simple watch example, When ever a config item changes
	// in etcd; immediately update our config
	args.WatchEtcd(parser.EtcdPath(), func(group, key, value string) {
		parser.Apply(args.NewOptions().Get(group).Set(key, value))
	})

	// Complex Example, where the config changes in etcd do
	// not get applied until 'config-version' is changed.
	stagedConf := args.NewOptions()
	// Watch etcd for any configuration changes
	args.WatchEtcd(parser.EtcdPath(), func(group, key, value string) {
		// Apply all the config
		stagedConf.Get(group).Set(key, value)
		if group == "" && key == "config-version" {
			// NOTE: If you are using opt.ThreadSafe() you can safely
			// ignore the 'opt' returned by Apply(). This is because Apply()
			// will update parsers internal pointer to the new version of the
			// config and subsequent calls to opt.ThreadSafe() will always
			// safely return the new version of the config

			// Apply the new config to the parser
			opt, err := parser.Apply(stagedConf)
			if err != nil {
				fmt.Printf("Probably a type cast error - %s\n", err.Error())
				return
			}
			// Clear the staged config values
			stagedConf = args.NewOptions()
			fmt.Printf("Config has been updated to version %d\n", opt.Int("config-version"))
		}
	})

	// TODO: Make a third example where we grab the updated etcd config only when 'config-version' changes
}
