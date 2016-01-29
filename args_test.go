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

	/*Describe("ParseArgs()", func() {
		cmdLine := []string{"--one", "--two", "--three"}
		parser := args.Parser()
		It("Should return error if Opt() was never called", func() {
			_, err := parser.ParseArgs(cmdLine)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Must create some options to parse with args.Opt() before calling arg.Parse()"))
		})
		It("Should count arguments", func() {
			parser.Opt("--one", args.Count())
			opt, _ := parser.ParseArgs(cmdLine)
			Expect(opt.Int("one")).To(Equal(1))
		})
	})*/

	Describe("Opt()", func() {
		parser := args.Parser()

		It("Should create optional rule --one", func() {
			parser.Opt("--one", args.Count())
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.IsPos).To(Equal(0))
		})

		It("Should create optional rule ++one", func() {
			parser.Opt("++one", args.Count())
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.IsPos).To(Equal(0))
		})

		It("Should create optional rule -one", func() {
			parser.Opt("-one", args.Count())
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.IsPos).To(Equal(0))
		})

		It("Should create optional rule +one", func() {
			parser.Opt("+one", args.Count())
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.IsPos).To(Equal(0))
		})
	})
})
