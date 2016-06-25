package args_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/thrawn01/args"
)

var _ = Describe("Options", func() {
	var opts *args.Options
	var log *TestLogger
	BeforeEach(func() {
		log = NewTestLogger()
		parser := args.NewParser()
		parser.SetLog(log)
		opts = parser.NewOptionsFromMap(args.DefaultOptionGroup,
			map[string]map[string]*args.OptionValue{
				args.DefaultOptionGroup: {
					"int":    &args.OptionValue{Value: 1, Seen: false},
					"bool":   &args.OptionValue{Value: true, Seen: false},
					"string": &args.OptionValue{Value: "one", Seen: false},
				},
				"endpoints": {
					"endpoint1": &args.OptionValue{Value: "host1", Seen: false},
					"endpoint2": &args.OptionValue{Value: "host2", Seen: false},
					"endpoint3": &args.OptionValue{Value: "host3", Seen: false},
				},
			})

	})
	Describe("log", func() {
		It("Should log to StdLogger when cast fails", func() {
			result := opts.Int("string")
			Expect(log.GetEntry()).To(Equal("Unable to Cast \"one\" to int for key 'string'|"))
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
	Describe("String()", func() {
		It("Should return values as string", func() {
			result := opts.String("string")
			Expect(log.GetEntry()).To(Equal(""))
			Expect(result).To(Equal("one"))
		})
		It("Should return default value if key doesn't exist", func() {
			result := opts.String("none")
			Expect(result).To(Equal(""))
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
	Describe("ToMap()", func() {
		It("Should return a map of the group options", func() {
			Expect(opts.Group("endpoints").ToMap()).To(Equal(map[string]interface{}{
				"endpoint1": "host1",
				"endpoint2": "host2",
				"endpoint3": "host3",
			}))
		})
		It("Should return an empty map if the group doesn't exist", func() {
			Expect(opts.Group("no-group").ToMap()).To(Equal(map[string]interface{}{}))
		})
	})
})
