package main

import (
	"fmt"
	"github.com/thrawn01/args"
	"os"
)

type Config struct {
	PowerLevel int
	Message    string
	Slice      []string
	Verbose    int
}

func main() {
	var conf Config

	// Create the parser
	parser := args.Parser(args.Name("example"))

	// Store Integers directly into a struct with a default value
	parser.Opt("--power-level", args.Alias("-p"), args.StoreInt(&conf.PowerLevel),
		args.Env("POWER_LEVEL"), args.Default("10000"), args.Help("set our power level"))

	// Options can begin with -name, --name or even ++name. Most alpha non word character are supported
	parser.Opt("++power", args.Alias("+p"), args.IsString(), args.Default("11,000"),
		args.Help("prefix demo"))

	// Use the args.Env() function to define an environment variable
	parser.Opt("--message", args.Alias("-m"), args.StoreStr(&conf.Message),
		args.Env("MESSAGE"), args.Default("over-ten-thousand"), args.Help("send a message"))

	// Pass a comma seperated list of strings and get a []string
	parser.Opt("--slice", args.Alias("-s"), args.StoreSlice(&conf.Slice),
		args.Env("LIST"), args.Default("one,two,three"), args.Help("list of messages"))

	// Count the number of times an argument is seen
	parser.Opt("--verbose", args.Alias("-v"), args.Count(), args.StoreInt(&conf.Verbose),
		args.Help("be verbose"))

	// Set bool to true if the argument is present on the command line
	parser.Opt("--debug", args.Alias("-d"), args.IsTrue(), args.Help("turn on Debug"))

	// Specify the type of the arg with IsInt(), IsString(), IsBool() or IsTrue()
	parser.Opt("--help", args.Alias("-h"), args.IsTrue(), args.Help("show this help message"))

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
	fmt.Println("")

	// Values can also be stored in a struct
	fmt.Printf("STRUCT Power   '%d'\n", conf.PowerLevel)
	fmt.Printf("STRUCT Message '%s'\n", conf.Message)
	fmt.Printf("STRUCT Slice   '%s'\n", conf.Slice)
	fmt.Printf("STRUCT Verbose '%d'\n", conf.Verbose)
	fmt.Printf("STRUCT Debug   '%t'\n", opt.Bool("debug"))
	fmt.Println("")

	iniFile := []byte(`
		power-level=20000
		message=OVER-THEN-THOUSAND!
		slice=three,four,five,six
		verbose=5
		debug=true
    `)

	// Make configuration simply by reading arguments from an INI file
	opt, err = parser.ParseIni(iniFile)
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
}
