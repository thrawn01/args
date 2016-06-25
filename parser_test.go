package args_test

import (
	"bytes"
	"encoding/base32"
	"fmt"
	"os"
	"path"
	"time"

	"golang.org/x/net/context"

	etcd "github.com/coreos/etcd/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	"github.com/thrawn01/args"
)

func okToTestEtcd() {
	if os.Getenv("ARGS_DOCKER_HOST") == "" {
		Skip("ARGS_DOCKER_HOST not set, skipped....")
	}
}

func etcdTestPath() string {
	var buf bytes.Buffer
	encoder := base32.NewEncoder(base32.StdEncoding, &buf)
	encoder.Write(uuid.NewRandom())
	encoder.Close()
	buf.Truncate(26)
	return path.Join("/args-tests", buf.String())
}

func etcdClientFactory() etcd.Client {
	if os.Getenv("ARGS_DOCKER_HOST") == "" {
		return nil
	}

	etcdUrl := fmt.Sprintf("http://%s:2379", os.Getenv("ARGS_DOCKER_HOST"))

	client, err := etcd.New(etcd.Config{
		Endpoints: []string{etcdUrl},
		Transport: etcd.DefaultTransport,
		// set timeout per request to fail fast when the target endpoint is unavailable
		HeaderTimeoutPerRequest: time.Second,
	})
	if err != nil {
		Fail(fmt.Sprintf("etcdApiFactory() - %s", err.Error()))
	}
	return client
}

func etcdSet(client etcd.Client, root, key, value string) {
	api := etcd.NewKeysAPI(client)
	slug := path.Join(root, key)
	//fmt.Printf("Setting '%s' key with '%s' value\n", slug, value)
	_, err := api.Set(context.Background(), slug, value, nil)
	if err != nil {
		Fail(fmt.Sprintf("etcdSet() - %s", err.Error()))
	}
	//fmt.Printf("Set is done. Metadata is %q\n", resp)
}

var _ = Describe("ArgParser", func() {
	Describe("ArgParser.ParseArgs(nil)", func() {
		parser := args.NewParser()
		It("Should return error if AddOption() was never called", func() {
			_, err := parser.ParseArgs(nil)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("Must create some options to match with args.AddOption() before calling arg.ParseArgs()"))
		})
	})
	Describe("ArgParser.AddOption()", func() {
		cmdLine := []string{"--one", "-two", "++three", "+four", "--power-level"}

		It("Should create optional rule --one", func() {
			parser := args.NewParser()
			parser.AddOption("--one").Count()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.IsPos).To(Equal(0))
		})
		It("Should create optional rule ++one", func() {
			parser := args.NewParser()
			parser.AddOption("++one").Count()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.IsPos).To(Equal(0))
		})

		It("Should create optional rule -one", func() {
			parser := args.NewParser()
			parser.AddOption("-one").Count()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.IsPos).To(Equal(0))
		})

		It("Should create optional rule +one", func() {
			parser := args.NewParser()
			parser.AddOption("+one").Count()
			rule := parser.GetRules()[0]
			Expect(rule.Name).To(Equal("one"))
			Expect(rule.IsPos).To(Equal(0))
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
			msg := parser.GenerateOptHelp()
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

	Describe("FromEtcd()", func() {
		client := etcdClientFactory()
		etcdRoot := etcdTestPath()
		var log *TestLogger
		It("Should fetch 'bind' value from /EtcdRoot/bind", func() {
			okToTestEtcd()

			// Must mark etcd config and command line options with Etcd()
			parser := args.NewParser(args.EtcdPath(etcdRoot))
			log = NewTestLogger()
			parser.SetLog(log)
			parser.AddConfig("--bind").Etcd()

			etcdSet(client, parser.EtcdRoot, "/bind", "thrawn01.org:3366")
			opts, err := parser.FromEtcd(client)
			Expect(err).To(BeNil())
			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.String("bind")).To(Equal("thrawn01.org:3366"))
		})
		It("Should fetch 'endpoints' values from /EtcdRoot/endpoints", func() {
			okToTestEtcd()

			parser := args.NewParser(args.EtcdPath(etcdRoot))
			log = NewTestLogger()
			parser.SetLog(log)
			parser.AddConfigGroup("endpoints").Etcd()

			etcdSet(client, parser.EtcdRoot, "/endpoints/endpoint1", "http://endpoint1.com:3366")
			etcdSet(client, parser.EtcdRoot, "/endpoints/endpoint2",
				"{ \"host\": \"endpoint2\", \"port\": \"3366\" }")

			opts, err := parser.FromEtcd(client)
			Expect(err).To(BeNil())
			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.Group("endpoints").ToMap()).To(Equal(map[string]interface{}{
				"endpoint1": "http://endpoint1.com:3366",
				"endpoint2": "{ \"host\": \"endpoint2\", \"port\": \"3366\" }",
			}))
		})
	})
	FDescribe("WatchEtcd", func() {
		var log *TestLogger
		var client etcd.Client
		var etcdRoot string
		var parser *args.ArgParser

		BeforeEach(func() {
			log = NewTestLogger()
			etcdRoot = etcdTestPath()
			parser = args.NewParser(args.EtcdPath(etcdRoot))
			parser.SetLog(log)
			client = etcdClientFactory()
		})
		It("Should watch /EtcdRoot/endpoints for new values", func() {
			okToTestEtcd()

			parser.AddConfigGroup("endpoints").Etcd()

			etcdSet(client, parser.EtcdRoot, "/endpoints/endpoint1", "http://endpoint1.com:3366")

			_, err := parser.FromEtcd(client)
			// Get a pointer to the current options
			opts := parser.GetOpts()

			Expect(err).To(BeNil())
			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.Group("endpoints").ToMap()).To(Equal(map[string]interface{}{
				"endpoint1": "http://endpoint1.com:3366",
			}))
			done := make(chan struct{})

			// TODO: change this func to accept an Update{} object
			cancelWatch := parser.WatchEtcd(client, func(group, key, value string) {
				fmt.Printf("callback - %s - %s - %s\n", group, key, value)
				// This takes an update object, and updates the opts with the latest which might
				// be what most people want. others will have the power to update how and when they like
				//parser.Apply(opts.Update(update))

				if group == "endpoints" {
					parser.Apply(opts.Group(group).Set(key, value, false))
				}
				close(done)
			})
			// Add a new endpoint
			etcdSet(client, parser.EtcdRoot, "/endpoints/endpoint2", "http://endpoint2.com:3366")
			// Wait until our call back is called
			<-done
			// TODO: This should close a channel, update our watch to multi channel waiting machine =)
			cancelWatch()
			// Get the updated options
			opts = parser.GetOpts()

			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.Group("endpoints").ToMap()).To(Equal(map[string]interface{}{
				"endpoint1": "http://endpoint1.com:3366",
				"endpoint2": "http://endpoint2.com:3366",
			}))

		})

	})
})
