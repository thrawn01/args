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
	DefaultTerminator string = "--"
	DefaultOptGroup   string = ""
)

// ***********************************************
//  Types
// ***********************************************
type CastFunc func(string, string) (interface{}, error)
type ActionFunc func(*Rule, string, []string, *int) error
type StoreFunc func(interface{})

// ***********************************************
// RuleModifier Object
// ***********************************************
type RuleModifier struct {
	rule *Rule
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

// ***********************************************
// Rule Object
// ***********************************************
type Rule struct {
	Count       int
	IsPos       int
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

func (self *Rule) Validate() error {
	return nil
}

func (self *Rule) GenerateHelpOpt() (string, string) {
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

func (self *Rule) ComputedValue(values *map[string]string) (interface{}, error) {
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
		return self.Cast(self.Name, *self.Default)
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
// GroupOptions Object
// ***********************************************
type GroupOptions struct {
	options map[string]*Options
}

func NewGroupOptions() *GroupOptions {
	return &GroupOptions{make(map[string]*Options)}
}

func (self *GroupOptions) Get(group string) *Options {
	opts, ok := self.options[group]
	if !ok {
		newOpts := NewOptions()
		self.options[group] = newOpts
		return newOpts
	}
	return opts
}

// ***********************************************
// Options Object
// ***********************************************

type OptionVal struct {
	Value interface{}
	Seen  bool // Argument was seen on the commandline
}

type Options struct {
	Values map[string]*OptionVal
}

func NewOptions() *Options {
	return &Options{make(map[string]*OptionVal)}
}

func (self *Options) Convert(key string, typeName string, convFunc func(value interface{})) {
	opt, ok := self.Values[key]
	if !ok {
		panic(fmt.Sprintf("No Such Option '%s' found", key))
	}
	defer func() {
		if msg := recover(); msg != nil {
			panic(fmt.Sprintf("Refusing to convert Option '%s' of type '%s' to '%s'",
				key, reflect.TypeOf(self.Values[key]), typeName))
		}
	}()
	convFunc(opt.Value)
}

func (self *Options) IsNil(key string) bool {
	if opt, ok := self.Values[key]; ok {
		return opt.Value == nil
	}
	return true
}

func (self *Options) Set(key string, value interface{}, seen bool) {
	self.Values[key] = &OptionVal{value, seen}
}

func (self *Options) NoArgs() bool {
	for _, opt := range self.Values {
		if opt.Seen == true {
			return false
		}
	}
	return true
}

func (self *Options) Int(key string) int {
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

func (self *Options) String(key string) string {
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

func (self *Options) Bool(key string) bool {
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
func (self *Options) Slice(key string) []string {
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
// ArgParser Object
// ***********************************************
type ParseModifier func(*ArgParser)

type ArgParser struct {
	Description string
	Name        string
	WordWrap    int
	mutex       sync.Mutex
	args        []string
	groupOpts   *GroupOptions
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
		NewGroupOptions(),
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
					return errors.New(fmt.Sprintf("Duplicate Opt() called with same name as '%s'", rule.Name))
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

func (self *ArgParser) Opt(name string) *RuleModifier {
	rule := &Rule{Cast: castString, Group: DefaultOptGroup}
	// Create a RuleModifier to configure the rule
	modifier := &RuleModifier{rule}
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
		rule.IsPos = 1
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
		return nil, errors.New("Must create some options to match with args.Opt()" +
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
func (self *ArgParser) Apply(values *map[string]string) (*Options, error) {
	results := NewGroupOptions()

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
		results.Get(rule.Group).Set(rule.Name, value, rule.Seen)
	}
	self.SetGroupOpts(results)
	return self.GetGroupOpts().Get(DefaultOptGroup), nil
}

func (self *ArgParser) SetGroupOpts(groupOpts *GroupOptions) {
	self.mutex.Lock()
	self.groupOpts = groupOpts
	self.mutex.Unlock()
}

func (self *ArgParser) GetGroupOpts() *GroupOptions {
	self.mutex.Lock()
	defer func() {
		self.mutex.Unlock()
	}()
	return self.groupOpts
}

func (self *ArgParser) GetOpts() *Options {
	return self.GetGroupOpts().Get(DefaultOptGroup)
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
	return self.Apply(&values)
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
		flags, message := rule.GenerateHelpOpt()
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

func castString(optName string, strValue string) (interface{}, error) {
	// If empty string is passed, give type init value
	if strValue == "" {
		return "", nil
	}
	return strValue, nil
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
