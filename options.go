package args

import (
	"bytes"
	"fmt"
	"strings"

	"errors"

	"reflect"

	"github.com/spf13/cast"
)

type Value interface {
	ToString(...int) string
	GetValue() interface{}
	GetRule() *Rule
	Seen() bool
}

type Options struct {
	log    StdLogger
	parser *ArgParser
	values map[string]Value
}

type RawValue struct {
	Value interface{}
	Rule  *Rule
}

func (self *RawValue) ToString(indent ...int) string {
	return fmt.Sprintf("%v", self.Value)
}

func (self *RawValue) GetValue() interface{} {
	return self.Value
}

func (self *RawValue) GetRule() *Rule {
	return self.Rule
}

func (self *RawValue) Seen() bool {
	if self.Rule.Flags&Seen != 0 {
		return true
	}
	return false
}

func (self *ArgParser) NewOptions() *Options {
	return &Options{
		values: make(map[string]Value),
		log:    self.log,
		parser: self,
	}
}

//func (self *ArgParser) NewOptionsFromMap(group string, groups map[string]map[string]*OptionValue) *Options {
func (self *ArgParser) NewOptionsFromMap(values map[string]interface{}) *Options {
	options := self.NewOptions()
	for key, value := range values {
		// If the value is a map of interfaces
		obj, ok := value.(map[string]interface{})
		if ok {
			// Convert them to options
			options.Set(key, self.NewOptionsFromMap(obj))
		} else {
			// Else set the value
			options.Set(key, value)
		}
	}
	return options
}

func (self *Options) GetOpts() *Options {
	return self.parser.GetOpts()
}

func (self *Options) GetValue() interface{} {
	return self
}

func (self *Options) GetRule() *Rule {
	return nil
}

func (self *Options) ToString(indent ...int) string {
	var buffer bytes.Buffer
	pad := strings.Repeat(" ", indent[0])

	for key, value := range self.values {
		buffer.WriteString(fmt.Sprintf("%s%s=%v\n", pad, key, value.ToString()))
	}
	return buffer.String()
}

func (self *Options) Group(key string) *Options {
	group, ok := self.values[key]
	// If group doesn't exist; always create it
	if !ok {
		group = self.parser.NewOptions()
		self.values[key] = group
	}
	// If user called Group() on this value, it *should* be an *Option
	options, ok := group.GetValue().(*Options)
	if !ok {
		self.log.Printf("Attempted to call Group(%s) on non *Option type %s",
			key, reflect.TypeOf(options))
	}
	return options
}

func (self *Options) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range self.values {
		// If the value is an *Option
		options, ok := value.(*Options)
		if ok {
			result[key] = options.ToMap()
		} else {
			result[key] = value
		}
	}
	return result
}

func (self *Options) Keys() []string {
	keys := make([]string, 0, len(self.values))
	for key := range self.values {
		keys = append(keys, key)
	}
	return keys
}

func (self *Options) Del(key string) *Options {
	delete(self.values, key)
	return self
}

// Just like Set() but also record the matching rule flags
func (self *Options) SetWithRule(key string, value interface{}, rule *Rule) *Options {
	self.values[key] = &RawValue{value, rule}
	return self
}

// Set an option with a key and value
func (self *Options) Set(key string, value interface{}) *Options {
	return self.SetWithRule(key, value, nil)
}

// Return true if any of the values in this Option object were seen on the command line
func (self *Options) Seen() bool {
	for _, opt := range self.values {
		if opt.Seen() {
			return true
		}
	}
	return false
}

/*
	Return true if none of the options where seen on the command line

	opts, _ := parser.ParseArgs(nil)
	if opts.NoArgs() {
		fmt.Printf("No arguments provided")
		os.Exit(-1)
	}
*/
func (self *Options) NoArgs() bool {
	return self.Seen()
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

func (self *Options) StringMap(key string) map[string]interface{} {
	return self.Group(key).ToMap()
}

func (self *Options) KeySlice(key string) []string {
	return self.Group(key).Keys()
}

func (self *Options) IsSet(key string) bool {
	if opt, ok := self.values[key]; ok {
		return !(opt.GetRule().Flags&NoValue != 0)
	}
	return false
}

// Returns true only if all of the keys given have values set
func (self *Options) Required(keys []string) error {
	for _, key := range keys {
		if !self.IsSet(key) {
			return errors.New(key)
		}
	}
	return nil
}

func (self *Options) HasKey(key string) bool {
	_, ok := self.values[key]
	return ok
}

func (self *Options) Get(key string) interface{} {
	if opt, ok := self.values[key]; ok {
		return opt.GetValue()
	}
	return nil
}

func (self *Options) InspectOpt(key string) Value {
	if opt, ok := self.values[key]; ok {
		return opt
	}
	return nil
}

func (self *Options) Interface(key string) interface{} {
	if opt, ok := self.values[key]; ok {
		return opt.GetValue()
	}
	return nil
}

func (self *Options) FromChangeEvent(event *ChangeEvent) *Options {
	if event.Deleted {
		self.Group(event.Group).Del(event.KeyName)
	} else {
		self.Group(event.Group).Set(event.KeyName, string(event.Value))
	}
	return self
}

// TODO: Add these getters
/*Float64(key string) : float64
StringMap(key string) : map[string]interface{}
StringMapString(key string) : map[string]string
Time(key string) : time.Time
Duration(key string) : time.Duration*/
