package args_test

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/thrawn01/args"
)

func TestArgs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Args Parser")
}

type TestLogger struct {
	result string
}

func NewTestLogger() *TestLogger {
	return &TestLogger{}
}

func (self *TestLogger) Print(stuff ...interface{}) {
	self.result = fmt.Sprint(stuff...)
}

func (self *TestLogger) Printf(format string, stuff ...interface{}) {
	self.result = fmt.Sprintf(format, stuff...)
}

func (self *TestLogger) Println(stuff ...interface{}) {
	self.result = fmt.Sprintln(stuff...)
}

func (self *TestLogger) GetEntry() string {
	return self.result
}

var _ = Describe("ArgParser", func() {
	Describe("ValuesFromIni()", func() {
		It("Should provide arg values from INI file", func() {
			parser := args.NewParser()
			parser.AddOption("--one").IsString()
			input := []byte("one=this is one value\ntwo=this is two value\n")
			opt, err := parser.FromIni(input)
			Expect(err).To(BeNil())
			Expect(opt.String("one")).To(Equal("this is one value"))
		})

		It("Should provide arg values from INI file after parsing the command line", func() {
			parser := args.NewParser()
			parser.AddOption("--one").IsString()
			parser.AddOption("--two").IsString()
			parser.AddOption("--three").IsString()
			cmdLine := []string{"--three", "this is three value"}
			opt, err := parser.ParseArgs(&cmdLine)
			input := []byte("one=this is one value\ntwo=this is two value\n")
			opt, err = parser.FromIni(input)
			Expect(err).To(BeNil())
			Expect(opt.String("one")).To(Equal("this is one value"))
			Expect(opt.String("three")).To(Equal("this is three value"))
		})

		It("Should not overide options supplied via the command line", func() {
			parser := args.NewParser()
			parser.AddOption("--one").IsString()
			parser.AddOption("--two").IsString()
			parser.AddOption("--three").IsString()
			cmdLine := []string{"--three", "this is three value", "--one", "this is from the cmd line"}
			opt, err := parser.ParseArgs(&cmdLine)
			input := []byte("one=this is one value\ntwo=this is two value\n")
			opt, err = parser.FromIni(input)
			Expect(err).To(BeNil())
			Expect(opt.String("one")).To(Equal("this is from the cmd line"))
			Expect(opt.String("three")).To(Equal("this is three value"))
		})

		It("Should clear any pre existing slices in the struct before assignment", func() {
			parser := args.NewParser()
			var list []string
			parser.AddOption("--list").StoreStringSlice(&list).Default("foo,bar,bit")

			opt, err := parser.ParseArgs(nil)
			Expect(err).To(BeNil())
			Expect(opt.StringSlice("list")).To(Equal([]string{"foo", "bar", "bit"}))
			Expect(list).To(Equal([]string{"foo", "bar", "bit"}))

			input := []byte("list=six,five,four\n")
			opt, err = parser.FromIni(input)
			Expect(err).To(BeNil())
			Expect(opt.StringSlice("list")).To(Equal([]string{"six", "five", "four"}))
			Expect(list).To(Equal([]string{"six", "five", "four"}))
		})
	})
	Describe("args.ParseArgs(nil)", func() {
		parser := args.NewParser()
		It("Should return error if AddOption() was never called", func() {
			_, err := parser.ParseArgs(nil)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Must create some options to match with args.AddOption() before calling arg.ParseArgs()"))
		})
	})
	Describe("args.AddOption()", func() {
		cmdLine := []string{"--one", "-two", "++three", "+four", "--power-level"}

		It("Should create optional rule --one", func() {
			parser := args.NewParser()
			parser.AddOption("--one").Count()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.IsPos).To(Equal(0))
		})
		It("Should create optional rule ++one", func() {
			parser := args.NewParser()
			parser.AddOption("++one").Count()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.IsPos).To(Equal(0))
		})

		It("Should create optional rule -one", func() {
			parser := args.NewParser()
			parser.AddOption("-one").Count()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.IsPos).To(Equal(0))
		})

		It("Should create optional rule +one", func() {
			parser := args.NewParser()
			parser.AddOption("+one").Count()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.IsPos).To(Equal(0))
		})

		It("Should match --one", func() {
			parser := args.NewParser()
			parser.AddOption("--one").Count()
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("one")).To(Equal(1))
		})
		It("Should match -two", func() {
			parser := args.NewParser()
			parser.AddOption("-two").Count()
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("two")).To(Equal(1))
		})
		It("Should match ++three", func() {
			parser := args.NewParser()
			parser.AddOption("++three").Count()
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("three")).To(Equal(1))
		})
		It("Should match +four", func() {
			parser := args.NewParser()
			parser.AddOption("+four").Count()
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("four")).To(Equal(1))
		})
		It("Should match --power-level", func() {
			parser := args.NewParser()
			parser.AddOption("--power-level").Count()
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(1))
		})
	})
	Describe("args.AddConfig()", func() {
		cmdLine := []string{"--power-level", "--power-level"}
		It("Should add new config only rule", func() {
			parser := args.NewParser()
			parser.AddConfig("power-level").Count().Help("My help message")

			// Should ignore command line options
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(0))

			// But Apply() a config file
			options := args.NewOptionsFromMap(args.DefaultOptionGroup, nil,
				map[string]map[string]*args.OptionValue{
					args.DefaultOptionGroup: {
						"power-level": &args.OptionValue{Value: 3, Seen: false},
					},
				})
			newOpt, _ := parser.Apply(options)
			// The old config still has the original non config applied version
			Expect(opt.Int("power-level")).To(Equal(0))
			// The new config has the value applied
			Expect(newOpt.Int("power-level")).To(Equal(3))
		})
	})
	Describe("args.AddRule()", func() {
		cmdLine := []string{"--power-level", "--power-level"}
		It("Should add new rules", func() {
			parser := args.NewParser()
			rule := args.NewRuleModifier(parser).Count().Help("My help message")
			parser.AddRule("--power-level", rule)
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(2))
		})
	})
	Describe("args.InGroup()", func() {
		cmdLine := []string{"--power-level", "--hostname", "mysql.com"}
		It("Should add a new group", func() {
			parser := args.NewParser()
			parser.AddOption("--power-level").Count()
			parser.InGroup("database").AddOption("--hostname")
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(1))
			Expect(opt.Group("database").String("hostname")).To(Equal("mysql.com"))
		})
	})
	Describe("args.AddRule()", func() {
		cmdLine := []string{"--power-level", "--power-level"}
		It("Should add new rules", func() {
			parser := args.NewParser()
			rule := args.NewRuleModifier(parser).Count().Help("My help message")
			parser.AddRule("--power-level", rule)
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(2))
		})
	})
	Describe("RuleModifier.AddConfig()", func() {
		cmdLine := []string{"--power-level", "--power-level", "--user"}
		It("Should add new config only rule", func() {
			parser := args.NewParser()
			parser.AddConfig("power-level").Count().Help("My help message")

			db := parser.InGroup("database")
			db.AddConfig("user").Help("database user")
			db.AddConfig("pass").Help("database password")

			// Should ignore command line options
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(0))

			// But Apply() a config file
			options := args.NewOptionsFromMap(args.DefaultOptionGroup, nil,
				map[string]map[string]*args.OptionValue{
					args.DefaultOptionGroup: {
						"power-level": &args.OptionValue{Value: 3, Seen: false},
					},
					"database": {
						"user": &args.OptionValue{Value: "my-user", Seen: false},
						"pass": &args.OptionValue{Value: "my-pass", Seen: false},
					},
				})
			newOpt, _ := parser.Apply(options)
			// The new config has the value applied
			Expect(newOpt.Int("power-level")).To(Equal(3))
			Expect(newOpt.Group("database").String("user")).To(Equal("my-user"))
			Expect(newOpt.Group("database").String("pass")).To(Equal("my-pass"))
		})
	})
	Describe("RuleModifier.InGroup()", func() {
		cmdLine := []string{"--power-level", "--hostname", "mysql.com"}
		It("Should add a new group", func() {
			parser := args.NewParser()
			parser.AddOption("--power-level").Count()
			parser.AddOption("--hostname").InGroup("database")
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(1))
			Expect(opt.Group("database").String("hostname")).To(Equal("mysql.com"))
		})
	})
	Describe("RuleModifier.Count()", func() {
		It("Should count one", func() {
			parser := args.NewParser()
			cmdLine := []string{"--verbose"}
			parser.AddOption("--verbose").Count()
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("verbose")).To(Equal(1))
		})
		It("Should count three times", func() {
			parser := args.NewParser()
			cmdLine := []string{"--verbose", "--verbose", "--verbose"}
			parser.AddOption("--verbose").Count()
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("verbose")).To(Equal(3))
		})
	})
	Describe("RuleModifier.IsInt()", func() {
		It("Should ensure value supplied is an integer", func() {
			parser := args.NewParser()
			parser.AddOption("--power-level").IsInt()

			cmdLine := []string{"--power-level", "10000"}
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(10000))
		})

		It("Should set err if the option value is not parsable as an integer", func() {
			parser := args.NewParser()
			cmdLine := []string{"--power-level", "over-ten-thousand"}
			parser.AddOption("--power-level").IsInt()
			_, err := parser.ParseArgs(&cmdLine)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Invalid value for '--power-level' - 'over-ten-thousand' is not an Integer"))
			//Expect(opt.Int("power-level")).To(Equal(0))
		})

		It("Should set err if no option value is provided", func() {
			parser := args.NewParser()
			cmdLine := []string{"--power-level"}
			parser.AddOption("--power-level").IsInt()
			_, err := parser.ParseArgs(&cmdLine)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Expected '--power-level' to have an argument"))
			//Expect(opt.Int("power-level")).To(Equal(0))
		})
	})
	Describe("RuleModifier.StoreInt()", func() {
		It("Should ensure value supplied is assigned to passed value", func() {
			parser := args.NewParser()
			var value int
			parser.AddOption("--power-level").StoreInt(&value)

			cmdLine := []string{"--power-level", "10000"}
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(10000))
			Expect(value).To(Equal(10000))
		})
	})
	Describe("RuleModifier.IsString()", func() {
		It("Should provide string value", func() {
			parser := args.NewParser()
			parser.AddOption("--power-level").IsString()

			cmdLine := []string{"--power-level", "over-ten-thousand"}
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.String("power-level")).To(Equal("over-ten-thousand"))
		})

		It("Should set err if no option value is provided", func() {
			parser := args.NewParser()
			cmdLine := []string{"--power-level"}
			parser.AddOption("--power-level").IsString()
			_, err := parser.ParseArgs(&cmdLine)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Expected '--power-level' to have an argument"))
		})
	})
	Describe("RuleModifier.StoreString()", func() {
		It("Should ensure value supplied is assigned to passed value", func() {
			parser := args.NewParser()
			var value string
			parser.AddOption("--power-level").StoreString(&value)

			cmdLine := []string{"--power-level", "over-ten-thousand"}
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.String("power-level")).To(Equal("over-ten-thousand"))
			Expect(value).To(Equal("over-ten-thousand"))
		})
	})
	Describe("RuleModifier.StoreStr()", func() {
		It("Should ensure value supplied is assigned to passed value", func() {
			parser := args.NewParser()
			var value string
			parser.AddOption("--power-level").StoreStr(&value)

			cmdLine := []string{"--power-level", "over-ten-thousand"}
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.String("power-level")).To(Equal("over-ten-thousand"))
			Expect(value).To(Equal("over-ten-thousand"))
		})
	})
	Describe("RuleModifier.StoreTrue()", func() {
		It("Should ensure value supplied is true when argument is seen", func() {
			parser := args.NewParser()
			var debug bool
			parser.AddOption("--debug").StoreTrue(&debug)

			cmdLine := []string{"--debug"}
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Bool("debug")).To(Equal(true))
			Expect(debug).To(Equal(true))
		})

		It("Should ensure value supplied is false when argument is NOT seen", func() {
			parser := args.NewParser()
			var debug bool
			parser.AddOption("--debug").StoreTrue(&debug)

			cmdLine := []string{"--something-else"}
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Bool("debug")).To(Equal(false))
			Expect(debug).To(Equal(false))
		})
	})
	Describe("RuleModifier.IsTrue()", func() {
		It("Should set true value when seen", func() {
			parser := args.NewParser()
			parser.AddOption("--help").IsTrue()

			cmdLine := []string{"--help"}
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Bool("help")).To(Equal(true))
		})

		It("Should set false when NOT seen", func() {
			parser := args.NewParser()
			cmdLine := []string{"--something-else"}
			parser.AddOption("--help").IsTrue()
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Bool("help")).To(Equal(false))
		})
	})
	Describe("RuleModifier.IsStringSlice()", func() {
		It("Should ensure []string provided is set when a comma separated list is provided", func() {
			parser := args.NewParser()
			parser.AddOption("--list").IsStringSlice()

			cmdLine := []string{"--list", "one,two,three"}
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.StringSlice("list")).To(Equal([]string{"one", "two", "three"}))
		})
	})
	Describe("RuleModifier.StoreStringSlice()", func() {
		It("Should ensure []string provided is set when a comma separated list is provided", func() {
			parser := args.NewParser()
			var list []string
			parser.AddOption("--list").StoreStringSlice(&list)

			cmdLine := []string{"--list", "one,two,three"}
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.StringSlice("list")).To(Equal([]string{"one", "two", "three"}))
			Expect(list).To(Equal([]string{"one", "two", "three"}))
		})

		It("Should ensure []string provided is set when a comma separated list is provided", func() {
			parser := args.NewParser()
			var list []string
			parser.AddOption("--list").StoreStringSlice(&list)

			cmdLine := []string{"--list", "one,two,three"}
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.StringSlice("list")).To(Equal([]string{"one", "two", "three"}))
			Expect(list).To(Equal([]string{"one", "two", "three"}))
		})

		It("Should ensure []interface{} provided is set if a single value is provided", func() {
			parser := args.NewParser()
			var list []string
			parser.AddOption("--list").StoreStringSlice(&list)

			cmdLine := []string{"--list", "one"}
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.StringSlice("list")).To(Equal([]string{"one"}))
			Expect(list).To(Equal([]string{"one"}))
		})

		It("Should set err if no list value is provided", func() {
			parser := args.NewParser()
			var list []string
			parser.AddOption("--list").StoreStringSlice(&list)

			cmdLine := []string{"--list"}
			_, err := parser.ParseArgs(&cmdLine)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Expected '--list' to have an argument"))
		})

	})
	Describe("RuleModifier.Default()", func() {
		It("Should ensure default values is supplied if no matching argument is found", func() {
			parser := args.NewParser()
			var value int
			parser.AddOption("--power-level").StoreInt(&value).Default("10")

			opt, err := parser.ParseArgs(nil)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(10))
			Expect(value).To(Equal(10))
		})

		It("Should panic if default value does not match AddOption() type", func() {
			parser := args.NewParser()
			panicCaught := false

			defer func() {
				msg := recover()
				Expect(msg).ToNot(BeNil())
				Expect(msg).To(ContainSubstring("args.Default"))
				panicCaught = true
			}()

			parser.AddOption("--power-level").IsInt().Default("over-ten-thousand")
			parser.ParseArgs(nil)
			Expect(panicCaught).To(Equal(true))
		})
	})
	Describe("RuleModifier.Env()", func() {
		AfterEach(func() {
			os.Unsetenv("POWER_LEVEL")
		})

		It("Should supply the environ value if argument was not passed", func() {
			parser := args.NewParser()
			var value int
			parser.AddOption("--power-level").StoreInt(&value).Env("POWER_LEVEL")

			os.Setenv("POWER_LEVEL", "10")

			opt, err := parser.ParseArgs(nil)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(10))
			Expect(value).To(Equal(10))
		})

		It("Should return an error if the environ value does not match the AddOption() type", func() {
			parser := args.NewParser()
			var value int
			parser.AddOption("--power-level").StoreInt(&value).Env("POWER_LEVEL")

			os.Setenv("POWER_LEVEL", "over-ten-thousand")

			_, err := parser.ParseArgs(nil)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Invalid value for 'POWER_LEVEL' - 'over-ten-thousand' is not an Integer"))
		})

		It("Should use the default value if argument was not passed and environment var was not set", func() {
			parser := args.NewParser()
			var value int
			parser.AddOption("--power-level").StoreInt(&value).Env("POWER_LEVEL").Default("1")

			opt, err := parser.ParseArgs(nil)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(1))
			Expect(value).To(Equal(1))
		})
	})
	Describe("parser.GenerateOptHelp()", func() {
		It("Should generate help messages given a set of rules", func() {
			parser := args.NewParser(args.WrapLen(80))
			parser.AddOption("--power-level").Alias("-p").Help("Specify our power level")
			parser.AddOption("--cat-level").
				Alias("-c").
				Help(`Lorem ipsum dolor sit amet, consectetur
			mollit anim id est laborum.`)
			msg := parser.GenerateOptHelp()
			Expect(msg).To(Equal("  -p, --power-level   Specify our power level " +
				"\n  -c, --cat-level     Lorem ipsum dolor sit amet, consecteturmollit anim id est" +
				"\n                      laborum. \n"))
		})
	})
	Describe("parser.GenerateHelp()", func() {
		It("Should generate help messages given a set of rules", func() {
			parser := args.NewParser(args.Name("dragon-ball"), args.WrapLen(80))
			parser.AddOption("--power-level").Alias("-p").Help("Specify our power level")
			parser.AddOption("--cat-level").Alias("-c").Help(`Lorem ipsum dolor sit amet, consectetur
				adipiscing elit, sed do eiusmod tempor incididunt ut labore et
				mollit anim id est laborum.`)
			//msg := parser.GenerateHelp()
			/*Expect(msg).To(Equal("Usage:\n" +
			" dragon-ball [OPTIONS]\n" +
			"\n" +
			"Options:\n" +
			"  -p, --power-level   Specify our power level\n" +
			"  -c, --cat-level     Lorem ipsum dolor sit amet, consecteturadipiscing elit, se\n" +
			"                     d do eiusmod tempor incididunt ut labore etmollit anim id\n" +
			"                      est laborum."))*/
		})
	})
	Describe("Helper.WordWrap()", func() {
		It("Should wrap the line including the indent length", func() {
			// Should show the message with the indentation of 10 characters on the next line
			msg := args.WordWrap(`Lorem ipsum dolor sit amet, consectetur
			adipiscing elit, sed do eiusmod tempor incididunt ut labore et
			mollit anim id est laborum.`, 10, 80)
			Expect(msg).To(Equal("Lorem ipsum dolor sit amet, consecteturadipiscing elit, sed do\n          eiusmod tempor incididunt ut labore etmollit anim id est laborum."))
		})
		It("Should wrap the line without the indent length", func() {
			// Should show the message with no indentation
			msg := args.WordWrap(`Lorem ipsum dolor sit amet, consectetur
			adipiscing elit, sed do eiusmod tempor incididunt ut labore et
			mollit anim id est laborum.`, 0, 80)
			Expect(msg).To(Equal("Lorem ipsum dolor sit amet, consecteturadipiscing elit, sed do eiusmod tempor\n incididunt ut labore etmollit anim id est laborum."))
		})
	})
	Describe("args.Dedent()", func() {
		It("Should un-indent a simple string", func() {
			text := args.Dedent(`Lorem ipsum dolor sit amet, consecteturadipiscing elit, sed
		 do eiusmod tempor incididunt ut labore etmollit anim id
		 est laborum.`)
			Expect(text).To(Equal("Lorem ipsum dolor sit amet, consecteturadipiscing elit, sed\ndo eiusmod tempor incididunt ut labore etmollit anim id\nest laborum."))
		})
		It("Should un-indent a string starting with a new line", func() {
			text := args.Dedent(`
		 Lorem ipsum dolor sit amet, consecteturadipiscing elit, sed
		 do eiusmod tempor incididunt ut labore etmollit anim id
		 est laborum.`)
			Expect(text).To(Equal("\nLorem ipsum dolor sit amet, consecteturadipiscing elit, sed\ndo eiusmod tempor incididunt ut labore etmollit anim id\nest laborum."))
		})
	})
	Describe("args.DedentTrim()", func() {
		It("Should un-indent a simple string and trim the result", func() {
			text := args.DedentTrim(`
	    Lorem ipsum dolor sit amet, consecteturadipiscing elit, sed
	    do eiusmod tempor incididunt ut labore etmollit anim id
	    est laborum.
	    `, "\n")
			Expect(text).To(Equal("Lorem ipsum dolor sit amet, consecteturadipiscing elit, sed\ndo eiusmod tempor incididunt ut labore etmollit anim id\nest laborum."))
		})
	})
})

var _ = Describe("Options", func() {
	var opts *args.Options
	var log *TestLogger
	BeforeEach(func() {
		log = NewTestLogger()
		opts = args.NewOptionsFromMap(args.DefaultOptionGroup, log,
			map[string]map[string]*args.OptionValue{
				args.DefaultOptionGroup: {
					"int":    &args.OptionValue{Value: 1, Seen: false},
					"bool":   &args.OptionValue{Value: true, Seen: false},
					"string": &args.OptionValue{Value: "one", Seen: false},
				},
			})

	})
	Describe("log", func() {
		It("Should log to StdLogger when cast fails", func() {
			result := opts.Int("string")
			Expect(log.GetEntry()).To(Equal("Unable to Cast \"one\" to int for key 'string'"))
			Expect(result).To(Equal(0))
		})
	})
	Describe("Int()", func() {
		It("Should convert values to integers", func() {
			result := opts.Int("int")
			Expect(log.GetEntry()).To(Equal(""))
			Expect(result).To(Equal(1))
		})
		It("Should return default value if key doesn't exist", func() {
			result := opts.Int("none")
			Expect(log.GetEntry()).To(Equal(""))
			Expect(result).To(Equal(0))
		})

	})
	Describe("Bool()", func() {
		It("Should convert values to boolean", func() {
			result := opts.Bool("bool")
			Expect(log.GetEntry()).To(Equal(""))
			Expect(result).To(Equal(true))
		})
		It("Should return default value if key doesn't exist", func() {
			result := opts.Bool("none")
			Expect(result).To(Equal(false))
		})

	})
	Describe("NoArgs()", func() {
		It("Should return true if no arguments on the command line", func() {
			parser := args.NewParser()
			parser.AddOption("--power-level").IsInt().Default("1")

			opt, err := parser.ParseArgs(nil)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(1))
			Expect(opt.NoArgs()).To(Equal(true))
		})
		It("Should return false if arguments on the command line", func() {
			parser := args.NewParser()
			parser.AddOption("--power-level").IsInt().Default("1")

			opt, err := parser.ParseArgs(&[]string{"--power-level", "2"})
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(2))
			Expect(opt.NoArgs()).To(Equal(false))
		})
	})
})
