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

var regexInValidPrefixChars = regexp.MustCompile(`\w|\s`)

type ParseModifier func(*Parser)

type Parser struct {
	stopParsingOnCommand bool
	description          string
	wordWrapLen          int
	isSubParser          bool
	prefixChars          []string
	helpAdded            bool
	envPrefix            string
	posCount             int
	attempts             int
	command              *Rule
	addHelp              bool
	options              *Options
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

	parser.helpAdded = self.helpAdded
	parser.addHelp = self.addHelp
	parser.options = self.options
	parser.rules = self.rules
	parser.args = self.args
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
	parser.isSubParser = true
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
					return errors.Errorf("Duplicate option '%s' defined", rule.Name)
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

func (self *Parser) ParseAndRun(args *[]string, data interface{}) (int, error) {
	_, err := self.Parse(args)
	if err != nil {
		if IsHelpError(err) {
			self.PrintHelp()
			return 1, nil
		}
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
func (self *Parser) ParseSimple(args *[]string) *Options {
	opt, err := self.Parse(args)

	// We could have a non critical error, in addition to the user asking for help
	if opt != nil && opt.Bool("help") {
		self.PrintHelp()
		return nil
	}
	// Print errors to stderr and include our help message
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return nil
	}
	return opt
}

// Parse the commandline, but also print help and exit if the user asked for --help
func (self *Parser) ParseOrExit(args *[]string) *Options {
	opt, err := self.Parse(args)

	// We could have a non critical error, in addition to the user asking for help
	if opt != nil && opt.Bool("help") {
		self.PrintHelp()
		os.Exit(1)
	}
	// Print errors to stderr and include our help message
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	return opt
}

// Parses command line arguments using os.Args if 'args' is nil
func (self *Parser) Parse(args *[]string) (*Options, error) {
	if args != nil {
		self.args = copyStringSlice(*args)
	} else if args == nil && !self.isSubParser {
		// If args is nil and we are not a subparser
		self.args = copyStringSlice(os.Args[1:])
	}

	if self.addHelp && !self.hasHelpOption() {
		// Add help option if --help or -h are not already taken by other options
		self.AddFlag("--help").Alias("-h").IsTrue().Help("Display this help message and exit")
		self.helpAdded = true
	}
	return self.parseUntil("--")
}

func (self *Parser) hasHelpOption() bool {
	for _, rule := range self.rules {
		if rule.Name == "help" {
			return true
		}
		for _, alias := range rule.Aliases {
			if alias == "-h" {
				return true
			}
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
			return nil, err
		}
		if rule == nil {
			continue
		}
		//fmt.Printf("Found rule - %+v\n", rule)

		// Remove the argument so a sub processor won't process it again, this avoids confusing behavior
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

	// When the user asks for --help
	if self.helpAdded && opts.Bool("help") {
		// Ignore the --help request if we see a sub command so the
		// sub command gets a change to process the --help request
		if self.command != nil {
			// root parsers that want to know if the -h option was provided
			// can still ask if the option was `WasSeen("help")`
			rule := opts.InspectOpt("help").GetRule()
			return opts.SetWithRule("help", false, rule), err
		}
		return opts, &HelpError{}
	}
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

func (self *Parser) SetOpts(options *Options) {
	self.mutex.Lock()
	self.options = options
	self.mutex.Unlock()
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
