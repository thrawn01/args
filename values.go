package args

import (
	"context"
	"errors"
	"os/user"
	"path/filepath"
	"reflect"

	"github.com/spf13/cast"
)

// The interface the user interacts with to retrieve values returned by the parser
type Values interface {
	Store
	Value

	// Value Accessors
	Keys() []Key
	Value(Key) Value

	// Safe Accessors (These accessors never return nil or panic)
	Group(string) Values
	String(string) string
	Int(string) int
	FilePath(key string) string
	Bool(key string) bool
	StringSlice(key string) []string
	StringMap(key string) map[string]string

	// Integrative Methods
	IsSet(key string) bool
	IsEnv(key string) bool
	IsArg(key string) bool
	IsDefault(key string) bool
	Seen() bool
	NoArgs() bool

	// Utility Methods
	ToMap() map[string]interface{}
	Del(Key) Values
}

// Implements the `Values` interface which contains `TypedValue` items. This struct is used after
// type validation when values are asserted to their final type and are ready to be consumed by the end user.
type TypedValues struct {
	values map[Key]Value
	log    StdLogger
	parser *PosParser
	key    Key
	src    SourceFlag
	rule   *Rule
}

// Create an empty `TypedValues` struct associated with this parser
func (s *PosParser) NewTypedValues(rule *Rule) *TypedValues {
	return &TypedValues{
		key:    Key{Group: DefaultOptionGroup},
		values: make(map[Key]Value),
		log:    s.log,
		parser: s,
		rule:   rule,
	}
}

// Create a new `TypedValues` struct from a map
func (s *PosParser) ValuesFromMap(src map[string]interface{}) *TypedValues {
	values := s.NewTypedValues(nil)
	values.src = FromMap

	for key, value := range src {
		// If the value is a map of interfaces
		obj, ok := value.(map[string]interface{})
		if ok {
			// Convert sub maps to `TypedValues`
			values.Set(context.Background(), Key{Group: key}, s.ValuesFromMap(obj))
		} else {
			values.Set(context.Background(),
				Key{
					Name:  key,
					Group: DefaultOptionGroup,
				},
				TypedValue{
					Value: value,
					Src:   FromMap,
				})
		}
	}
	return values
}

// ------------------------------------------------------
// ------------------------------------------------------------
// Value Methods
// ------------------------------------------------------------
// ------------------------------------------------------

// Required for `TypedValues` to implement the `Value` interface
func (s *TypedValues) GetValue() interface{} {
	return s
}

// Required for `TypedValues` to implement the `Value` interface
func (s *TypedValues) GetRule() *Rule {
	return s.rule
}

// Required for `TypedValues` to implement the `Value` interface
func (s *TypedValues) GetKey() Key {
	return s.key
}

// Required for `TypedValues` to implement the `Value` interface
func (s *TypedValues) Interface() interface{} {
	return s
}

// Required for `TypedValues` to implement the `Value` interface
func (s *TypedValues) Source() SourceFlag {
	return s.src
}

// ------------------------------------------------------
// ------------------------------------------------------------
// Store Methods
// ------------------------------------------------------------
// ------------------------------------------------------

// Required for `TypedValues` to implement the `Store` interface
func (s *TypedValues) Close() {}

// Required to implement the `Store` interface; Gets the value associated with this key
func (s *TypedValues) Get(ctx context.Context, key Key) (Value, error) {
	if val, ok := s.values[key]; ok {
		return val, nil
	}
	return nil, nil
}

// Required to implement the `Store` interface; Sets the value provided
func (s *TypedValues) Set(ctx context.Context, key Key, value Value) error {
	s.values[key] = value
	return nil
}

// Required for `TypedValues` to implement the `Store` interface
func (s *TypedValues) Watch(ctx context.Context, root string) (<-chan ChangeEvent, error) {
	return nil, errors.New("not implemented")
}

// Required for `TypedValues` to implement the `Store` interface;
func (s *TypedValues) List(ctx context.Context, key Key) ([]Value, error) {
	group := s.Group(key.String())
	var results []Value

	for _, key := range group.Keys() {
		results = append(results, group.Value(key))
	}
	return results, nil
}

// ------------------------------------------------------
// ------------------------------------------------------------
// Value Accessors (These accessors can return nil)
// ------------------------------------------------------------
// ------------------------------------------------------

// Get the `Value` struct for this key; Return nil if not found
func (s *TypedValues) Value(key Key) Value {
	return s.values[key]
}

// Return a list of all keys for the `Values` object
func (s *TypedValues) Keys() []Key {
	keys := make([]Key, 0, len(s.values))
	for key := range s.values {
		keys = append(keys, key)
	}
	return keys
}

// ------------------------------------------------------
// ------------------------------------------------------------
// Safe Accessors (These accessors never return nil or panic)
// ------------------------------------------------------------
// ------------------------------------------------------

// Returns a `Values` interface that contains the values for the group specified;
// Returns an empty `Values` object if the specified group does not exist.
func (s *TypedValues) Group(key string) Values {
	// "" is not a valid group
	if key == "" {
		return s
	}

	value, ok := s.values[Key{Name: key}]
	// If group doesn't exist; always create it
	if !ok {
		value = s.parser.NewTypedValues(nil)
		s.values[Key{Name: key}] = value
	}

	// If user called Group() on this value, it *should* be a type that implements the `Values` interface
	obj, ok := value.(Values)
	if !ok {
		s.log.Printf("Attempted to call Group(%s) on non `Values` type %s", key, reflect.TypeOf(value.Interface()))
		return s.parser.NewTypedValues(nil)
	}
	return obj
}

// Returns the requested value as a string
func (s *TypedValues) String(key string) string {
	if val, ok := s.values[Key{Name: key}]; ok {
		value, err := cast.ToStringE(val.Interface())
		if err != nil {
			s.log.Printf("Values: %s for key '%s'", err.Error(), key)
		}
		return value
	}
	s.log.Printf("Values: no such key '%s'", key)
	return ""
}

// Returns the requested value as an int
func (s *TypedValues) Int(key string) int {
	if val, ok := s.values[Key{Name: key}]; ok {
		value, err := cast.ToIntE(val.Interface())
		if err != nil {
			s.log.Printf("Values: %s for key '%s'", err.Error(), key)
		}
		return value
	}
	s.log.Printf("Values: no such key '%s'", key)
	return 0
}

// Assumes the Value is a string path and performs tilde '~' expansion if necessary
func (s *TypedValues) FilePath(key string) string {
	val, ok := s.values[Key{Name: key}]
	if !ok {
		s.log.Printf("Values: no such key '%s'", key)
		return ""
	}

	path, err := cast.ToStringE(val.Interface())
	if err != nil {
		s.log.Printf("Values: %s for key '%s'", err.Error(), key)
	}

	if len(path) == 0 || path[0] != '~' {
		return path
	}

	usr, err := user.Current()
	if err != nil {
		s.log.Printf("'%s': while determining user for '%s' expansion: %s", key, path, err)
		return path
	}
	return filepath.Join(usr.HomeDir, path[1:])
}

func (s *TypedValues) Bool(key string) bool {
	if val, ok := s.values[Key{Name: key}]; ok {
		value, err := cast.ToBoolE(val.Interface())
		if err != nil {
			s.log.Printf("Values: %s for key '%s'", err.Error(), key)
		}
		return value
	}
	s.log.Printf("Values: no such key '%s'", key)
	return false
}

func (s *TypedValues) StringSlice(key string) []string {
	if val, ok := s.values[Key{Name: key}]; ok {
		value, err := cast.ToStringSliceE(val.Interface())
		if err != nil {
			s.log.Printf("%s for key '%s'", err.Error(), key)
		}
		return value
	}
	s.log.Printf("Values: no such key '%s'", key)
	return []string{}
}

func (s *TypedValues) StringMap(key string) map[string]string {
	group := s.Group(key)

	result := make(map[string]string)
	for _, key := range group.Keys() {
		result[key.Name] = group.String(key.Name)
	}
	return result
}

// TODO: Add these getters
/*Float64(key string) : float64
Time(key string) : time.Time
Duration(key string) : time.Duration*/

// ------------------------------------------------------
// ------------------------------------------------------------
// Integrative Methods
// ------------------------------------------------------------
// ------------------------------------------------------

// Returns true if the argument value is set; Use IsDefault(), IsEnv(), IsArg()
// to determine where the parser found the value
func (s *TypedValues) IsSet(key string) bool {
	_, ok := s.values[Key{Name: key}]
	return ok
	/*		rule := val.GetRule()
		if rule == nil {
			return false
		}
		return !(rule.Flags&HasNoValue != 0)
	}
	return false*/
}

// Returns true if this argument is set via the environment
func (s *TypedValues) IsEnv(key string) bool {
	if val, ok := s.values[Key{Name: key}]; ok {
		return val.Source()&FromEnv != 0
	}
	return false
}

// Returns true if this argument is set via the command line
func (s *TypedValues) IsArg(key string) bool {
	if val, ok := s.values[Key{Name: key}]; ok {
		return val.Source()&FromArgv != 0
	}
	return false
}

// Returns true if this argument is set via the default value
func (s *TypedValues) IsDefault(key string) bool {
	if val, ok := s.values[Key{Name: key}]; ok {
		return val.Source()&FromDefault != 0
	}
	return false
}

// Return true if any of the values where parsed from argv
func (s *TypedValues) Seen() bool {
	for _, val := range s.values {
		if val.Source() == FromArgv {
			return true
		}
	}
	return false
}

//	Return true if none of the values where parsed from argv
func (s *TypedValues) NoArgs() bool {
	return !s.Seen()
}

// ------------------------------------------------------
// ------------------------------------------------------------
// Utility Methods
// ------------------------------------------------------------
// ------------------------------------------------------

// Return a map representation of the `Values` object
func (s *TypedValues) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range s.values {
		val, ok := value.(Values)
		if ok {
			result[key.Group] = val.ToMap()
		} else {
			result[key.Name] = value.Interface()
		}
	}
	return result
}

// Delete the given key from `Values`
func (s *TypedValues) Del(key Key) Values {
	delete(s.values, key)
	return s
}

// Returns true if this argument was set via the command line or was set by an environment variable
func (s *TypedValues) WasSeen(key string) bool {
	if val, ok := s.values[Key{Name: key}]; ok {
		rule := val.GetRule()
		if rule == nil {
			return false
		}
		return (rule.Flags&WasSeenInArgv != 0) || (rule.Flags&IsEnvValue != 0)
	}
	return false
}

// Returns true only if all of the keys given have values set
func (s *TypedValues) Required(keys []string) error {
	for _, key := range keys {
		if !s.IsSet(key) {
			return errors.New(key)
		}
	}
	return nil
}

func (s *TypedValues) HasKey(key string) bool {
	_, ok := s.values[Key{Name: key}]
	return ok
}

func (s *TypedValues) FromChangeEvent(event ChangeEvent) *TypedValues {
	if event.Deleted {
		s.Group(event.Key.Group).Del(event.Key)
	} else {
		s.Group(event.Key.Group).Set(context.Background(), event.Key, event.Value)
	}
	return s
}

// TODO: Once we add subcommand support to PosParser
// Set the sub command list returned by `SubCommands()`
/*func (s *TypedValues) SetSubCommands(values []string) {
	s.Set("!sub-commands", values)
}

// Return a list of sub commands that the user provided to reach the sub parser these val are for
func (s *TypedValues) SubCommands() []string {
	return s.Get("!sub-commands").([]string)
}*/

// ------------------------------------------------------
// ------------------------------------------------------------
// Typed Implementation of Value
// ------------------------------------------------------------
// ------------------------------------------------------

type Value interface {
	Interface() interface{}
	GetRule() *Rule
	Source() SourceFlag
}

// Implements the Value interface
type TypedValue struct {
	Value interface{}
	Rule  *Rule
	Src   SourceFlag
}

func (s TypedValue) Interface() interface{} { return s.Value }
func (s TypedValue) GetRule() *Rule         { return s.Rule }
func (s TypedValue) Source() SourceFlag     { return s.Src }

// ------------------------------------------------------
// ------------------------------------------------------------
// String Implementation of Value
// ------------------------------------------------------------
// ------------------------------------------------------

type StringValue struct {
	Key   Key
	Value string
	Rule  *Rule
	Src   SourceFlag
}

func (s StringValue) Interface() interface{} { return s.Value }
func (s StringValue) GetRule() *Rule         { return s.Rule }
func (s StringValue) GetKey() Key            { return s.Key }
func (s StringValue) Source() SourceFlag     { return s.Src }
