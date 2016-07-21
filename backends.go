package args

import (
	"path"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"golang.org/x/net/context"
)

const MAX_BACKOFF_WAIT = 2 * time.Second

type ChangeEvent interface {
	Err() error
	Key() string
	Group() string
	Value() []byte
	Deleted() bool
	Rule() *Rule
	SetRule(*Rule)
}

// A ChangeEvent is a representation of an key=value update, delete or expire. Args attempts to match
// a rule to the change and includes the matched rule in the ChangeEvent. If args is unable to match
// a with this change, then ChangeEvent.Rule will be nil
type Event struct {
	Rule    *Rule
	Group   string
	Key     string
	Value   []byte
	Deleted bool
	Error   error
}

type Pair struct {
	Key   string
	Value []byte
}

func NewPair(key string, value []byte) *Pair {
	return &Pair{
		Key:   key,
		Value: value,
	}
}

type WatchCancelFunc func()

type Backend interface {
	// Get retrieves a value from a K/V store for the provided key.
	Get(ctx context.Context, key string) (Pair, error)

	// List retrieves all keys and values under a provided key.
	List(ctx context.Context, key string) ([]Pair, error)

	// Set the provided key to value.
	Set(ctx context.Context, key string, value []byte) error

	// Watch monitors store for changes to key.
	Watch(ctx context.Context, key string) <-chan ChangeEvent

	// Return the root key used to store all other keys in the backend
	GetRootKey() string
}

func (self *ArgParser) FromStore(backend Backend) (*Options, error) {

	// Build the rules keys
	self.buildBackendKeys(backend.GetRootKey())

	options, err := self.ParseStore(backend)
	if err != nil {
		return options, err
	}
	// Apply the etcd values to the commandline and environment variables
	return self.Apply(options)
}

func (self *ArgParser) ParseStore(backend Backend) (*Options, error) {
	values := self.NewOptions()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer func() { cancel() }()
	//
	for _, rule := range self.rules {
		if rule.HasFlags(IsConfigGroup) {
			pairs, err := backend.List(ctx, rule.EtcdPath)
			if err != nil {
				self.error("args.FromStore().List() fetching '%s' - '%s'", rule.EtcdPath, err.Error())
				continue
			}
			// Iterate through all the key=values for this group
			for _, pair := range pairs {
				values.Group(rule.Group).Set(path.Base(string(pair.Key)), string(pair.Value))
			}
			continue
		}
		pair, err := backend.Get(ctx, rule.EtcdPath)
		if err != nil {
			self.error("args.ParseEtcd(): Failed to fetch key '%s' - '%s'", rule.EtcdPath, err.Error())
			continue
		}
		values.Group(rule.Group).Set(rule.Name, string(pair.Value))
	}
	return values, nil
}

func (self *ArgParser) matchBackendRule(key string) *Rule {
	for _, rule := range self.rules {
		comparePath := key
		if rule.HasFlags(IsConfigGroup) {
			comparePath = path.Dir(key)
		}
		if comparePath == rule.EtcdPath {
			return rule
		}
	}
	return nil
}

// Generate rule.EtcdPath for all rules using the parsers set EtcRoot
func (self *ArgParser) buildBackendKeys(root string) {
	for _, rule := range self.rules {
		// Do this so users are not surprised root isn't prefixed with "/"
		root = "/" + strings.TrimPrefix(root, "/")
		// Build the full etcd key path
		rule.EtcdPath = rule.EtcdKeyPath(root)
	}
}

func (self *ArgParser) Watch(backend Backend, callBack func(ChangeEvent, error)) WatchCancelFunc {
	var isRunning sync.WaitGroup
	done := make(chan struct{})

	// Build the rules keys
	self.buildBackendKeys(backend.GetRootKey())

	isRunning.Add(1)
	go func() {
		var event ChangeEvent
		var ok bool
		for {
			// Always attempt to watch, until the user tells us to stop
			ctx, cancel := context.WithCancel(context.Background())

			watchChan := backend.Watch(ctx, backend.GetRootKey())
			isRunning.Done() // Notify we are watching
			for {
				select {
				case event, ok = <-watchChan:
					if !ok {
						goto Retry
					}
					if event.Err() {
						callBack(nil, errors.Wrap(event.Err(), "ArgParser.Watch()"))
						goto Retry
					}

					rule := self.matchBackendRule(event.Key())
					if rule != nil {
						event.SetRule(rule)
					}
					callBack(event, nil)
				case <-done:
					cancel()
					return
				}
			}
		Retry:
			// Cancel our current context and sleep
			cancel()
			self.Sleep()
		}
	}()

	// Wait until the goroutine is running before we return, this ensures any updates
	// our application might make to etcd will be picked up by WatchEtcd()
	isRunning.Wait()
	return func() { close(done) }
}

func (self *ArgParser) Sleep() {
	self.attempts = self.attempts + 1
	delay := time.Duration(self.attempts) * 2 * time.Millisecond
	if delay > MAX_BACKOFF_WAIT {
		delay = MAX_BACKOFF_WAIT
	}
	self.log.Printf("WatchEtcd Retry in %v ...", delay)
	time.Sleep(delay)
}
