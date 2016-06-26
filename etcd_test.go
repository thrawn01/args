package args_test

import (
	"bytes"
	"encoding/base32"
	"fmt"
	"os"
	"path"
	"time"

	etcd "github.com/coreos/etcd/clientv3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	"github.com/thrawn01/args"
	"golang.org/x/net/context"
)

func okToTestEtcd() {
	if os.Getenv("ARGS_DOCKER_HOST") == "" {
		Skip("ARGS_DOCKER_HOST not set, skipped....")
	}
}

func newRootPath() string {
	var buf bytes.Buffer
	encoder := base32.NewEncoder(base32.StdEncoding, &buf)
	encoder.Write(uuid.NewRandom())
	encoder.Close()
	buf.Truncate(26)
	return path.Join("/args-tests", buf.String())
}

func etcdClientFactory() *etcd.Client {
	if os.Getenv("ARGS_DOCKER_HOST") == "" {
		return nil
	}

	client, err := etcd.New(etcd.Config{
		Endpoints: []string{
			fmt.Sprintf("%s:2379", os.Getenv("ARGS_DOCKER_HOST")),
		},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		Fail(fmt.Sprintf("etcdApiFactory() - %s", err.Error()))
	}
	return client
}

func etcdPut(client *etcd.Client, root, key, value string) {
	// Context Timeout for 2 seconds
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	// Set the value in the etcd store
	_, err := client.Put(ctx, path.Join(root, key), value)
	if err != nil {
		Fail(fmt.Sprintf("etcdPut() - %s", err.Error()))
	}
}

var _ = Describe("ArgParser", func() {
	var client *etcd.Client
	var etcdRoot string
	var log *TestLogger

	BeforeEach(func() {
		client = etcdClientFactory()
		etcdRoot = newRootPath()
		log = NewTestLogger()
	})

	AfterEach(func() {
		if client != nil {
			client.Close()
		}
	})

	Describe("FromEtcd()", func() {
		It("Should default to /please-set-a-name if no args.Name() or args.EtcdPath() specified", func() {
			okToTestEtcd()

			parser := args.NewParser()
			parser.SetLog(log)
			parser.AddConfig("--bind")
			_, err := parser.FromEtcd(client)
			Expect(err).To(BeNil())
			Expect(parser.EtcdRoot).To(Equal("/please-set-a-name"))
		})
		It("Should use args.Name() if args.EtcdPath() not specified", func() {
			okToTestEtcd()

			parser := args.NewParser(args.Name("my-name"))
			parser.SetLog(log)
			parser.AddConfig("--bind")
			_, err := parser.FromEtcd(client)
			Expect(err).To(BeNil())
			Expect(parser.EtcdRoot).To(Equal("/my-name"))
		})
		It("Should fetch 'bind' value from /EtcdRoot/bind", func() {
			okToTestEtcd()

			parser := args.NewParser(args.EtcdPath(etcdRoot))
			parser.SetLog(log)
			parser.AddConfig("--bind")

			etcdPut(client, parser.EtcdRoot, "/DEFAULT/bind", "thrawn01.org:3366")
			opts, err := parser.FromEtcd(client)
			Expect(err).To(BeNil())
			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.String("bind")).To(Equal("thrawn01.org:3366"))
		})
		It("Should fetch 'endpoints' values from /EtcdRoot/endpoints", func() {
			okToTestEtcd()

			parser := args.NewParser(args.EtcdPath(etcdRoot))
			parser.SetLog(log)
			parser.AddConfigGroup("endpoints")

			etcdPut(client, parser.EtcdRoot, "/endpoints/endpoint1", "http://endpoint1.com:3366")

			opts, err := parser.FromEtcd(client)
			Expect(err).To(BeNil())
			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.Group("endpoints").ToMap()).To(Equal(map[string]interface{}{
				"endpoint1": "http://endpoint1.com:3366",
			}))

			etcdPut(client, parser.EtcdRoot, "/endpoints/endpoint2",
				"{ \"host\": \"endpoint2\", \"port\": \"3366\" }")

			opts, err = parser.FromEtcd(client)
			Expect(err).To(BeNil())
			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.Group("endpoints").ToMap()).To(Equal(map[string]interface{}{
				"endpoint1": "http://endpoint1.com:3366",
				"endpoint2": "{ \"host\": \"endpoint2\", \"port\": \"3366\" }",
			}))
		})
		It("Should be ok if config option not found in etcd store", func() {
			okToTestEtcd()

			parser := args.NewParser(args.EtcdPath(etcdRoot))
			parser.SetLog(log)
			parser.AddConfig("--bind")

			etcdPut(client, parser.EtcdRoot, "/not-found", "foo")
			opts, err := parser.FromEtcd(client)
			Expect(err).To(BeNil())
			Expect(log.GetEntry()).To(ContainSubstring("not found"))
			Expect(opts.String("bind")).To(Equal(""))
		})
	})
	Describe("WatchEtcd", func() {
		It("Should watch /EtcdRoot/endpoints for new values", func() {
			okToTestEtcd()

			parser := args.NewParser(args.EtcdPath(etcdRoot))
			parser.SetLog(log)
			parser.AddConfigGroup("endpoints")

			etcdPut(client, parser.EtcdRoot, "/endpoints/endpoint1", "http://endpoint1.com:3366")

			_, err := parser.FromEtcd(client)
			opts := parser.GetOpts()
			Expect(err).To(BeNil())
			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.Group("endpoints").ToMap()).To(Equal(map[string]interface{}{
				"endpoint1": "http://endpoint1.com:3366",
			}))

			done := make(chan struct{})

			// TODO: change this func to accept an Update{} object
			cancelWatch := parser.WatchEtcd(client, func(event *args.ChangeEvent) {
				// Always check for errors
				if event.Err != nil {
					fmt.Printf("Watch Error - %s\n", event.Err.Error())
					close(done)
					return
				}
				parser.Apply(opts.FromChangeEvent(event))
				// Tell the test to continue, Change event was handled
				close(done)
			})
			// Add a new endpoint
			etcdPut(client, parser.EtcdRoot, "/endpoints/endpoint2", "http://endpoint2.com:3366")
			// Wait until the change event is handled
			<-done
			// Stop the watch
			cancelWatch()
			// Get the updated options
			opts = parser.GetOpts()

			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.Group("endpoints").ToMap()).To(Equal(map[string]interface{}{
				"endpoint1": "http://endpoint1.com:3366",
				"endpoint2": "http://endpoint2.com:3366",
			}))

		})
		// TODO
		It("Should continue to attempt to reconnect if the etcd client disconnects", func() {})
		// TODO
		It("Should apply any change using opt.FromChangeEvent()", func() {})
	})
})
