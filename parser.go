package args

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"sync"
	"time"
)

const MAX_BACKOFF_WAIT = 2 * time.Second

type ParseModifier func(*ArgParser)

type ArgParser struct {
	EnvPrefix   string
	EtcdRoot    string
	Description string
	Name        string
	WordWrap    int
	mutex       sync.Mutex
	args        []string
	options     *Options
	rules       Rules
	err         error
	idx         int
	attempts    int
	log         StdLogger
}

// Creates a new instance of the argument parser
func NewParser(modifiers ...ParseModifier) *ArgParser {
	parser := &ArgParser{
		"",
		"",
		"",
		"",
		200,
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
		// If it's not a config only option
		if !rule.IsConfig {
			// If must be a positional
			rule.IsPos = 1
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
			return self.Apply(nil)
		}
		// Match our arguments with rules expected
		//fmt.Printf("====== Attempting to match: %d:%s - ", self.idx, self.args[self.idx])
		matched, err := self.match(self.rules)
		if err != nil {
			return nil, err
		}

		if !matched {
			//fmt.Printf("Failed to Match\n")
			// TODO: If we didn't match any options and user asked us to fail on
			// unmatched arguments return an error here
		}
	}
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

		// Config group is an adhoc group of key=values which
		// do not have a specific type defined
		if rule.IsConfigGroup {
			for _, key := range values.Group(rule.Group).Keys() {
				value := values.Group(rule.Group).Get(key)
				results.Group(rule.Group).Set(key, value, rule.Seen)
			}
		} else {
			results.Group(rule.Group).Set(rule.Name, value, rule.Seen)
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

func (self *ArgParser) match(rules Rules) (bool, error) {
	// Find a Rule that matches this argument
	for _, rule := range rules {
		matched, err := rule.Match(self.args, &self.idx)
		if err != nil {
			// This Rule did match our argument but had an error
			return true, err
		}
		if matched {
			//fmt.Printf("Matched '%s' with '%s'\n", rule.Name, rule.Value)
			return true, nil
		}
	}
	// No Rules matched our arguments and there was no error
	return false, nil
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
