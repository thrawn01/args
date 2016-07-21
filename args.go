package args

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

const (
	DefaultTerminator  string = "--"
	DefaultOptionGroup string = ""
)

var DefaultLogger *NullLogger = &NullLogger{}

// We only need part of the standard logging functions
type StdLogger interface {
	Print(...interface{})
	Printf(string, ...interface{})
	Println(...interface{})
}

type NullLogger struct{}

func (self *NullLogger) Print(...interface{})          {}
func (self *NullLogger) Printf(string, ...interface{}) {}
func (self *NullLogger) Println(...interface{})        {}

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

func EnvPrefix(prefix string) ParseModifier {
	return func(parser *ArgParser) {
		parser.EnvPrefix = prefix
	}
}

func EtcdPath(path string) ParseModifier {
	return func(parser *ArgParser) {
		parser.EtcdRoot = path
	}
}

func NoHelp() ParseModifier {
	return func(parser *ArgParser) {
		parser.addHelp = false
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

func castString(name string, value interface{}) (interface{}, error) {
	// If value is nil, return the type default
	if value == nil {
		return "", nil
	}

	// If value is not an string
	if reflect.TypeOf(value).Kind() != reflect.String {
		return 0, errors.New(fmt.Sprintf("Invalid value for '%s' - '%s' is not a String", name, value))
	}
	return value, nil
}

func castInt(name string, value interface{}) (interface{}, error) {
	// If value is nil, return the type default
	if value == nil {
		return 0, nil
	}

	// If it's already an integer of some sort
	kind := reflect.TypeOf(value).Kind()
	switch kind {
	case reflect.Int:
		return value, nil
	case reflect.Int8:
		return int(value.(int8)), nil
	case reflect.Int16:
		return int(value.(int16)), nil
	case reflect.Int32:
		return int(value.(int32)), nil
	case reflect.Int64:
		return int(value.(int64)), nil
	case reflect.Uint8:
		return int(value.(uint8)), nil
	case reflect.Uint16:
		return int(value.(uint16)), nil
	case reflect.Uint32:
		return int(value.(uint32)), nil
	case reflect.Uint64:
		return int(value.(uint64)), nil
	}
	// If it's not an integer, it better be a string that we can cast
	if kind != reflect.String {
		return 0, errors.New(fmt.Sprintf("Invalid value for '%s' - '%s' is not a Integer or Castable string", name, value))
	}
	strValue := value.(string)

	intValue, err := strconv.ParseInt(strValue, 10, 64)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("Invalid value for '%s' - '%s' is not an Integer", name, strValue))
	}
	return int(intValue), nil
}

func castBool(name string, value interface{}) (interface{}, error) {
	// If value is nil, return the type default
	if value == nil {
		return false, nil
	}

	kind := reflect.TypeOf(value).Kind()
	if kind == reflect.Bool {
		return value, nil
	}
	// If it's not a boolean, it better be a string that we can cast
	if kind != reflect.String {
		return 0, errors.New(fmt.Sprintf("Invalid value for '%s' - '%s' is not a Boolean or Castable string", name, value))
	}
	strValue := value.(string)

	boolValue, err := strconv.ParseBool(strValue)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Invalid value for '%s' - '%s' is not a Boolean", name, strValue))
	}
	return bool(boolValue), nil
}

func castStringSlice(name string, value interface{}) (interface{}, error) {
	// If value is nil, return the type default
	if value == nil {
		return []string{}, nil
	}

	kind := reflect.TypeOf(value).Kind()
	if kind == reflect.Slice {
		sliceKind := reflect.TypeOf(value).Elem().Kind()
		// Is already a []string
		if sliceKind == reflect.String {
			return value, nil
		}
		return []string{}, errors.New(fmt.Sprintf("Invalid slice type for '%s' - '%s'  not a String", name, value))
	}

	if kind != reflect.String {
		return []string{}, errors.New(fmt.Sprintf("Invalid slice type for '%s' - '%s' is not a []string or parsable comma delimited string", name, value))
	}
	strValue := value.(string)

	// If no comma is found, then assume this is a single value
	if strings.Index(strValue, ",") == -1 {
		return []string{strValue}, nil
	}

	// Split the values separated by comma's
	return strings.Split(strValue, ","), nil
}

// Returns true if the error was because help message was printed
func AskedForHelp(err error) bool {
	obj, ok := err.(HelpErrorInterface)
	return ok && obj.IsHelpError()
}

type HelpErrorInterface interface {
	IsHelpError() bool
}

type HelpError struct{}

func (e *HelpError) Error() string {
	return ""
}

func (e *HelpError) IsHelpError() bool {
	return true
}

// A ChangeEvent is a representation of an etcd key=value update, delete or expire. Args attempts to match
// a rule to the etcd change and includes the matched rule in the ChangeEvent. If args is unable to match
// a with this change, then ChangeEvent.Rule will be nil
type ChangeEvent struct {
	Rule    *Rule
	Group   string
	Key     string
	Value   string
	Deleted bool
}
