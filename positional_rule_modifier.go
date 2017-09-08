package args

import "reflect"

type PosRuleModifier struct {
	rule   *Rule
	parser *PosParser
}

// TODO: Delete
func NewPosRuleModifier(parser *PosParser) *PosRuleModifier {
	return &PosRuleModifier{newRule(), parser}
}

// TODO: Delete
func newPosRuleModifier(rule *Rule, parser *PosParser) *PosRuleModifier {
	return &PosRuleModifier{rule, parser}
}

func (self *PosRuleModifier) GetRule() *Rule {
	return self.rule
}

func (self *PosRuleModifier) IsString() *PosRuleModifier {
	self.rule.SetFlag(IsExpectingValue)
	self.rule.Cast = castString
	return self
}

// If the option is seen on the command line, the value is 'true'
func (self *PosRuleModifier) IsTrue() *PosRuleModifier {
	self.rule.ClearFlag(IsExpectingValue)
	self.rule.Cast = castBool
	return self
}

func (self *PosRuleModifier) IsBool() *PosRuleModifier {
	self.rule.SetFlag(IsExpectingValue)
	self.rule.Cast = castBool
	self.rule.Value = false
	return self
}

func (self *PosRuleModifier) Default(value string) *PosRuleModifier {
	self.rule.Default = &value
	return self
}

func (self *PosRuleModifier) StoreInt(dest *int) *PosRuleModifier {
	// Implies IsInt()
	self.rule.SetFlag(IsExpectingValue)
	self.rule.Cast = castInt
	self.rule.StoreValue = func(value interface{}) {
		*dest = value.(int)
	}
	return self
}

func (self *PosRuleModifier) IsInt() *PosRuleModifier {
	self.rule.SetFlag(IsExpectingValue)
	self.rule.Cast = castInt
	return self
}

func (self *PosRuleModifier) StoreTrue(dest *bool) *PosRuleModifier {
	self.rule.ClearFlag(IsExpectingValue)
	self.rule.Cast = castBool
	self.rule.StoreValue = func(value interface{}) {
		*dest = value.(bool)
	}
	return self
}

func (self *PosRuleModifier) IsStringSlice() *PosRuleModifier {
	self.rule.SetFlag(IsExpectingValue)
	self.rule.Cast = castStringSlice
	self.rule.SetFlag(IsGreedy)
	return self
}

func (self *PosRuleModifier) IsStringMap() *PosRuleModifier {
	self.rule.SetFlag(IsExpectingValue)
	self.rule.Cast = castStringMap
	self.rule.SetFlag(IsGreedy)
	return self
}

// TODO: Make this less horribad, and use more reflection to make the interface simpler
// It should also take more than just []string but also []int... etc...
func (self *PosRuleModifier) StoreStringSlice(dest *[]string) *PosRuleModifier {
	self.rule.SetFlag(IsExpectingValue)
	self.rule.Cast = castStringSlice
	self.rule.StoreValue = func(src interface{}) {
		// First clear the current slice if any
		*dest = nil
		// This should never happen if we validate the types
		srcType := reflect.TypeOf(src)
		if srcType.Kind() != reflect.Slice {
			self.parser.log.Printf("Attempted to store '%s' which is not a slice", srcType.Kind())
		}
		for _, value := range src.([]string) {
			*dest = append(*dest, value)
		}
	}
	return self
}

func (self *PosRuleModifier) StoreStringMap(dest *map[string]string) *PosRuleModifier {
	self.rule.SetFlag(IsExpectingValue)
	self.rule.Cast = castStringMap
	self.rule.StoreValue = func(src interface{}) {
		// clear the current before assignment
		*dest = nil
		*dest = src.(map[string]string)
	}
	return self
}

// Indicates this option has an alias it can go by
func (self *PosRuleModifier) Alias(name string) *PosRuleModifier {
	self.rule.AddAlias(name, self.parser.prefixChars)
	return self
}

// Add the abbreviated version of the option (-a, -b, -c, etc...)
func (self *PosRuleModifier) Short(name string) *PosRuleModifier {
	self.rule.AddAlias(name, []string{"-"})
	return self
}

// Makes this option or positional argument required
func (self *PosRuleModifier) Required() *PosRuleModifier {
	self.rule.SetFlag(IsRequired)
	return self
}

// Value of this option can only be one of the provided choices; Required() is implied
func (self *PosRuleModifier) Choices(choices []string) *PosRuleModifier {
	self.rule.SetFlag(IsRequired)
	self.rule.Choices = choices
	return self
}

func (self *PosRuleModifier) StoreStr(dest *string) *PosRuleModifier {
	return self.StoreString(dest)
}

func (self *PosRuleModifier) StoreString(dest *string) *PosRuleModifier {
	self.rule.SetFlag(IsExpectingValue)
	// Implies IsString()
	self.rule.Cast = castString
	self.rule.StoreValue = func(value interface{}) {
		*dest = value.(string)
	}
	return self
}

func (self *PosRuleModifier) Count() *PosRuleModifier {
	self.rule.ClearFlag(IsExpectingValue)
	self.rule.SetFlag(IsCountFlag)
	self.rule.Cast = castInt
	return self
}

func (self *PosRuleModifier) Env(varName string) *PosRuleModifier {
	//self.rule.EnvVars = append(self.rule.EnvVars, self.parser.envPrefix+varName)
	return self
}

func (self *PosRuleModifier) Help(message string) *PosRuleModifier {
	self.rule.RuleDesc = message
	return self
}

func (self *PosRuleModifier) InGroup(group string) *PosRuleModifier {
	self.rule.Group = group
	return self
}

// TODO: Add support for groups
/*func (self *PosRuleModifier) AddConfigGroup(group string) *PosRuleModifier {
	var newRule Rule
	newRule = *self.rule
	newRule.SetFlag(IsConfigGroup)
	newRule.Group = group
	// Make a new PosRuleModifier using self as the template
	return self.parser.addRule(group, newPosRuleModifier(&newRule, self.parser))
}

func (self *PosRuleModifier) AddFlag(name string) *PosRuleModifier {
	var newRule Rule
	newRule = *self.rule
	newRule.SetFlag(IsFlag)
	// Make a new PosRuleModifier using self as the template
	return self.parser.addRule(name, newPosRuleModifier(&newRule, self.parser))
}

func (self *PosRuleModifier) AddConfig(name string) *PosRuleModifier {
	var newRule Rule
	newRule = *self.rule
	// Make a new Rule using self.rule as the template
	newRule.SetFlag(IsConfig)
	return self.parser.addRule(name, newPosRuleModifier(&newRule, self.parser))
}*/
