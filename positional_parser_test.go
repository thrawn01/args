package args_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/thrawn01/args"
)

func TestNewPosParser(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Args PosParser")
}

var _ = Describe("Parser", func() {
	var log *TestLogger

	BeforeEach(func() {
		log = NewTestLogger()
	})

	FDescribe("Parser.AddFlag()", func() {
		cmdLine := []string{"--one", "-two", "++three", "+four", "--power-level"}

		It("Should create optional rule --one", func() {
			parser := args.NewPosParser()
			parser.AddFlag("--one").Count()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.Order).To(Equal(0))
		})
		It("Should create optional rule ++one", func() {
			parser := args.NewPosParser()
			parser.AddFlag("++one").Count()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.Order).To(Equal(0))
		})

		It("Should create optional rule -one", func() {
			parser := args.NewPosParser()
			parser.AddFlag("-one").Count()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.Order).To(Equal(0))
		})

		It("Should create optional rule +one", func() {
			parser := args.NewPosParser()
			parser.AddFlag("+one").Count()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.Order).To(Equal(0))
		})

		FIt("Should match --one", func() {
			parser := args.NewPosParser()
			parser.AddFlag("--one").Count()

			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.Order).To(Equal(0))

			_, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			//Expect(opt.Int("one")).To(Equal(1))
		})
		It("Should match -two", func() {
			parser := args.NewPosParser()
			parser.AddFlag("-two").Count()
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("two")).To(Equal(1))
		})
		It("Should match ++three", func() {
			parser := args.NewPosParser()
			parser.AddFlag("++three").Count()
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("three")).To(Equal(1))
		})
		It("Should match +four", func() {
			parser := args.NewPosParser()
			parser.AddFlag("+four").Count()
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("four")).To(Equal(1))
		})
		It("Should match --power-level", func() {
			parser := args.NewPosParser()
			parser.AddFlag("--power-level").Count()
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(1))
		})
		It("Should match 'no-docker'", func() {
			cmdLine := []string{"--no-docker"}

			parser := args.NewPosParser()
			parser.AddFlag("no-docker").Count()
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Bool("no-docker")).To(Equal(true))
		})
		It("Should raise an error if a option is required but not provided", func() {
			parser := args.NewPosParser()
			parser.AddFlag("--power-level").Required()
			cmdLine := []string{""}
			_, err := parser.Parse(cmdLine)

			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(Equal("option '--power-level' is required"))
		})
	})
})
