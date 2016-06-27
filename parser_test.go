package args_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/thrawn01/args"
)

var _ = Describe("ArgParser", func() {
	Describe("ArgParser.ParseArgs(nil)", func() {
		It("Should return error if AddOption() was never called", func() {
			parser := args.NewParser(args.NoHelp())
			_, err := parser.ParseArgs(nil)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Must create some options to match with args.AddOption() before calling arg.ParseArgs()"))
		})
		It("Should add Help option if none provided", func() {
			parser := args.NewParser()
			_, err := parser.ParseArgs(nil)
			Expect(err).To(BeNil())
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("help"))
		})
	})
	Describe("ArgParser.AddOption()", func() {
		cmdLine := []string{"--one", "-two", "++three", "+four", "--power-level"}

		It("Should create optional rule --one", func() {
			parser := args.NewParser()
			parser.AddOption("--one").Count()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.Order).To(Equal(0))
		})
		It("Should create optional rule ++one", func() {
			parser := args.NewParser()
			parser.AddOption("++one").Count()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.Order).To(Equal(0))
		})

		It("Should create optional rule -one", func() {
			parser := args.NewParser()
			parser.AddOption("-one").Count()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.Order).To(Equal(0))
		})

		It("Should create optional rule +one", func() {
			parser := args.NewParser()
			parser.AddOption("+one").Count()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.Order).To(Equal(0))
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
		It("Should raise an error if a option is required but not provided", func() {
			parser := args.NewParser()
			parser.AddOption("--power-level").Required()
			cmdLine := []string{""}
			_, err := parser.ParseArgs(&cmdLine)

			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(Equal("option '--power-level' is required"))
		})
	})

	Describe("ArgParser.AddPositional()", func() {
		cmdLine := []string{"one", "two", "three", "four"}

		It("Should create positional rule first", func() {
			parser := args.NewParser()
			parser.AddPositional("first").IsString()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("first"))
			Expect(rule.Order).To(Equal(1))
		})
		It("Should match first position 'one'", func() {
			parser := args.NewParser()
			parser.AddPositional("first").IsString()
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.String("first")).To(Equal("one"))
		})
		It("Should match first positionals in order of declaration", func() {
			parser := args.NewParser()
			parser.AddPositional("first").IsString()
			parser.AddPositional("second").IsString()
			parser.AddPositional("third").IsString()
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.String("first")).To(Equal("one"))
			Expect(opt.String("second")).To(Equal("two"))
			Expect(opt.String("third")).To(Equal("three"))
		})
		It("Should handle no positionals if declared", func() {
			parser := args.NewParser()
			parser.AddPositional("first").IsString()
			parser.AddPositional("second").IsString()
			parser.AddPositional("third").IsString()

			cmdLine := []string{"one", "two"}
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.String("first")).To(Equal("one"))
			Expect(opt.String("second")).To(Equal("two"))
			Expect(opt.String("third")).To(Equal(""))
		})
		It("Should mixing optionals and positionals", func() {
			parser := args.NewParser()
			parser.AddOption("--verbose").IsTrue()
			parser.AddOption("--first").IsString()
			parser.AddPositional("second").IsString()
			parser.AddPositional("third").IsString()

			cmdLine := []string{"--first", "one", "two", "--verbose"}
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.String("first")).To(Equal("one"))
			Expect(opt.String("second")).To(Equal("two"))
			Expect(opt.String("third")).To(Equal(""))
			Expect(opt.Bool("verbose")).To(Equal(true))
		})
		It("Should raise an error if an optional and a positional share the same name", func() {
			parser := args.NewParser()
			parser.AddOption("--first").IsString()
			parser.AddPositional("first").IsString()

			cmdLine := []string{"--first", "one", "one"}
			_, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(Equal("Duplicate option with same name as 'first'"))
		})
		It("Should raise an error if a positional is required but not provided", func() {
			parser := args.NewParser()
			parser.AddPositional("first").Required()
			parser.AddPositional("second").Required()

			cmdLine := []string{"one"}
			_, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(Equal("positional 'second' is required"))
		})
	})
	Describe("ArgParser.AddConfig()", func() {
		cmdLine := []string{"--power-level", "--power-level"}
		It("Should add new config only rule", func() {
			parser := args.NewParser()
			parser.AddConfig("power-level").Count().Help("My help message")

			// Should ignore command line options
			opt, err := parser.ParseArgs(&cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(0))

			// But Apply() a config file
			options := parser.NewOptionsFromMap(args.DefaultOptionGroup,
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
	Describe("ArgParser.InGroup()", func() {
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
	Describe("ArgParser.AddRule()", func() {
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
	Describe("ArgParser.GenerateOptHelp()", func() {
		It("Should generate help messages given a set of rules", func() {
			parser := args.NewParser(args.WrapLen(80))
			parser.AddOption("--power-level").Alias("-p").Help("Specify our power level")
			parser.AddOption("--cat-level").
				Alias("-c").
				Help(`Lorem ipsum dolor sit amet, consectetur
			mollit anim id est laborum.`)
			msg := parser.GenerateHelpSection(args.IsOption)
			Expect(msg).To(Equal("  -p, --power-level   Specify our power level " +
				"\n  -c, --cat-level     Lorem ipsum dolor sit amet, consecteturmollit anim id est" +
				"\n                      laborum. \n"))
		})
	})
	Describe("ArgParser.GenerateHelp()", func() {
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
	Describe("ArgParser.AddCommand()", func() {
		It("Should run a command if seen on the command line", func() {
			parser := args.NewParser()
			called := false
			parser.AddCommand("command1", func(parent *args.ArgParser, data interface{}) int {
				called = true
				return 0
			})
			cmdLine := []string{"command1"}
			retCode, err := parser.ParseAndRun(&cmdLine, nil)
			Expect(err).To(BeNil())
			Expect(retCode).To(Equal(0))
			Expect(called).To(Equal(true))
		})
		It("Should not confuse a command with a following positional", func() {
			parser := args.NewParser()
			called := 0
			parser.AddCommand("set", func(parent *args.ArgParser, data interface{}) int {
				called++
				return 0
			})
			cmdLine := []string{"set", "set"}
			retCode, err := parser.ParseAndRun(&cmdLine, nil)
			Expect(err).To(BeNil())
			Expect(retCode).To(Equal(0))
			Expect(called).To(Equal(1))
		})
		It("Should provide a sub parser with that will not confuse a following positional", func() {
			parser := args.NewParser()
			called := 0
			parser.AddCommand("set", func(parent *args.ArgParser, data interface{}) int {
				parent.AddPositional("first").Required()
				parent.AddPositional("second").Required()
				opts, err := parent.ParseArgs(nil)
				Expect(err).To(BeNil())
				Expect(opts.String("first")).To(Equal("foo"))
				Expect(opts.String("second")).To(Equal("bar"))

				called++
				return 0
			})
			cmdLine := []string{"set", "foo", "bar"}
			retCode, err := parser.ParseAndRun(&cmdLine, nil)
			Expect(err).To(BeNil())
			Expect(retCode).To(Equal(0))
			Expect(called).To(Equal(1))

		})
		It("Should allow sub commands to be a thing", func() {
			parser := args.NewParser()
			called := 0
			parser.AddCommand("volume", func(parent *args.ArgParser, data interface{}) int {
				parent.AddCommand("create", func(subParent *args.ArgParser, data interface{}) int {
					subParent.AddPositional("volume-name").Required()
					opts, err := subParent.ParseArgs(nil)
					Expect(err).To(BeNil())
					Expect(opts.String("volume-name")).To(Equal("my-new-volume"))

					called++
					return 0
				})
				retCode, err := parent.ParseAndRun(nil, nil)
				Expect(err).To(BeNil())
				Expect(retCode).To(Equal(0))
				return retCode
			})
			cmdLine := []string{"volume", "create", "my-new-volume"}
			retCode, err := parser.ParseAndRun(&cmdLine, nil)
			Expect(err).To(BeNil())
			Expect(retCode).To(Equal(0))
			Expect(called).To(Equal(1))

		})
	})
})
