package args

import (
	"bytes"
	"errors"
	"fmt"
	"os/user"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/spf13/cast"
)

type Value interface {
	ToString(...int) string
	GetValue() interface{}
	GetRule() *Rule
	Seen() bool
}

type Options struct {
	values map[string]Value
	log    StdLogger
	parser *Parser
}

type RawValue struct {
	Value interface{}
	Rule  *Rule
}

func (rv *RawValue) ToString(indent ...int) string {
	return fmt.Sprintf("%v", rv.Value)
}

func (rv *RawValue) GetValue() interface{} {
	return rv.Value
}

func (rv *RawValue) GetRule() *Rule {
	return rv.Rule
}

func (rv *RawValue) Seen() bool {
	if rv.Rule == nil {
		return false
	}
	if rv.Rule.Flags&WasSeenInArgv != 0 {
		return true
	}
	return false
}

func (p *Parser) NewOptions() *Options {
	return &Options{
		values: make(map[string]Value),
		log:    p.log,
		parser: p,
	}
}

func (p *Parser) NewOptionsFromMap(values map[string]interface{}) *Options {
	options := p.NewOptions()
	for key, value := range values {
		// If the value is a map of interfaces
		obj, ok := value.(map[string]interface{})
		if ok {
			// Convert them to options
			options.SetWithOptions(key, p.NewOptionsFromMap(obj))
		} else {
			// Else set the value
			options.Set(key, value)
		}
	}
	return options
}

func (o *Options) GetOpts() *Options {
	return o.parser.GetOpts()
}

func (o *Options) GetValue() interface{} {
	return o
}

func (o *Options) GetRule() *Rule {
	return nil
}

func (o *Options) ToString(indented ...int) string {
	var buffer bytes.Buffer
	indent := 2
	if len(indented) != 0 {
		indent = indented[0]
	}

	buffer.WriteString("{\n")
	pad := strings.Repeat(" ", indent)

	// Sort the values so testing is consistent
	var keys []string
	for key := range o.values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		buffer.WriteString(fmt.Sprintf("%s'%s' = %s\n", pad, key, o.values[key].ToString(indent+2)))
	}
	buffer.WriteString(pad[2:] + "}")
	return buffer.String()
}

func (o *Options) Group(key string) *Options {
	// "" is not a valid group
	if key == "" {
		return o
	}

	group, ok := o.values[key]
	// If group doesn't exist; always create it
	if !ok {
		group = o.parser.NewOptions()
		o.values[key] = group
	}
	// If user called Group() on this value, it *should* be an
	// *Option, map[string]string or map[string]interface{}
	options := o.ToOption(group.GetValue())
	if options == nil {
		o.log.Printf("Attempted to call Group(%s) on non *Option or map[string]interface type %s",
			key, reflect.TypeOf(group.GetValue()))
		// Do this so we don't panic if we can't cast this group
		options = o.parser.NewOptions()
	}
	return options
}

// Given an interface of map[string]string or map[string]string
// or *Option return an *Options with the same content.
// return nil if un-successful
func (o *Options) ToOption(from interface{}) *Options {
	if options, ok := from.(*Options); ok {
		return options
	}
	if stringMap, ok := from.(map[string]string); ok {
		result := make(map[string]interface{})
		for key, value := range stringMap {
			result[key] = value
		}
		return o.parser.NewOptionsFromMap(result)
	}

	if interfaceMap, ok := from.(map[string]interface{}); ok {
		return o.parser.NewOptionsFromMap(interfaceMap)
	}
	return nil
}

func (o *Options) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range o.values {
		// If the value is an *Option
		options, ok := value.(*Options)
		if ok {
			result[key] = options.ToMap()
		} else {
			result[key] = value.GetValue()
		}
	}
	return result
}

func (o *Options) Keys() []string {
	keys := make([]string, 0, len(o.values))
	for key := range o.values {
		keys = append(keys, key)
	}
	return keys
}

func (o *Options) Del(key string) *Options {
	delete(o.values, key)
	return o
}

func (o *Options) SetWithOptions(key string, value *Options) *Options {
	o.values[key] = value
	return o
}

// Just like Set() but also record the matching rule flags
func (o *Options) SetWithRule(key string, value interface{}, rule *Rule) *Options {
	o.values[key] = &RawValue{value, rule}
	return o
}

// Set an option with a key and value
func (o *Options) Set(key string, value interface{}) *Options {
	return o.SetWithRule(key, value, nil)
}

// Set the sub command list returned by `SubCommands()`
func (o *Options) SetSubCommands(values []string) {
	o.Set("!sub-commands", values)
}

// Return a list of sub commands that the user provided to reach the sub parser these options are for
func (o *Options) SubCommands() []string {
	return o.Get("!sub-commands").([]string)
}

// Return true if any of the values in this Option object were seen on the command line
func (o *Options) Seen() bool {
	for _, opt := range o.values {
		if opt.Seen() {
			return true
		}
	}
	return false
}

/*
	Return true if none of the options where seen on the command line

	opts, _ := parser.Parse(nil)
	if opts.NoArgs() {
		fmt.Printf("No arguments provided")
		os.Exit(-1)
	}
*/
func (o *Options) NoArgs() bool {
	return !o.Seen()
}

func (o *Options) Int(key string) int {
	value, err := cast.ToIntE(o.Interface(key))
	if err != nil {
		o.log.Printf("%s for key '%s'", err.Error(), key)
	}
	return value
}

func (o *Options) String(key string) string {
	value, err := cast.ToStringE(o.Interface(key))
	if err != nil {
		o.log.Printf("%s for key '%s'", err.Error(), key)
	}
	return value
}

// Assumes the option is a string path and performs tilde '~' expansion if necessary
func (o *Options) FilePath(key string) string {
	path, err := cast.ToStringE(o.Interface(key))
	if err != nil {
		o.log.Printf("%s for key '%s'", err.Error(), key)
	}

	if len(path) == 0 || path[0] != '~' {
		return path
	}

	usr, err := user.Current()
	if err != nil {
		o.log.Printf("'%s': while determining user for '%s' expansion: %s", key, path, err)
		return path
	}
	return filepath.Join(usr.HomeDir, path[1:])
}

func (o *Options) Bool(key string) bool {
	value, err := cast.ToBoolE(o.Interface(key))
	if err != nil {
		o.log.Printf("%s for key '%s'", err.Error(), key)
	}
	return value
}

func (o *Options) StringSlice(key string) []string {
	value, err := cast.ToStringSliceE(o.Interface(key))
	if err != nil {
		o.log.Printf("%s for key '%s'", err.Error(), key)
	}
	return value
}

func (o *Options) StringMap(key string) map[string]string {
	group := o.Group(key)

	result := make(map[string]string)
	for _, key := range group.Keys() {
		result[key] = group.String(key)
	}
	return result
}

func (o *Options) KeySlice(key string) []string {
	return o.Group(key).Keys()
}

// Returns true if the argument value is set.
// Use IsDefault(), IsEnv(), IsArg() to determine how the parser set the value
func (o *Options) IsSet(key string) bool {
	if opt, ok := o.values[key]; ok {
		rule := opt.GetRule()
		if rule == nil {
			return false
		}
		return !(rule.Flags&HasNoValue != 0)
	}
	return false
}

// Returns true if this argument is set via the environment
func (o *Options) IsEnv(key string) bool {
	if opt, ok := o.values[key]; ok {
		rule := opt.GetRule()
		if rule == nil {
			return false
		}
		return rule.Flags&IsEnvValue != 0
	}
	return false
}

// Returns true if this argument is set via the command line
func (o *Options) IsArg(key string) bool {
	if opt, ok := o.values[key]; ok {
		rule := opt.GetRule()
		if rule == nil {
			return false
		}
		return rule.Flags&WasSeenInArgv != 0
	}
	return false
}

// Returns true if this argument is set via the default value
func (o *Options) IsDefault(key string) bool {
	if opt, ok := o.values[key]; ok {
		rule := opt.GetRule()
		if rule == nil {
			return false
		}
		return rule.Flags&IsDefaultValue != 0
	}
	return false
}

// Returns true if this argument was set via the command line or was set by an environment variable
func (o *Options) WasSeen(key string) bool {
	if opt, ok := o.values[key]; ok {
		rule := opt.GetRule()
		if rule == nil {
			return false
		}
		return (rule.Flags&WasSeenInArgv != 0) || (rule.Flags&IsEnvValue != 0)
	}
	return false
}

// Returns true only if all of the keys given have values set
func (o *Options) Required(keys []string) error {
	for _, key := range keys {
		if !o.IsSet(key) {
			return errors.New(key)
		}
	}
	return nil
}

func (o *Options) HasKey(key string) bool {
	_, ok := o.values[key]
	return ok
}

func (o *Options) Get(key string) interface{} {
	if opt, ok := o.values[key]; ok {
		return opt.GetValue()
	}
	return nil
}

func (o *Options) InspectOpt(key string) Value {
	if opt, ok := o.values[key]; ok {
		return opt
	}
	return nil
}

func (o *Options) Interface(key string) interface{} {
	if opt, ok := o.values[key]; ok {
		return opt.GetValue()
	}
	return nil
}

func (o *Options) FromChangeEvent(event ChangeEvent) *Options {
	if event.Deleted {
		o.Group(event.Key.Group).Del(event.Key.Name)
	} else {
		o.Group(event.Key.Group).Set(event.Key.Name, event.Value)
	}
	return o
}

// TODO: Add these getters
/*Float64(key string) : float64
Time(key string) : time.Time
Duration(key string) : time.Duration*/
