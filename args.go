package args

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
)

const (
	DefaultTerminator string = "--"
)

// ***********************************************
// 					 Types
// ***********************************************
type RuleModifier func(*Rule)
type CastFunc func(string, string) (interface{}, error)
type ActionFunc func(*Rule, string, []string, *int) error
type StoreFunc func(interface{})

// ***********************************************
// 				Rule Object
// ***********************************************
type Rule struct {
	IsPos      int
	Name       string
	Value      interface{}
	Default    interface{}
	Aliases    []string
	Cast       CastFunc
	Action     ActionFunc
	StoreValue StoreFunc
	// Stuff and Junk
}

func (self *Rule) Validate() error {
	return nil
}

func (self *Rule) MatchesAlias(args []string, idx *int) (bool, string) {
	for _, alias := range self.Aliases {
		if args[*idx] == alias {
			return true, args[*idx]
		}
	}
	return false, ""
}

func (self *Rule) Match(args []string, idx *int) (bool, error) {
	matched, alias := self.MatchesAlias(args, idx)
	fmt.Printf("Matched: %s - %s\n", matched, alias)
	if !matched {
		return false, nil
	}

	// If user defined an action
	if self.Action != nil {
		err := self.Action(self, alias, args, idx)
		if err != nil {
			return true, err
		}
		return true, nil
	}

	// If no actions are specified assume a value follows this argument and should be converted
	*idx++
	if len(args) <= *idx {
		return true, errors.New(fmt.Sprintf("Expected '%s' to have an argument", alias))
	}
	//fmt.Printf("arg: %s value: %s\n", alias, args[*idx])
	value, err := self.Cast(alias, args[*idx])
	if err != nil {
		return true, err
	}
	self.Value = value
	return true, nil
}

func (self *Rule) GetValue() interface{} {
	// If Rule Matched Argument on command line
	if self.Value != nil {
		return self.Value
	}
	// If Rule Matched Environment variable
	// Apply default if available
	if self.Default != nil {
		return self.Default
	}
	return nil
}

// ***********************************************
// 				Rules Object
// ***********************************************
type Rules []*Rule

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

func (self Options) IsNil(key string) bool {
	value, ok := self[key]
	if !ok {
		return true
	}
	return value == nil
}

func (self Options) Int(key string) int {
	// TODO: This should convert int to string using strconv
	var result int
	self.Convert(key, func(value interface{}) {
		if value != nil {
			result = value.(int)
			return
		}
		// Avoid panic, return 0 if no value
		result = 0
	})
	return result
}

// ***********************************************
// 				ArgParser Object
// ***********************************************

type ArgParser struct {
	args  []string
	rules Rules
	err   error
	idx   int
}

var isOptional = regexp.MustCompile(`^(\W+)([\w|-]*)$`)

func (self *ArgParser) ValidateRules() error {
	for idx, rule := range self.rules {
		// Duplicate rule check
		next := idx + 1
		if next < len(self.rules) {
			for ; next < len(self.rules); next++ {
				if rule.Name == self.rules[next].Name {
					return errors.New(fmt.Sprintf("Duplicate Opt() called with same name as '%s'", rule.Name))
				}
			}
		}

		// Ensure user didn't set a bad default value
		if rule.Cast != nil && rule.Default != nil {
			cast, err := rule.Cast("args.Default()", rule.Default.(string))
			if err != nil {
				panic(err.Error())
			}
			rule.Default = cast
		}
	}
	return nil
}

func (self *ArgParser) Opt(name string, modifiers ...RuleModifier) {
	rule := &Rule{}
	// If name begins with a non word charater, assume it's an optional argument
	if isOptional.MatchString(name) {
		// Attempt to extract the name
		group := isOptional.FindStringSubmatch(name)
		if group == nil {
			fmt.Printf("Failed to find argument name\n")
			self.err = errors.New(fmt.Sprintf("Invalid optional argument name '%s'", name))
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
		modify(rule)
	}
	// Append our rules to the list
	self.rules = append(self.rules, rule)
	// Make sure conflicting/duplicate rules where not used
	self.err = self.ValidateRules()
}

func (self *ArgParser) GetRules() Rules {
	return self.rules
}

// Parses command line arguments using os.Args
func (self *ArgParser) Parse() (*Options, error) {
	return self.ParseArgs(os.Args[1:])
}

func (self *ArgParser) ParseArgs(args []string) (*Options, error) {
	return self.ParseUntil(args, "--")
}

func (self *ArgParser) ParseUntil(args []string, terminator string) (*Options, error) {
	self.args = args
	self.idx = 0

	// Sanity Check
	if len(self.rules) == 0 {
		self.err = errors.New("Must create some options to match with args.Opt()" +
			" before calling arg.Parse()")
	}

	// If any of the Opt() Calls reported an error
	if self.err != nil {
		return nil, self.err
	}

	// Sort the rules so positional rules are parsed last
	sort.Sort(self.rules)

	// Process command line arguments until we find our terminator
	for ; self.idx < len(self.args); self.idx++ {
		if self.args[self.idx] == terminator {
			goto collectResults
		}
		// Match our arguments with rules expected
		fmt.Printf("====== Attempting to match: %d:%s - ", self.idx, self.args[self.idx])
		matched, err := self.match(self.rules)
		if err != nil {
			return nil, err
		}

		if !matched {
			fmt.Printf("Failed to Match\n")
			// TODO: If we didn't match any options and user asked us to fail on
			// unmatched arguments return an error here
		}
	}
collectResults:
	results := &Options{}

	// Get the computed value after applying all rules
	for _, rule := range self.rules {
		value := rule.GetValue()
		// If we have a Store() for this rule apply it here
		if rule.StoreValue != nil {
			rule.StoreValue(value)
		}
		(*results)[rule.Name] = value
	}

	return results, nil
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
			fmt.Printf("Matched '%s' with '%s'\n", rule.Name, rule.Value)
			return true, nil
		}
	}
	// No Rules matched our arguments and there was no error
	return false, nil
}

func (self *ArgParser) printRules() {
	for _, rule := range self.rules {
		fmt.Printf("Rule: %s - '%s'\n", rule.Name, rule.Value)
	}
}

// ***********************************************
// 				PUBLIC FUNCTIONS
// ***********************************************

// Creates a new instance of the argument parser
func Parser() *ArgParser {
	return &ArgParser{}
}

// Indicates this option has an alias it can go by
func Alias(optName string) RuleModifier {
	return func(rule *Rule) {
		fmt.Printf("Alias(%s)\n", optName)
	}
}

func Count() RuleModifier {
	return func(rule *Rule) {
		rule.Action = func(rule *Rule, alias string, args []string, idx *int) error {
			// If user asked us to count the instances of this argument
			rule.Value = rule.Value.(int) + 1
			return nil
		}
		if rule.Value == nil {
			rule.Value = 0
		}
	}
}

func castInt(optName string, strValue string) (interface{}, error) {
	value, err := strconv.ParseInt(strValue, 10, 64)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("Invalid value for '%s' - '%s' is not an Integer", optName, strValue))
	}
	return int(value), nil
}

func Int() RuleModifier {
	return func(rule *Rule) {
		rule.Cast = castInt
		rule.Value = 0
	}
}

func Default(value string) RuleModifier {
	return func(rule *Rule) {
		rule.Default = value
	}
}

func StoreInt(dest *int) RuleModifier {
	// Implies Int()
	return func(rule *Rule) {
		rule.Cast = castInt
		rule.StoreValue = func(value interface{}) {
			fmt.Printf("Value: %s\n", value)
			*dest = value.(int)
		}
	}
}
