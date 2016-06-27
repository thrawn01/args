package args

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"sync"
)

type ParseModifier func(*ArgParser)

type ArgParser struct {
	Command              *Rule
	EnvPrefix            string
	EtcdRoot             string
	Description          string
	Name                 string
	WordWrap             int
	StopParsingOnCommand bool
	mutex                sync.Mutex
	args                 []string
	options              *Options
	rules                Rules
	err                  error
	idx                  int
	attempts             int
	log                  StdLogger
}

// Creates a new instance of the argument parser
func NewParser(modifiers ...ParseModifier) *ArgParser {
	// TODO: Fix this... is stupid
	parser := &ArgParser{
		nil,
		"",
		"",
		"",
		"",
		200,
		false,
		sync.Mutex{},
		[]string{},
		nil,
		nil,
		nil,
		0,
		0,
		DefaultLogger,
	}
	for _, modify := range modifiers {
		modify(parser)
	}
	return parser
}

var isOptional = regexp.MustCompile(`^(\W+)([\w|-]*)$`)

func (self *ArgParser) SetLog(logger StdLogger) {
	self.log = logger
}

func (self *ArgParser) GetLog() StdLogger {
	return self.log
}

// Takes the current parser and return a new parser with
// any arguments already parsed removed from argv and none of the rules of the parent
func (self *ArgParser) SubParser() *ArgParser {
	return nil
}

func (self *ArgParser) ValidateRules() error {
	for idx, rule := range self.rules {
		// Duplicate rule check
		next := idx + 1
		if next < len(self.rules) {
			for ; next < len(self.rules); next++ {
				if rule.Name == self.rules[next].Name {
					return errors.New(fmt.Sprintf("Duplicate option with same name as '%s'", rule.Name))
				}
			}
		}

		// Ensure user didn't set a bad default value
		if rule.Cast != nil && rule.Default != nil {
			_, err := rule.Cast("args.Default()", *rule.Default)
			if err != nil {
				panic(err.Error())
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
	return self.AddRule(name, NewRuleModifier(self))
}

func (self *ArgParser) Cfg(name string) *RuleModifier {
	return self.AddConfig(name)
}

func (self *ArgParser) AddConfig(name string) *RuleModifier {
	rule := newRule()
	rule.IsConfig = true
	return self.AddRule(name, newRuleModifier(rule, self))
}

func (self *ArgParser) AddPositional(name string) *RuleModifier {
	rule := newRule()
	rule.IsPos++
	rule.NotGreedy = true
	return self.AddRule(name, newRuleModifier(rule, self))
}

func (self *ArgParser) AddCommand(name string, cmdFunc CommandFunc) *RuleModifier {
	rule := newRule()
	rule.Type = CommandRule
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
	if isOptional.MatchString(name) {
		// Attempt to extract the name
		group := isOptional.FindStringSubmatch(name)
		if group == nil {
			panic(fmt.Sprintf("Invalid optional argument name '%s'", name))
		} else {
			rule.Aliases = append(rule.Aliases, name)
			rule.Name = group[2]
		}
	} else {
		if rule.Type == CommandRule {
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

// Parses command line arguments using os.Args if 'args' is nil
func (self *ArgParser) ParseArgs(args *[]string) (*Options, error) {
	if args == nil {
		return self.parseUntil(os.Args[1:], "--")
	}
	return self.parseUntil(*args, "--")
}

func (self *ArgParser) ParseAndRun(args *[]string, data interface{}) (int, error) {
	_, err := self.ParseArgs(args)
	if err != nil {
		return -1, err
	}

	// If user didn't provide a command via the commandline
	if self.Command == nil {
		self.PrintHelp()
		return -1, nil
	}

	retCode := self.Command.CommandFunc(self, data)
	return retCode, nil
}

func (self *ArgParser) parseUntil(args []string, terminator string) (*Options, error) {
	self.args = args
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
			if rule.Type == CommandRule {
				if self.Command == nil {
					//fmt.Printf("Set Command\n")
					self.Command = rule
					// Remove the command from our arguments before preceding
					self.args = append(self.args[:self.idx], self.args[self.idx+1:]...)
					if self.idx != 0 {
						self.idx--
					}
					// If user asked us to stop parsing arguments after finding a command
					if self.StopParsingOnCommand {
						goto Apply
					}
				} else {
					// Only one command is allowed at a time. This other match
					// must be a positional argument or a subcommand
					rule.Seen = false
				}
			}
		}
		//fmt.Printf("Failed to Match\n")
		// TODO: If we didn't match any options and user asked us to fail on
		// unmatched arguments return an error here
	}
Apply:
	return self.Apply(nil)
}

// Gather all the values from our rules, then apply the passed in map to any rules that don't have a computed value.
func (self *ArgParser) Apply(values *Options) (*Options, error) {
	results := self.NewOptions()

	// for each of the rules
	for _, rule := range self.rules {
		// Get the computed value
		value, err := rule.ComputedValue(values)
		if err != nil {
			return nil, err
		}
		// If we have a Store() for this rule apply it here
		if rule.StoreValue != nil {
			rule.StoreValue(value)
		}

		// Special Case here for Config Groups
		if rule.IsConfigGroup && values != nil {
			for _, key := range values.Group(rule.Group).Keys() {
				value := values.Group(rule.Group).Get(key)
				results.Group(rule.Group).SetSeen(key, value, rule.Seen)
			}
		} else {
			results.Group(rule.Group).SetSeen(rule.Name, value, rule.Seen)
		}
	}
	self.SetOpts(results)
	return self.GetOpts(), nil
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

func (self *ArgParser) printRules() {
	for _, rule := range self.rules {
		fmt.Printf("Rule: %s - '%+v'\n", rule.Name, rule.Value)
	}
}

func (self *ArgParser) PrintHelp() {
	fmt.Println(self.GenerateHelp())
}

func (self *ArgParser) GenerateHelp() string {
	var result bytes.Buffer
	// TODO: Improve this once we have positional arguments
	result.WriteString("Usage:\n")
	// Super generic usage message
	result.WriteString(fmt.Sprintf("  %s [OPTIONS]\n", self.Name))
	result.WriteString("\n")
	result.WriteString(WordWrap(self.Description, 0, 80))
	result.WriteString("\n")
	result.WriteString("\nOptions:\n")
	result.WriteString(self.GenerateOptHelp())
	return result.String()
}

func (self *ArgParser) GenerateOptHelp() string {
	var result bytes.Buffer

	type HelpMsg struct {
		Flags   string
		Message string
	}
	var options []HelpMsg

	// Ask each rule to generate a Help message for the options
	maxLen := 0
	for _, rule := range self.rules {
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
