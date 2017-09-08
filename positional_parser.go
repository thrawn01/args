package args

import (
	"context"
	"os"
	"sort"

	"strconv"

	"github.com/pkg/errors"
)

type PosParser struct {
	// Sorted list of parsing rules
	rules Rules
	// Stored parsing results
	store Store
	// Flags that modify parser behavior
	flags ParseFlag
	// Prefix applied to all to rules that match environment variables
	envPrefix string
	// Prefix characters applied to all AddFlag() or AddArgument() names that do not specify a
	// prefix in their name. IE: AddFlag("help") will match, "--help" and "-help" if prefixChars
	// is set too []string{"--", "-"}
	prefixChars []string
	// If defined will log parse and type errors to  this logger
	log StdLogger
	// Our parent parser if this instance is a sub-parser
	parent *PosParser
}

// Creates a new instance of the argument parser
func NewPosParser() *PosParser {
	return &PosParser{}
}

func (s *PosParser) validateRules(rules Rules) (Rules, error) {
	var validate Rules

	// If were passed some rules, append them
	if rules != nil {
		validate = append(s.rules, rules...)
	}

	// Validate with our parents rules if exist
	if s.parent != nil {
		return s.parent.validateRules(validate)
	}

	return nil, validate.ValidateRules()
}

func (s *PosParser) parseARGV(argv []string) (Store, error) {
	store := NewStringStore()

	var tokens Tokens

	// Parse the CLI into a list of tokens
	err := s.parse(&tokens, argv)
	if err != nil {
		return store, err
	}

	// Create a store from the collected argv tokens
	ctx := context.Background()
	for _, rule := range s.rules {
		// Count the number of times this flag occurred
		if rule.HasFlag(IsCountFlag) {
			var count int
			for range tokens.Matched(rule) {
				count++
			}
			store.Set(ctx,
				Key{
					Name:  rule.Name,
					Group: rule.Group,
				},
				StringValue{
					Value: strconv.Itoa(count),
					Src:   FromArgv,
				})
		}
	}
	return store, err
}

type Token struct {
	RawFlag string
	Value   *string
	Rule    *Rule
}

type Tokens []*Token

// Returns the first token it find that has the specified rule
func (s Tokens) HasRule(rule *Rule) *Token {
	for _, token := range s {
		if token.Rule == rule {
			return token
		}
	}
	return nil
}

// Returns all tokens that match the specified rule
func (s Tokens) Matched(rule *Rule) Tokens {
	var tokens Tokens
	for _, token := range s {
		if token.Rule == rule {
			tokens = append(tokens, token)
		}
	}
	return tokens
}

func (s *PosParser) parse(tokens *Tokens, argv []string) error {
	var token Token

	if len(argv) == 0 {
		// No more args to parse
		return nil
	}

	for _, rule := range s.rules {
		// Ignore config rules
		if rule.HasFlag(IsConfig) {
			continue
		}

		// If this is an flag rule
		if rule.HasFlag(IsFlag) {
			// Match any aliases for this rule
			for _, alias := range rule.Aliases {
				// TODO: flag could be '--foo=bar' or '-ffffff'

				// If the flag matches an alias
				if argv[0] == alias {
					token.RawFlag = argv[0]
					token.Rule = rule

					// consume the next arg as the value for this flag
					if rule.HasFlag(IsExpectingValue) && len(argv) > 1 {
						argv = argv[1:]
						token.Value = &argv[0]
					}
					goto NEXT

				}
			}
		}

		// If this is an argument
		if rule.HasFlag(IsArgument) {
			// If it's greedy
			if rule.HasFlag(IsGreedy) {
				token.Value = &argv[0]
				goto NEXT
			}
			// This argument has already been seen and has a value
			if tokens.HasRule(rule) == nil {
				continue
			}
			// Record the argument was seen
			token.Rule = rule
			token.Value = &argv[0]
			goto NEXT
		}
		continue
	NEXT:
		*tokens = append(*tokens, &token)
		return s.parse(tokens, argv[1:])
	}
	return s.parse(tokens, argv[1:])
}

func (s *PosParser) ParseENV() (Store, error) {
	return nil, nil
}

// Parses command line arguments using os.Args if 'args' is nil.
func (s *PosParser) Parse(argv []string) (Values, error) {
	var err error

	if argv == nil {
		argv = os.Args
	}

	// Sanity Check
	if len(s.rules) == 0 {
		return s.GetValues(), errors.New("No rules defined; call AddFlag() or AddArgument() before calling Parse()")
	}

	// If user requested we add a help flag, and if one is not already defined
	if hasFlags(s.flags, AddHelpFlag) && !s.HasHelpFlag() {
		s.AddFlag("--help").Alias("-h").IsTrue().Help("Display a help message and exit")
	}

	// Check for duplicate rules, and combine any rules from parent parsers
	if s.rules, err = s.validateRules(nil); err != nil {
		return s.GetValues(), err
	}

	// Sort the rules so positional rules are evaluated last
	sort.Sort(s.rules)

	// Returns a store with values parsed from argv
	store, err := s.parseARGV(argv)
	if err != nil {
		return s.GetValues(), err
	}

	// Apply the parsed store with our current store
	if err := s.Apply(store); err != nil {
		return s.GetValues(), err
	}

	/*

		// Returns a store with values from the environment
		envStore, err := s.ParseENV()
		if err != nil {
			return s.GetArgs(), err
		}
		// Create a new store to apply our parsed values to
		store := NewStringStore()

		// Apply environment values first
		err = store.Apply(envStore)

		// Apply argv values next
		err = store.Apply(argStore)

		// Apply the combined store with our current store
		if err := s.Apply(store); err != nil {
			return s.GetArgs(), err
		}*/

	// Return a pointer to the latest version of the values
	return s.GetValues(), nil
}

// Return the current version of the parsed arguments
func (s *PosParser) GetValues() Values {
	return nil
}

// Returns the current list of rules for this parser. If you want to modify a rule
// use GetRule() instead.
func (s *PosParser) GetRules() Rules {
	return s.rules
}

// Using the rules defined in the parser, apply values in the given `Store` to the parser.
// If a value already exists in the parser it is replaced with the values provided by the `Store`
func (s *PosParser) Apply(store Store) error {

	// Using the rules defined in the parser; fetch values from the store
	store, err := s.FromStore(store)
	if err != nil {
		return err
	}

	// Apply default values to the store
	for _, rule := range s.rules {
		// If this rule has a default value
		if rule.Default != nil {
			_, err := store.Get(context.Background(), rule.Key())
			// If no values defined for this rule
			if IsNotFoundErr(err) {
				store.Set(context.Background(), rule.Key(), TypedValue{
					Value: *rule.Default,
					Rule:  rule,
					Src:   FromDefault,
				})
			}
		}
	}

	// Ensure the store abides by the rules the parser has for values
	if err := s.ValidateStore(store); err != nil {
		return err
	}

	// Cast the values to their final type
	s.store, err = s.CastValues(store)
	return err
}

// Given a store, validate the store values conform to the rules defined by the parser.
func (s *PosParser) ValidateStore(store Store) error {

	// Ensure required values are supplied
	for _, rule := range s.rules {
		if rule.HasFlag(IsRequired) {
			_, err := store.Get(context.Background(), rule.Key())
			if err != nil {
				if IsNotFoundErr(err) {
					return errors.New(rule.RequiredMessage())
				}
				return err
			}
		}
	}
	return nil
}

// Given a store, return a new store with the values from the provided store cast to the types specified by their rules.
func (s *PosParser) CastValues(store Store) (Store, error) {
	result := s.NewTypedValues(nil)

	values, err := store.List(context.Background(), Key{})
	if err != nil {
		return nil, err
	}

	// Convert values to the type specified in the rules
	for _, value := range values {
		rule := value.GetRule()
		// Only validate values with rules attached
		if rule == nil {
			continue
		}

		if rule.HasFlag(IsConfigGroup) {
			// TODO: Handle ConfigGroup
			continue
		}

		// Convert values to specified types
		newValue, err := rule.Cast(rule.Name, nil, value.Interface())
		if err != nil {
			return nil, err
		}

		result.Set(context.Background(), value.GetRule().Key(),
			TypedValue{
				Value: newValue,
				Rule:  rule,
				Src:   value.Source(),
			})
	}

	return result, nil
}

// Using the rules defined in the parser; fetch values from the store
// and return a new `Store` of the fetched values with the rules attached.
func (s *PosParser) FromStore(store Store) (Store, error) {
	results := s.NewTypedValues(nil)

	// Fetch only values the parser has rules for
	ctx, cancel := context.WithTimeout(context.Background(), StoreTimeout)
	defer func() { cancel() }()

	for _, rule := range s.rules {
		key := rule.Key()
		if rule.HasFlag(IsConfigGroup) {
			values, err := store.List(ctx, key)
			if err != nil {
				//TODO: s.info("args.ParseBackend(): Failed to list '%s' - '%s'", key.Group, err.Error())
				return nil, err
			}

			// Create a new group
			results.Set(ctx, key, s.NewTypedValues(rule))

			// Iterate through all the key=values pairs for this group
			for _, value := range values {
				results.Group(key.Group).Set(ctx, key,
					TypedValue{
						Value: value.Interface(),
						Rule:  rule,
						Src:   value.Source(),
					})
			}
			continue
		}

		value, err := store.Get(ctx, key)
		if err != nil {
			// TODO: self.info("args.ParseBackend(): Failed to fetch key '%s' - %s", key.Name, err.Error())
			return nil, err
		}
		results.Group(key.Group).Set(ctx, key,
			TypedValue{
				Value: value.Interface(),
				Rule:  rule,
				Src:   value.Source(),
			})
	}

	return results, nil
}

// Return true if the parser has the 'help' flag defined
func (s *PosParser) HasHelpFlag() bool {
	return s.rules.GetRule("help") != nil
}

// Add a new flag to the parser rules and return a modifier for further configuration
func (s *PosParser) AddFlag(name string) *PosRuleModifier {
	rule := &Rule{Cast: castString, Group: DefaultOptionGroup, EnvPrefix: s.envPrefix}
	rule.SetFlag(IsFlag)
	// Add to the aliases we can match on the command line and return the un-prefixed name of the alias
	rule.Name = rule.AddAlias(name, s.prefixChars)
	s.rules = append(s.rules, rule)
	return &PosRuleModifier{rule, s}
}
