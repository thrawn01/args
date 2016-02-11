package args_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/thrawn01/args"
	"os"
	"testing"
)

func TestArgs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Args Parser")
}

var _ = Describe("ArgParser", func() {

	Describe("Options.Convert()", func() {
		It("Should convert options to integers", func() {
			opts := args.Options{"one": 1}
			var result int
			opts.Convert("one", "int", func(value interface{}) {
				result = value.(int)
			})
			Expect(result).To(Equal(1))
		})

		It("Should raise panic if unable to cast an option", func() {
			opts := args.Options{"one": ""}
			panicCaught := false
			result := 0

			defer func() {
				msg := recover()
				Expect(msg).ToNot(BeNil())
				Expect(msg).To(ContainSubstring("Refusing"))
				panicCaught = true
			}()

			opts.Convert("one", "int", func(value interface{}) {
				result = value.(int)
			})
			Expect(panicCaught).To(Equal(true))
		})

	})
	Describe("ValuesFromIni()", func() {
		It("Should provide arg values from INI file", func() {
			parser := args.Parser()
			parser.Opt("--one", args.IsString())
			input := []byte("one=this is one value\ntwo=this is two value\n")
			opt, err := parser.ParseIni(input)
			Expect(err).To(BeNil())
			Expect(opt.String("one")).To(Equal("this is one value"))
		})

		It("Should provide arg values from INI file after parsing the command line", func() {
			parser := args.Parser()
			parser.Opt("--one", args.IsString())
			parser.Opt("--two", args.IsString())
			parser.Opt("--three", args.IsString())
			cmdLine := []string{"--three", "this is three value"}
			opt, err := parser.ParseSlice(cmdLine)
			input := []byte("one=this is one value\ntwo=this is two value\n")
			opt, err = parser.ParseIni(input)
			Expect(err).To(BeNil())
			Expect(opt.String("one")).To(Equal("this is one value"))
			Expect(opt.String("three")).To(Equal("this is three value"))
		})

		It("Should not overide options supplied via the command line", func() {
			parser := args.Parser()
			parser.Opt("--one", args.IsString())
			parser.Opt("--two", args.IsString())
			parser.Opt("--three", args.IsString())
			cmdLine := []string{"--three", "this is three value", "--one", "this is from the cmd line"}
			opt, err := parser.ParseSlice(cmdLine)
			input := []byte("one=this is one value\ntwo=this is two value\n")
			opt, err = parser.ParseIni(input)
			Expect(err).To(BeNil())
			Expect(opt.String("one")).To(Equal("this is from the cmd line"))
			Expect(opt.String("three")).To(Equal("this is three value"))
		})

	})

	Describe("args.ParseArgs()", func() {
		parser := args.Parser()
		It("Should return error if Opt() was never called", func() {
			_, err := parser.ParseArgs()
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Must create some options to match with args.Opt() before calling arg.ParseArgs()"))
		})
	})

	Describe("args.Opt()", func() {
		cmdLine := []string{"--one", "-two", "++three", "+four", "--power-level"}

		It("Should create optional rule --one", func() {
			parser := args.Parser()
			parser.Opt("--one", args.Count())
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.IsPos).To(Equal(0))
		})

		It("Should create optional rule ++one", func() {
			parser := args.Parser()
			parser.Opt("++one", args.Count())
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.IsPos).To(Equal(0))
		})

		It("Should create optional rule -one", func() {
			parser := args.Parser()
			parser.Opt("-one", args.Count())
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.IsPos).To(Equal(0))
		})

		It("Should create optional rule +one", func() {
			parser := args.Parser()
			parser.Opt("+one", args.Count())
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.IsPos).To(Equal(0))
		})

		It("Should match --one", func() {
			parser := args.Parser()
			parser.Opt("--one", args.Count())
			opt, err := parser.ParseSlice(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("one")).To(Equal(1))
		})
		It("Should match -two", func() {
			parser := args.Parser()
			parser.Opt("-two", args.Count())
			opt, err := parser.ParseSlice(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("two")).To(Equal(1))
		})
		It("Should match ++three", func() {
			parser := args.Parser()
			parser.Opt("++three", args.Count())
			opt, err := parser.ParseSlice(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("three")).To(Equal(1))
		})
		It("Should match +four", func() {
			parser := args.Parser()
			parser.Opt("+four", args.Count())
			opt, err := parser.ParseSlice(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("four")).To(Equal(1))
		})
		It("Should match --power-level", func() {
			parser := args.Parser()
			parser.Opt("--power-level", args.Count())
			opt, err := parser.ParseSlice(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(1))
		})
	})

	Describe("args.Count()", func() {
		It("Should count one", func() {
			parser := args.Parser()
			cmdLine := []string{"--verbose"}
			parser.Opt("--verbose", args.Count())
			opt, err := parser.ParseSlice(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("verbose")).To(Equal(1))
		})
		It("Should count three times", func() {
			parser := args.Parser()
			cmdLine := []string{"--verbose", "--verbose", "--verbose"}
			parser.Opt("--verbose", args.Count())
			opt, err := parser.ParseSlice(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("verbose")).To(Equal(3))
		})
	})

	Describe("args.IsInt()", func() {
		It("Should ensure value supplied is an integer", func() {
			parser := args.Parser()
			parser.Opt("--power-level", args.IsInt())

			cmdLine := []string{"--power-level", "10000"}
			opt, err := parser.ParseSlice(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(10000))
		})

		It("Should set err if the option value is not parsable as an integer", func() {
			parser := args.Parser()
			cmdLine := []string{"--power-level", "over-ten-thousand"}
			parser.Opt("--power-level", args.IsInt())
			_, err := parser.ParseSlice(cmdLine)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Invalid value for '--power-level' - 'over-ten-thousand' is not an Integer"))
			//Expect(opt.Int("power-level")).To(Equal(0))
		})

		It("Should set err if no option value is provided", func() {
			parser := args.Parser()
			cmdLine := []string{"--power-level"}
			parser.Opt("--power-level", args.IsInt())
			_, err := parser.ParseSlice(cmdLine)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Expected '--power-level' to have an argument"))
			//Expect(opt.Int("power-level")).To(Equal(0))
		})
	})

	Describe("args.StoreInt()", func() {
		It("Should ensure value supplied is assigned to passed value", func() {
			parser := args.Parser()
			var value int
			parser.Opt("--power-level", args.StoreInt(&value))

			cmdLine := []string{"--power-level", "10000"}
			opt, err := parser.ParseSlice(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(10000))
			Expect(value).To(Equal(10000))
		})
	})

	Describe("args.IsString()", func() {
		It("Should provide string value", func() {
			parser := args.Parser()
			parser.Opt("--power-level", args.IsString())

			cmdLine := []string{"--power-level", "over-ten-thousand"}
			opt, err := parser.ParseSlice(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.String("power-level")).To(Equal("over-ten-thousand"))
		})

		It("Should set err if no option value is provided", func() {
			parser := args.Parser()
			cmdLine := []string{"--power-level"}
			parser.Opt("--power-level", args.IsString())
			_, err := parser.ParseSlice(cmdLine)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Expected '--power-level' to have an argument"))
		})
	})

	Describe("args.StoreString()", func() {
		It("Should ensure value supplied is assigned to passed value", func() {
			parser := args.Parser()
			var value string
			parser.Opt("--power-level", args.StoreString(&value))

			cmdLine := []string{"--power-level", "over-ten-thousand"}
			opt, err := parser.ParseSlice(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.String("power-level")).To(Equal("over-ten-thousand"))
			Expect(value).To(Equal("over-ten-thousand"))
		})
	})

	Describe("args.StoreStr()", func() {
		It("Should ensure value supplied is assigned to passed value", func() {
			parser := args.Parser()
			var value string
			parser.Opt("--power-level", args.StoreStr(&value))

			cmdLine := []string{"--power-level", "over-ten-thousand"}
			opt, err := parser.ParseSlice(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.String("power-level")).To(Equal("over-ten-thousand"))
			Expect(value).To(Equal("over-ten-thousand"))
		})
	})

	Describe("args.StoreTrue()", func() {
		It("Should ensure value supplied is true when argument is seen", func() {
			parser := args.Parser()
			var debug bool
			parser.Opt("--debug", args.StoreTrue(&debug))

			cmdLine := []string{"--debug"}
			opt, err := parser.ParseSlice(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Bool("debug")).To(Equal(true))
			Expect(debug).To(Equal(true))
		})

		It("Should ensure value supplied is false when argument is NOT seen", func() {
			parser := args.Parser()
			var debug bool
			parser.Opt("--debug", args.StoreTrue(&debug))

			cmdLine := []string{"--something-else"}
			opt, err := parser.ParseSlice(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Bool("debug")).To(Equal(false))
			Expect(debug).To(Equal(false))
		})
	})

	Describe("args.IsTrue()", func() {
		It("Should set true value when seen", func() {
			parser := args.Parser()
			parser.Opt("--help", args.IsTrue())

			cmdLine := []string{"--help"}
			opt, err := parser.ParseSlice(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Bool("help")).To(Equal(true))
		})

		It("Should set false when NOT seen", func() {
			parser := args.Parser()
			cmdLine := []string{"--something-else"}
			parser.Opt("--help", args.IsTrue())
			opt, err := parser.ParseSlice(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Bool("help")).To(Equal(false))
		})
	})

	Describe("args.StoreSlice()", func() {
		It("Should ensure []string provided is set when a comma separated list is provided", func() {
			parser := args.Parser()
			var list []string
			parser.Opt("--list", args.StoreSlice(&list))

			cmdLine := []string{"--list", "one,two,three"}
			opt, err := parser.ParseSlice(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Slice("list")).To(Equal([]string{"one", "two", "three"}))
			Expect(list).To(Equal([]string{"one", "two", "three"}))
		})

		It("Should ensure []interface{} provided is set if a single value is provided", func() {
			parser := args.Parser()
			var list []string
			parser.Opt("--list", args.StoreSlice(&list))

			cmdLine := []string{"--list", "one"}
			opt, err := parser.ParseSlice(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Slice("list")).To(Equal([]string{"one"}))
			Expect(list).To(Equal([]string{"one"}))
		})

		It("Should set err if no list value is provided", func() {
			parser := args.Parser()
			var list []string
			parser.Opt("--list", args.StoreSlice(&list))

			cmdLine := []string{"--list"}
			_, err := parser.ParseSlice(cmdLine)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Expected '--list' to have an argument"))
		})
	})

	Describe("args.Default()", func() {
		It("Should ensure default values is supplied if no matching argument is found", func() {
			parser := args.Parser()
			var value int
			parser.Opt("--power-level", args.StoreInt(&value), args.Default("10"))

			opt, err := parser.ParseSlice([]string{})
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(10))
			Expect(value).To(Equal(10))
		})

		It("Should panic if default value does not match Opt() type", func() {
			parser := args.Parser()
			panicCaught := false

			defer func() {
				msg := recover()
				Expect(msg).ToNot(BeNil())
				Expect(msg).To(ContainSubstring("args.Default"))
				panicCaught = true
			}()

			parser.Opt("--power-level", args.IsInt(), args.Default("over-ten-thousand"))
			Expect(panicCaught).To(Equal(true))
		})
	})

	Describe("args.Env()", func() {
		AfterEach(func() {
			os.Unsetenv("POWER_LEVEL")
		})

		It("Should supply the environ value if argument was not passed", func() {
			parser := args.Parser()
			var value int
			parser.Opt("--power-level", args.StoreInt(&value), args.Env("POWER_LEVEL"))

			os.Setenv("POWER_LEVEL", "10")

			opt, err := parser.ParseSlice([]string{})
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(10))
			Expect(value).To(Equal(10))
		})

		It("Should return an error if the environ value does not match the Opt() type", func() {
			parser := args.Parser()
			var value int
			parser.Opt("--power-level", args.StoreInt(&value), args.Env("POWER_LEVEL"))

			os.Setenv("POWER_LEVEL", "over-ten-thousand")

			_, err := parser.ParseSlice([]string{})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Invalid value for 'POWER_LEVEL' - 'over-ten-thousand' is not an Integer"))
		})

		It("Should use the default value if argument was not passed and environment var was not set", func() {
			parser := args.Parser()
			var value int
			parser.Opt("--power-level", args.StoreInt(&value), args.Env("POWER_LEVEL"), args.Default("1"))

			opt, err := parser.ParseSlice([]string{})
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(1))
			Expect(value).To(Equal(1))
		})
	})

	Describe("parser.GenerateOptHelp()", func() {
		It("Should generate help messages given a set of rules", func() {
			parser := args.Parser(args.WrapLen(80))
			parser.Opt("--power-level", args.Alias("-p"), args.Help("Specify our power level"))
			parser.Opt("--cat-level", args.Alias("-c"), args.Help(`Lorem ipsum dolor sit amet, consectetur
			mollit anim id est laborum.`))
			msg := parser.GenerateOptHelp()
			Expect(msg).To(Equal("  -p, --power-level   Specify our power level " +
				"\n  -c, --cat-level     Lorem ipsum dolor sit amet, consecteturmollit anim id est" +
				"\n                      laborum. \n"))
		})
	})

	/*Describe("parser.GenerateHelp()", func() {
			It("Should generate help messages given a set of rules", func() {
				parser := args.Parser(args.Name("dragon-ball"), args.WrapLen(80))
				parser.Opt("--power-level", args.Alias("-p"), args.Help("Specify our power level"))
				parser.Opt("--cat-level", args.Alias("-c"), args.Help(`Lorem ipsum dolor sit amet, consectetur
				adipiscing elit, sed do eiusmod tempor incididunt ut labore et
				mollit anim id est laborum.`))
				msg := parser.GenerateHelp()
				Expect(msg).To(Equal(args.Dedent(`
	                Usage:
	                dragon-ball [OPTIONS]

	                Options:
	                  --power-level, -p   Specify our power level
	                  --cat-level, -c     Lorem ipsum dolor sit amet, consecteturadipiscing elit, se
	                                     d do eiusmod tempor incididunt ut labore etmollit anim id
	                                      est laborum.
	            `)))
			})
		})*/

	Describe("Helper.WordWrap()", func() {
		It("Should wrap the line including the indent length", func() {
			// Should show the message with the indentation of 10 characters on the next line
			msg := args.WordWrap(`Lorem ipsum dolor sit amet, consectetur
			adipiscing elit, sed do eiusmod tempor incididunt ut labore et
			mollit anim id est laborum.`, 10, 80)
			Expect(msg).To(Equal(args.DedentTrim(`
            Lorem ipsum dolor sit amet, consecteturadipiscing elit, sed do
                      eiusmod tempor incididunt ut labore etmollit anim id est laborum.`, "\n")))
		})
		It("Should wrap the line without the indent length", func() {
			// Should show the message with no indentation
			msg := args.WordWrap(`Lorem ipsum dolor sit amet, consectetur
			adipiscing elit, sed do eiusmod tempor incididunt ut labore et
			mollit anim id est laborum.`, 0, 80)
			Expect(msg).To(Equal(args.DedentTrim(`
			Lorem ipsum dolor sit amet, consecteturadipiscing elit, sed do eiusmod tempor
			 incididunt ut labore etmollit anim id est laborum.`, "\n")))
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
