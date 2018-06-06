package args_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/thrawn01/args"
	"log"
)

var watchChan chan args.ChangeEvent

type TestBackend struct {
	keys  map[string]args.Pair
	lists map[string][]args.Pair
	close chan struct{}
}

func NewTestBackend() args.Backend {
	return &TestBackend{
		keys: map[string]args.Pair{
			"/root/bind": {Key: args.Key{Name: "bind"}, Value: "thrawn01.org:3366"}},
		lists: map[string][]args.Pair{
			"/root/endpoints": {
				{
					Key:   args.Key{Group: "endpoints", Name: "endpoint1"},
					Value: "http://endpoint1.com:3366",
				},
				{
					Key:   args.Key{Group: "endpoints", Name: "endpoint2"},
					Value: `{ "host": "endpoint2", "port": "3366" }`,
				},
			},
			"/root/watch": {
				{
					Key:   args.Key{Group: "watch", Name: "endpoint1"},
					Value: "http://endpoint1.com:3366",
				},
			},
		},
	}
}

func fullKey(key args.Key) string {
	return fmt.Sprintf("/root/%s", key.Join("/"))
}

func (tb *TestBackend) Get(ctx context.Context, key args.Key) (args.Pair, error) {
	pair, ok := tb.keys[fullKey(key)]
	if !ok {
		return args.Pair{}, errors.New(fmt.Sprintf("'%s' not found", fullKey(key)))
	}
	return pair, nil
}

func (tb *TestBackend) List(ctx context.Context, key args.Key) ([]args.Pair, error) {
	pairs, ok := tb.lists[fullKey(key)]
	if !ok {
		return []args.Pair{}, errors.New(fmt.Sprintf("'%s' not found", fullKey(key)))
	}
	return pairs, nil
}

func (tb *TestBackend) Set(ctx context.Context, key args.Key, value string) error {
	tb.keys[fullKey(key)] = args.Pair{Key: key, Value: value}
	return nil
}

// Watch monitors store for changes to key.
func (tb *TestBackend) Watch(ctx context.Context, key string) (<-chan args.ChangeEvent, error) {
	changeChan := make(chan args.ChangeEvent, 2)

	go func() {
		var event args.ChangeEvent
		select {
		case event = <-watchChan:
			changeChan <- event
		case <-tb.close:
			close(changeChan)
			return
		}
	}()
	return changeChan, nil
}

func (tb *TestBackend) Close() {
	if tb.close != nil {
		close(tb.close)
	}
}

func (tb *TestBackend) GetRootKey() string {
	return "/root"
}

func NewChangeEvent(key args.Key, value string) args.ChangeEvent {
	return args.ChangeEvent{
		Key:     key,
		Value:   value,
		Deleted: false,
		Err:     nil,
	}
}

var _ = Describe("backend", func() {
	var log *TestLogger
	var backend args.Backend

	BeforeEach(func() {
		backend = NewTestBackend()
		log = NewTestLogger()
		watchChan = make(chan args.ChangeEvent, 1)
	})

	AfterEach(func() {
		if backend != nil {
			backend.Close()
		}
	})

	Describe("args.FromBackend()", func() {
		It("Should fetch 'bind' value from backend", func() {
			parser := args.NewParser()
			parser.Log(log)
			parser.AddConfig("bind")

			opts, err := parser.FromBackend(backend)
			Expect(err).To(BeNil())
			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.String("bind")).To(Equal("thrawn01.org:3366"))
		})
		It("Should use List() when fetching Config Groups", func() {
			parser := args.NewParser()
			parser.Log(log)
			parser.AddConfigGroup("endpoints")

			opts, err := parser.FromBackend(backend)
			Expect(err).To(BeNil())
			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.Group("endpoints").ToMap()).To(Equal(map[string]interface{}{
				"endpoint1": "http://endpoint1.com:3366",
				"endpoint2": `{ "host": "endpoint2", "port": "3366" }`,
			}))
		})
		/*It("Should return an error if config option not found in the backend", func() {
			parser := args.NewParser()
			parser.Log(log)
			parser.AddConfig("--missing")

			opts, err := parser.FromBackend(backend)
			Expect(err).To(BeNil())
			Expect(log.GetEntry()).To(ContainSubstring("not found"))
			Expect(opts.String("missing")).To(Equal(""))
		})*/
		It("Should call Watch() to watch for new values", func() {
			parser := args.NewParser()
			parser.Log(log)
			parser.AddConfigGroup("watch")

			_, err := parser.FromBackend(backend)
			opts := parser.GetOpts()
			Expect(err).To(BeNil())
			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.Group("watch").ToMap()).To(Equal(map[string]interface{}{
				"endpoint1": "http://endpoint1.com:3366",
			}))

			done := make(chan struct{})

			cancelWatch := parser.Watch(backend, func(event args.ChangeEvent, err error) {
				// Always check for errors
				if err != nil {
					fmt.Printf("Watch Error - %s\n", err.Error())
					close(done)
					return
				}
				parser.Apply(opts.FromChangeEvent(event))
				// Tell the test to continue, Change event was handled
				close(done)
			})
			// Add a new endpoint
			watchChan <- NewChangeEvent(args.Key{Group: "watch", Name: "endpoint2"}, "http://endpoint2.com:3366")
			// Wait until the change event is handled
			<-done
			// Stop the watch
			cancelWatch()
			// Get the updated options
			opts = parser.GetOpts()

			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.Group("watch").ToMap()).To(Equal(map[string]interface{}{
				"endpoint1": "http://endpoint1.com:3366",
				"endpoint2": "http://endpoint2.com:3366",
			}))
		})
	})
})

func ExampleParser_FromBackend() {
	// Simple backend, usually an INI, YAML, ETCD, or CONSOL backend
	backend := args.NewHasMapBackend(map[string]string {
		"/root/foo": "bar",
		"/root/kit": "kat",
	}, "/root/")

	parser := args.NewParser()
	parser.AddFlag("foo")
	parser.AddFlag("kit")

	// Parse our command line args first
	_, err := parser.Parse([]string{"--foo", "bash"})
	if err != nil {
		log.Fatal(err)
	}

	// Now apply our backend values, any existing values from the
	// command line always take precedence
	opts, err := parser.FromBackend(backend)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("foo = %s\n", opts.String("foo"))
	fmt.Printf("kit = %s\n", opts.String("kit"))

	// Output:
	// foo = bash
	// kit = kat
}

