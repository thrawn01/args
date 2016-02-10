package args

import (
	"errors"
	"fmt"
	"github.com/go-ini/ini"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
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
	Count       int
	IsPos       int
	Name        string
	HelpMsg     string
	VarName     string
	Value       interface{}
	Default     interface{}
	Aliases     []string
	EnvironVars []string
	Cast        CastFunc
	Action      ActionFunc
	StoreValue  StoreFunc
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
	//fmt.Printf("Matched: %s - %s\n", matched, alias)
	if !matched {
		return false, nil
	}
	//self.FlagSeen = true

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

func (self *Rule) GetValuesFrom(values *map[string]string) (interface{}, error) {
	// TODO: Do this better
	if self.Count != 0 {
		self.Value = self.Count
	}

	// If Rule Matched Argument on command line
	if self.Value != nil {
		return self.Value, nil
	}

	// If Rule Matched Environment variable
	value, err := self.GetEnvValue()
	if err != nil {
		return nil, err
	}
	if value != nil {
		return value, nil
	}

	// If provided our map of values, use that
	if values != nil {
		if value, ok := (*values)[self.Name]; ok {
			return self.Cast(self.Name, value)
		}
	}

	// Apply default if available
	if self.Default != nil {
		return self.Default, nil
	}
	// Return the default value for our type choice
	value, _ = self.Cast("", "")
	return value, nil
}

func (self *Rule) GetEnvValue() (interface{}, error) {
	if self.EnvironVars == nil {
		return nil, nil
	}

	for _, varName := range self.EnvironVars {
		//if value, ok := os.LookupEnv(varName); ok {
		if value := os.Getenv(varName); value != "" {
			return self.Cast(varName, value)
		}
	}
	return nil, nil
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
// 				Options Object
// ***********************************************

type Options map[string]interface{}

func (self Options) Convert(key string, typeName string, convFunc func(value interface{})) {
	value, ok := self[key]
	if !ok {
		panic(fmt.Sprintf("No Such Option '%s' found", key))
	}
	defer func() {
		if msg := recover(); msg != nil {
			panic(fmt.Sprintf("Refusing to convert Option '%s' of type '%s' to '%s'",
				key, reflect.TypeOf(self[key]), typeName))
		}
	}()
	convFunc(value)
}

func (self Options) IsNil(key string) bool {
	if value, ok := self[key]; ok {
		return value == nil
	}
	return true
}

func (self Options) Int(key string) int {
	var result int
	self.Convert(key, "int", func(value interface{}) {
		if value != nil {
			result = value.(int)
			return
		}
		// Avoid panic, return 0 if no value
		result = 0
	})
	return result
}

func (self Options) String(key string) string {
	var result string
	self.Convert(key, "string", func(value interface{}) {
		if value != nil {
			result = value.(string)
			return
		}
		// Avoid panic, return "" if no value
		result = ""
	})
	return result
}

func (self Options) Bool(key string) bool {
	var result bool
	self.Convert(key, "bool", func(value interface{}) {
		if value != nil {
			result = value.(bool)
			return
		}
		// Avoid panic, return false if no value
		result = false
	})
	return result
}

// TODO: Should support more than just []string
func (self Options) Slice(key string) []string {
	var result []string
	self.Convert(key, "[]string", func(value interface{}) {
		if value != nil {
			result = value.([]string)
			return
		}
		// Avoid panic, return []string{} if no value
		result = []string{}
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
	rule := &Rule{Cast: castString}
	// If name begins with a non word charater, assume it's an optional argument
	if isOptional.MatchString(name) {
		// Attempt to extract the name
		group := isOptional.FindStringSubmatch(name)
		if group == nil {
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
func (self *ArgParser) ParseArgs() (*Options, error) {
	return self.ParseSlice(os.Args[1:])
}

func (self *ArgParser) ParseSlice(args []string) (*Options, error) {
	return self.ParseUntil(args, "--")
}

func (self *ArgParser) ParseUntil(args []string, terminator string) (*Options, error) {
	self.args = args
	self.idx = 0

	// Sanity Check
	if len(self.rules) == 0 {
		self.err = errors.New("Must create some options to match with args.Opt()" +
			" before calling arg.ParseArgs()")
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
			return self.collectResults(nil)
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
	return self.collectResults(nil)
}

func (self *ArgParser) collectResults(values *map[string]string) (*Options, error) {
	results := &Options{}

	// Get the computed value after applying all rules
	for _, rule := range self.rules {
		value, err := rule.GetValuesFrom(values)
		if err != nil {
			return nil, err
		}
		// If we have a Store() for this rule apply it here
		if rule.StoreValue != nil {
			rule.StoreValue(value)
		}
		(*results)[rule.Name] = value
	}

	return results, nil
}

func (self *ArgParser) ParseIni(input []byte) (*Options, error) {
	// Parse the file return a map of the contents
	cfg, err := ini.Load(input)
	if err != nil {
		return nil, err
	}
	values := make(map[string]string)
	for _, key := range cfg.Section("").KeyStrings() {
		values[key] = cfg.Section("").Key(key).String()
	}
	// Apply the ini file values to the commandline and environment variables
	return self.collectResults(&values)
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
func Alias(aliasName string) RuleModifier {
	return func(rule *Rule) {
		rule.Aliases = append(rule.Aliases, aliasName)
	}
}

func Count() RuleModifier {
	return func(rule *Rule) {
		rule.Action = func(rule *Rule, alias string, args []string, idx *int) error {
			// If user asked us to count the instances of this argument
			rule.Count = rule.Count + 1
			return nil
		}
	}
}

// If the option is seen on the command line, the value is 'true'
func IsTrue() RuleModifier {
	return func(rule *Rule) {
		rule.Action = func(rule *Rule, alias string, args []string, idx *int) error {
			rule.Value = true
			return nil
		}
		rule.Cast = castBool
	}
}

func castString(optName string, strValue string) (interface{}, error) {
	// If empty string is passed, give type init value
	if strValue == "" {
		return "", nil
	}
	return strValue, nil
}

func IsString() RuleModifier {
	return func(rule *Rule) {
		rule.Cast = castString
	}
}

func castInt(optName string, strValue string) (interface{}, error) {
	// If empty string is passed, give type init value
	if strValue == "" {
		return 0, nil
	}

	value, err := strconv.ParseInt(strValue, 10, 64)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("Invalid value for '%s' - '%s' is not an Integer", optName, strValue))
	}
	return int(value), nil
}

func IsInt() RuleModifier {
	return func(rule *Rule) {
		rule.Cast = castInt
	}
}

func castBool(optName string, strValue string) (interface{}, error) {
	// If empty string is passed, give type init value
	if strValue == "" {
		return false, nil
	}

	value, err := strconv.ParseBool(strValue)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("Invalid value for '%s' - '%s' is not a Boolean", optName, strValue))
	}
	return bool(value), nil
}

func IsBool() RuleModifier {
	return func(rule *Rule) {
		rule.Cast = castBool
		rule.Value = false
	}
}

func Default(value string) RuleModifier {
	return func(rule *Rule) {
		rule.Default = value
	}
}

func StoreInt(dest *int) RuleModifier {
	// Implies IsInt()
	return func(rule *Rule) {
		rule.Cast = castInt
		rule.StoreValue = func(value interface{}) {
			*dest = value.(int)
		}
	}
}

func StoreTrue(dest *bool) RuleModifier {
	return func(rule *Rule) {
		rule.Action = func(rule *Rule, alias string, args []string, idx *int) error {
			rule.Value = true
			return nil
		}
		rule.Cast = castBool
		rule.StoreValue = func(value interface{}) {
			*dest = value.(bool)
		}
	}
}

func castStringSlice(optName string, strValue string) (interface{}, error) {
	// If empty string is passed, give type init value
	if strValue == "" {
		return []string{}, nil
	}

	// If no comma is found, then assume this is a single value
	if strings.Index(strValue, ",") == -1 {
		return []string{strValue}, nil
	}

	// Split the values separated by comma's
	return strings.Split(strValue, ","), nil
}

// TODO: Make this less horribad, and use more reflection to make the interface simpler
// It should also take more than just []string but also []int... etc...
func StoreSlice(dest *[]string) RuleModifier {
	return func(rule *Rule) {
		rule.Cast = castStringSlice
		rule.StoreValue = func(src interface{}) {
			// This should never happen if we validate the types
			srcType := reflect.TypeOf(src)
			if srcType.Kind() != reflect.Slice {
				panic(fmt.Sprintf("Attempted to store '%s' which is not a slice", srcType.Kind()))
			}
			for _, value := range src.([]string) {
				*dest = append(*dest, value)
			}
		}
	}
}

func StoreStr(dest *string) RuleModifier {
	return StoreString(dest)
}

func StoreString(dest *string) RuleModifier {
	// Implies IsString()
	return func(rule *Rule) {
		rule.Cast = castString
		rule.StoreValue = func(value interface{}) {
			*dest = value.(string)
		}
	}
}

func Env(varName string) RuleModifier {
	return func(rule *Rule) {
		rule.EnvironVars = append(rule.EnvironVars, varName)
	}
}

func VarName(varName string) RuleModifier {
	return func(rule *Rule) {
		rule.VarName = varName
	}
}

func Help(message string) RuleModifier {
	return func(rule *Rule) {
		rule.HelpMsg = message
	}
}
