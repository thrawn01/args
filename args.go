package args

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
)

const (
	DefaultTerminator string = "--"
)

// ***********************************************
// 					 Types
// ***********************************************
type RuleModifier func(*Rule)

// ***********************************************
// 				Rule Object
// ***********************************************
const (
	countAction int = iota
	appendAction
)

type Rule struct {
	IsPos   int
	Action  int
	Name    string
	Value   interface{}
	Aliases []string
	// Stuff and Junk
}

func (self *Rule) Validate() error {
	return nil
}

func (self *Rule) MatchesAlias(args []string, idx *int) bool {
	for _, alias := range self.Aliases {
		if args[*idx] == alias {
			return true
		}
	}
	return false
}

func (self *Rule) Match(args []string, idx *int) (bool, error) {
	fmt.Printf("Rule.Match(%s)\n", args[*idx])
	if !self.MatchesAlias(args, idx) {
		return false, nil
	}

	if self.Action == countAction {
		if self.Value == nil {
			self.Value = 1
			return true, nil
		}
		self.Value = self.Value.(int) + 1
	}
	return false, nil
}

// ***********************************************
// 				Rules Object
// ***********************************************
type Rules []Rule

func (self Rules) Len() int {
	return len(self)
}

func (self Rules) Less(left, right int) bool {
	return self[left].IsPos < self[right].IsPos
}

func (self Rules) Swap(left, right int) {
	self[left], self[right] = self[right], self[left]
}

// ***********************************************
// 				Result Object
// ***********************************************

type Options map[string]interface{}

func (self Options) Convert(key string, convFunc func(value interface{})) {
	value, ok := self[key]
	if !ok {
		panic(fmt.Sprintf("No Such Option '%s' found", key))
	}
	defer func() {
		if msg := recover(); msg != nil {
			panic(fmt.Sprintf("Refusing to convert Option '%s' of type '%s' to an int", key, reflect.TypeOf(self[key])))
		}
	}()
	convFunc(value)
}

func (self Options) Int(key string) int {
	// TODO: This should convert int to string using strconv
	var result int
	self.Convert(key, func(value interface{}) {
		result = value.(int)
	})
	return result
}

// ***********************************************
// 				ArgParser Object
// ***********************************************

type ArgParser struct {
	args    []string
	results Options
	rules   Rules
	err     error
	idx     int
}

var isOptional = regexp.MustCompile(`^(\W+)(\w*)$`)
var extractName = regexp.MustCompile(`^(\W+)(\w*)$`)

func (self *ArgParser) Opt(name string, modifiers ...RuleModifier) {
	rule := Rule{}
	// If name begins with a non word charater, assume it's an optional argument
	if isOptional.MatchString(name) {
		// Attempt to extract the name
		group := isOptional.FindStringSubmatch(name)
		if group == nil {
			self.err = errors.New(fmt.Sprintf("Invalid Optional argument name '%s'", name))
			return
		} else {
			rule.Aliases = append(rule.Aliases, name)
			rule.Name = group[2]
		}
	} else {
		rule.IsPos = 1
		rule.Name = name
	}

	for _, modify := range modifiers {
		// The modifiers know how to modify a rule
		modify(&rule)
	}
	// Make sure conflicting rules where not used
	self.err = rule.Validate()
	// Append our rules to the list
	self.rules = append(self.rules, rule)
}

func (self *ArgParser) GetRules() Rules {
	return self.rules
}

// Parses command line arguments using os.Args
func (self *ArgParser) Parse() (Options, error) {
	return self.ParseArgs(os.Args[1:])
}

func (self *ArgParser) ParseArgs(args []string) (Options, error) {
	return self.ParseUntil(args, "--")
}

func (self *ArgParser) ParseUntil(args []string, terminator string) (Options, error) {
	self.args = args
	self.idx = 0

	// Sanity Check
	if len(self.rules) == 0 {
		self.err = errors.New("Must create some options to parse with args.Opt()" +
			" before calling arg.Parse()")
	}

	// If any of the Opt() Calls reported an error
	if self.err != nil {
		return self.results, self.err
	}

	// Sort the rules so positional rules are parsed last
	sort.Sort(self.rules)

	// Process command line arguments until we find our terminator
	for ; self.idx < len(self.args); self.idx++ {
		fmt.Printf("Looking at: %d:%s\n", self.idx, self.args[self.idx])
		if self.args[self.idx] == terminator {
			fmt.Printf("Terminator Reached\n")
			return self.results, nil
		}
		// Match our arguments with rules expected
		matched, err := self.match(self.rules)
		if err != nil {
			return self.results, err
		}

		if matched {
			continue
		}

		// TODO: If we didn't match either and user asked us to fail on
		// unmatched arguments return an error here
	}
	return self.results, nil
}

func (self *ArgParser) match(rules Rules) (bool, error) {
	// Find a Rule for this argument
	for _, rule := range rules {
		matched, err := rule.Match(self.args, &self.idx)
		if err != nil {
			// This Rule did match our argument but had an error
			return true, err
		}
		if matched {
			fmt.Printf("Matched '%s'", rule.Name)
			// This Rule matched our argument
			self.results[rule.Name] = rule.Value
			return true, nil
		}
	}
	// No Rules match our arguments and there was no error
	return false, nil
}

// ***********************************************
// 				PUBLIC FUNCTIONS
// ***********************************************

// Creates a new instance of the argument parser
func Parser() ArgParser {
	return ArgParser{}
}

// Indicates this option has an alias it can go by
func Alias(optName string) RuleModifier {
	return func(rule *Rule) {
		fmt.Printf("Alias(%s)\n", optName)
	}
}

func Count() RuleModifier {
	return func(rule *Rule) {
		//fmt.Printf("Count()\n")
	}
}
