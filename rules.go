package args

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

type RuleModifier struct {
	rule   *Rule
	parser *ArgParser
}

func NewRuleModifier(parser *ArgParser) *RuleModifier {
	return &RuleModifier{newRule(), parser}
}

func newRuleModifier(rule *Rule, parser *ArgParser) *RuleModifier {
	return &RuleModifier{rule, parser}
}

func (self *RuleModifier) GetRule() *Rule {
	return self.rule
}

func (self *RuleModifier) IsString() *RuleModifier {
	self.rule.Cast = castString
	return self
}

// If the option is seen on the command line, the value is 'true'
func (self *RuleModifier) IsTrue() *RuleModifier {
	self.rule.Action = func(rule *Rule, alias string, args []string, idx *int) error {
		rule.Value = true
		return nil
	}
	self.rule.Cast = castBool
	return self
}

func (self *RuleModifier) IsBool() *RuleModifier {
	self.rule.Cast = castBool
	self.rule.Value = false
	return self
}

func (self *RuleModifier) Default(value string) *RuleModifier {
	self.rule.Default = &value
	return self
}

func (self *RuleModifier) StoreInt(dest *int) *RuleModifier {
	// Implies IsInt()
	self.rule.Cast = castInt
	self.rule.StoreValue = func(value interface{}) {
		*dest = value.(int)
	}
	return self
}

func (self *RuleModifier) IsInt() *RuleModifier {
	self.rule.Cast = castInt
	return self
}

func (self *RuleModifier) StoreTrue(dest *bool) *RuleModifier {
	self.rule.Action = func(rule *Rule, alias string, args []string, idx *int) error {
		rule.Value = true
		return nil
	}
	self.rule.Cast = castBool
	self.rule.StoreValue = func(value interface{}) {
		*dest = value.(bool)
	}
	return self
}

func (self *RuleModifier) IsStringSlice() *RuleModifier {
	self.rule.Cast = castStringSlice
	return self
}

// TODO: Make this less horribad, and use more reflection to make the interface simpler
// It should also take more than just []string but also []int... etc...
func (self *RuleModifier) StoreStringSlice(dest *[]string) *RuleModifier {
	self.rule.Cast = castStringSlice
	self.rule.StoreValue = func(src interface{}) {
		// First clear the current slice if any
		*dest = nil
		// This should never happen if we validate the types
		srcType := reflect.TypeOf(src)
		if srcType.Kind() != reflect.Slice {
			self.parser.GetLog().Printf("Attempted to store '%s' which is not a slice", srcType.Kind())
		}
		for _, value := range src.([]string) {
			*dest = append(*dest, value)
		}
	}
	return self
}

// Indicates this option has an alias it can go by
func (self *RuleModifier) Alias(aliasName string) *RuleModifier {
	self.rule.Aliases = append(self.rule.Aliases, aliasName)
	return self
}

// Makes this option or positional argument required
func (self *RuleModifier) Required() *RuleModifier {
	self.rule.SetFlags(IsRequired)
	return self
}

func (self *RuleModifier) StoreStr(dest *string) *RuleModifier {
	return self.StoreString(dest)
}

func (self *RuleModifier) StoreString(dest *string) *RuleModifier {
	// Implies IsString()
	self.rule.Cast = castString
	self.rule.StoreValue = func(value interface{}) {
		*dest = value.(string)
	}
	return self
}

func (self *RuleModifier) Count() *RuleModifier {
	self.rule.Action = func(rule *Rule, alias string, args []string, idx *int) error {
		// If user asked us to count the instances of this argument
		rule.Count = rule.Count + 1
		return nil
	}
	self.rule.Cast = castInt
	return self
}

func (self *RuleModifier) Env(varName string) *RuleModifier {
	self.rule.EnvVars = append(self.rule.EnvVars, varName)
	return self
}

func (self *RuleModifier) VarName(varName string) *RuleModifier {
	self.rule.VarName = varName
	return self
}

func (self *RuleModifier) Help(message string) *RuleModifier {
	self.rule.RuleDesc = message
	return self
}

func (self *RuleModifier) InGroup(group string) *RuleModifier {
	self.rule.Group = group
	return self
}

func (self *RuleModifier) AddConfigGroup(group string) *RuleModifier {
	var newRule Rule
	newRule = *self.rule
	newRule.SetFlags(IsConfigGroup)
	newRule.Group = group
	// Make a new RuleModifier using self as the template
	return self.parser.AddRule(group, newRuleModifier(&newRule, self.parser))
}

func (self *RuleModifier) Opt(name string) *RuleModifier {
	return self.AddOption(name)
}

func (self *RuleModifier) AddOption(name string) *RuleModifier {
	var newRule Rule
	newRule = *self.rule
	newRule.SetFlags(IsOption)
	// Make a new RuleModifier using self as the template
	return self.parser.AddRule(name, newRuleModifier(&newRule, self.parser))
}

func (self *RuleModifier) Cfg(name string) *RuleModifier {
	return self.AddConfig(name)
}

func (self *RuleModifier) AddConfig(name string) *RuleModifier {
	var newRule Rule
	newRule = *self.rule
	// Make a new Rule using self.rule as the template
	newRule.SetFlags(IsConfig)
	return self.parser.AddRule(name, newRuleModifier(&newRule, self.parser))
}

func (self *RuleModifier) Key(key string) *RuleModifier {
	self.rule.Key = key
	return self
}

// ***********************************************
// Rule Object
// ***********************************************

type CastFunc func(string, interface{}, interface{}) (interface{}, error)
type ActionFunc func(*Rule, string, []string, *int) error
type StoreFunc func(interface{})
type CommandFunc func(*ArgParser, interface{}) int

const (
	IsCommand int64 = 1 << iota
	IsPositional
	IsConfig
	IsConfigGroup
	IsRequired
	IsOption
	NoValue
	Seen
)

type Rule struct {
	Count       int
	Order       int
	Name        string
	RuleDesc    string
	VarName     string
	Value       interface{}
	Default     *string
	Aliases     []string
	EnvVars     []string
	EnvPrefix   string
	Cast        CastFunc
	Action      ActionFunc
	StoreValue  StoreFunc
	CommandFunc CommandFunc
	Group       string
	Key         string
	BackendKey  string
	NotGreedy   bool
	Flags       int64
}

func newRule() *Rule {
	return &Rule{Cast: castString, Group: DefaultOptionGroup}
}

func (self *Rule) HasFlags(flag int64) bool {
	return self.Flags&flag != 0
}

func (self *Rule) SetFlags(flag int64) {
	self.Flags = (self.Flags | flag)
}

func (self *Rule) ClearFlags(flag int64) {
	mask := (self.Flags ^ flag)
	self.Flags &= mask
}

func (self *Rule) Validate() error {
	return nil
}

func (self *Rule) GenerateUsage() string {
	switch {
	case self.Flags&IsOption != 0:
		if self.HasFlags(IsRequired) {
			return fmt.Sprintf("%s", self.Aliases[0])
		}
		return fmt.Sprintf("[%s]", self.Aliases[0])
	case self.Flags&IsPositional != 0:
		if self.HasFlags(IsRequired) {
			return fmt.Sprintf("%s", self.Name)
		}
		return fmt.Sprintf("[%s]", self.Name)
	}
	return ""
}

func (self *Rule) GenerateHelp() (string, string) {
	var parens []string
	paren := ""

	if !self.HasFlags(IsCommand) {
		if self.Default != nil {
			parens = append(parens, fmt.Sprintf("Default=%s", *self.Default))
		}
		if len(self.EnvVars) != 0 {
			envs := strings.Join(self.EnvVars, ",")
			parens = append(parens, fmt.Sprintf("Env=%s", envs))
		}
		if len(parens) != 0 {
			paren = fmt.Sprintf("(%s)", strings.Join(parens, " "))
		}
	}

	if self.HasFlags(IsPositional) {
		return ("  " + self.Name), self.RuleDesc
	}
	// TODO: This sort should happen when we validate rules
	sort.Sort(sort.Reverse(sort.StringSlice(self.Aliases)))
	return ("  " + strings.Join(self.Aliases, ", ")), (self.RuleDesc + " " + paren)
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
	name := self.Name
	var matched bool

	if self.HasFlags(IsConfig) {
		return false, nil
	}

	if self.HasFlags(IsPositional) {
		// If we are a positional and we have already been seen, and not greedy
		if self.HasFlags(Seen) && self.NotGreedy {
			// Do not match this argument
			return false, nil
		}
		// TODO: Handle Greedy
	} else {
		//fmt.Printf("Matched Positional Arg: %s\n", args[*idx])
		matched, name = self.MatchesAlias(args, idx)
		//fmt.Printf("Matched Optional Arg: %v - %s\n", matched, alias)
		if !matched {
			return false, nil
		}
	}
	self.SetFlags(Seen)

	// If user defined an action
	if self.Action != nil {
		err := self.Action(self, name, args, idx)
		if err != nil {
			return true, err
		}
		return true, nil
	}

	if !self.HasFlags(IsPositional) {
		// If no actions are specified assume a value follows this argument and should be converted
		*idx++
		if len(args) <= *idx {
			return true, errors.New(fmt.Sprintf("Expected '%s' to have an argument", name))
		}
	}

	// If we get here, this argument is associated with either an option value or a positional
	//fmt.Printf("arg: %s value: %s\n", alias, args[*idx])
	value, err := self.Cast(name, self.Value, self.UnEscape(args[*idx]))
	if err != nil {
		return true, err
	}
	self.Value = value
	return true, nil
}

func (self *Rule) UnEscape(str string) string {
	return strings.Replace(str, "\\", "", -1)
}

// Returns the appropriate required warning to display to the user
func (self *Rule) RequiredMessage() string {
	switch {
	case self.Flags&IsPositional != 0:
		return fmt.Sprintf("positional '%s' is required", self.Name)
	case self.Flags&IsConfig != 0:
		return fmt.Sprintf("config '%s' is required", self.Name)
	default:
		return fmt.Sprintf("option '%s' is required", self.Aliases[0])
	}
}

func (self *Rule) ComputedValue(values *Options) (interface{}, error) {
	if self.Count != 0 {
		self.Value = self.Count
	}

	// If rule matched argument on command line
	if self.HasFlags(Seen) {
		return self.Value, nil
	}

	// If rule matched environment variable
	value, err := self.GetEnvValue()
	if err != nil {
		return nil, err
	}

	if value != nil {
		return value, nil
	}

	// TODO: Move this logic from here, This method should be all about getting the value
	if self.HasFlags(IsConfigGroup) {
		return nil, nil
	}

	// If provided our map of values, use that
	if values != nil {
		group := values.Group(self.Group)
		if group.HasKey(self.Name) {
			self.ClearFlags(NoValue)
			return self.Cast(self.Name, group.Get(self.Name))
		}
	}

	// Apply default if available
	if self.Default != nil {
		return self.Cast(self.Name, *self.Default)
	}

	// TODO: Move this logic from here, This method should be all about getting the value
	if self.HasFlags(IsRequired) {
		return nil, errors.New(self.RequiredMessage())
	}

	// Flag that we found no value for this rule
	self.SetFlags(NoValue)

	// Return the default value for our type choice
	value, _ = self.Cast(self.Name, nil)
	return value, nil
}

func (self *Rule) GetEnvValue() (interface{}, error) {
	if self.EnvVars == nil {
		return nil, nil
	}

	for _, varName := range self.EnvVars {
		varName := self.EnvPrefix + varName
		//if value, ok := os.LookupEnv(varName); ok {
		if value := os.Getenv(varName); value != "" {
			return self.Cast(varName, value)
		}
	}
	return nil, nil
}

func (self *Rule) BackendKeyPath(rootPath string) string {
	rootPath = strings.TrimPrefix(rootPath, "/")
	if self.Key == "" {
		self.Key = self.Name
	}

	if self.HasFlags(IsConfigGroup) {
		return path.Join("/", rootPath, self.Group)
	}
	if self.Group == DefaultOptionGroup {
		return path.Join("/", rootPath, "DEFAULT", self.Key)
	}
	return path.Join("/", rootPath, self.Group, self.Key)
}

// ***********************************************
// Rules Object
// ***********************************************
type Rules []*Rule

func (self Rules) Len() int {
	return len(self)
}

func (self Rules) Less(left, right int) bool {
	return self[left].Order < self[right].Order
}

func (self Rules) Swap(left, right int) {
	self[left], self[right] = self[right], self[left]
}
