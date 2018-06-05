package args

import "reflect"

type RuleModifier struct {
	rule   *Rule
	parser *Parser
}

func NewRuleModifier(parser *Parser) *RuleModifier {
	return &RuleModifier{newRule(), parser}
}

func newRuleModifier(rule *Rule, parser *Parser) *RuleModifier {
	return &RuleModifier{rule, parser}
}

func (rm *RuleModifier) GetRule() *Rule {
	return rm.rule
}

func (rm *RuleModifier) IsString() *RuleModifier {
	rm.rule.Cast = castString
	return rm
}

// If the option is seen on the command line, the value is 'true'
func (rm *RuleModifier) IsTrue() *RuleModifier {
	rm.rule.Action = func(rule *Rule, alias string, args []string, idx *int) error {
		rule.Value = true
		return nil
	}
	rm.rule.Cast = castBool
	return rm
}

func (rm *RuleModifier) IsBool() *RuleModifier {
	rm.rule.Cast = castBool
	rm.rule.Value = false
	return rm
}

func (rm *RuleModifier) Default(value string) *RuleModifier {
	rm.rule.Default = &value
	return rm
}

func (rm *RuleModifier) StoreInt(dest *int) *RuleModifier {
	// Implies IsInt()
	rm.rule.Cast = castInt
	rm.rule.StoreValue = func(value interface{}) {
		*dest = value.(int)
	}
	return rm
}

func (rm *RuleModifier) IsInt() *RuleModifier {
	rm.rule.Cast = castInt
	return rm
}

func (rm *RuleModifier) StoreTrue(dest *bool) *RuleModifier {
	rm.rule.Action = func(rule *Rule, alias string, args []string, idx *int) error {
		rule.Value = true
		return nil
	}
	rm.rule.Cast = castBool
	rm.rule.StoreValue = func(value interface{}) {
		*dest = value.(bool)
	}
	return rm
}

func (rm *RuleModifier) IsStringSlice() *RuleModifier {
	rm.rule.Cast = castStringSlice
	rm.rule.SetFlag(IsGreedy)
	return rm
}

func (rm *RuleModifier) IsStringMap() *RuleModifier {
	rm.rule.Cast = castStringMap
	rm.rule.SetFlag(IsGreedy)
	return rm
}

// TODO: Make this less horribad, and use more reflection to make the interface simpler
// It should also take more than just []string but also []int... etc...
func (rm *RuleModifier) StoreStringSlice(dest *[]string) *RuleModifier {
	rm.rule.Cast = castStringSlice
	rm.rule.StoreValue = func(src interface{}) {
		// First clear the current slice if any
		*dest = nil
		// This should never happen if we validate the types
		srcType := reflect.TypeOf(src)
		if srcType.Kind() != reflect.Slice {
			rm.parser.log.Printf("Attempted to store '%s' which is not a slice", srcType.Kind())
		}
		for _, value := range src.([]string) {
			*dest = append(*dest, value)
		}
	}
	return rm
}

func (rm *RuleModifier) StoreStringMap(dest *map[string]string) *RuleModifier {
	rm.rule.Cast = castStringMap
	rm.rule.StoreValue = func(src interface{}) {
		// clear the current before assignment
		*dest = nil
		*dest = src.(map[string]string)
	}
	return rm
}

// Indicates this option has an alias it can go by
func (rm *RuleModifier) Alias(name string) *RuleModifier {
	rm.rule.AddAlias(name, rm.parser.prefixChars)
	return rm
}

// Add the abbreviated version of the option (-a, -b, -c, etc...)
func (rm *RuleModifier) Short(name string) *RuleModifier {
	rm.rule.AddAlias(name, []string{"-"})
	return rm
}

// Makes this option or positional argument required
func (rm *RuleModifier) Required() *RuleModifier {
	rm.rule.SetFlag(IsRequired)
	return rm
}

// Value of this option can only be one of the provided choices; Required() is implied
func (rm *RuleModifier) Choices(choices []string) *RuleModifier {
	rm.rule.SetFlag(IsRequired)
	rm.rule.Choices = choices
	return rm
}

func (rm *RuleModifier) StoreStr(dest *string) *RuleModifier {
	return rm.StoreString(dest)
}

func (rm *RuleModifier) StoreString(dest *string) *RuleModifier {
	// Implies IsString()
	rm.rule.Cast = castString
	rm.rule.StoreValue = func(value interface{}) {
		*dest = value.(string)
	}
	return rm
}

func (rm *RuleModifier) Count() *RuleModifier {
	rm.rule.Action = func(rule *Rule, alias string, args []string, idx *int) error {
		// If user asked us to count the instances of this argument
		rule.Count = rule.Count + 1
		return nil
	}
	rm.rule.Cast = castInt
	return rm
}

func (rm *RuleModifier) Env(varName string) *RuleModifier {
	rm.rule.EnvVars = append(rm.rule.EnvVars, rm.parser.envPrefix+varName)
	return rm
}

func (rm *RuleModifier) Help(message string) *RuleModifier {
	rm.rule.RuleDesc = message
	return rm
}

func (rm *RuleModifier) InGroup(group string) *RuleModifier {
	rm.rule.Group = group
	return rm
}

func (rm *RuleModifier) AddConfigGroup(group string) *RuleModifier {
	var newRule Rule
	newRule = *rm.rule
	newRule.SetFlag(IsConfigGroup)
	newRule.Group = group
	// Make a new RuleModifier using rm as the template
	return rm.parser.addRule(group, newRuleModifier(&newRule, rm.parser))
}

func (rm *RuleModifier) AddFlag(name string) *RuleModifier {
	var newRule Rule
	newRule = *rm.rule
	newRule.SetFlag(IsFlag)
	// Make a new RuleModifier using rm as the template
	return rm.parser.addRule(name, newRuleModifier(&newRule, rm.parser))
}

func (rm *RuleModifier) AddConfig(name string) *RuleModifier {
	var newRule Rule
	newRule = *rm.rule
	// Make a new Rule using rm.rule as the template
	newRule.SetFlag(IsConfig)
	return rm.parser.addRule(name, newRuleModifier(&newRule, rm.parser))
}
