package args

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"sort"
	"sync"

	"github.com/fatih/structs"
	"github.com/pkg/errors"
)

var regexIsOptional = regexp.MustCompile(`^(\W+)([\w|-]*)$`)

type ParseModifier func(*ArgParser)

type ArgParser struct {
	Command              *Rule
	EnvPrefix            string
	EtcdRoot             string
	Description          string
	Name                 string
	WordWrap             int
	IsSubParser          bool
	StopParsingOnCommand bool
	HelpIO               *os.File
	helpAdded            bool
	mutex                sync.Mutex
	addHelp              bool
	args                 []string
	options              *Options
	rules                Rules
	err                  error
	idx                  int
	posCount             int
	attempts             int
	log                  StdLogger
}

// Creates a new instance of the argument parser
func NewParser(modifiers ...ParseModifier) *ArgParser {
	parser := &ArgParser{
		WordWrap: 200,
		mutex:    sync.Mutex{},
		log:      DefaultLogger,
		addHelp:  true,
		HelpIO:   os.Stdout,
	}
	for _, modify := range modifiers {
		modify(parser)
	}
	return parser
}

// Takes the current parser and return a new parser with
// appropriate for use within a command function
func (self *ArgParser) SubParser() *ArgParser {
	parser := NewParser()
	src := structs.New(self)
	dest := structs.New(parser)

	// Copy all the public values
	for _, field := range src.Fields() {
		if field.IsExported() {
			dest.Field(field.Name()).Set(field.Value())
		}
	}

	parser.args = self.args
	parser.rules = self.rules
	parser.log = self.log
	parser.helpAdded = self.helpAdded
	parser.addHelp = self.addHelp
	parser.options = self.options

	// Remove all Commands from our rules
	for i := len(parser.rules) - 1; i >= 0; i-- {
		if parser.rules[i].HasFlags(IsCommand) {
			// Remove the rule
			parser.rules = append(parser.rules[:i], parser.rules[i+1:]...)
		}
	}
	// Clear the selected Commands
	parser.Command = nil
	parser.IsSubParser = true
	return parser
}

func (self *ArgParser) SetLog(logger StdLogger) {
	self.log = logger
}

func (self *ArgParser) GetLog() StdLogger {
	return self.log
}

func (self *ArgParser) error(format string, args ...interface{}) {
	if self.log != nil {
		self.log.Printf(format, args...)
	}
}

func (self *ArgParser) ValidateRules() error {
	for idx, rule := range self.rules {
		// Duplicate rule check
		next := idx + 1
		if next < len(self.rules) {
			for ; next < len(self.rules); next++ {
				// If the name and groups are the same
				if rule.Name == self.rules[next].Name && rule.Group == self.rules[next].Group {
					return errors.New(fmt.Sprintf("Duplicate option '%s' defined", rule.Name))
				}
			}
		}
		// Ensure user didn't set a bad default value
		if rule.Cast != nil && rule.Default != nil {
			_, err := rule.Cast(rule.Name, *rule.Default)
			if err != nil {
				return errors.Wrap(err, "Bad default value")
			}
		}
	}
	return nil
}

func (self *ArgParser) InGroup(group string) *RuleModifier {
	return NewRuleModifier(self).InGroup(group)
}

func (self *ArgParser) AddConfigGroup(group string) *RuleModifier {
	return NewRuleModifier(self).AddConfigGroup(group)
}

func (self *ArgParser) Opt(name string) *RuleModifier {
	return self.AddOption(name)
}

func (self *ArgParser) AddOption(name string) *RuleModifier {
	rule := newRule()
	rule.SetFlags(IsOption)
	return self.AddRule(name, newRuleModifier(rule, self))
}

func (self *ArgParser) Cfg(name string) *RuleModifier {
	return self.AddConfig(name)
}

func (self *ArgParser) AddConfig(name string) *RuleModifier {
	rule := newRule()
	rule.SetFlags(IsConfig)
	return self.AddRule(name, newRuleModifier(rule, self))
}

func (self *ArgParser) AddPositional(name string) *RuleModifier {
	rule := newRule()
	self.posCount++
	rule.Order = self.posCount
	rule.SetFlags(IsPositional)
	rule.NotGreedy = true
	return self.AddRule(name, newRuleModifier(rule, self))
}

func (self *ArgParser) AddCommand(name string, cmdFunc CommandFunc) *RuleModifier {
	rule := newRule()
	rule.SetFlags(IsCommand)
	rule.CommandFunc = cmdFunc
	rule.Action = func(rule *Rule, alias string, args []string, idx *int) error {
		return nil
	}
	// Make a new RuleModifier using self as the template
	return self.AddRule(name, newRuleModifier(rule, self))
}

func (self *ArgParser) AddRule(name string, modifier *RuleModifier) *RuleModifier {
	rule := modifier.GetRule()

	// Apply the Environment Prefix to all new rules
	rule.EnvPrefix = self.EnvPrefix

	// If name begins with a non word character, assume it's an optional argument
	if regexIsOptional.MatchString(name) {
		// Attempt to extract the name
		group := regexIsOptional.FindStringSubmatch(name)
		if group == nil {
			panic(fmt.Sprintf("Invalid optional argument name '%s'", name))
		} else {
			rule.Aliases = append(rule.Aliases, name)
			rule.Name = group[2]
		}
	} else {
		if rule.HasFlags(IsCommand) {
			rule.Aliases = append(rule.Aliases, name)
		}
		rule.Name = name
	}
	// Append the rule our list of rules
	self.rules = append(self.rules, rule)
	return modifier
}

func (self *ArgParser) GetRules() Rules {
	return self.rules
}

func (self *ArgParser) ParseAndRun(args *[]string, data interface{}) (int, error) {
	_, err := self.ParseArgs(args)
	if err != nil {
		return -1, err
	}
	return self.RunCommand(data)
}

// Run the command chosen via the command line, err != nil
// if no command was found on the commandline
func (self *ArgParser) RunCommand(data interface{}) (int, error) {
	// If user didn't provide a command via the commandline
	if self.Command == nil {
		self.PrintHelp()
		return -1, nil
	}

	parser := self.SubParser()
	retCode := self.Command.CommandFunc(parser, data)
	return retCode, nil
}

func (self *ArgParser) HasHelpOption() bool {
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

// Will parse the commandline, but also print help and exit if the user asked for --help
func (self *ArgParser) ParseArgsSimple(args *[]string) *Options {
	opt, err := self.ParseArgs(args)

	// We could have a non critical error, in addition to the user asking for help
	if opt != nil && opt.Bool("help") {
		self.PrintHelp()
		os.Exit(1)
	}
	// Print errors to stderr and include our help message
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		self.PrintHelp()
		os.Exit(-1)
	}
	return opt
}

// Parses command line arguments using os.Args if 'args' is nil
func (self *ArgParser) ParseArgs(args *[]string) (*Options, error) {
	if args != nil {
		self.args = *args
	} else if args == nil && !self.IsSubParser {
		// If args is nil and we are not a subparser
		self.args = os.Args[1:]
	}

	if self.addHelp && !self.HasHelpOption() {
		// Add help option if --help or -h are not already taken by other options
		self.AddOption("--help").Alias("-h").IsTrue().Help("Display this help message and exit")
		self.helpAdded = true
	}
	return self.parseUntil(self.args, "--")
}

func (self *ArgParser) parseUntil(args []string, terminator string) (*Options, error) {
	self.idx = 0

	// Sanity Check
	if len(self.rules) == 0 {
		return nil, errors.New("Must create some options to match with args.AddOption()" +
			" before calling arg.ParseArgs()")
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
		rule, err := self.match(self.rules)
		if err != nil {
			return nil, err
		}
		if rule != nil {
			//fmt.Printf("Found rule - %+v\n", rule)
			// If we matched a command
			if rule.HasFlags(IsCommand) {
				// If we already found a command token on the commandline
				if self.Command != nil {
					// Ignore this match, it must be a sub command or a positional argument
					rule.Seen = false
				}
				self.Command = rule
				// Remove the command argument so we don't process it again in our sub parser
				self.args = append(self.args[:self.idx], self.args[self.idx+1:]...)
				self.idx--
				// If user asked us to stop parsing arguments after finding a command
				// This might be useful if the user wants arguments found before the command
				// to apply only to the parent processor
				if self.StopParsingOnCommand {
					goto Apply
				}
			}
		}
	}
Apply:
	opts, err := self.Apply(nil)
	// TODO: Wrap post parsing validation stuff into a method
	// TODO: This should include the isRequired check
	// return self.PostValidation(self.Apply(nil))

	// If we specified a command, we probably do not want help even if we see -h on the commandline
	if self.helpAdded && opts.Bool("help") && self.Command == nil {
		return opts, &HelpError{}
	}
	return opts, err
}

// Gather all the values from our rules, then apply the passed in map to any rules that don't have a computed value.
func (self *ArgParser) Apply(values *Options) (*Options, error) {
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
		if rule.HasFlags(IsConfigGroup) && values != nil {
			for _, key := range values.Group(rule.Group).Keys() {
				value := values.Group(rule.Group).Get(key)
				results.Group(rule.Group).SetSeen(key, value, rule.Seen)
			}
		} else {
			results.Group(rule.Group).SetSeen(rule.Name, value, rule.Seen)
		}
	}

	self.SetOpts(results)
	return self.GetOpts(), self.err
}

func (self *ArgParser) SetOpts(options *Options) {
	self.mutex.Lock()
	self.options = options
	self.mutex.Unlock()
}

func (self *ArgParser) GetOpts() *Options {
	self.mutex.Lock()
	defer func() {
		self.mutex.Unlock()
	}()
	return self.options
}

func (self *ArgParser) match(rules Rules) (*Rule, error) {
	// Find a Rule that matches this argument
	for _, rule := range rules {
		matched, err := rule.Match(self.args, &self.idx)
		if err != nil {
			// This Rule did match our argument but had an error
			return rule, err
		}
		if matched {
			//fmt.Printf("Matched '%s' with '%s'\n", rule.Name, rule.Value)
			return rule, nil
		}
	}
	// No Rules matched our arguments and there was no error
	return nil, nil
}

func (self *ArgParser) PrintRules() {
	for _, rule := range self.rules {
		fmt.Printf("Rule: %s - '%+v'\n", rule.Name, rule)
	}
}

func (self *ArgParser) PrintHelp() {
	fmt.Fprintln(self.HelpIO, self.GenerateHelp())
}

func (self *ArgParser) GenerateHelp() string {
	var result bytes.Buffer
	// TODO: Improve this once we have positional arguments
	// Super generic usage message
	result.WriteString(fmt.Sprintf("Usage: %s %s %s\n", self.Name,
		self.GenerateUsage(IsOption),
		self.GenerateUsage(IsPositional)))
	result.WriteString("\n")
	result.WriteString(WordWrap(self.Description, 0, 80))
	result.WriteString("\n")

	commands := self.GenerateHelpSection(IsCommand)
	if commands != "" {
		result.WriteString("\nCommands:\n")
		result.WriteString(commands)
	}

	positional := self.GenerateHelpSection(IsPositional)
	if positional != "" {
		result.WriteString("\nPositionals:\n")
		result.WriteString(positional)
	}

	options := self.GenerateHelpSection(IsOption)
	if options != "" {
		result.WriteString("\nOptions:\n")
		result.WriteString(options)
	}
	return result.String()
}

func (self *ArgParser) GenerateUsage(flags int64) string {
	var result bytes.Buffer

	// TODO: Should only return [OPTIONS] if there are too many options
	// TODO: to display on a single line
	if flags == IsOption {
		return "[OPTIONS]"
	}

	for _, rule := range self.rules {
		if !rule.HasFlags(flags) {
			continue
		}
		result.WriteString(rule.GenerateUsage() + " ")
	}
	return result.String()
}

type HelpMsg struct {
	Flags   string
	Message string
}

func (self *ArgParser) GenerateHelpSection(flags int64) string {
	var result bytes.Buffer
	var options []HelpMsg

	// Ask each rule to generate a Help message for the options
	maxLen := 0
	for _, rule := range self.rules {
		if !rule.HasFlags(flags) {
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
		message := WordWrap(opt.Message, indent, self.WordWrap)
		result.WriteString(fmt.Sprintf(flagFmt, opt.Flags, message))
	}
	return result.String()
}
