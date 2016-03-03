## Argument Parser
Because I was un-happy about all the other arg parsers

## Usage
```go
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
}

func main() {
	var conf Config

	// Create the parser
	parser := args.Parser(args.Name("example"))

	parser.Opt("--power-level").StoreInt(&conf.PowerLevel).
		Env("POWER_LEVEL").Default("10000").Help("set our power level")

	// Store Integers directly into a struct with a default value
	parser.Opt("--power-level").Alias("-p").StoreInt(&conf.PowerLevel).
		Env("POWER_LEVEL").Default("10000").Help("set our power level")

	// Options can begin with -name, --name or even ++name. Most alpha non word character are supported
	parser.Opt("++power").Alias("+p").IsString().Default("11,000").Help("prefix demo")

	// Use the args.Env() function to define an environment variable
	parser.Opt("--message").Alias("-m").StoreStr(&conf.Message).
		Env("MESSAGE").Default("over-ten-thousand").Help("send a message")

	// Pass a comma separated list of strings and get a []string
	parser.Opt("--slice").Alias("-s").StoreSlice(&conf.Slice).Env("LIST").
		Default("one,two,three").Help("list of messages"))

	// Count the number of times an argument is seen
	parser.Opt("--verbose").Alias("-v").Count().StoreInt(&conf.Verbose).Help("be verbose")

	// Set bool to true if the argument is present on the command line
	parser.Opt("--debug").Alias("-d").IsTrue().Help("turn on Debug")

	// Specify the type of the arg with IsInt(), IsString(), IsBool() or IsTrue()
	parser.Opt("--help").Alias("-h").IsTrue().Help("show this help message")

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
```

Running this program produces this output

```
CAST Power     '10000'
CAST Message   'over-ten-thousand'
CAST Slice     '[one two three]'
CAST Verbose   '0'
CAST Debug     'false'

STRUCT Power   '10000'
STRUCT Message 'over-ten-thousand'
STRUCT Slice   '[one two three]'
STRUCT Verbose '0'
STRUCT Debug   'false'

INI Power      '20000'
INI Message    'OVER-THEN-THOUSAND!'
INI Slice      '[three four five six]'
INI Verbose    '5'
INI Debug      'true'

Usage:
  example [OPTIONS]

Options:
  -p, --power-level   set our power level (Default=10000 Env=POWER_LEVEL)
  +p, ++power         prefix demo (Default=11,000)
  -m, --message       send a message (Default=over-ten-thousand Env=MESSAGE)
  -s, --slice         list of messages (Default=one,two,three Env=LIST)
  -v, --verbose       be verbose
  -d, --debug         turn on Debug
  -h, --help          show this help message

exit status 255
```

## Stuff that works
* Support list of strings '--list my,list,of,things'
* Support Counting the number of times an arg has been seen
* Support for Storing Strings,Ints,Booleans in a struct
* Support Default Arguments
* Support Reading arguments from an ini file
* Support different types of optional prefixes (--, -, ++, +, etc..)
* Generate Help Message

## TODO
* Custom Help and Usage
* Support Positional Arguments
* Support SubParsing
* Support counting arguments in this format -vvvv
* Support list of ints,floats,etc.. '--list my,list,of,things'
* Support map type '--map={1:"thing", 2:"thing"}'
* Support float type '--float=3.14'
* Support '-arg=value'
* Support Parent Parsing
* Support '-aV' where 'a' is the argument and 'V' is the value
* DeDent thingy


