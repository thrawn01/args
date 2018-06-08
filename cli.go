package args

import (
	"io/ioutil"

	"os"

	"fmt"
	"net/http"
	"strings"

	"encoding/json"
	"regexp"

	"github.com/pkg/errors"
)

// A collection of CLI build helpers

// Load contents from the specified file
func LoadFile(fileName string) ([]byte, error) {
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read '%s'", fileName)
	}
	return content, nil
}

// Returns true if the file has ModeCharDevice set. This is useful when determining if
// a CLI is receiving piped data.
//
//   var contents string
//   var err error
//
//   // If stdin is getting piped data, read from stdin
//   if args.IsCharDevice(os.Stdin) {
//       contents, err = ioutil.ReadAll(os.Stdin)
//   } else {
//       // load from file given instead
//       contents, err = args.LoadFile(opts.String("input-file"))
//   }
func IsCharDevice(file *os.File) bool {
	stat, err := file.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

// Returns a curl command representation of the passed http.Request
func CurlString(req *http.Request, payload *[]byte) string {
	parts := []string{"curl", "-i", "-X", req.Method, req.URL.String()}
	for key, value := range req.Header {
		parts = append(parts, fmt.Sprintf("-H \"%s: %s\"", key, value[0]))
	}

	if payload != nil {
		parts = append(parts, fmt.Sprintf(" -d '%s'", string(*payload)))
	}
	return strings.Join(parts, " ")
}

// Given an indented string remove any common leading whitespace from every
// line in text. Works much like python's `textwrap.dedent()` function.
// (Mixing Spaces and Tabs will have undesired effects)
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

// Exactly like `Dedent()` but trims trailing `cutset` characters
func DedentTrim(input string, cutset string) string {
	return strings.Trim(Dedent(input), cutset)
}

// Formats the text `msg` into the form of a paragraph, wrapping the text to the
// character `length` specified. Indenting the second line `indent` number of spaces
func WordWrap(msg string, indent int, length int) string {
	// Remove any previous formatting
	regex, _ := regexp.Compile(" {2,}|\n|\t")
	msg = regex.ReplaceAllString(msg, "")

	wordWrapLen := length - indent
	if wordWrapLen <= 0 {
		panic(fmt.Sprintf("Flag indent spacing '%d' exceeds wordwrap length '%d'\n", indent, length))
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

// Given a comma separated string, return a slice of string items.
// Return the entire string as the first item if no comma is found.
func StringToSlice(value string, modifiers ...func(s string) string) []string {
	result := strings.Split(value, ",")
	// Apply the modifiers
	for _, modifier := range modifiers {
		for idx, item := range result {
			result[idx] = modifier(item)
		}
	}
	return result
}

// Given a comma separated string of key values in the form `key=value`.
// Return a map of key values as strings, Also excepts JSON for more complex
// quoted or escaped data.
func StringToMap(value string) (map[string]string, error) {
	tokenizer := newKeyValueTokenizer(value)
	result := make(map[string]string)

	var lvalue, rvalue, expression string
	for {
		lvalue = tokenizer.Next()
		if lvalue == "" {
			return result, errors.New(fmt.Sprintf("Expected key at pos '%d' but found none; "+
				"map values should be 'key=value' separated by commas", tokenizer.Pos))
		}
		if strings.HasPrefix(lvalue, "{") {
			// Assume this is JSON format and attempt to un-marshal
			return jsonToMap(value)
		}

		expression = tokenizer.Next()
		if expression != "=" {
			return result, errors.New(fmt.Sprintf("Expected '=' after '%s' but found '%s'; "+
				"map values should be 'key=value' separated by commas", lvalue, expression))
		}
		rvalue = tokenizer.Next()
		if rvalue == "" {
			return result, errors.New(fmt.Sprintf("Expected value after '%s' but found none; "+
				"map values should be 'key=value' separated by commas", expression))
		}
		result[lvalue] = rvalue

		// Are there anymore tokens?
		delimiter := tokenizer.Next()
		if delimiter == "" {
			break
		}

		// Should be a comma next
		if delimiter != "," {
			return result, errors.New(fmt.Sprintf("Expected ',' after '%s' but found '%s'; "+
				"map values should be 'key=value' separated by commas", rvalue, delimiter))
		}
	}
	return result, nil
}

func jsonToMap(value string) (map[string]string, error) {
	result := make(map[string]string)
	err := json.Unmarshal([]byte(value), &result)
	if err != nil {
		return result, errors.New(fmt.Sprintf("JSON map decoding for '%s' failed with '%s'; "+
			`JSON map values should be in form '{"key":"value", "foo":"bar"}'`, value, err))
	}
	return result, nil
}

// TODO: Add SetDefault
