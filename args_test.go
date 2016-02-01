package args_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/thrawn01/args"
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
			opts.Convert("one", func(value interface{}) {
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

			opts.Convert("one", func(value interface{}) {
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
			Expect(err.Error()).To(Equal("Must create some options to parse with args.Opt() before calling arg.Parse()"))
		})
	})

	Describe("args.Opt()", func() {
		cmdLine := []string{"--one", "-two", "++three", "+four"}

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
})
