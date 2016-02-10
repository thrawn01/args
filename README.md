## Argument Parser
Because I was un-happy about all the other arg parsers

## Usage
```go

	var Config struct {
		PowerLevel  int
		Message		string
		Slice		[]string
	}
	var conf Config

	// Create the parser
	parser := args.Parser()

	// Define The Options
	parser.Opt("--power-level", args.Alias("-p"), args.StoreInt(&conf.PowerLevel),
		args.Env("POWER_LEVEL"), args.Default("10000"), args.Help("Set our power level"))
	parser.Opt("--message", args.Alias("-m"), args.StoreStr(&conf.Message),
		args.Env("MESSAGE"), args.Default("over-ten-thousand"), args.Help("Send a message"))
	parser.Opt("--slice", args.Alias("-s"), args.StoreSlice(&conf.Slice),
		args.Env("LIST"), args.Default("one,two,three"), args.Help("Send a message"))

	opt, err := parser.Parse()
	if err != nil {
		fatal(err.Error())
	}

	fmt.Printf("Power: %d\n", opt.Int("power-level"))
	fmt.Printf("Message %s\n", opt.String("message"))
	fmt.Printf("Slice %s\n", opt.Slice("slice"))

	fmt.Printf("Power: %d\n", conf.PowerLevel)
	fmt.Printf("Message %s\n", conf.Message)
	fmt.Printf("Slice %s\n", conf.Slice)

	// Read arguments from an INI file;
	// Can support any type of file, Built in YAML/JSON support planned.
	var iniFile := []byte(`
	power-level=20000
	message=OVER THEN THOUSAND!
	slice=three,four,five,six
	`)

	parser.ParseIni(iniFile)

	// Values from the config file are used if the argument is not present on
	// the commandline
	fmt.Printf("Power: %d\n", conf.PowerLevel)
	fmt.Printf("Message %s\n", conf.Message)
	fmt.Printf("Slice %s\n", conf.Slice)
```

## Stuff that works
* Support list of strings '--list my,list,of,things'
* Support Counting the number of times an arg has been seen
* Support for Storing Strings,Ints,Booleans in a struct
* Support Default Arguments
* Support Reading arguments from an ini file

## TODO
* Generate Help Message
* Custom Help and Usage
* Support Positional Arguments
* Support SubParsing
* Support list of ints,floats,etc.. '--list my,list,of,things'
* Support map type '--map={1:"thing", 2:"thing"}'
* Support float type '--float=3.14'
* Support '-arg=value'
* Support Parent Parsing
* Support '-aV' where 'a' is the argument and 'V' is the value
* DeDent thingy


