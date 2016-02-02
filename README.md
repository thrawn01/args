## Argument Parser
Because I was un-happy about all the other arg parsers

## Usage
```go

	var Config struct {
		PowerLevel  int
		Message		string
	}
	var conf Config

	// Create the parser
	parser := args.Parser()

	// Define The Options
	parser.Opt("--power-level", args.StoreInt(&conf.PowerLevel), args.Env("POWER_LEVEL"), args.Default(10000))
	parser.Opt("--message", args.StoreString(&conf.Message), args.Env("MESSAGE"), args.Default("over-ten-thousand"))

	opt, err := parser.Parse()
	if err != nil {
		fatal(err.Error())
	}

	fmt.Printf("Power: %d\n", opt.Int("power-level"))
	fmt.Printf("Message %s\n", opt.String("message"))

	fmt.Printf("Power: %d\n", conf.PowerLevel)
	fmt.Printf("Message %s\n", conf.Message)

```

## TODO
* Generate Help Message
* Custom Help and Usage
* Support Aliases
* Support Positional Arguments
* Support SubParsing
* Support list type '--list=my,list,of,things'
* Support map type '--map={1:"thing", 2:"thing"}'
* Support '-arg=value'
* Support Parent Parsing
* Support '-aV' where 'a' is the argument and 'V' is the value
* DeDent thingy


