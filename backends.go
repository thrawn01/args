package args

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const maxBackOffWait = 2 * time.Second

// A ChangeEvent is a representation of an key=value update, delete or expire. Args attempts to match
// a rule to the change and includes the matched rule in the ChangeEvent. If args is unable to match
// a with this change, then ChangeEvent.Rule will be nil
type ChangeEvent struct {
	Key     Key
	Value   string
	Deleted bool
	Err     error
	Rule    *Rule
}

// Represents the key portion of the key value `Pair` which
// includes the `Group` the key is contained in.
type Key struct {
	Group string
	Name  string
}

// Join the group and the key with the provided separator to form a string key
//	key := Key{Group: "group", Key: "key"}
//	key.Join("/") // Returns "group/key"
func (s Key) Join(sep string) string {
	if s.Name == "" {
		return s.Group
	}
	if s.Group == "" {
		return s.Name
	}
	return fmt.Sprintf("%s%s%s", s.Group, sep, s.Name)
}

// Represents a key value pair for use in `Backend` implementations
type Pair struct {
	Key   Key
	Value string
}

type WatchCancelFunc func()

// Struct's that implement this interface can be passed to the parser via `parser.FromBackend()`
// this allows users to source values for arguments from sources other than `os.Args`.
// See `parser.FromBackend()` for example
type Backend interface {
	// Get retrieves a value from a K/V store for the provided key.
	Get(ctx context.Context, key Key) (Pair, error)

	// List retrieves all keys and values that match the `Key.Group` given.
	List(ctx context.Context, key Key) ([]Pair, error)

	// Set the provided key to value.
	Set(ctx context.Context, key Key, value string) error

	// Watch monitors store for changes to key.
	Watch(ctx context.Context, root string) (<-chan ChangeEvent, error)

	// Return the root key used to store all other keys in the backend
	GetRootKey() string

	// Closes the connection to the backend and cancels all watches
	Close()
}

func (p *Parser) FromBackend(backend Backend) (*Options, error) {
	options, err := p.ParseBackend(backend)
	if err != nil {
		return options, err
	}
	// Apply the backend values to the commandline and environment variables
	return p.Apply(options)
}

func (p *Parser) ParseBackend(backend Backend) (*Options, error) {
	values := p.NewOptions()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer func() { cancel() }()
	//
	for _, rule := range p.rules {
		key := rule.BackendKey()
		if rule.HasFlag(IsConfigGroup) {
			pairs, err := backend.List(ctx, key)
			if err != nil {
				p.info("args.ParseBackend(): Failed to list '%s' - '%s'", key.Group, err.Error())
				continue
			}
			// Iterate through all the key=values pairs for this group
			for _, pair := range pairs {
				values.Group(pair.Key.Group).Set(pair.Key.Name, pair.Value)
			}
			continue
		}
		pair, err := backend.Get(ctx, key)
		if err != nil {
			// This can be a normal occurrence, and probably shouldn't be logged
			//p.info("args.ParseBackend(): Failed to fetch key '%s' - %s", key.Name, err.Error())
			continue
		}
		values.Group(pair.Key.Group).Set(pair.Key.Name, pair.Value)
	}
	return values, nil
}

func (p *Parser) Watch(backend Backend, callBack func(ChangeEvent, error)) WatchCancelFunc {
	var isRunning sync.WaitGroup
	var once sync.Once
	done := make(chan struct{})

	isRunning.Add(1)
	go func() {
		for {
			// Always attempt to watch, until the user tells us to stop
			ctx, cancel := context.WithCancel(context.Background())
			watchChan, err := backend.Watch(ctx, backend.GetRootKey())
			once.Do(func() { isRunning.Done() }) // Notify we are watching
			if err != nil {
				callBack(ChangeEvent{}, err)
				goto Retry
			}
			for {
				select {
				case event, ok := <-watchChan:
					if !ok {
						goto Retry
					}

					if event.Err != nil {
						callBack(ChangeEvent{}, errors.Wrap(event.Err, "backends.Watch()"))
						goto Retry
					}

					// find the rule this key is for
					rule := p.findRule(event.Key)
					if rule != nil {
						event.Rule = rule
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
			p.sleep()
		}
	}()

	// Wait until the go-routine is running before we return, this ensures any updates
	// our application might need from the backend picked up by Watch()
	isRunning.Wait()
	return func() {
		if done != nil {
			close(done)
		}
	}
}

func (p *Parser) findRule(key Key) *Rule {
	for _, rule := range p.rules {
		if rule.HasFlag(IsConfigGroup) {
			if rule.Group == key.Group {
				return rule
			}
		} else {
			if rule.Group == key.Group && rule.Name == key.Name {
				return rule
			}
		}
	}
	return nil
}

func (p *Parser) sleep() {
	p.attempts = p.attempts + 1
	delay := time.Duration(p.attempts) * 2 * time.Millisecond
	if delay > maxBackOffWait {
		delay = maxBackOffWait
	}
	p.log.Printf("Backend Retry in %v ...", delay)
	time.Sleep(delay)
}
