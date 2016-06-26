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
	ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
	//defer cancel()

	// Set the value in the etcd store
	_, err := client.Put(ctx, path.Join(root, key), value)
	if err != nil {
		Fail(fmt.Sprintf("etcdPut() - %s", err.Error()))
	}
}

var _ = Describe("ArgParser", func() {
	FDescribe("FromEtcd()", func() {
		client := etcdClientFactory()
		//defer client.Close()
		etcdRoot := newRootPath()
		var log *TestLogger

		It("Should fetch 'bind' value from /EtcdRoot/bind", func() {
			okToTestEtcd()

			// Must mark etcd config and command line options with Etcd()
			parser := args.NewParser(args.EtcdPath(etcdRoot))
			log = NewTestLogger()
			parser.SetLog(log)
			parser.AddConfig("--bind").Etcd()

			etcdPut(client, parser.EtcdRoot, "/bind", "thrawn01.org:3366")
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

			etcdPut(client, parser.EtcdRoot, "/endpoints/endpoint1", "http://endpoint1.com:3366")
			etcdPut(client, parser.EtcdRoot, "/endpoints/endpoint2",
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
		client := etcdClientFactory()
		defer func() { client.Close() }()
		etcdRoot := newRootPath()
		var log *TestLogger

		It("Should watch /EtcdRoot/endpoints for new values", func() {
			okToTestEtcd()

			parser := args.NewParser(args.EtcdPath(etcdRoot))
			log = NewTestLogger()
			parser.SetLog(log)

			parser.AddConfigGroup("endpoints").Etcd()

			etcdPut(client, parser.EtcdRoot, "/endpoints/endpoint1", "http://endpoint1.com:3366")

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
			etcdPut(client, parser.EtcdRoot, "/endpoints/endpoint2", "http://endpoint2.com:3366")
			// Wait until our call back is called
			<-done
			// TODO: This should close a channel, update our watch to multi channel waiting machine =)
			fmt.Printf("Calling Cancel\n")
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
