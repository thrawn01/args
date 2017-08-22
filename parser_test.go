package args_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/thrawn01/args"
)

var _ = Describe("Parser", func() {
	var log *TestLogger

	BeforeEach(func() {
		log = NewTestLogger()
	})

	Describe("Parser.Parse(nil)", func() {
		It("Should return error if AddFlag() or AddArgument() was never called", func() {
			parser := args.NewParser().AddHelp(false)
			_, err := parser.Parse(nil)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Must create some options to match with before calling arg.Parse()"))
		})
		It("Should return error if duplicate flag or arguments defined", func() {
			parser := args.NewParser().AddHelp(false)
			parser.AddFlag("--time").Alias("-t")
			parser.AddFlag("--trigger").Alias("-t")
			_, err := parser.Parse(nil)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Duplicate alias '-t' for 'time' redefined by 'trigger'"))
		})
		It("Should add Help option if none provided", func() {
			parser := args.NewParser()
			_, err := parser.Parse(nil)
			Expect(err).To(BeNil())
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("help"))
		})
	})
	Describe("Parser.AddHelp()", func() {
		It("If set to false should not create a --help option", func() {
			parser := args.NewParser()
			for _, rule := range parser.GetRules() {
				Expect(rule.Name).To(Not(Equal("help")))
			}
		})
	})
	Describe("Parser.AddOption()", func() {
		cmdLine := []string{"--one", "-two", "++three", "+four", "--power-level"}

		It("Should create optional rule --one", func() {
			parser := args.NewParser()
			parser.AddFlag("--one").Count()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.Order).To(Equal(0))
		})
		It("Should create optional rule ++one", func() {
			parser := args.NewParser()
			parser.AddFlag("++one").Count()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.Order).To(Equal(0))
		})

		It("Should create optional rule -one", func() {
			parser := args.NewParser()
			parser.AddFlag("-one").Count()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.Order).To(Equal(0))
		})

		It("Should create optional rule +one", func() {
			parser := args.NewParser()
			parser.AddFlag("+one").Count()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.Order).To(Equal(0))
		})

		It("Should match --one", func() {
			parser := args.NewParser()
			parser.AddFlag("--one").Count()
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("one")).To(Equal(1))
		})
		It("Should match -two", func() {
			parser := args.NewParser()
			parser.AddFlag("-two").Count()
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("two")).To(Equal(1))
		})
		It("Should match ++three", func() {
			parser := args.NewParser()
			parser.AddFlag("++three").Count()
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("three")).To(Equal(1))
		})
		It("Should match +four", func() {
			parser := args.NewParser()
			parser.AddFlag("+four").Count()
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("four")).To(Equal(1))
		})
		It("Should match --power-level", func() {
			parser := args.NewParser()
			parser.AddFlag("--power-level").Count()
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(1))
		})
		It("Should match 'no-docker'", func() {
			cmdLine := []string{"--no-docker"}

			parser := args.NewParser()
			parser.AddFlag("no-docker").Count()
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Bool("no-docker")).To(Equal(true))
		})
		It("Should raise an error if a option is required but not provided", func() {
			parser := args.NewParser()
			parser.AddFlag("--power-level").Required()
			cmdLine := []string{""}
			_, err := parser.Parse(cmdLine)

			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(Equal("option '--power-level' is required"))
		})
	})

	Describe("Parser.IsStringSlice()", func() {
		It("Should allow slices in a comma delimited string", func() {
			parser := args.NewParser()
			parser.AddFlag("--list").IsStringSlice().Default("foo,bar,bit")

			// Test Default Value
			opt, err := parser.Parse(nil)
			Expect(err).To(BeNil())
			Expect(opt.StringSlice("list")).To(Equal([]string{"foo", "bar", "bit"}))

			// Provided on the command line
			cmdLine := []string{"--list", "belt,car,table"}
			opt, err = parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.StringSlice("list")).To(Equal([]string{"belt", "car", "table"}))
		})
		It("Should allow slices in a comma delimited string saved to a variable", func() {
			parser := args.NewParser()
			var list []string
			parser.AddFlag("--list").StoreStringSlice(&list).Default("foo,bar,bit")

			cmdLine := []string{"--list", "belt,car,table"}
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.StringSlice("list")).To(Equal([]string{"belt", "car", "table"}))
			Expect(list).To(Equal([]string{"belt", "car", "table"}))
		})
		It("Should allow multiple iterations of the same option to create a slice", func() {
			parser := args.NewParser()
			parser.AddFlag("--list").IsStringSlice()

			cmdLine := []string{"--list", "bee", "--list", "cat", "--list", "dad"}
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.StringSlice("list")).To(Equal([]string{"bee", "cat", "dad"}))
		})
		It("Should allow multiple iterations of the same option to create a slice - with var", func() {
			parser := args.NewParser()
			var list []string
			parser.AddFlag("--list").StoreStringSlice(&list)

			cmdLine := []string{"--list", "bee", "--list", "cat", "--list", "dad"}
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.StringSlice("list")).To(Equal([]string{"bee", "cat", "dad"}))
			Expect(list).To(Equal([]string{"bee", "cat", "dad"}))
		})
		It("Should allow multiple iterations of the same argument to create a slice", func() {
			parser := args.NewParser()
			parser.AddArgument("list").IsStringSlice()

			cmdLine := []string{"bee", "cat", "dad"}
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.StringSlice("list")).To(Equal([]string{"bee", "cat", "dad"}))
		})
	})
	Describe("Parser.IsStringMap()", func() {
		It("Should handle slice apply from alternate sources", func() {
			parser := args.NewParser()
			parser.AddFlag("--list").IsStringSlice()

			options := parser.NewOptionsFromMap(
				map[string]interface{}{
					"list": []string{"bee", "cat", "dad"},
				})
			opt, err := parser.Apply(options)
			Expect(err).To(BeNil())
			Expect(opt.StringSlice("list")).To(Equal([]string{"bee", "cat", "dad"}))
		})
		It("Should error if apply on map is invalid type", func() {
			parser := args.NewParser()
			parser.AddFlag("--list").IsStringSlice()

			options := parser.NewOptionsFromMap(
				map[string]interface{}{
					"list": 1,
				})
			_, err := parser.Apply(options)
			Expect(err).To(Not(BeNil()))
		})
		It("Should not error if key contains non alpha char", func() {
			parser := args.NewParser()
			parser.AddFlag("--map").IsStringMap()
			parser.AddFlag("--foo")

			cmdLine := []string{"--map", "http.ip=192.168.1.1"}
			opt, err := parser.Parse(cmdLine)
			Expect(opt.StringMap("map")).To(Equal(map[string]string{"http.ip": "192.168.1.1"}))
			Expect(err).To(BeNil())
		})
		It("Should not error if key or value contains an escaped equal or comma", func() {
			parser := args.NewParser()
			parser.AddFlag("--map").IsStringMap()
			parser.AddFlag("--foo")

			cmdLine := []string{"--map", `http\=ip=192.168.1.1`}
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.StringMap("map")).To(Equal(map[string]string{"http=ip": "192.168.1.1"}))
		})
		It("Should error not error if no map value is supplied", func() {
			parser := args.NewParser()
			parser.AddFlag("--list").IsStringMap()
			parser.AddFlag("--foo")

			_, err := parser.Parse(nil)
			Expect(err).To(BeNil())
		})
		It("Should allow string map with '=' expression in a comma delimited string", func() {
			parser := args.NewParser()
			parser.AddFlag("--map").IsStringMap().Default("foo=bar,bar=foo")

			// Test Default Value
			opt, err := parser.Parse(nil)
			Expect(opt).To(Not(BeNil()))

			Expect(err).To(BeNil())
			Expect(opt.StringMap("map")).To(Equal(map[string]string{"foo": "bar", "bar": "foo"}))

			// Provided on the command line
			cmdLine := []string{"--map", "belt=car,table=cloth"}
			opt, err = parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.StringMap("map")).To(Equal(map[string]string{"belt": "car", "table": "cloth"}))
		})
		It("Should store string map into a struct", func() {
			parser := args.NewParser()
			var destMap map[string]string
			parser.AddFlag("--map").StoreStringMap(&destMap).Default("foo=bar,bar=foo")

			// Test Default Value
			opt, err := parser.Parse(nil)
			Expect(opt).To(Not(BeNil()))

			Expect(err).To(BeNil())
			Expect(destMap).To(Equal(map[string]string{"foo": "bar", "bar": "foo"}))

			// Provided on the command line
			cmdLine := []string{"--map", "belt=car,table=cloth"}
			opt, err = parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(destMap).To(Equal(map[string]string{"belt": "car", "table": "cloth"}))
		})
		It("Should allow string map with JSON string", func() {
			parser := args.NewParser()
			parser.AddFlag("--map").IsStringMap().Default(`{"foo":"bar", "bar":"foo"}`)

			// Test Default Value
			opt, err := parser.Parse(nil)
			Expect(err).To(BeNil())
			Expect(opt.StringMap("map")).To(Equal(map[string]string{"foo": "bar", "bar": "foo"}))

			// Provided on the command line
			cmdLine := []string{"--map", `{"belt":"car","table":"cloth"}`}
			opt, err = parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.StringMap("map")).To(Equal(map[string]string{"belt": "car", "table": "cloth"}))
		})
		It("Should allow multiple iterations of the same argument to create a map", func() {
			parser := args.NewParser()
			parser.AddFlag("--map").IsStringMap()

			cmdLine := []string{"--map", "blue=bell", "--map", "cat=dog", "--map", "dad=boy"}
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.StringMap("map")).To(Equal(map[string]string{
				"blue": "bell",
				"cat":  "dog",
				"dad":  "boy",
			}))
		})
		It("Should handle map apply from alternate sources", func() {
			parser := args.NewParser()
			parser.AddFlag("--map").IsStringMap()

			options := parser.NewOptionsFromMap(
				map[string]interface{}{
					"map": map[string]string{"key": "value"},
				})
			opt, err := parser.Apply(options)
			Expect(err).To(BeNil())
			Expect(opt.StringMap("map")).To(Equal(map[string]string{
				"key": "value",
			}))
		})
		It("Should error if apply on map is invalid type", func() {
			parser := args.NewParser()
			parser.AddFlag("--map").IsStringMap()

			options := parser.NewOptionsFromMap(
				map[string]interface{}{
					"map": 1,
				})
			_, err := parser.Apply(options)
			Expect(err).To(Not(BeNil()))
		})
		It("Should fail with incomplete key=values", func() {
			parser := args.NewParser()
			parser.AddFlag("--map").IsStringMap()

			cmdLine := []string{"--map", "belt"}
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(Not(BeNil()))

			cmdLine = []string{"--map", "belt="}
			opt, err = parser.Parse(cmdLine)
			Expect(err).To(Not(BeNil()))

			cmdLine = []string{"--map", "belt=blue;,"}
			opt, err = parser.Parse(cmdLine)
			Expect(err).To(Not(BeNil()))

			cmdLine = []string{"--map", "belt=car,table=cloth"}
			opt, err = parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.StringMap("map")).To(Equal(map[string]string{"belt": "car", "table": "cloth"}))
		})

		It("Should allow multiple iterations of the same argument to create a map with JSON", func() {
			parser := args.NewParser()
			parser.AddFlag("--map").IsStringMap()

			cmdLine := []string{
				"--map", `{"blue":"bell"}`,
				"--map", `{"cat":"dog"}`,
				"--map", `{"dad":"boy"}`,
			}
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.StringMap("map")).To(Equal(map[string]string{
				"blue": "bell",
				"cat":  "dog",
				"dad":  "boy",
			}))
		})

	})
	Describe("Parser.PrefixChars()", func() {
		It("Should return allow user to modify which prefix characters to match optional args with", func() {
			cmdLine := []string{"-first", "first", "+second", "second"}
			parser := args.NewParser().PrefixChars([]string{"+", "-"})
			parser.AddFlag("first")
			parser.AddFlag("second")
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.String("first")).To(Equal("first"))
			Expect(opt.String("second")).To(Equal("second"))
		})
		It("Should return not allow invalid prefix characters", func() {
			parser := args.NewParser().PrefixChars([]string{"_"})
			_, err := parser.Parse(nil)
			Expect(err.Error()).To(Equal("invalid PrefixChars() '_'; alpha, underscore and whitespace not allowed"))

			parser = args.NewParser().PrefixChars([]string{"a"})
			_, err = parser.Parse(nil)
			Expect(err.Error()).To(Equal("invalid PrefixChars() 'a'; alpha, underscore and whitespace not allowed"))

			parser = args.NewParser().PrefixChars([]string{" "})
			_, err = parser.Parse(nil)
			Expect(err.Error()).To(Equal("invalid PrefixChars() ' '; alpha, underscore and whitespace not allowed"))

			parser = args.NewParser().PrefixChars([]string{""})
			_, err = parser.Parse(nil)
			Expect(err.Error()).To(Equal("invalid PrefixChars() prefix cannot be empty"))
		})

	})
	Describe("Parser.ModifyRule()", func() {
		It("Should return allow user to modify an existing rule ", func() {
			parser := args.NewParser()
			parser.AddFlag("first").IsString().Default("one")
			opt, err := parser.Parse(nil)
			Expect(err).To(BeNil())
			Expect(opt.String("first")).To(Equal("one"))
			// Modify the rule and parse again
			parser.ModifyRule("first").Default("two")
			opt, err = parser.Parse(nil)
			Expect(err).To(BeNil())
			Expect(opt.String("first")).To(Equal("two"))
		})
	})
	Describe("Parser.GetRule()", func() {
		It("Should return allow user to modify an existing rule ", func() {
			parser := args.NewParser()
			parser.AddFlag("first").IsString().Default("one")
			opt, err := parser.Parse(nil)
			Expect(err).To(BeNil())
			Expect(opt.String("first")).To(Equal("one"))
			// Get the rule
			rule := parser.GetRule("first")
			Expect(rule.Name).To(Equal("first"))
		})
	})
	Describe("Parser.SubCommands()", func() {
		cmdLine := []string{"one", "-v", "two", "three", "-t", "four"}
		It("Should return a list of parent sub commands", func() {
			var parents []string
			parser := args.NewParser()
			parser.AddCommand("one", func(parser *args.Parser, _ interface{}) (int, error) {
				parser.AddCommand("two", func(parser *args.Parser, _ interface{}) (int, error) {
					parents = parser.SubCommands()
					return 0, nil
				})
				return parser.ParseAndRun(nil, nil)
			})
			parser.ParseAndRun(cmdLine, nil)
			Expect(parents).To(Equal([]string{"one", "two"}))
		})
	})

	Describe("Parser.AddArgument()", func() {
		cmdLine := []string{"one", "two", "three", "four"}

		It("Should create argument rule first", func() {
			parser := args.NewParser()
			parser.AddArgument("first").IsString()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("first"))
			Expect(rule.Order).To(Equal(1))
		})
		It("Should match first argument 'one'", func() {
			parser := args.NewParser()
			parser.AddArgument("first").IsString()
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.String("first")).To(Equal("one"))
		})
		It("Should match first argument in order of declaration", func() {
			parser := args.NewParser()
			parser.AddArgument("first").IsString()
			parser.AddArgument("second").IsString()
			parser.AddArgument("third").IsString()
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.String("first")).To(Equal("one"))
			Expect(opt.String("second")).To(Equal("two"))
			Expect(opt.String("third")).To(Equal("three"))
		})
		It("Should handle no arguments if declared", func() {
			parser := args.NewParser()
			parser.AddArgument("first").IsString()
			parser.AddArgument("second").IsString()
			parser.AddArgument("third").IsString()

			cmdLine := []string{"one", "two"}
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.String("first")).To(Equal("one"))
			Expect(opt.String("second")).To(Equal("two"))
			Expect(opt.String("third")).To(Equal(""))
		})
		It("Should mixing optionals and arguments", func() {
			parser := args.NewParser()
			parser.AddFlag("--verbose").IsTrue()
			parser.AddFlag("--first").IsString()
			parser.AddArgument("second").IsString()
			parser.AddArgument("third").IsString()

			cmdLine := []string{"--first", "one", "two", "--verbose"}
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.String("first")).To(Equal("one"))
			Expect(opt.String("second")).To(Equal("two"))
			Expect(opt.String("third")).To(Equal(""))
			Expect(opt.Bool("verbose")).To(Equal(true))
		})
		It("Should raise an error if an optional and an argument share the same name", func() {
			parser := args.NewParser()
			parser.AddFlag("--first").IsString()
			parser.AddArgument("first").IsString()

			cmdLine := []string{"--first", "one", "one"}
			_, err := parser.Parse(cmdLine)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(Equal("Duplicate argument or flag 'first' defined"))
		})
		It("Should raise if options and configs share the same name", func() {
			parser := args.NewParser()
			parser.AddFlag("--debug").IsTrue()
			parser.AddConfig("debug").IsBool()

			_, err := parser.Parse(nil)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(Equal("Duplicate argument or flag 'debug' defined"))
		})
		It("Should raise an error if a argument is required but not provided", func() {
			parser := args.NewParser()
			parser.AddArgument("first").Required()
			parser.AddArgument("second").Required()

			cmdLine := []string{"one"}
			_, err := parser.Parse(cmdLine)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(Equal("argument 'second' is required"))
		})
		It("Should raise an error if a slice argument is followed by another argument", func() {
			parser := args.NewParser()
			parser.AddArgument("first").IsStringSlice()
			parser.AddArgument("second")

			cmdLine := []string{"one"}
			_, err := parser.Parse(cmdLine)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(Equal("'second' is ambiguous when " +
				"following greedy argument 'first'"))
		})
		It("Should raise an error if a slice argument is followed by another slice argument", func() {
			parser := args.NewParser()
			parser.AddArgument("first").IsStringSlice()
			parser.AddArgument("second").IsStringSlice()

			cmdLine := []string{"one"}
			_, err := parser.Parse(cmdLine)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(Equal("'second' is ambiguous when " +
				"following greedy argument 'first'"))
		})
		It("Should raise an error if the name contains invalid characters", func() {
			parser := args.NewParser()
			parser.AddArgument("*thing").IsStringSlice()
			_, err := parser.Parse(nil)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(Equal("Bad argument or flag '*thing'; contains invalid characters"))
		})
	})
	Describe("Parser.AddConfig()", func() {
		cmdLine := []string{"--power-level", "--power-level"}
		It("Should add new config only rule", func() {
			parser := args.NewParser()
			parser.AddConfig("power-level").Count().Help("My help message")

			// Should ignore command line options
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(0))

			// But Apply() a config file
			options := parser.NewOptionsFromMap(
				map[string]interface{}{
					"power-level": 3,
				})
			newOpt, _ := parser.Apply(options)
			// The old config still has the original non config applied version
			Expect(opt.Int("power-level")).To(Equal(0))
			// The new config has the value applied
			Expect(newOpt.Int("power-level")).To(Equal(3))
		})
	})
	Describe("Parser.InGroup()", func() {
		cmdLine := []string{"--power-level", "--hostname", "mysql.com"}
		It("Should add a new group", func() {
			parser := args.NewParser()
			parser.AddFlag("--power-level").Count()
			parser.InGroup("database").AddFlag("--hostname")
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.Int("power-level")).To(Equal(1))
			Expect(opt.Group("database").String("hostname")).To(Equal("mysql.com"))
		})
	})
	Describe("Parser.GenerateOptHelp()", func() {
		It("Should generate help messages given a set of rules", func() {
			parser := args.NewParser().WordWrap(80)
			parser.AddFlag("--power-level").Alias("-p").Help("Specify our power level")
			parser.AddFlag("--cat-level").
				Alias("-c").
				Help(`Lorem ipsum dolor sit amet, consectetur
			mollit anim id est laborum.`)
			msg := parser.GenerateHelp()
			Expect(msg).To(ContainSubstring("  -p, --power-level   Specify our power level" +
				"\n  -c, --cat-level     Lorem ipsum dolor sit amet, consecteturmollit anim id est" +
				"\n                      laborum.\n"))
		})
	})
	Describe("Parser.Epilog()", func() {
		It("Should generate help message with an epilog at the end", func() {
			parser := args.NewParser().Epilog("Now with more magical cow powers")
			parser.AddFlag("--power-level").Alias("-p").Help("Specify our power level")
			msg := parser.GenerateHelp()
			Expect(msg).To(ContainSubstring("Usage:  [OPTIONS]"))
			Expect(msg).To(ContainSubstring("Now with more magical cow powers"))
			Expect(msg).To(ContainSubstring("-p, --power-level   Specify our power level"))
		})
	})
	Describe("Parser.Usage()", func() {
		It("Should generate help message with custom useage", func() {
			parser := args.NewParser().Usage("-asdfbsx [ARGUMENT] -- [CONTAINER ARGS]...")
			parser.AddFlag("--power-level").Alias("-p").Help("Specify our power level")
			msg := parser.GenerateHelp()
			Expect(msg).To(ContainSubstring("Usage: -asdfbsx [ARGUMENT] -- [CONTAINER ARGS]..."))
			Expect(msg).To(ContainSubstring("-p, --power-level   Specify our power level"))
		})
	})
	Describe("Parser.GenerateHelp()", func() {
		It("Should generate help messages given a set of rules", func() {
			parser := args.NewParser().
				Name("dragon-ball").
				Desc("Small Description").
				EnvPrefix("APP_").
				WordWrap(80)

			parser.AddFlag("--environ").Default("1").Alias("-e").Env("ENV").Help("Default thing")
			parser.AddFlag("--default").Default("0").Alias("-d").Help("Default thing")
			parser.AddFlag("--power-level").Alias("-p").Help("Specify our power level")
			parser.AddFlag("--cat-level").Alias("-c").Help(`Lorem ipsum dolor sit amet, consectetur
				adipiscing elit, sed do eiusmod tempor incididunt ut labore et
				mollit anim id est laborum.`)
			msg := parser.GenerateHelp()
			Expect(msg).To(ContainSubstring("Usage: dragon-ball [OPTIONS]"))
			Expect(msg).To(ContainSubstring("Small Description"))
			Expect(msg).To(ContainSubstring("-e, --environ       Default thing (Default=1, Env=APP_ENV)"))
			Expect(msg).To(ContainSubstring("-d, --default       Default thing (Default=0)"))
			Expect(msg).To(ContainSubstring("-p, --power-level   Specify our power level"))
		})
		It("Should generate formated description if flag is set", func() {
			desc := `
			Custom formated description ----------------------------------------------------------- over 80

			With lots of new lines
			`
			parser := args.NewParser().
				Desc(desc, args.IsFormatted).
				Name("dragon-ball").
				WordWrap(80)

			parser.AddFlag("--environ").Default("1").Alias("-e").Help("Default thing")
			msg := parser.GenerateHelp()
			Expect(msg).To(ContainSubstring("Custom formated description --------------------" +
				"--------------------------------------- over 80"))
		})
	})
	Describe("Parser.AddCommand()", func() {
		It("Should run a command if seen on the command line", func() {
			parser := args.NewParser()
			called := false
			parser.AddCommand("command1", func(parent *args.Parser, data interface{}) (int, error) {
				called = true
				return 0, nil
			})
			cmdLine := []string{"command1"}
			retCode, err := parser.ParseAndRun(cmdLine, nil)
			Expect(err).To(BeNil())
			Expect(retCode).To(Equal(0))
			Expect(called).To(Equal(true))
		})
		It("Should not confuse a command with a following argument", func() {
			parser := args.NewParser()
			called := 0
			parser.AddCommand("set", func(parent *args.Parser, data interface{}) (int, error) {
				called++
				return 0, nil
			})
			cmdLine := []string{"set", "set"}
			retCode, err := parser.ParseAndRun(cmdLine, nil)
			Expect(err).To(BeNil())
			Expect(retCode).To(Equal(0))
			Expect(called).To(Equal(1))
		})
		It("Should provide a sub parser with that will not confuse a following argument", func() {
			parser := args.NewParser()
			called := 0
			parser.AddCommand("set", func(parent *args.Parser, data interface{}) (int, error) {
				parent.AddArgument("first").Required()
				parent.AddArgument("second").Required()
				opts, err := parent.Parse(nil)
				Expect(err).To(BeNil())
				Expect(opts.String("first")).To(Equal("foo"))
				Expect(opts.String("second")).To(Equal("bar"))

				called++
				return 0, nil
			})
			cmdLine := []string{"set", "foo", "bar"}
			retCode, err := parser.ParseAndRun(cmdLine, nil)
			Expect(err).To(BeNil())
			Expect(retCode).To(Equal(0))
			Expect(called).To(Equal(1))
		})
		It("Should allow sub commands to be a thing", func() {
			parser := args.NewParser()
			called := 0
			parser.AddCommand("volume", func(parent *args.Parser, data interface{}) (int, error) {
				parent.AddCommand("create", func(subParent *args.Parser, data interface{}) (int, error) {
					subParent.AddArgument("volume-name").Required()
					opts, err := subParent.Parse(nil)
					Expect(err).To(BeNil())
					Expect(opts.String("volume-name")).To(Equal("my-new-volume"))

					called++
					return 0, nil
				})
				retCode, err := parent.ParseAndRun(nil, nil)
				Expect(err).To(BeNil())
				Expect(retCode).To(Equal(0))
				return retCode, nil
			})
			cmdLine := []string{"volume", "create", "my-new-volume"}
			retCode, err := parser.ParseAndRun(cmdLine, nil)
			Expect(err).To(BeNil())
			Expect(retCode).To(Equal(0))
			Expect(called).To(Equal(1))
		})
		It("Should respect auto added help option in commands", func() {
			parser := args.NewParser()
			// Capture the help message via Pipe()
			_, ioWriter, _ := os.Pipe()
			parser.SetHelpIO(ioWriter)

			called := 0
			parser.AddCommand("set", func(parent *args.Parser, data interface{}) (int, error) {
				parent.AddArgument("first").Required()
				opts, err := parent.Parse(nil)
				Expect(opts.Bool("help")).To(Equal(true))

				// Should be a parse validation error (missing required argument)
				Expect(err).To(Not(BeNil()))

				called++
				return 0, nil
			})
			cmdLine := []string{"set", "-h"}
			retCode, err := parser.ParseAndRun(cmdLine, nil)
			Expect(err).To(BeNil())
			Expect(retCode).To(Equal(0))
			Expect(called).To(Equal(1))
		})
		It("Root parser should ignore help option if a sub command was provided", func() {
			// ignoring help gives a sub command a chance to provide help

			parser := args.NewParser()
			// Capture the help message via Pipe()
			_, ioWriter, _ := os.Pipe()
			parser.SetHelpIO(ioWriter)

			called := 0
			parser.AddCommand("set", func(parent *args.Parser, data interface{}) (int, error) {
				parent.AddArgument("first").Required()
				opts, err := parent.Parse(nil)
				Expect(opts.Bool("help")).To(Equal(true))
				Expect(err).To(Not(BeNil()))
				Expect(err.Error()).To(Equal("argument 'first' is required"))

				called++
				return 0, nil
			})

			// Parse at this point will indicate `--help` was asked for and
			// a sub command was specified
			cmdLine := []string{"set", "-h"}
			opt := parser.ParseSimple(cmdLine)

			Expect(opt).To(Not(BeNil()))
			Expect(opt.Bool("help")).To(Equal(true))
			Expect(opt.SubCommands()).To(Equal([]string{"set"}))

			// Since opt.SubCommands() indicates a subcommand was specified we should pass control to
			// the subcommand, Thus allowing the sub command `set` to provide help
			retCode, err := parser.RunCommand(nil)
			Expect(err).To(BeNil())
			Expect(retCode).To(Equal(0))
			Expect(called).To(Equal(1))
		})
	})
	Describe("Parser.GetArgs()", func() {
		It("Should return all un-matched arguments and options", func() {
			parser := args.NewParser()
			parser.AddArgument("image")
			parser.AddFlag("-output").Alias("-o").Required()
			parser.AddFlag("-runtime").Default("docker")

			cmdLine := []string{"golang:1.6", "build", "-o",
				"amd64-my-prog", "-installsuffix", "static", "./..."}
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.String("output")).To(Equal("amd64-my-prog"))
			Expect(opt.String("image")).To(Equal("golang:1.6"))
			Expect(parser.GetArgs()).To(Equal([]string{"build", "-installsuffix", "static", "./..."}))
		})
		It("Should return all empty if all arguments and options matched", func() {
			parser := args.NewParser()
			parser.AddFlag("--list").IsStringSlice()

			cmdLine := []string{"--list", "bee", "--list", "cat", "--list", "dad"}
			opt, err := parser.Parse(cmdLine)
			Expect(err).To(BeNil())
			Expect(opt.StringSlice("list")).To(Equal([]string{"bee", "cat", "dad"}))
			Expect(parser.GetArgs()).To(Equal([]string{}))
		})
	})
})
