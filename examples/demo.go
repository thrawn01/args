package main

import (
	"github.com/thrawn01/args"
)

type Config struct {
	Bind string
}

func main() {
	var conf Config

	//parser := args.Parser(args.WarnUnknownArgs(), args.UsageName("my-program"))
	parser := args.Parser()

	// Positional Argument, there can be more than one argument and the result will be an array
	parser.Pos("count", args.Append(), args.IsInt(), args.Help("Count the numbers provided"))

	// Optional Positional Argument, the order the positional arguments are
	// defined is the order they are expected on the command line
	parser.Pos("infile", args.Optional(), args.IsString(), args.Help("Output file, If none supplied reads from stdin"))
	parser.Pos("outfile", args.Optional(), args.IsString(), args.Help("Input file, If none supplied writes to stdout"))

	// Counts the number of instances a flag is seen and stores the count as an integer
	parser.Opt("--verbose", args.Alias("-v"), args.Count(),
		args.Help("Prints Stuff and things"))

	// String with default option and Environment option (VarName defines the help message value name)
	parser.Opt("--bind", args.Alias("-b"), args.IsString(), args.VarName("ADDRESS"), args.Default("localhost:8080"),
		args.Env("BIND_ADDRESS"), args.Store(&conf.Bind), args.Help("The network interface to bind too"))

	// Unusual option indicators
	parser.Opt("++unusual", args.Alias("+u"), args.IsBool(), args.Default(false),
		args.Help("Unusual option indicator"))
	parser.Opt("-short", args.Alias("-s"), args.IsBool(), args.Default(false),
		args.Help("Short option indicator"))

	// Booleans
	parser.Opt("--debug", args.Alias("-d"), args.IsBool(), args.Default(false),
		args.Help("Turns on debug messages"))

	// Append stores an array, and appends each argument value to the array. This allows an option to be specified multiple times.
	parser.Opt("--endpoint", args.Alias("-e"), args.Append(), args.IsString(),
		args.Default([]string{"http://google.com"}), args.Help("The endpoints to proxy for"))

	// Take an option and split the arguments IE: --terms one,two,three
	parser.Opt("--terms", args.Alias("-t"), args.Array(), args.IsString(),
		args.Default([]string{"foo", "bar"}), args.Help("The terms used to search google with"))

	// Give the user choices
	parser.Opt("--choice", args.Alias("-c"), args.Choice(), args.IsInt(),
		args.Default([]int{1, 2, 3}), args.Help("You can only choose 1,2 or 3"))

	// Give the option is required (Required options can not have defaults)
	parser.Opt("--required", args.Alias("-r"), args.Required(), args.IsString(),
		args.Help("This option is required"))

	// Display Help
	parser.Opt("--help", args.Alias("-h"), args.IsBool(), args.Help("Display this help message"))

	//opts, err := parser.ParseArgs([]string{"-p", "-vvv"})
	opts, err := parser.Parse()
	if err != nil {
		fmt.Println("-- %s", err.Error())
		opts.PrintUsage()
		os.Exit(-1)
	}

	// If user asked for --help or there were no options passed
	if opts.NoArgs() && opts.Bool("help") {
		opts.PrintHelp()
		os.Exit(-1)
	}

	fmt.Printf("Bind Address from opts: %s", opts.String("bind"))
	fmt.Printf("Bind Address from Config struct: %s", conf.Bind)

	fmt.Printf("Verbose Count: %d", opts.Int("verbose"))
	fmt.Printf("Debug Boolean: %b", opts.Bool("debug"))

	for _, endpoint := range opts.Array("endpoint").([]string) {
		fmt.Println("Endpoint: %s", endpoint)
	}

	// Parent that provides shared options
	parent := args.Parser()
	parent.Opt("verbose", args.Alias("-v"), args.Count(), args.Help("Be verbose"))

	// Sub Parser, Any arguments past the positional arg 'sub1' will be considered by this sub
	sub := parser.SubParser("volume")

	// parser and will inherit all the opts from the parent parser
	list := sub.SubParser("list", args.Inherit(parent))

	list.Opt("filter", args.Alias("-f"), args.IsBool(), args.Help("Filter the list by this thingy"))

}
