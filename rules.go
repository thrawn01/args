package args

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

var regexHasPrefix = regexp.MustCompile(`^(\W+)([\w|-]*)$`)

// ***********************************************
// Rule Object
// ***********************************************

type castFunc func(string, interface{}, interface{}) (interface{}, error)
type actionFunc func(*Rule, string, []string, *int) error
type storeFunc func(interface{})
type commandFunc func(*Parser, interface{}) (int, error)

type RuleFlag int64

const (
	IsCommand RuleFlag = 1 << iota
	IsArgument
	IsConfig
	IsConfigGroup
	IsRequired
	IsFlag
	IsGreedy
	HasNoValue
	IsDefaultValue
	IsEnvValue
	WasSeenInArgv
)

type Rule struct {
	Count       int
	Order       int
	Name        string
	RuleDesc    string
	Value       interface{}
	Default     *string
	Aliases     []string
	EnvVars     []string
	Choices     []string
	EnvPrefix   string
	Cast        castFunc
	Action      actionFunc
	StoreValue  storeFunc
	CommandFunc commandFunc
	Group       string
	NotGreedy   bool
	Flags       RuleFlag
}

func newRule() *Rule {
	return &Rule{Cast: castString, Group: DefaultOptionGroup}
}

func (r *Rule) AddAlias(name string, prefixes []string) string {
	switch true {
	case r.HasFlag(IsCommand):
		r.Aliases = append(r.Aliases, name)
		name = fmt.Sprintf("!cmd-%s", name)
	case r.HasFlag(IsFlag):
		// If name begins with a non word character, then user included the prefix in the name
		if regexHasPrefix.MatchString(name) {
			// Attempt to extract the name from the prefix
			group := regexHasPrefix.FindStringSubmatch(name)
			r.Aliases = append(r.Aliases, name)
			name = group[2]
		} else {
			if len(prefixes) != 0 {
				// User specified an prefix to apply to all non prefixed options
				for _, prefix := range prefixes {
					r.Aliases = append(r.Aliases, fmt.Sprintf("%s%s", prefix, name))
				}
			} else {
				// TODO: If it's a short name, less than or equal 2 characters, assume a '-' prefix
				// Apply default '--' prefix if none specified
				r.Aliases = append(r.Aliases, fmt.Sprintf("--%s", name))
			}
		}
	}
	return name
}

func (r *Rule) HasFlag(flag RuleFlag) bool {
	return r.Flags&flag != 0
}

func (r *Rule) SetFlag(flag RuleFlag) {
	r.Flags = r.Flags | flag
}

func (r *Rule) ClearFlag(flag RuleFlag) {
	mask := r.Flags ^ flag
	r.Flags &= mask
}

func (r *Rule) Validate() error {
	return nil
}

func (r *Rule) GenerateUsage() string {
	switch {
	case r.Flags&IsFlag != 0:
		if r.HasFlag(IsRequired) {
			return fmt.Sprintf("%s", r.Aliases[0])
		}
		return fmt.Sprintf("[%s]", r.Aliases[0])
	case r.Flags&IsArgument != 0:
		if r.HasFlag(IsRequired) {
			return fmt.Sprintf("<%s>", r.Name)
		}
		return fmt.Sprintf("[%s]", r.Name)
	}
	return ""
}

func (r *Rule) GenerateHelp() (string, string) {
	var parens []string
	paren := ""

	if !r.HasFlag(IsCommand) {
		if r.Default != nil {
			parens = append(parens, fmt.Sprintf("Default=%s", *r.Default))
		}
		if len(r.EnvVars) != 0 {
			envs := strings.Join(r.EnvVars, ",")
			parens = append(parens, fmt.Sprintf("Env=%s", envs))
		}
		if len(parens) != 0 {
			paren = fmt.Sprintf(" (%s)", strings.Join(parens, ", "))
		}
	}

	if r.HasFlag(IsArgument) {
		return "  " + r.Name, r.RuleDesc
	}
	// TODO: This sort should happen when we validate rules
	sort.Sort(sort.Reverse(sort.StringSlice(r.Aliases)))
	return "  " + strings.Join(r.Aliases, ", "), r.RuleDesc + paren
}

func (r *Rule) MatchesAlias(args []string, idx *int) (bool, string) {
	for _, alias := range r.Aliases {
		if args[*idx] == alias {
			return true, args[*idx]
		}
	}
	return false, ""
}

func (r *Rule) Match(args []string, idx *int) (bool, error) {
	name := r.Name
	var matched bool

	if r.HasFlag(IsConfig) {
		return false, nil
	}

	// If this is an argument
	if r.HasFlag(IsArgument) {
		// And we have already seen this argument and it's not greedy
		if r.HasFlag(WasSeenInArgv) && !r.HasFlag(IsGreedy) {
			return false, nil
		}
	} else {
		// Match any known aliases
		matched, name = r.MatchesAlias(args, idx)
		if !matched {
			return false, nil
		}
	}
	r.SetFlag(WasSeenInArgv)

	// If user defined an action
	if r.Action != nil {
		return true, r.Action(r, name, args, idx)
	}

	// If no actions are specified assume a value follows this argument
	if !r.HasFlag(IsArgument) {
		*idx++
		if len(args) <= *idx {
			return true, errors.New(fmt.Sprintf("Expected '%s' to have an argument", name))
		}
	}

	// If we get here, this argument is associated with either an option value or an positional argument
	value, err := r.Cast(name, r.Value, args[*idx])
	if err != nil {
		return true, err
	}
	r.Value = value
	return true, nil
}

// Returns the appropriate required warning to display to the user
func (r *Rule) RequiredMessage() string {
	switch {
	case r.Flags&IsArgument != 0:
		return fmt.Sprintf("argument '%s' is required", r.Name)
	case r.Flags&IsConfig != 0:
		return fmt.Sprintf("config '%s' is required", r.Name)
	default:
		return fmt.Sprintf("option '%s' is required", r.Aliases[0])
	}
}

func (r *Rule) ComputedValue(values *Options) (interface{}, error) {
	if r.Count != 0 {
		r.Value = r.Count
	}

	// If rule matched argument on command line
	if r.HasFlag(WasSeenInArgv) {
		return r.Value, nil
	}

	// If rule matched environment variable
	value, err := r.GetEnvValue()
	if err != nil {
		return nil, err
	}

	if value != nil {
		// Flag the value is from the environment
		r.SetFlag(IsEnvValue)
		return value, nil
	}

	// TODO: Move this logic from here, This method should be all about getting the value
	if r.HasFlag(IsConfigGroup) {
		return nil, nil
	}

	// If provided our map of values, use that
	if values != nil {
		group := values.Group(r.Group)
		if group.HasKey(r.Name) {
			r.ClearFlag(HasNoValue)
			return r.Cast(r.Name, r.Value, group.Get(r.Name))
		}
	}

	// Apply default if available
	if r.Default != nil {
		r.SetFlag(IsDefaultValue)
		return r.Cast(r.Name, r.Value, *r.Default)
	}

	// TODO: Move this logic from here, This method should be all about getting the value
	if r.HasFlag(IsRequired) {
		return nil, errors.New(r.RequiredMessage())
	}

	// Flag that we found no value for this rule
	r.SetFlag(HasNoValue)

	// Return the default value for our type choice
	value, _ = r.Cast(r.Name, r.Value, nil)
	return value, nil
}

func (r *Rule) GetEnvValue() (interface{}, error) {
	if r.EnvVars == nil {
		return nil, nil
	}

	for _, varName := range r.EnvVars {
		//if value, ok := os.LookupEnv(varName); ok {
		if value := os.Getenv(varName); value != "" {
			return r.Cast(varName, r.Value, value)
		}
	}
	return nil, nil
}

func (r *Rule) BackendKey() Key {
	// Do this so users are not surprised root isn't prefixed with "/"
	//rootPath = "/" + strings.TrimPrefix(rootPath, "/")

	// Just return the group
	if r.HasFlag(IsConfigGroup) {
		return Key{Group: r.Group}
	}

	return Key{Group: r.Group, Name: r.Name}
}

// ***********************************************
// Rules Object
// ***********************************************
type Rules []*Rule

func (rs Rules) Len() int {
	return len(rs)
}

func (rs Rules) Less(left, right int) bool {
	return rs[left].Order < rs[right].Order
}

func (rs Rules) Swap(left, right int) {
	rs[left], rs[right] = rs[right], rs[left]
}
