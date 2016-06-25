package args_test

import (
	"bytes"
	"encoding/base32"
	"fmt"
	"os"
	"path"
	"time"

	etcd "github.com/coreos/etcd/client"
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
	Describe("WatchEtcd", func() {
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
			cancelWatch := parser.WatchEtcd(client, func(event *args.ChangeEvent) {
				fmt.Printf("callback - %s - %s - %s\n", event.Group, event.Key, event.Value)
				// This takes an update object, and updates the opts with the latest which might
				// be what most people want. others will have the power to update how and when they like
				//parser.Apply(opts.Update(update))

				if event.Group == "endpoints" {
					parser.Apply(opts.Group(event.Group).Set(event.Key, event.Value, false))
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
