package args_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/thrawn01/args"
)

var _ = Describe("ArgParser", func() {
	Describe("FromIni()", func() {
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
})
