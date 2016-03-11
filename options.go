package args

import (
	"bytes"
	"fmt"

	"github.com/spf13/cast"
)

// ***********************************************
// Options Object
// ***********************************************
type Options struct {
	group  string
	log    StdLogger
	parser *ArgParser
	values map[string]*OptionValue
	groups map[string]*Options
}

type OptionValue struct {
	Value interface{}
	Seen  bool // Argument was seen on the commandline
}

func (self *ArgParser) NewOptions(group string) *Options {
	groups := make(map[string]*Options)
	new := &Options{
		group,
		self.log,
		self,
		make(map[string]*OptionValue),
		groups,
	}
	// Add the new Options{} to the group of options
	groups[group] = new
	return new
}

func (self *ArgParser) NewOptionsWithGroups(group string, groups map[string]*Options) *Options {
	new := &Options{
		group,
		self.log,
		self,
		make(map[string]*OptionValue),
		groups,
	}
	// Add the new Options{} to the group of options
	groups[group] = new
	return new
}

func (self *ArgParser) NewOptionsFromMap(group string, groups map[string]map[string]*OptionValue) *Options {
	options := self.NewOptions(group)
	for groupName, values := range groups {
		grp := options.Group(groupName)
		for key, opt := range values {
			grp.Set(key, opt.Value, opt.Seen)
		}
	}
	return options
}

func (self *Options) ThreadSafe() *Options {
	return self.parser.GetOpts()
}

func (self *Options) ValuesToString() string {
	var buffer bytes.Buffer
	groupName := self.group
	if groupName == "" {
		groupName = "\"\""
	}
	buffer.WriteString(fmt.Sprintf("%s:\n", groupName))
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
	if group == DefaultOptionGroup {
		return self
	}
	opts, ok := self.groups[group]
	if !ok {
		// TODO: Validate group name has valid characters or at least
		// doesn't have ':' in the name which would conflict with Compare()

		// If group doesn't exist; create it
		new := self.parser.NewOptionsWithGroups(group, self.groups)
		self.groups[group] = new
		return new
	}
	return opts
}

func (self *Options) Set(key string, value interface{}, seen bool) *Options {
	self.values[key] = &OptionValue{value, seen}
	return self
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

func (self *Options) Int(key string) int {
	value, err := cast.ToIntE(self.Interface(key))
	if err != nil {
		self.log.Printf("%s for key '%s'", err.Error(), key)
	}
	return value
}

func (self *Options) String(key string) string {
	value, err := cast.ToStringE(self.Interface(key))
	if err != nil {
		self.log.Printf("%s for key '%s'", err.Error(), key)
	}
	return value
}

func (self *Options) Bool(key string) bool {
	value, err := cast.ToBoolE(self.Interface(key))
	if err != nil {
		self.log.Printf("%s for key '%s'", err.Error(), key)
	}
	return value
}

func (self *Options) StringSlice(key string) []string {
	value, err := cast.ToStringSliceE(self.Interface(key))
	if err != nil {
		self.log.Printf("%s for key '%s'", err.Error(), key)
	}
	return value
}

func (self *Options) IsSet(key string) bool {
	if opt, ok := self.values[key]; ok {
		return opt.Value != nil
	}
	return false
}

func (self *Options) HasKey(key string) bool {
	_, ok := self.values[key]
	return ok
}

func (self *Options) Get(key string) interface{} {
	if opt, ok := self.values[key]; ok {
		return opt.Value
	}
	return nil
}

func (self *Options) Interface(key string) interface{} {
	if opt, ok := self.values[key]; ok {
		return opt.Value
	}
	return nil
}

// TODO: Add these getters
/*Float64(key string) : float64
StringMap(key string) : map[string]interface{}
StringMapString(key string) : map[string]string
Time(key string) : time.Time
Duration(key string) : time.Duration*/
