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
func (self *Parser) SubParser() *Parser {
	parser := NewParser()
	src := structs.New(self)
	dest := structs.New(parser)

	// Copy all the public values
	for _, field := range src.Fields() {
		if field.IsExported() {
			dest.Field(field.Name()).Set(field.Value())
		}
	}

	parser.addHelp = self.addHelp
	parser.options = self.options
	parser.rules = self.rules
	parser.args = self.args
	parser.parent = self
	parser.log = self.log

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

func (self *Parser) Log(logger StdLogger) {
	self.log = logger
}

func (self *Parser) GetLog() StdLogger {
	return self.log
}

func (self *Parser) Name(value string) *Parser {
	self.name = value
	return self
}

func (self *Parser) GetName() string {
	return self.name
}

func (self *Parser) Desc(value string, flags ...ParseFlag) *Parser {
	self.description = value
	for _, flag := range flags {
		setFlags(&self.flags, flag)
	}
	return self
}

func (self *Parser) GetDesc() string {
	return self.description
}

func (self *Parser) WordWrap(value int) *Parser {
	self.wordWrapLen = value
	return self
}

func (self *Parser) GetWordWrap() int {
	return self.wordWrapLen
}

func (self *Parser) EnvPrefix(value string) *Parser {
	self.envPrefix = value
	return self
}

func (self *Parser) GetEnvPrefix() string {
	return self.envPrefix
}

func (self *Parser) AddHelp(value bool) *Parser {
	self.addHelp = value
	return self
}

func (self *Parser) GetAddHelp() bool {
	return self.addHelp
}

func (self *Parser) Epilog(value string) *Parser {
	self.epilog = value
	return self
}

func (self *Parser) GetEpilog() string {
	return self.epilog
}

func (self *Parser) Usage(value string) *Parser {
	self.usage = value
	return self
}

func (self *Parser) GetUsage() string {
	return self.usage
}

func (self *Parser) PrefixChars(values []string) *Parser {

	for _, prefix := range values {
		if len(prefix) == 0 {
			self.err = errors.New("invalid PrefixChars() prefix cannot be empty")
			return self
		}
		if regexInValidPrefixChars.MatchString(prefix) {
			self.err = fmt.Errorf("invalid PrefixChars() '%s';"+
				" alpha, underscore and whitespace not allowed", prefix)
		}
	}
	self.prefixChars = values
	return self
}

func (self *Parser) GetPrefixChars() []string {
	return self.prefixChars
}

func (self *Parser) SetHelpIO(file *os.File) {
	self.helpIO = file
}

func (self *Parser) info(format string, args ...interface{}) {
	if self.log != nil {
		self.log.Printf(format, args...)
	}
}

func (self *Parser) ValidateRules() error {
	var greedyRule *Rule
	for idx, rule := range self.rules {
		// Duplicate rule check
		next := idx + 1
		if next < len(self.rules) {
			for ; next < len(self.rules); next++ {
				// If the name and groups are the same
				if rule.Name == self.rules[next].Name && rule.Group == self.rules[next].Group {
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

func (self *Parser) InGroup(group string) *RuleModifier {
	return NewRuleModifier(self).InGroup(group)
}

func (self *Parser) AddConfigGroup(group string) *RuleModifier {
	return NewRuleModifier(self).AddConfigGroup(group)
}

func (self *Parser) AddFlag(name string) *RuleModifier {
	rule := newRule()
	rule.SetFlag(IsFlag)
	return self.addRule(name, newRuleModifier(rule, self))
}

func (self *Parser) AddConfig(name string) *RuleModifier {
	rule := newRule()
	rule.SetFlag(IsConfig)
	return self.addRule(name, newRuleModifier(rule, self))
}

func (self *Parser) AddArg(name string) *RuleModifier {
	return self.AddArgument(name)
}

func (self *Parser) AddArgument(name string) *RuleModifier {
	rule := newRule()
	self.posCount++
	rule.Order = self.posCount
	rule.SetFlag(IsArgument)
	return self.addRule(name, newRuleModifier(rule, self))
}

func (self *Parser) AddCommand(name string, cmdFunc CommandFunc) *RuleModifier {
	rule := newRule()
	rule.SetFlag(IsCommand)
	rule.CommandFunc = cmdFunc
	rule.Action = func(rule *Rule, alias string, args []string, idx *int) error {
		return nil
	}
	// Make a new RuleModifier using self as the template
	return self.addRule(name, newRuleModifier(rule, self))
}

func (self *Parser) addRule(name string, modifier *RuleModifier) *RuleModifier {
	rule := modifier.GetRule()
	// Apply the Environment Prefix to all new rules
	rule.EnvPrefix = self.envPrefix
	// Add to the aliases we can match on the command line and return the name of the alias added
	rule.Name = rule.AddAlias(name, self.prefixChars)
	// Append the rule our list of rules
	self.rules = append(self.rules, rule)
	return modifier
}

// Returns the current list of rules for this parser. If you want to modify a rule
// use GetRule() instead.
func (self *Parser) GetRules() Rules {
	return self.rules
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
func (self *Parser) ModifyRule(name string) *RuleModifier {
	for _, rule := range self.rules {
		if rule.Name == name {
			return newRuleModifier(rule, self)
		}
	}
	return nil
}

// Allow the user to inspect a parser rule
func (self *Parser) GetRule(name string) *Rule {
	for _, rule := range self.rules {
		if rule.Name == name {
			return rule
		}
	}
	return nil
}

func (self *Parser) ParseAndRun(args []string, data interface{}) (int, error) {
	opts, err := self.Parse(args)

	// NOTE: Always check for help before handling errors, the user should always
	// be able ask for help regardless of validation or parse errors
	// TODO: Add this note to the args.Parse() documentation

	// If user asked for help without specifying a command
	// print the root parsers help and exit
	if opts.Bool("help") && len(opts.SubCommands()) == 0 {
		self.PrintHelp()
		return 0, nil
	}

	// If there was a validation or parse error
	if err != nil {
		return 1, err
	}

	return self.RunCommand(data)
}

// Run the command chosen via the command line, err != nil
// if no command was found on the commandline
func (self *Parser) RunCommand(data interface{}) (int, error) {
	// If user didn't provide a command via the commandline
	if self.command == nil {
		self.PrintHelp()
		return 1, nil
	}

	parser := self.SubParser()
	retCode, err := self.command.CommandFunc(parser, data)
	return retCode, err
}

// Parses the command line and prints errors and help if needed
// if user asked for --help print the help message and return nil.
// if there was an error parsing, print the error to stderr and return ni
//	opts := parser.ParseSimple(nil)
//	if opts != nil {
//		return 0, nil
//	}
func (self *Parser) ParseSimple(args []string) *Options {
	opts, err := self.Parse(args)

	// If user asked for help without specifying a command
	// print the root parsers help and exit
	if opts.Bool("help") && len(opts.SubCommands()) == 0 {
		self.PrintHelp()
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
func (self *Parser) ParseOrExit(args []string) *Options {
	opt, err := self.Parse(args)

	// We could have a non critical error, in addition to the user asking for help
	if opt != nil && opt.Bool("help") {
		self.PrintHelp()
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
func (self *Parser) Parse(args []string) (*Options, error) {
	if args != nil {
		// Make a copy of the args we are parsing
		self.args = copyStringSlice(args)
	} else if args == nil && self.parent == nil {
		// If args is nil and we are not a sub parser
		self.args = copyStringSlice(os.Args[1:])
	}

	// If the user asked us to add 'help' flag and 'help' is not already defined by the user
	if self.addHelp && !self.HasHelpFlag() {
		// Add help flag if --help or -h are not already taken by other options
		self.AddFlag("--help").Alias("-h").IsTrue().Help("Display this help message and exit")
	}
	return self.parseUntil("--")
}

// Return true if the parser has the 'help' flag defined
func (self *Parser) HasHelpFlag() bool {
	for _, rule := range self.rules {
		if rule.Name == "help" {
			return true
		}
	}
	return false
}

func (self *Parser) parseUntil(terminator string) (*Options, error) {
	self.idx = 0

	// Sanity Check
	if len(self.rules) == 0 {
		return nil, errors.New("Must create some options to match with before calling arg.Parse()")
	}

	if err := self.ValidateRules(); err != nil {
		return nil, err
	}

	// Sort the rules so positional rules are parsed last
	sort.Sort(self.rules)

	// Process command line arguments until we find our terminator
	for ; self.idx < len(self.args); self.idx++ {

		if self.args[self.idx] == terminator {
			goto Apply
		}
		// Match our arguments with rules expected
		//fmt.Printf("====== Attempting to match: %d:%s - ", self.idx, self.args[self.idx])

		// Some options have arguments, this is the idx of the option if it matches
		startIdx := self.idx
		rule, err := self.matchRules(self.rules)
		if err != nil {
			// errors here are cast and expected argument errors;
			// users should never expect *Options to be nil
			return self.NewOptions(), err
		}
		if rule == nil {
			continue
		}
		//fmt.Printf("Found rule - %+v\n", rule)

		// TODO: I don't think this is useful anymore, consider removing this feature
		// Remove the argument so a sub parser won't parse it again, this avoids confusing behavior
		// for sub parsers. IE: [prog -o option sub-command -o option] the first -o will not
		// be confused with the second -o since we remove it from args here
		self.args = append(self.args[:startIdx], self.args[self.idx+1:]...)
		self.idx += startIdx - (self.idx + 1)

		// If we matched a command
		if rule.HasFlag(IsCommand) {
			// If we already found a command token on the commandline
			if self.command != nil {
				// Ignore this match, it must be a sub command or a positional argument
				rule.ClearFlag(WasSeenInArgv)
			}
			self.command = rule
			// If user asked us to stop parsing arguments after finding a command
			// This might be useful if the user wants arguments found before the command
			// to apply only to the parent processor
			if self.stopParsingOnCommand {
				goto Apply
			}
		}
	}
Apply:
	opts, err := self.Apply(nil)
	// TODO: Wrap post parsing validation stuff into a method
	// TODO: This should include the isRequired check
	// return self.PostValidation(self.Apply(nil))
	return opts, err
}

// Gather all the values from our rules, then apply the passed in options to any rules that don't have a computed value.
func (self *Parser) Apply(values *Options) (*Options, error) {
	results := self.NewOptions()

	// for each of the rules
	for _, rule := range self.rules {
		// Get the computed value
		value, err := rule.ComputedValue(values)
		if err != nil {
			self.err = err
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

	self.SetOpts(results)
	return self.GetOpts(), self.err
}

// Return the parent parser if there is one, else return nil
func (self *Parser) Parent() *Parser {
	return self.parent
}

// Build a list of parent parsers for this sub parser, return empty list if this parser is the root parser
func (self *Parser) Parents() []*Parser {
	result := make([]*Parser, 0)
	findSubParsers(self, &result)
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

func (self *Parser) SetOpts(options *Options) {
	commands := self.SubCommands()
	options.SetSubCommands(commands)

	self.mutex.Lock()
	self.options = options
	self.mutex.Unlock()
}

// Build a list of sub commands that the user provided to reach this sub parser;
// return an empty list if this parser has no sub commands associated with it
func (self *Parser) SubCommands() []string {
	var results []string
	for _, parser := range self.Parents() {
		if parser.command == nil {
			return results
		}
		// Get the command name, without the special naming
		results = append(results, strings.Trim(parser.command.Name, "!cmd-"))
	}
	return results
}

func (self *Parser) GetOpts() *Options {
	self.mutex.Lock()
	defer func() {
		self.mutex.Unlock()
	}()
	return self.options
}

// Return the un-parsed portion of the argument array. These are arguments that where not
// matched by any AddOption() or AddArgument() rules defined by the user.
func (self *Parser) GetArgs() []string {
	return copyStringSlice(self.args)
}

func (self *Parser) matchRules(rules Rules) (*Rule, error) {
	// Find a Rule that matches this argument
	for _, rule := range rules {
		matched, err := rule.Match(self.args, &self.idx)
		// If no rule was matched
		if !matched {
			continue
		}
		return rule, err
	}
	// No Rules matched our arguments and there was no error
	return nil, nil
}

func (self *Parser) PrintRules() {
	for _, rule := range self.rules {
		fmt.Printf("Rule: %s - '%+v'\n", rule.Name, rule)
	}
}

func (self *Parser) PrintHelp() {
	fmt.Fprintln(self.helpIO, self.GenerateHelp())
}

func (self *Parser) GenerateHelp() string {
	var result bytes.Buffer
	// TODO: Improve this once we have arguments
	// Super generic usage message
	if self.usage != "" {
		result.WriteString(fmt.Sprintf("Usage: %s\n", self.usage))
	} else {
		result.WriteString(fmt.Sprintf("Usage: %s %s %s\n", self.name,
			self.GenerateUsage(IsFlag),
			self.GenerateUsage(IsArgument)))
	}

	if self.description != "" {
		result.WriteString("\n")

		if hasFlags(self.flags, IsFormatted) {
			result.WriteString(self.description)
		} else {
			result.WriteString(WordWrap(self.description, 0, 80))
		}
		result.WriteString("\n")
	}

	commands := self.GenerateHelpSection(IsCommand)
	if commands != "" {
		result.WriteString("\nCommands:\n")
		result.WriteString(commands)
	}

	argument := self.GenerateHelpSection(IsArgument)
	if argument != "" {
		result.WriteString("\nArguments:\n")
		result.WriteString(argument)
	}

	options := self.GenerateHelpSection(IsFlag)
	if options != "" {
		result.WriteString("\nOptions:\n")
		result.WriteString(options)
	}

	if self.epilog != "" {
		result.WriteString(self.epilog)
	}
	return result.String()
}

func (self *Parser) GenerateUsage(flags RuleFlag) string {
	var result bytes.Buffer

	// TODO: Should only return [OPTIONS] if there are too many options to display on a single line
	if flags == IsFlag {
		return "[OPTIONS]"
	}

	for _, rule := range self.rules {
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

func (self *Parser) GenerateHelpSection(flags RuleFlag) string {
	var result bytes.Buffer
	var options []HelpMsg

	// Ask each rule to generate a Help message for the options
	maxLen := 0
	for _, rule := range self.rules {
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
		message := WordWrap(opt.Message, indent, self.wordWrapLen)
		result.WriteString(fmt.Sprintf(flagFmt, opt.Flags, message))
	}
	return result.String()
}
