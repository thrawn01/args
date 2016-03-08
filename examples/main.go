package main

import (
	"fmt"
	"os"

	"github.com/thrawn01/args"
)

type Config struct {
	PowerLevel int
	Message    string
	Slice      []string
	Verbose    int
	DbHost     string
}

func main() {
	var conf Config

	// Create the parser with program name 'example'
	// and environment variables prefixed with APP_
	parser := args.Parser(args.Name("example"), args.EnvPrefix("APP_"))

	parser.Opt("--power-level").StoreInt(&conf.PowerLevel).
		Env("POWER_LEVEL").Default("10000").Help("set our power level")

	// Store Integers directly into a struct with a default value
	parser.Opt("--power-level").Alias("-p").StoreInt(&conf.PowerLevel).
		Env("POWER_LEVEL").Default("10000").Help("set our power level")

	// Options can begin with -name, --name or even ++name.
	// Most non word characters are supported
	parser.Opt("++config-file").Alias("+c").IsString().
		Default("/path/to/config").Help("path to config file")

	// Use the args.Env() function to define an environment variable
	// NOTE: Since the parser was passed args.EnvPrefix("APP_") the actual
	// environment variable name is 'APP_MESSAGE'
	parser.Opt("--message").Alias("-m").StoreStr(&conf.Message).
		Env("MESSAGE").Default("over-ten-thousand").Help("send a message")

	// Pass a comma separated list of strings and get a []string
	parser.Opt("--slice").Alias("-s").StoreSlice(&conf.Slice).Env("LIST").
		Default("one,two,three").Help("list of messages")

	// Count the number of times an argument is seen
	parser.Opt("--verbose").Alias("-v").Count().StoreInt(&conf.Verbose).Help("be verbose")

	// Set bool to true if the argument is present on the command line
	parser.Opt("--debug").Alias("-d").IsTrue().Help("turn on Debug")

	// Specify the type of the arg with IsInt(), IsString(), IsBool() or IsTrue()
	parser.Opt("--help").Alias("-h").IsTrue().Help("show this help message")

	// 'Conf' options are not set via the command line but can be set
	// via a config file or an environment variable
	parser.Conf("twelve-factor").Env("TWELVE_FACTOR").Help("Demo of config options")

	// Define a 'database' subgroup
	db = parser.Group("database")
	// Add command line Options to the subgroup
	db.Opt("--host").Alias("-dH").StoreStr(&conf.DbHost).
		Default("localhost").Help("database hostname")

	// Add subgroup specific config. 'Conf' options are not set via the
	// command line but can be set via a config file or anything that calls parser.Apply()
	db.Conf("debug").IsTrue().Help("enable database debug")

	// 'Conf' option names are not allowed to start with a non word character like
	// '--' or '++' so they can not be confused with command line options
	db.Conf("database").IsString().Default("myDatabase").Help("name of database to use")

	// If no type 'IsString', 'IsInt' is specified, defaults to 'IsString'
	db.Conf("user").Help("database user")
	db.Conf("pass").Help("database password")

	// Pass our own argument list, or nil to parse os.Args[]
	opt, err := parser.ParseArgs(nil)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}

	// Fetch values by using the Cast functions
	fmt.Printf("CAST Power     '%d'\n", opt.Int("power-level"))
	fmt.Printf("CAST Message   '%s'\n", opt.String("message"))
	fmt.Printf("CAST Slice     '%s'\n", opt.Slice("slice"))
	fmt.Printf("CAST Verbose   '%d'\n", opt.Int("verbose"))
	fmt.Printf("CAST Debug     '%t'\n", opt.Bool("debug"))

	// Fetch Group values
	dbOpt := opt.Group("database")
	fmt.Printf("CAST DB Host   '%s'\n", dbOpt.String("host"))
	fmt.Printf("CAST DB Debug  '%t'\n", dbOpt.Bool("debug"))
	fmt.Printf("CAST DB User   '%s'\n", dbOpt.String("user"))
	fmt.Printf("CAST DB Pass   '%s'\n", dbOpt.String("pass"))

	fmt.Println("")

	// Values can also be stored in a struct
	fmt.Printf("STRUCT Power   '%d'\n", conf.PowerLevel)
	fmt.Printf("STRUCT Message '%s'\n", conf.Message)
	fmt.Printf("STRUCT Slice   '%s'\n", conf.Slice)
	fmt.Printf("STRUCT Verbose '%d'\n", conf.Verbose)
	fmt.Printf("STRUCT Debug   '%t'\n", opt.Bool("debug"))
	fmt.Printf("STRUCT DbHost  '%s'\n", conf.DbHost)
	fmt.Println("")

	iniFile := []byte(`
		power-level=20000
		message=OVER-TEN-THOUSAND!
		slice=three,four,five,six
		verbose=5
		debug=true

		[database]
		debug=false
		host=mysql.thrawn01.org
		user=my-username
		pass=my-password
	`)

	// Make configuration simply by reading arguments from an INI file
	opt, err = parser.FromIni(iniFile)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}

	// Values from the config file are used only if the argument is not present
	// on the commandline
	fmt.Printf("INI Power      '%d'\n", conf.PowerLevel)
	fmt.Printf("INI Message    '%s'\n", conf.Message)
	fmt.Printf("INI Slice      '%s'\n", conf.Slice)
	fmt.Printf("INI Verbose    '%d'\n", conf.Verbose)
	fmt.Printf("INI Debug      '%t'\n", opt.Bool("debug"))
	fmt.Println("")

	// If user asked for --help or there were no options passed
	if opt.NoArgs() || opt.Bool("help") {
		parser.PrintHelp()
		os.Exit(-1)
	}

	// Thread safe access to the options are available, calls to ThreadSafe()
	// ensure no data race occurs if the config is updated outside of the go routine
	go func() {
		// Use ThreadSafe() inside http.Handler to get race free access to your config
		localOpt := opt.ThreadSafe()
		fmt.Printf("THREAD SAFE DBHostname '%s'\n", localOpt.Group("database").String("host"))
		fmt.Printf("THREAD SAFE Verbose '%d'\n", localOpt.String("verbose"))
	}()

	// Update the config
	updatedIniFile := []byte(`
		verbose=0

		[database]
		host=postgresql.thrawn01.org
	`)
	opt, err = parser.FromIni(updatedIniFile)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}
	fmt.Printf("UPDATED CONFIG DBHostname '%s'\n", opt.Group("database").String("host"))
	fmt.Printf("UPDATED CONFIG Verbose '%d'\n", opt.String("verbose"))

	// Simple example
	args.WatchFile(opt.String("config-file"), func() {
		opt, err = parser.FromIni(opt.String("config-file"))
		if err != nil {
			fmt.Printf("Failed to update config - %s\n", err.Error())
			return
		}
	})

	// Complex Example
	args.WatchFile(opt.String("config-file"), func() {
		rawValues, err = parser.ParseIni(opt.String("config-file"))
		if err != nil {
			fmt.Printf("Failed to update config - %s\n", err.Error())
			return
		}
		// Apply these raw values to the parser rules
		opt, err = parser.Apply(rawValues)
		if err != nil {
			fmt.Printf("Probably a type cast error - %s\n", err.Error())
			return
		}
		// Compare the parsed file with our current values
		changedKeys := opt.Compare(rawValues)
		// Iterate through all the keys that changed
		for key := range changedKeys {
			// If the database host changed
			if key == "database:host" {
				// Re-init the database connection
				initDb(opt)
			}
		}

	})
}
