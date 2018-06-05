package args

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/fatih/structs"
	"github.com/pkg/errors"
)

type ParseFlag int64

const (
	IsFormatted ParseFlag = 1 << iota
)

var regexInValidPrefixChars = regexp.MustCompile(`[\w\s]`)
var regexInValidRuleName = regexp.MustCompile(`[!"#$&'/()*;<>{|}\\\\~\s]`)

type ParseModifier func(*Parser)

type Parser struct {
	stopParsingOnCommand bool
	description          string
	wordWrapLen          int
	prefixChars          []string
	envPrefix            string
	posCount             int
	attempts             int
	command              *Rule
	addHelp              bool
	options              *Options
	parent               *Parser
	helpIO               *os.File
	epilog               string
	usage                string
	mutex                sync.Mutex
	flags                ParseFlag
	rules                Rules
	args                 []string
	name                 string
	err                  error
	idx                  int
	log                  StdLogger
}

// Creates a new instance of the argument parser
func NewParser() *Parser {
	parser := &Parser{
		wordWrapLen: 200,
		mutex:       sync.Mutex{},
		log:         DefaultLogger,
		addHelp:     true,
		helpIO:      os.Stdout,
	}
	return parser
}

// Takes the current parser and return a new parser
// appropriate for use within a command function
func (p *Parser) SubParser() *Parser {
	parser := NewParser()
	src := structs.New(p)
	dest := structs.New(parser)

	// Copy all the public values
	for _, field := range src.Fields() {
		if field.IsExported() {
			dest.Field(field.Name()).Set(field.Value())
		}
	}

	parser.addHelp = p.addHelp
	parser.options = p.options
	parser.rules = p.rules
	parser.args = p.args
	parser.parent = p
	parser.log = p.log

	// Remove all Commands from our rules
	for i := len(parser.rules) - 1; i >= 0; i-- {
		if parser.rules[i].HasFlag(IsCommand) {
			// Remove the rule
			parser.rules = append(parser.rules[:i], parser.rules[i+1:]...)
		}
	}
	// Clear the selected Commands
	parser.command = nil
	return parser
}

func (p *Parser) Log(logger StdLogger) {
	p.log = logger
}

func (p *Parser) Name(value string) *Parser {
	p.name = value
	return p
}

func (p *Parser) Desc(value string, flags ...ParseFlag) *Parser {
	p.description = value
	for _, flag := range flags {
		setFlags(&p.flags, flag)
	}
	return p
}

func (p *Parser) WordWrap(value int) *Parser {
	p.wordWrapLen = value
	return p
}

func (p *Parser) EnvPrefix(value string) *Parser {
	p.envPrefix = value
	return p
}

func (p *Parser) AddHelp(value bool) *Parser {
	p.addHelp = value
	return p
}

func (p *Parser) Epilog(value string) *Parser {
	p.epilog = value
	return p
}

func (p *Parser) Usage(value string) *Parser {
	p.usage = value
	return p
}

func (p *Parser) PrefixChars(values []string) *Parser {

	for _, prefix := range values {
		if len(prefix) == 0 {
			p.err = errors.New("invalid PrefixChars() prefix cannot be empty")
			return p
		}
		if regexInValidPrefixChars.MatchString(prefix) {
			p.err = fmt.Errorf("invalid PrefixChars() '%s';"+
				" alpha, underscore and whitespace not allowed", prefix)
		}
	}
	p.prefixChars = values
	return p
}

func (p *Parser) SetHelpIO(file *os.File) {
	p.helpIO = file
}

func (p *Parser) info(format string, args ...interface{}) {
	if p.log != nil {
		p.log.Printf(format, args...)
	}
}

func (p *Parser) validateRules() error {
	var greedyRule *Rule
	for idx, rule := range p.rules {
		// Duplicate rule check
		next := idx + 1
		if next < len(p.rules) {
			for ; next < len(p.rules); next++ {
				// If the name and groups are the same
				if rule.Name == p.rules[next].Name && rule.Group == p.rules[next].Group {
					return errors.Errorf("Duplicate argument or flag '%s' defined", rule.Name)
				}
				// If the alias is a duplicate
				for _, alias := range p.rules[next].Aliases {
					var duplicate string

					// if rule.Aliases contains 'alias'
					for _, item := range rule.Aliases {
						if item == alias {
							duplicate = alias
						}
					}
					if len(duplicate) != 0 {
						return errors.Errorf("Duplicate alias '%s' for '%s' redefined by '%s'",
							duplicate, rule.Name, p.rules[next].Name)
					}
				}
				if rule.Name == p.rules[next].Name && rule.Group == p.rules[next].Group {
					return errors.Errorf("Duplicate argument or flag '%s' defined", rule.Name)
				}
			}
		}
		// Ensure user didn't set a bad default value
		if rule.Cast != nil && rule.Default != nil {
			_, err := rule.Cast(rule.Name, nil, *rule.Default)
			if err != nil {
				return errors.Wrap(err, "Bad default value")
			}
		}
		// Check for invalid option and argument names
		if regexInValidRuleName.MatchString(rule.Name) {
			if !strings.HasPrefix(rule.Name, "!cmd-") {
				return errors.Errorf("Bad argument or flag '%s'; contains invalid characters",
					rule.Name)
			}
		}

		if !rule.HasFlag(IsArgument) {
			continue
		}
		// If we already found a greedy rule, no other argument should follow
		if greedyRule != nil {
			return errors.Errorf("'%s' is ambiguous when following greedy argument '%s'",
				rule.Name, greedyRule.Name)
		}
		// Check for ambiguous greedy arguments
		if rule.HasFlag(IsGreedy) {
			if greedyRule == nil {
				greedyRule = rule
			}
		}
	}
	return nil
}

func (p *Parser) InGroup(group string) *RuleModifier {
	return NewRuleModifier(p).InGroup(group)
}

func (p *Parser) AddConfigGroup(group string) *RuleModifier {
	return NewRuleModifier(p).AddConfigGroup(group)
}

func (p *Parser) AddFlag(name string) *RuleModifier {
	rule := newRule()
	rule.SetFlag(IsFlag)
	return p.addRule(name, newRuleModifier(rule, p))
}

func (p *Parser) AddConfig(name string) *RuleModifier {
	rule := newRule()
	rule.SetFlag(IsConfig)
	return p.addRule(name, newRuleModifier(rule, p))
}

func (p *Parser) AddArgument(name string) *RuleModifier {
	rule := newRule()
	p.posCount++
	rule.Order = p.posCount
	rule.SetFlag(IsArgument)
	return p.addRule(name, newRuleModifier(rule, p))
}

func (p *Parser) AddCommand(name string, cmdFunc CommandFunc) *RuleModifier {
	rule := newRule()
	rule.SetFlag(IsCommand)
	rule.CommandFunc = cmdFunc
	rule.Action = func(rule *Rule, alias string, args []string, idx *int) error {
		return nil
	}
	// Make a new RuleModifier using self as the template
	return p.addRule(name, newRuleModifier(rule, p))
}

func (p *Parser) addRule(name string, modifier *RuleModifier) *RuleModifier {
	rule := modifier.GetRule()
	// Apply the Environment Prefix to all new rules
	rule.EnvPrefix = p.envPrefix
	// Add to the aliases we can match on the command line and return the name of the alias added
	rule.Name = rule.AddAlias(name, p.prefixChars)
	// Append the rule our list of rules
	p.rules = append(p.rules, rule)
	return modifier
}

// Returns the current list of rules for this parser. If you want to modify a rule
// use GetRule() instead.
func (p *Parser) GetRules() Rules {
	return p.rules
}

// Allow the user to modify an existing parser rule
//	parser := args.NewParser()
//	parser.AddOption("--endpoint").Default("http://localhost:19092")
//	parser.AddOption("--grpc").IsTrue()
// 	opts := parser.`ParseSimple(nil)
//
//	if opts.Bool("grpc") && !opts.WasSeen("endpoint") {
//		parser.ModifyRule("endpoint").SetDefault("localhost:19091")
//		opts = parser.ParseSimple(nil)
//	}
func (p *Parser) ModifyRule(name string) *RuleModifier {
	for _, rule := range p.rules {
		if rule.Name == name {
			return newRuleModifier(rule, p)
		}
	}
	return nil
}

// Allow the user to inspect a parser rule
func (p *Parser) GetRule(name string) *Rule {
	for _, rule := range p.rules {
		if rule.Name == name {
			return rule
		}
	}
	return nil
}

func (p *Parser) ParseAndRun(args []string, data interface{}) (int, error) {
	opts, err := p.Parse(args)

	// NOTE: Always check for help before handling errors, the user should always
	// be able ask for help regardless of validation or parse errors
	// TODO: Add this note to the args.Parse() documentation

	// If user asked for help without specifying a command
	// print the root parsers help and exit
	if opts.Bool("help") && len(opts.SubCommands()) == 0 {
		p.PrintHelp()
		return 0, nil
	}

	// If there was a validation or parse error
	if err != nil {
		return 1, err
	}

	return p.RunCommand(data)
}

// Run the command chosen via the command line, err != nil
// if no command was found on the commandline
func (p *Parser) RunCommand(data interface{}) (int, error) {
	// If user didn't provide a command via the commandline
	if p.command == nil {
		p.PrintHelp()
		return 1, nil
	}

	parser := p.SubParser()
	retCode, err := p.command.CommandFunc(parser, data)
	return retCode, err
}

// Parses the command line and prints errors and help if needed
// if user asked for --help print the help message and return nil.
// if there was an error parsing, print the error to stderr and return ni
//	opts := parser.ParseSimple(nil)
//	if opts != nil {
//		return 0, nil
//	}
func (p *Parser) ParseSimple(args []string) *Options {
	opts, err := p.Parse(args)

	// If user asked for help without specifying a command
	// print the root parsers help and exit
	if opts.Bool("help") && len(opts.SubCommands()) == 0 {
		p.PrintHelp()
		return nil
	}
	// Print errors to stderr and include our help message
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return nil
	}
	return opts
}

// Parse the commandline, but also print help and exit if the user asked for --help
func (p *Parser) ParseOrExit(args []string) *Options {
	opt, err := p.Parse(args)

	// We could have a non critical error, in addition to the user asking for help
	if opt != nil && opt.Bool("help") {
		p.PrintHelp()
		os.Exit(0)
	}
	// Print errors to stderr and include our help message
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	return opt
}

// Parses command line arguments using os.Args if 'args' is nil
func (p *Parser) Parse(args []string) (*Options, error) {
	if args != nil {
		// Make a copy of the args we are parsing
		p.args = copyStringSlice(args)
	} else if args == nil && p.parent == nil {
		// If args is nil and we are not a sub parser
		p.args = copyStringSlice(os.Args[1:])
	}

	// If the user asked us to add 'help' flag and 'help' is not already defined by the user
	if p.addHelp && !p.HasHelpFlag() {
		// Add help flag if --help or -h are not already taken by other options
		p.AddFlag("--help").Alias("-h").IsTrue().Help("Display this help message and exit")
	}
	return p.parseUntil("--")
}

// Return true if the parser has the 'help' flag defined
func (p *Parser) HasHelpFlag() bool {
	for _, rule := range p.rules {
		if rule.Name == "help" {
			return true
		}
	}
	return false
}

func (p *Parser) parseUntil(terminator string) (*Options, error) {
	p.idx = 0
	empty := p.NewOptions()

	// Sanity Check
	if len(p.rules) == 0 {
		return empty, errors.New("Must create some options to match with before calling arg.Parse()")
	}

	if err := p.validateRules(); err != nil {
		return empty, err
	}

	// Sort the rules so positional rules are parsed last
	sort.Sort(p.rules)

	// Process command line arguments until we find our terminator
	for ; p.idx < len(p.args); p.idx++ {

		if p.args[p.idx] == terminator {
			goto Apply
		}
		// Match our arguments with rules expected
		//fmt.Printf("====== Attempting to match: %d:%s - ", p.idx, p.args[p.idx])

		// Some options have arguments, this is the idx of the option if it matches
		startIdx := p.idx
		rule, err := p.matchRules(p.rules)
		if err != nil {
			// errors here are cast and expected argument errors;
			// users should never expect *Options to be nil
			return empty, err
		}
		if rule == nil {
			continue
		}
		//fmt.Printf("Found rule - %+v\n", rule)

		// TODO: I don't think this is useful anymore, consider removing this feature
		// Remove the argument so a sub parser won't parse it again, this avoids confusing behavior
		// for sub parsers. IE: [prog -o option sub-command -o option] the first -o will not
		// be confused with the second -o since we remove it from args here
		p.args = append(p.args[:startIdx], p.args[p.idx+1:]...)
		p.idx += startIdx - (p.idx + 1)

		// If we matched a command
		if rule.HasFlag(IsCommand) {
			// If we already found a command token on the commandline
			if p.command != nil {
				// Ignore this match, it must be a sub command or a positional argument
				rule.ClearFlag(WasSeenInArgv)
			}
			p.command = rule
			// If user asked us to stop parsing arguments after finding a command
			// This might be useful if the user wants arguments found before the command
			// to apply only to the parent processor
			if p.stopParsingOnCommand {
				goto Apply
			}
		}
	}
Apply:
	opts, err := p.Apply(nil)
	// TODO: Wrap post parsing validation stuff into a method
	// TODO: This should include the isRequired check
	// return p.PostValidation(p.Apply(nil))
	return opts, err
}

// Gather all the values from our rules, then apply the passed in options to any rules that don't have a computed value.
func (p *Parser) Apply(values *Options) (*Options, error) {
	results := p.NewOptions()

	// for each of the rules
	for _, rule := range p.rules {
		// Get the computed value
		value, err := rule.ComputedValue(values)
		if err != nil {
			p.err = err
			continue
		}

		// If we have a Store() for this rule apply it here
		if rule.StoreValue != nil {
			rule.StoreValue(value)
		}

		// Special Case here for Config Groups
		if rule.HasFlag(IsConfigGroup) && values != nil {
			for _, key := range values.Group(rule.Group).Keys() {
				value := values.Group(rule.Group).Get(key)
				results.Group(rule.Group).SetWithRule(key, value, rule)
			}
		} else {
			results.Group(rule.Group).SetWithRule(rule.Name, value, rule)

			// Choices check
			if rule.Choices != nil {
				strValue := results.Group(rule.Group).String(rule.Name)
				if !containsString(strValue, rule.Choices) {
					err := errors.Errorf("'%s' is an invalid argument for '%s' choose from (%s)",
						strValue, rule.Name, strings.Join(rule.Choices, ", "))
					return results, err
				}
			}
		}
	}

	p.setOpts(results)
	return p.GetOpts(), p.err
}

// Return the parent parser if there is one, else return nil
func (p *Parser) Parent() *Parser {
	return p.parent
}

// Build a list of parent parsers for this sub parser, return empty list if this parser is the root parser
func (p *Parser) Parents() []*Parser {
	result := make([]*Parser, 0)
	findSubParsers(p, &result)
	return result
}

func findSubParsers(parser *Parser, result *[]*Parser) {
	if parser == nil {
		return
	}
	// Push the parser to then beginning of the slice
	*result = append([]*Parser{parser}, *result...)
	findSubParsers(parser.parent, result)
}

func (p *Parser) setOpts(options *Options) {
	commands := p.SubCommands()
	options.SetSubCommands(commands)

	p.mutex.Lock()
	p.options = options
	p.mutex.Unlock()
}

// Build a list of sub commands that the user provided to reach this sub parser;
// return an empty list if this parser has no sub commands associated with it
func (p *Parser) SubCommands() []string {
	var results []string
	for _, parser := range p.Parents() {
		if parser.command == nil {
			return results
		}
		// Get the command name, without the special naming
		results = append(results, strings.Trim(parser.command.Name, "!cmd-"))
	}
	return results
}

func (p *Parser) GetOpts() *Options {
	p.mutex.Lock()
	defer func() {
		p.mutex.Unlock()
	}()
	return p.options
}

// Return the un-parsed portion of the argument array. These are arguments that where not
// matched by any AddOption() or AddArgument() rules defined by the user.
func (p *Parser) GetArgs() []string {
	return copyStringSlice(p.args)
}

func (p *Parser) matchRules(rules Rules) (*Rule, error) {
	// Find a Rule that matches this argument
	for _, rule := range rules {
		matched, err := rule.Match(p.args, &p.idx)
		// If no rule was matched
		if !matched {
			continue
		}
		return rule, err
	}
	// No Rules matched our arguments and there was no error
	return nil, nil
}

func (p *Parser) PrintRules() {
	for _, rule := range p.rules {
		fmt.Printf("Rule: %s - '%+v'\n", rule.Name, rule)
	}
}

func (p *Parser) PrintHelp() {
	fmt.Fprintln(p.helpIO, p.GenerateHelp())
}

func (p *Parser) GenerateHelp() string {
	var result bytes.Buffer
	// TODO: Improve this once we have arguments
	// Super generic usage message
	if p.usage != "" {
		result.WriteString(fmt.Sprintf("Usage: %s\n", p.usage))
	} else {
		result.WriteString(fmt.Sprintf("Usage: %s %s %s\n", p.name,
			p.generateUsage(IsFlag),
			p.generateUsage(IsArgument)))
	}

	if p.description != "" {
		result.WriteString("\n")

		if hasFlags(p.flags, IsFormatted) {
			result.WriteString(p.description)
		} else {
			result.WriteString(WordWrap(p.description, 0, 80))
		}
		result.WriteString("\n")
	}

	commands := p.generateHelpSection(IsCommand)
	if commands != "" {
		result.WriteString("\nCommands:\n")
		result.WriteString(commands)
	}

	argument := p.generateHelpSection(IsArgument)
	if argument != "" {
		result.WriteString("\nArguments:\n")
		result.WriteString(argument)
	}

	options := p.generateHelpSection(IsFlag)
	if options != "" {
		result.WriteString("\nOptions:\n")
		result.WriteString(options)
	}

	if p.epilog != "" {
		result.WriteString(p.epilog)
	}
	return result.String()
}

func (p *Parser) generateUsage(flags RuleFlag) string {
	var result bytes.Buffer

	// TODO: Should only return [OPTIONS] if there are too many options to display on a single line
	if flags == IsFlag {
		return "[OPTIONS]"
	}

	for _, rule := range p.rules {
		if !rule.HasFlag(flags) {
			continue
		}
		result.WriteString(" " + rule.GenerateUsage())
	}
	return result.String()
}

type HelpMsg struct {
	Flags   string
	Message string
}

func (p *Parser) generateHelpSection(flags RuleFlag) string {
	var result bytes.Buffer
	var options []HelpMsg

	// Ask each rule to generate a Help message for the options
	maxLen := 0
	for _, rule := range p.rules {
		if !rule.HasFlag(flags) {
			continue
		}
		flags, message := rule.GenerateHelp()
		if len(flags) > maxLen {
			maxLen = len(flags)
		}
		options = append(options, HelpMsg{flags, message})
	}

	// Set our indent length
	indent := maxLen + 3
	flagFmt := fmt.Sprintf("%%-%ds%%s\n", indent)

	for _, opt := range options {
		message := WordWrap(opt.Message, indent, p.wordWrapLen)
		result.WriteString(fmt.Sprintf(flagFmt, opt.Flags, message))
	}
	return result.String()
}
