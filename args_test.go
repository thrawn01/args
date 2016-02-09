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

	Describe("args.Parse()", func() {
		parser := args.Parser()
		It("Should return error if Opt() was never called", func() {
			_, err := parser.Parse()
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Must create some options to match with args.Opt() before calling arg.Parse()"))
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
			opt, err := parser.ParseArgs(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("one")).To(Equal(1))
		})
		It("Should match -two", func() {
			parser := args.Parser()
			parser.Opt("-two", args.Count())
			opt, err := parser.ParseArgs(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("two")).To(Equal(1))
		})
		It("Should match ++three", func() {
			parser := args.Parser()
			parser.Opt("++three", args.Count())
			opt, err := parser.ParseArgs(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("three")).To(Equal(1))
		})
		It("Should match +four", func() {
			parser := args.Parser()
			parser.Opt("+four", args.Count())
			opt, err := parser.ParseArgs(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("four")).To(Equal(1))
		})
		It("Should match --power-level", func() {
			parser := args.Parser()
			parser.Opt("--power-level", args.Count())
			opt, err := parser.ParseArgs(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(1))
		})
	})

	Describe("args.Count()", func() {
		It("Should count one", func() {
			parser := args.Parser()
			cmdLine := []string{"--verbose"}
			parser.Opt("--verbose", args.Count())
			opt, err := parser.ParseArgs(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("verbose")).To(Equal(1))
		})
		It("Should count three times", func() {
			parser := args.Parser()
			cmdLine := []string{"--verbose", "--verbose", "--verbose"}
			parser.Opt("--verbose", args.Count())
			opt, err := parser.ParseArgs(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("verbose")).To(Equal(3))
		})
	})

	Describe("args.IsInt()", func() {
		It("Should ensure value supplied is an integer", func() {
			parser := args.Parser()
			parser.Opt("--power-level", args.IsInt())

			cmdLine := []string{"--power-level", "10000"}
			opt, err := parser.ParseArgs(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(10000))
		})

		It("Should set err if the option value is not parsable as an integer", func() {
			parser := args.Parser()
			cmdLine := []string{"--power-level", "over-ten-thousand"}
			parser.Opt("--power-level", args.IsInt())
			_, err := parser.ParseArgs(cmdLine)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Invalid value for '--power-level' - 'over-ten-thousand' is not an Integer"))
			//Expect(opt.Int("power-level")).To(Equal(0))
		})

		It("Should set err if no option value is provided", func() {
			parser := args.Parser()
			cmdLine := []string{"--power-level"}
			parser.Opt("--power-level", args.IsInt())
			_, err := parser.ParseArgs(cmdLine)
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
			opt, err := parser.ParseArgs(cmdLine)
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
			opt, err := parser.ParseArgs(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.String("power-level")).To(Equal("over-ten-thousand"))
		})

		It("Should set err if no option value is provided", func() {
			parser := args.Parser()
			cmdLine := []string{"--power-level"}
			parser.Opt("--power-level", args.IsString())
			_, err := parser.ParseArgs(cmdLine)
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
			opt, err := parser.ParseArgs(cmdLine)
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
			opt, err := parser.ParseArgs(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.String("power-level")).To(Equal("over-ten-thousand"))
			Expect(value).To(Equal("over-ten-thousand"))
		})
	})

	Describe("args.Default()", func() {
		It("Should ensure default values is supplied if no matching argument is found", func() {
			parser := args.Parser()
			var value int
			parser.Opt("--power-level", args.StoreInt(&value), args.Default("10"))

			opt, err := parser.ParseArgs([]string{})
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

			opt, err := parser.ParseArgs([]string{})
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(10))
			Expect(value).To(Equal(10))
		})

		It("Should return an error if the environ value does not match the Opt() type", func() {
			parser := args.Parser()
			var value int
			parser.Opt("--power-level", args.StoreInt(&value), args.Env("POWER_LEVEL"))

			os.Setenv("POWER_LEVEL", "over-ten-thousand")

			_, err := parser.ParseArgs([]string{})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Invalid value for 'POWER_LEVEL' - 'over-ten-thousand' is not an Integer"))
		})

		It("Should use the default value if argument was not passed and environment var was not set", func() {
			parser := args.Parser()
			var value int
			parser.Opt("--power-level", args.StoreInt(&value), args.Env("POWER_LEVEL"), args.Default("1"))

			opt, err := parser.ParseArgs([]string{})
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(1))
			Expect(value).To(Equal(1))
		})
	})
})
