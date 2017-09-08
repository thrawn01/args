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
var regexInValidRuleName = regexp.MustCompile(`[!"#$&'/()*;<>{|}\\\\~\s]`)

// ***********************************************
// Rule Object
// ***********************************************

type CastFunc func(string, interface{}, interface{}) (interface{}, error)
type ActionFunc func(*Rule, string, []string, *int) error
type StoreFunc func(interface{})
type CommandFunc func(*Parser, interface{}) (int, error)

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
	IsExpectingValue
	IsCountFlag
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
	Cast        CastFunc
	Action      ActionFunc
	StoreValue  StoreFunc
	CommandFunc CommandFunc
	Group       string
	NotGreedy   bool
	Flags       RuleFlag
}

func newRule() *Rule {
	return &Rule{Cast: castString, Group: DefaultOptionGroup}
}

func (self *Rule) AddAlias(name string, prefixes []string) string {
	switch true {
	case self.HasFlag(IsCommand):
		self.Aliases = append(self.Aliases, name)
		name = fmt.Sprintf("!cmd-%s", name)
	case self.HasFlag(IsFlag):
		// If name begins with a non word character, then user included the prefix in the name
		if regexHasPrefix.MatchString(name) {
			// Attempt to extract the name from the prefix
			group := regexHasPrefix.FindStringSubmatch(name)
			self.Aliases = append(self.Aliases, name)
			name = group[2]
		} else {
			if len(prefixes) != 0 {
				// User specified an prefix to apply to all non prefixed options
				for _, prefix := range prefixes {
					self.Aliases = append(self.Aliases, fmt.Sprintf("%s%s", prefix, name))
				}
			} else {
				// TODO: If it's a short name, less than or equal 2 characters, assume a '-' prefix
				// Apply default '--' prefix if none specified
				self.Aliases = append(self.Aliases, fmt.Sprintf("--%s", name))
			}
		}
	}
	return name
}

func (self *Rule) HasFlag(flag RuleFlag) bool {
	return self.Flags&flag != 0
}

func (self *Rule) SetFlag(flag RuleFlag) {
	self.Flags = (self.Flags | flag)
}

func (self *Rule) ClearFlag(flag RuleFlag) {
	mask := (self.Flags ^ flag)
	self.Flags &= mask
}

func (self *Rule) Validate() error {
	return nil
}

func (self *Rule) GenerateUsage() string {
	switch {
	case self.Flags&IsFlag != 0:
		if self.HasFlag(IsRequired) {
			return fmt.Sprintf("%s", self.Aliases[0])
		}
		return fmt.Sprintf("[%s]", self.Aliases[0])
	case self.Flags&IsArgument != 0:
		if self.HasFlag(IsRequired) {
			return fmt.Sprintf("<%s>", self.Name)
		}
		return fmt.Sprintf("[%s]", self.Name)
	}
	return ""
}

func (self *Rule) GenerateHelp() (string, string) {
	var parens []string
	paren := ""

	if !self.HasFlag(IsCommand) {
		if self.Default != nil {
			parens = append(parens, fmt.Sprintf("Default=%s", *self.Default))
		}
		if len(self.EnvVars) != 0 {
			envs := strings.Join(self.EnvVars, ",")
			parens = append(parens, fmt.Sprintf("Env=%s", envs))
		}
		if len(parens) != 0 {
			paren = fmt.Sprintf(" (%s)", strings.Join(parens, ", "))
		}
	}

	if self.HasFlag(IsArgument) {
		return ("  " + self.Name), self.RuleDesc
	}
	// TODO: This sort should happen when we validate rules
	sort.Sort(sort.Reverse(sort.StringSlice(self.Aliases)))
	return ("  " + strings.Join(self.Aliases, ", ")), (self.RuleDesc + paren)
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

	if self.HasFlag(IsConfig) {
		return false, nil
	}

	// If this is an argument
	if self.HasFlag(IsArgument) {
		// And we have already seen this argument and it's not greedy
		if self.HasFlag(WasSeenInArgv) && !self.HasFlag(IsGreedy) {
			return false, nil
		}
	} else {
		// Match any known aliases
		matched, name = self.MatchesAlias(args, idx)
		if !matched {
			return false, nil
		}
	}
	self.SetFlag(WasSeenInArgv)

	// If user defined an action
	if self.Action != nil {
		return true, self.Action(self, name, args, idx)
	}

	// If no actions are specified assume a value follows this argument
	if !self.HasFlag(IsArgument) {
		*idx++
		if len(args) <= *idx {
			return true, errors.New(fmt.Sprintf("Expected '%s' to have an argument", name))
		}
	}

	// If we get here, this argument is associated with either an option value or an positional argument
	value, err := self.Cast(name, self.Value, args[*idx])
	if err != nil {
		return true, err
	}
	self.Value = value
	return true, nil
}

// Returns the appropriate required warning to display to the user
func (self *Rule) RequiredMessage() string {
	switch {
	case self.Flags&IsArgument != 0:
		return fmt.Sprintf("argument '%s' is required", self.Name)
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
	if self.HasFlag(WasSeenInArgv) {
		return self.Value, nil
	}

	// If rule matched environment variable
	value, err := self.GetEnvValue()
	if err != nil {
		return nil, err
	}

	if value != nil {
		// Flag the value is from the environment
		self.SetFlag(IsEnvValue)
		return value, nil
	}

	// TODO: Move this logic from here, This method should be all about getting the value
	if self.HasFlag(IsConfigGroup) {
		return nil, nil
	}

	// If provided our map of values, use that
	if values != nil {
		group := values.Group(self.Group)
		if group.HasKey(self.Name) {
			self.ClearFlag(HasNoValue)
			return self.Cast(self.Name, self.Value, group.Get(self.Name))
		}
	}

	// Apply default if available
	if self.Default != nil {
		self.SetFlag(IsDefaultValue)
		return self.Cast(self.Name, self.Value, *self.Default)
	}

	// TODO: Move this logic from here, This method should be all about getting the value
	if self.HasFlag(IsRequired) {
		return nil, errors.New(self.RequiredMessage())
	}

	// Flag that we found no value for this rule
	self.SetFlag(HasNoValue)

	// Return the default value for our type choice
	value, _ = self.Cast(self.Name, self.Value, nil)
	return value, nil
}

func (self *Rule) GetEnvValue() (interface{}, error) {
	if self.EnvVars == nil {
		return nil, nil
	}

	for _, varName := range self.EnvVars {
		//if value, ok := os.LookupEnv(varName); ok {
		if value := os.Getenv(varName); value != "" {
			return self.Cast(varName, self.Value, value)
		}
	}
	return nil, nil
}

func (self *Rule) Key() Key {
	// Do this so users are not surprised root isn't prefixed with "/"
	//rootPath = "/" + strings.TrimPrefix(rootPath, "/")

	// Just return the group
	if self.HasFlag(IsConfigGroup) {
		return Key{Group: self.Group}
	}

	return Key{Group: self.Group, Name: self.Name}
}

// ***********************************************
// Rules Object
// ***********************************************
type Rules []*Rule

func (s Rules) Len() int {
	return len(s)
}

func (s Rules) Less(left, right int) bool {
	return s[left].Order < s[right].Order
}

func (s Rules) Swap(left, right int) {
	s[left], s[right] = s[right], s[left]
}

func (s Rules) ValidateRules() error {
	var greedyRule *Rule
	for idx, rule := range s {
		// Duplicate rule check
		next := idx + 1
		if next < len(s) {
			for ; next < len(s); next++ {
				// If the name and groups are the same
				if rule.Name == s[next].Name && rule.Group == s[next].Group {
					return errors.Errorf("Duplicate argument or flag '%s' defined", rule.Name)
				}
				// If the alias is a duplicate
				for _, alias := range s[next].Aliases {
					var duplicate string

					// if rule.Aliases contains 'alias'
					for _, item := range rule.Aliases {
						if item == alias {
							duplicate = alias
						}
					}
					if len(duplicate) != 0 {
						return errors.Errorf("Duplicate alias '%s' for '%s' redefined by '%s'",
							duplicate, rule.Name, s[next].Name)
					}
				}
				if rule.Name == s[next].Name && rule.Group == s[next].Group {
					return errors.Errorf("Duplicate argument or flag '%s' defined", rule.Name)
				}
			}
		}
		// Ensure user didn't set a bad default value
		if rule.Cast != nil && rule.Default != nil {
			_, err := rule.Cast(rule.Name, nil, *rule.Default)
			if err != nil {
				return errors.Wrap(err, "Bad default value")
			}
		}
		// Check for invalid option and argument names
		if regexInValidRuleName.MatchString(rule.Name) {
			if !strings.HasPrefix(rule.Name, "!cmd-") {
				return errors.Errorf("Bad argument or flag '%s'; contains invalid characters",
					rule.Name)
			}
		}

		if !rule.HasFlag(IsArgument) {
			continue
		}
		// If we already found a greedy rule, no other argument should follow
		if greedyRule != nil {
			return errors.Errorf("'%s' is ambiguous when following greedy argument '%s'",
				rule.Name, greedyRule.Name)
		}
		// Check for ambiguous greedy arguments
		if rule.HasFlag(IsGreedy) {
			if greedyRule == nil {
				greedyRule = rule
			}
		}
	}
	return nil
}

func (s Rules) GetRule(name string) *Rule {
	for _, rule := range s {
		if rule.Name == name {
			return rule
		}
	}
	return nil
}
