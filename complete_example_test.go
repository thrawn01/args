package args_test

import (
	"fmt"
	"github.com/thrawn01/args"
)

type Config struct {
	PowerLevel  int
	Message     string
	StringSlice []string
	Verbose     int
	DbHost      string
	TheQuestion string
	TheAnswer   int
}

func Example_complete() {
	var conf Config

	// Create the parser with program name 'example'
	// and environment variables prefixed with APP_
	parser := args.NewParser().
		Name("demo").
		EnvPrefix("APP_").
		Desc("This is a demo app to showcase some features of args")

	// Store Integers directly into a struct with a default value
	parser.AddFlag("--power-level").
		Help("set our power level").
		StoreInt(&conf.PowerLevel).
		Env("POWER_LEVEL").
		Default("10000").
		Alias("-p")

	// Command line flags can begin with -name, --name or even ++name
	// Most non word characters are supported
	parser.AddFlag("++config-file").
		Help("path to config file").
		Default("/path/to/config").
		Alias("+c").
		IsString()

	// Add an environment variable as a possible source for the argument
	// NOTE: Since the parser was passed args.EnvPrefix("APP_") the actual
	// environment variable name is 'APP_MESSAGE'
	parser.AddFlag("--message").
		Default("over-ten-thousand").
		Help("send a message").
		StoreStr(&conf.Message).
		Env("MESSAGE").
		Alias("-m")

	// Pass a comma separated list of strings and get a []string slice
	parser.AddFlag("--slice").Alias("-s").
		Help("list of things separated by a comma").
		StoreStringSlice(&conf.StringSlice).
		Default("one,two, three").
		Env("LIST")

	// Count the number of times an option is seen
	parser.AddFlag("--verbose").
		Help("be verbose").
		StoreInt(&conf.Verbose).
		Alias("-v").
		Count()

	// Set bool to true if the option is present on the command line
	parser.AddFlag("--debug").
		Help("turn on Debug").
		Alias("-d").
		IsTrue()

	// Specify the type of the arg with IsInt(), IsString(), IsBool() or IsTrue()
	parser.AddFlag("--help").
		Help("show this help message").
		Alias("-h").
		IsTrue()

	// Add Required arguments
	parser.AddArgument("the-question").
		Help("Before you have an answer").
		StoreStr(&conf.TheQuestion).
		Required()

	// Add Optional arguments
	parser.AddArgument("the-answer").
		Help("It must be 42").
		StoreInt(&conf.TheAnswer).
		Default("42").
		IsInt()

	// 'Conf' flags are not set via the command line but can be set
	// via a config file or an environment variable
	parser.AddConfig("twelve-factor").
		Env("TWELVE_FACTOR").
		Help("Demo of config flags")

	// Define a 'database' subgroup
	db := parser.InGroup("database")

	// Add command line flags to the subgroup
	db.AddFlag("--host").Alias("-dH").StoreStr(&conf.DbHost).
		Default("localhost").Help("database hostname")

	// Add subgroup specific config. 'Conf' flags are not set via the
	// command line but can be set via a config file or anything that calls parser.Apply()
	db.AddConfig("debug").IsTrue().Help("enable database debug")

	// 'Conf' option names are not allowed to start with a non word character like
	// '--' or '++' so they can not be confused with command line flags
	db.AddConfig("database").IsString().Default("myDatabase").Help("name of database to use")

	// If no type is specified, defaults to 'IsString'
	db.AddConfig("user").Help("database user")
	db.AddConfig("pass").Help("database password")

	// Pass our own argument list, or nil to parse os.Args[]
	opt := parser.ParseOrExit(nil)

	// NOTE: ParseOrExit() is just a convenience, you can call
	// parser.Parse(nil) directly and handle the errors
	// yourself if you have more complicated use case

	// Demo default variables in a struct
	fmt.Printf("Power        '%d'\n", conf.PowerLevel)
	fmt.Printf("Message      '%s'\n", conf.Message)
	fmt.Printf("String Slice '%s'\n", conf.StringSlice)
	fmt.Printf("DbHost       '%s'\n", conf.DbHost)
	fmt.Printf("TheAnswer    '%d'\n", conf.TheAnswer)
	fmt.Println("")

	fmt.Println("")
	fmt.Println("==================")
	fmt.Println(" Direct Cast")
	fmt.Println("==================")

	// Fetch values by using the Cast functions
	fmt.Printf("Power               '%d'\n", opt.Int("power-level"))
	fmt.Printf("Message             '%s'\n", opt.String("message"))
	fmt.Printf("String Slice        '%s'\n", opt.StringSlice("slice"))
	fmt.Printf("Verbose             '%d'\n", opt.Int("verbose"))
	fmt.Printf("Debug               '%t'\n", opt.Bool("debug"))
	fmt.Printf("TheAnswer           '%d'\n", opt.Int("the-answer"))
	fmt.Printf("TheAnswer as String '%s'\n", opt.String("the-answer"))

	fmt.Println("")
	fmt.Println("==================")
	fmt.Println(" Database Group")
	fmt.Println("==================")

	// Fetch Group values
	dbAddOption := opt.Group("database")
	fmt.Printf("CAST DB Host   '%s'\n", dbAddOption.String("host"))
	fmt.Printf("CAST DB Debug  '%t'\n", dbAddOption.Bool("debug"))
	fmt.Printf("CAST DB User   '%s'\n", dbAddOption.String("user"))
	fmt.Printf("CAST DB Pass   '%s'\n", dbAddOption.String("pass"))

	fmt.Println("")
}
