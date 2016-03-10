package args

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"sync"

	"github.com/go-ini/ini"
)

const (
	DefaultTerminator  string = "--"
	DefaultOptionGroup string = ""
)

// ***********************************************
//  Types
// ***********************************************
type CastFunc func(string, interface{}) (interface{}, error)
type ActionFunc func(*Rule, string, []string, *int) error
type StoreFunc func(interface{})

// ***********************************************
// RuleModifier Object
// ***********************************************
type RuleModifier struct {
	rule   *Rule
	parser *ArgParser
}

func NewRuleModifier(parser *ArgParser) *RuleModifier {
	return &RuleModifier{newRule(), parser}
}

func newRuleModifier(rule *Rule, parser *ArgParser) *RuleModifier {
	return &RuleModifier{newRule(), parser}
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

// TODO: Make this less horribad, and use more reflection to make the interface simpler
// It should also take more than just []string but also []int... etc...
func (self *RuleModifier) StoreSlice(dest *[]string) *RuleModifier {
	self.rule.Cast = castStringSlice
	self.rule.StoreValue = func(src interface{}) {
		// First clear the currenty slice if any
		*dest = nil
		// This should never happen if we validate the types
		srcType := reflect.TypeOf(src)
		if srcType.Kind() != reflect.Slice {
			panic(fmt.Sprintf("Attempted to store '%s' which is not a slice", srcType.Kind()))
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
	self.rule.EnvironVars = append(self.rule.EnvironVars, varName)
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

func (self *RuleModifier) AddOption(name string) *RuleModifier {
	return self.parser.AddRule(name, self)
}

// ***********************************************
// Rule Object
// ***********************************************
type Rule struct {
	Count       int
	IsPos       int
	IsConfig    bool
	Name        string
	RuleDesc    string
	VarName     string
	Value       interface{}
	Seen        bool
	Default     *string
	Aliases     []string
	EnvironVars []string
	Cast        CastFunc
	Action      ActionFunc
	StoreValue  StoreFunc
	Group       string
}

func newRule() *Rule {
	return &Rule{Cast: castString, Group: DefaultOptionGroup}
}

func (self *Rule) Validate() error {
	return nil
}

func (self *Rule) GenerateHelp() (string, string) {
	var parens []string
	paren := ""
	if self.Default != nil {
		parens = append(parens, fmt.Sprintf("Default=%s", *self.Default))
	}
	if len(self.EnvironVars) != 0 {
		envs := strings.Join(self.EnvironVars, ",")
		parens = append(parens, fmt.Sprintf("Env=%s", envs))
	}
	if len(parens) != 0 {
		paren = fmt.Sprintf("(%s)", strings.Join(parens, " "))
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
	if self.IsConfig {
		return false, nil
	}
	matched, alias := self.MatchesAlias(args, idx)
	//fmt.Printf("Matched: %s - %s\n", matched, alias)
	if !matched {
		return false, nil
	}
	self.Seen = true

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

func (self *Rule) ComputedValue(values *Options) (interface{}, error) {
	// TODO: Do count better?
	if self.Count != 0 {
		self.Value = self.Count
	}

	// If Rule Matched Argument on command line
	if self.Seen {
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
		if values.HasKey(self.Name) {
			return self.Cast(self.Name, values.RawValue(self.Name))
		}
	}

	// Apply default if available
	if self.Default != nil {
		return self.Cast(self.Name, *self.Default)
	}
	// Return the default value for our type choice
	value, _ = self.Cast(self.Name, nil)
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
// Rules Object
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
// Options Object
// ***********************************************
type Options struct {
	group  string
	values map[string]*OptionValue
	groups map[string]*Options
}

type OptionValue struct {
	Value interface{}
	Seen  bool // Argument was seen on the commandline
}

func NewOptions(group string) *Options {
	groups := make(map[string]*Options)
	new := &Options{
		group,
		make(map[string]*OptionValue),
		groups,
	}
	// Add the new Options{} to the group of options
	groups[group] = new
	return new
}

func NewOptionsWithGroups(group string, groups map[string]*Options) *Options {
	new := &Options{
		group,
		make(map[string]*OptionValue),
		groups,
	}
	// Add the new Options{} to the group of options
	groups[group] = new
	return new
}

func NewOptionsFromMap(group string, groups map[string]map[string]*OptionValue) *Options {
	options := NewOptions(group)
	for groupName, values := range groups {
		grp := options.Group(groupName)
		for key, opt := range values {
			grp.Set(key, opt.Value, opt.Seen)
		}
	}
	return options
}

func (self *Options) ValuesToString() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("%s:\n", self.group))
	for key, value := range self.values {
		buffer.WriteString(fmt.Sprintf("   %s=%s\n", key, value.Value))
	}
	return buffer.String()
}

func (self *Options) GroupsToString() string {
	var buffer bytes.Buffer
	for _, group := range self.groups {
		buffer.WriteString(group.ValuesToString())
	}
	return buffer.String()
}

func (self *Options) Keys() []string {
	keys := make([]string, 0, len(self.values))
	for key := range self.values {
		keys = append(keys, key)
	}
	return keys
}

func (self *Options) Group(group string) *Options {
	// If they asked for the default group, and I'm the default group return myself
	if group == DefaultOptionGroup && self.group == group {
		return self
	}
	opts, ok := self.groups[group]
	if !ok {
		// TODO: Validate group name has valid characters or at least
		// doesn't have ':' in the name which would conflict with Compare()

		// If group doesn't exist; create it
		new := NewOptionsWithGroups(group, self.groups)
		self.groups[group] = new
		return new
	}
	return opts
}

func (self *Options) Set(key string, value interface{}, seen bool) *Options {
	self.values[key] = &OptionValue{value, seen}
	return self
}

func (self *Options) Get(key string) string {
	return self.String(key)
}

// Return true if any of the values in this Option object were seen on the command line
func (self *Options) ValuesSeen() bool {
	for _, opt := range self.values {
		if opt.Seen == true {
			return true
		}
	}
	return false
}

/*
	Return true if no arguments where seen on the command line

	opts, _ := parser.ParseArgs(nil)
	if opts.NoArgs() {
		fmt.Printf("No arguments provided")
		parser.PrintHelp()
		os.Exit(-1)
	}
*/
func (self *Options) NoArgs() bool {
	for _, group := range self.groups {
		if group.ValuesSeen() {
			return false
		}
	}
	return !self.ValuesSeen()
}

func (self *Options) convert(key string, typeName string, convFunc func(value interface{})) {
	opt, ok := self.values[key]
	if !ok {
		panic(fmt.Sprintf("No Such Value '%s' found", key))
	}
	defer func() {
		if msg := recover(); msg != nil {
			panic(fmt.Sprintf("Refusing to convert Option '%s' of type '%s' with value '%s' to '%s'",
				key, reflect.TypeOf(self.values[key].Value), self.values[key].Value, typeName))
		}
	}()
	convFunc(opt.Value)
}

func (self *Options) Int(key string) int {
	var result int
	self.convert(key, "int", func(value interface{}) {
		if value != nil {
			result = value.(int)
			return
		}
		// Avoid panic, return 0 if no value
		result = 0
	})
	return result
}

func (self *Options) String(key string) string {
	var result string
	self.convert(key, "string", func(value interface{}) {
		if value != nil {
			result = value.(string)
			return
		}
		// Avoid panic, return "" if no value
		result = ""
	})
	return result
}

func (self *Options) Bool(key string) bool {
	var result bool
	self.convert(key, "bool", func(value interface{}) {
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
func (self *Options) Slice(key string) []string {
	var result []string
	self.convert(key, "[]string", func(value interface{}) {
		if value != nil {
			result = value.([]string)
			return
		}
		// Avoid panic, return []string{} if no value
		result = []string{}
	})
	return result
}

func (self *Options) IsNil(key string) bool {
	if opt, ok := self.values[key]; ok {
		return opt.Value == nil
	}
	return true
}

func (self *Options) HasKey(key string) bool {
	_, ok := self.values[key]
	return ok
}

func (self *Options) RawValue(key string) interface{} {
	if opt, ok := self.values[key]; ok {
		return opt.Value
	}
	return nil
}

// ***********************************************
// ArgParser Object
// ***********************************************
type ParseModifier func(*ArgParser)

type ArgParser struct {
	Description string
	Name        string
	WordWrap    int
	mutex       sync.Mutex
	args        []string
	options     *Options
	rules       Rules
	err         error
	idx         int
}

// Creates a new instance of the argument parser
func NewParser(modifiers ...ParseModifier) *ArgParser {
	parser := &ArgParser{
		"",
		"",
		200,
		sync.Mutex{},
		[]string{},
		NewOptions(""),
		nil,
		nil,
		0,
	}
	for _, modify := range modifiers {
		modify(parser)
	}
	return parser
}

var isOptional = regexp.MustCompile(`^(\W+)([\w|-]*)$`)

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
	results := NewOptions("")

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
		results.Group(rule.Group).Set(rule.Name, value, rule.Seen)
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

func (self *ArgParser) ParseIni(input []byte) (*Options, error) {
	// Parse the file return a map of the contents
	cfg, err := ini.Load(input)
	if err != nil {
		return nil, err
	}
	values := NewOptions("")
	for _, section := range cfg.Sections() {
		group := cfg.Section(section.Name())
		for _, key := range group.KeyStrings() {
			// Always use our default option group name for the DEFAULT section
			name := section.Name()
			if name == "DEFAULT" {
				name = DefaultOptionGroup
			}
			values.Group(name).Set(key, group.Key(key).String(), false)
		}

	}
	// Apply the ini file values to the commandline and environment variables
	return self.Apply(values)
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

// ***********************************************
// PUBLIC FUNCTIONS
// ***********************************************

func Name(name string) ParseModifier {
	return func(parser *ArgParser) {
		parser.Name = name
	}
}

func Desc(desc string) ParseModifier {
	return func(parser *ArgParser) {
		parser.Description = desc
	}
}

func WrapLen(length int) ParseModifier {
	return func(parser *ArgParser) {
		parser.WordWrap = length
	}
}

// ***********************************************
// Public Word Formatting Functions
// ***********************************************

// Mixing Spaces and Tabs will have undesired effects
func Dedent(input string) string {
	text := []byte(input)

	// find the first \n::space:: combo
	leadingWhitespace := regexp.MustCompile(`(?m)^[ \t]+`)
	idx := leadingWhitespace.FindIndex(text)
	if idx == nil {
		fmt.Printf("Unable to find \\n::space:: combo\n")
		return input
	}
	//fmt.Printf("idx: '%d:%d'\n", idx[0], idx[1])

	// Create a regex to match any the number of spaces we first found
	gobbleRegex := fmt.Sprintf("(?m)^[ \t]{%d}?", (idx[1] - idx[0]))
	//fmt.Printf("gobbleRegex: '%s'\n", gobbleRegex)
	gobbleIndents := regexp.MustCompile(gobbleRegex)
	// Find any identical spaces and remove them
	dedented := gobbleIndents.ReplaceAll(text, []byte{})
	return string(dedented)
}

func DedentTrim(input string, cutset string) string {
	return strings.Trim(Dedent(input), cutset)
}

func WordWrap(msg string, indent int, wordWrap int) string {
	// Remove any previous formating
	regex, _ := regexp.Compile(" {2,}|\n|\t")
	msg = regex.ReplaceAllString(msg, "")

	wordWrapLen := wordWrap - indent
	if wordWrapLen <= 0 {
		panic(fmt.Sprintf("Flag indent spacing '%d' exceeds wordwrap length '%d'\n", indent, wordWrap))
	}

	if len(msg) < wordWrapLen {
		return msg
	}

	// Split the msg into lines
	var lines []string
	var eol int
	for i := 0; i < len(msg); {
		eol = i + wordWrapLen
		// If the End Of Line exceeds the message length + our peek at the next character
		if (eol + 1) >= len(msg) {
			// Slice until the end of the message
			lines = append(lines, msg[i:len(msg)])
			i = len(msg)
			break
		}
		// Slice this portion of the message into a single line
		line := msg[i:eol]
		// If the next character past eol is not a space
		// (Usually means we are in the middle of a word)
		if msg[eol+1] != ' ' {
			// Find the last space before the word wrap
			idx := strings.LastIndex(line, " ")
			eol = i + idx
		}
		lines = append(lines, msg[i:eol])
		i = eol
	}
	var spacer string
	if indent <= 0 {
		spacer = fmt.Sprintf("\n%%s")
	} else {
		spacer = fmt.Sprintf("\n%%-%ds", indent-1)
	}

	//fmt.Print("fmt: %s\n", spacer)
	seperator := fmt.Sprintf(spacer, "")
	return strings.Join(lines, seperator)
}

func castString(name string, value interface{}) (interface{}, error) {
	// If value is nil, return the type default
	if value == nil {
		return "", nil
	}

	// If value is not an string
	if reflect.TypeOf(value).Kind() != reflect.String {
		return 0, errors.New(fmt.Sprintf("Invalid value for '%s' - '%s' is not a String", name, value))
	}
	return value, nil
}

func castInt(name string, value interface{}) (interface{}, error) {
	// If value is nil, return the type default
	if value == nil {
		return 0, nil
	}

	// If it's already an integer of some sort
	kind := reflect.TypeOf(value).Kind()
	switch kind {
	case reflect.Int:
		return value, nil
	case reflect.Int8:
		return int(value.(int8)), nil
	case reflect.Int16:
		return int(value.(int16)), nil
	case reflect.Int32:
		return int(value.(int32)), nil
	case reflect.Int64:
		return int(value.(int64)), nil
	case reflect.Uint8:
		return int(value.(uint8)), nil
	case reflect.Uint16:
		return int(value.(uint16)), nil
	case reflect.Uint32:
		return int(value.(uint32)), nil
	case reflect.Uint64:
		return int(value.(uint64)), nil
	}
	// If it's not an integer, it better be a string that we can cast
	if kind != reflect.String {
		return 0, errors.New(fmt.Sprintf("Invalid value for '%s' - '%s' is not a Integer or Castable string", name, value))
	}
	strValue := value.(string)

	intValue, err := strconv.ParseInt(strValue, 10, 64)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("Invalid value for '%s' - '%s' is not an Integer", name, strValue))
	}
	return int(intValue), nil
}

func castBool(name string, value interface{}) (interface{}, error) {
	// If value is nil, return the type default
	if value == nil {
		return false, nil
	}

	kind := reflect.TypeOf(value).Kind()
	if kind == reflect.Bool {
		return value, nil
	}
	// If it's not a boolean, it better be a string that we can cast
	if kind != reflect.String {
		return 0, errors.New(fmt.Sprintf("Invalid value for '%s' - '%s' is not a Boolean or Castable string", name, value))
	}
	strValue := value.(string)

	boolValue, err := strconv.ParseBool(strValue)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Invalid value for '%s' - '%s' is not a Boolean", name, strValue))
	}
	return bool(boolValue), nil
}

func castStringSlice(name string, value interface{}) (interface{}, error) {
	// If value is nil, return the type default
	if value == nil {
		return []string{}, nil
	}

	kind := reflect.TypeOf(value).Kind()
	if kind == reflect.Slice {
		sliceKind := reflect.TypeOf(value).Elem().Kind()
		// Is already a []string
		if sliceKind == reflect.String {
			return value, nil
		}
		return []string{}, errors.New(fmt.Sprintf("Invalid slice type for '%s' - '%s'  not a String", name, value))
	}

	if kind != reflect.String {
		return []string{}, errors.New(fmt.Sprintf("Invalid slice type for '%s' - '%s' is not a []string or parsable comma delimited string", name, value))
	}
	strValue := value.(string)

	// If no comma is found, then assume this is a single value
	if strings.Index(strValue, ",") == -1 {
		return []string{strValue}, nil
	}

	// Split the values separated by comma's
	return strings.Split(strValue, ","), nil
}
