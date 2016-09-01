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

		opts = parser.NewOptionsFromMap(
			map[string]interface{}{
				"int":    1,
				"bool":   true,
				"string": "one",
				"endpoints": map[string]interface{}{
					"endpoint1": "host1",
					"endpoint2": "host2",
					"endpoint3": "host3",
				},
				"deeply": map[string]interface{}{
					"nested": map[string]interface{}{
						"thing": "foo",
					},
					"foo": "bar",
				},
			},
		)

		/*opts.String("string")                       // == "one"
		opts.Group("endpoints")                     // == *args.Options
		opts.Group("endpoints").String("endpoint1") // == "host1"
		opts.Group("endpoints").ToMap()             // {"endpoint1": "host1", ...}
		opts.StringMap("endpoints")                 // {"endpoint1": "host1", ...}
		opts.KeySlice("endpoints")                  // [ "endpoint1", "endpoint2", ]
		opts.StringSlice("endpoints")               // [ "host1", "host2", "host3" ]*/

		// Leaves the door open for IntSlice(), IntMap(), etc....

		/*opts = parser.NewOptionsFromMap(args.DefaultOptionGroup,
		map[string]map[string]*args.OptionValue{
			args.DefaultOptionGroup: {
				"int":    &args.OptionValue{Value: 1, Flags: 0},
				"bool":   &args.OptionValue{Value: true, Flags: 0},
				"string": &args.OptionValue{Value: "one", Flags: 0},
			},
			"endpoints": {
				"endpoint1": &args.OptionValue{Value: "host1", Flags: 0},
				"endpoint2": &args.OptionValue{Value: "host2", Flags: 0},
				"endpoint3": &args.OptionValue{Value: "host3", Flags: 0},
			},
		})*/

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

	Describe("IsSet()", func() {
		It("Should return true if the value is not a cast default", func() {
			parser := args.NewParser()
			parser.AddOption("--is-set").IsInt().Default("1")
			parser.AddOption("--not-set")
			opt, err := parser.ParseArgs(nil)
			Expect(err).To(BeNil())
			Expect(opt.IsSet("is-set")).To(Equal(true))
			Expect(opt.IsSet("not-set")).To(Equal(false))

		})
	})

	Describe("InspectOpt()", func() {
		It("Should return the option object requested", func() {
			parser := args.NewParser()
			parser.AddOption("--is-set").IsInt().Default("1")
			parser.AddOption("--not-set")
			opt, err := parser.ParseArgs(nil)
			Expect(err).To(BeNil())
			option := opt.InspectOpt("is-set")
			Expect(option.GetValue().(int)).To(Equal(1))
			Expect(option.GetRule().Flags).To(Equal(int64(32)))
		})
	})

	Describe("Required()", func() {
		It("Should return nil if all values are provided", func() {
			parser := args.NewParser()
			parser.AddOption("--is-set").IsInt().Default("1")
			parser.AddOption("--is-provided")
			parser.AddOption("--not-set")
			opt, err := parser.ParseArgs(&[]string{"--is-provided", "foo"})
			Expect(err).To(BeNil())

			// All options required have values
			Expect(opt.Required([]string{"is-set", "is-provided"})).To(BeNil())

			// Option 'not-set' is missing is not provided
			err = opt.Required([]string{"is-set", "is-provided", "not-set"})
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(Equal("not-set"))
		})
	})

	Describe("ToString()", func() {
		It("Should return a string representation of the options", func() {
			output := opts.ToString()
			Expect(output).To(Equal(`{
  'bool' = true
  'deeply' = {
    'foo' = bar
    'nested' = {
      'thing' = foo
    }
  }
  'endpoints' = {
    'endpoint1' = host1
    'endpoint2' = host2
    'endpoint3' = host3
  }
  'int' = 1
  'string' = one
}`))
		})
	})

})
