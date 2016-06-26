package args

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	etcd "github.com/coreos/etcd/clientv3"
	"golang.org/x/net/context"
)

const MAX_BACKOFF_WAIT = 2 * time.Second

// Which options to pass to etcd client depends on the rule type
func (self *ArgParser) chooseOption(rule *Rule) etcd.OpOption {
	if rule.IsConfigGroup {
		return etcd.WithPrefix()
	}
	return func(op *etcd.Op) {}
}

func (self *ArgParser) ParseEtcd(client *etcd.Client) (*Options, error) {
	values := self.NewOptions()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer func() { cancel() }()

	for _, rule := range self.rules {
		resp, err := client.Get(ctx, rule.EtcdPath, self.chooseOption(rule))
		if err != nil {
			if self.log != nil {
				self.log.Printf("args.ParseEtcd(): Failed to fetch key '%s' - '%s'",
					rule.EtcdPath, err.Error())
			}
			continue
		}
		// Does this mean it wasn't found?
		if len(resp.Kvs) == 0 {
			self.log.Printf("args.ParseEtcd(): key '%s' not found", rule.EtcdPath)
			continue
		}
		if rule.IsConfigGroup {
			// Iterate through all the key=values for this group
			for _, node := range resp.Kvs {
				values.Group(rule.Group).Set(path.Base(string(node.Key)), string(node.Value))
			}
		} else if len(resp.Kvs) == 1 {
			values.Group(rule.Group).Set(rule.Name, string(resp.Kvs[0].Value))
		} else {
			values.Group(rule.Group).Set(rule.Name, string(resp.Kvs[0].Value))
			self.log.Printf("args.ParseEtcd(): Expected 1 Key=Value response but got multiple for key '%s'",
				rule.EtcdPath)
		}
	}
	return values, nil
}

// Generate rule.EtcdPath for all rules using the parsers set EtcRoot
func (self *ArgParser) generateEtcdPathKeys() {
	for _, rule := range self.rules {
		if self.EtcdRoot == "" {
			if self.Name == "" {
				self.EtcdRoot = "please-set-a-name"
			} else {
				self.EtcdRoot = self.Name
			}
		}
		// Do this so users are not surprised self.EtcdRoot isn't prefixed with "/"
		self.EtcdRoot = "/" + strings.TrimPrefix(self.EtcdRoot, "/")
		// Build the full etcd key path
		rule.EtcdPath = rule.EtcdKeyPath(self.EtcdRoot)
	}
}

func (self *ArgParser) FromEtcd(client *etcd.Client) (*Options, error) {
	self.generateEtcdPathKeys()

	options, err := self.ParseEtcd(client)
	if err != nil {
		return options, err
	}
	// Apply the etcd values to the commandline and environment variables
	return self.Apply(options)
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

type WatchCancelFunc func()

func (self *ArgParser) WatchEtcd(client *etcd.Client, callBack func(*ChangeEvent)) WatchCancelFunc {
	var isRunning sync.WaitGroup
	done := make(chan struct{})

	self.generateEtcdPathKeys()

	isRunning.Add(1)
	go func() {
		var resp etcd.WatchResponse
		var ok bool
		for {
			// Always attempt to watch, until the user tells us to stop
			ctx, cancel := context.WithCancel(context.Background())
			watchChan := client.Watch(ctx, self.EtcdRoot, etcd.WithPrefix())
			isRunning.Done() // Notify we are watching
			for {
				select {
				case resp, ok = <-watchChan:
					if !ok {
						goto Retry
					}
					if resp.Canceled {
						msg := fmt.Sprintf("args.WatchEtcd(): Etcd Cancelled watch with '%s'", resp.Err())
						self.log.Printf(msg)
						callBack(&ChangeEvent{Err: errors.New(msg)})
					}
					for _, event := range resp.Events {
						callBack(NewChangeEvent(self.rules, event, nil))
					}
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

// A ChangeEvent is a representation of an etcd key=value update, delete or expire. Args attempts to match
// a rule to the etcd change and includes the matched rule in the ChangeEvent. If args is unable to match
// a with this change, then ChangeEvent.Rule will be nil
type ChangeEvent struct {
	Rule    *Rule
	Group   string
	Key     string
	Value   string
	Deleted bool
	Err     error
}

func findEtcdRule(etcdPath string, rules Rules) *Rule {
	for _, rule := range rules {
		if etcdPath == rule.EtcdPath {
			return rule
		}
	}
	return nil
}

// Given args.Rules and etcd.Response, attempt to match the response to the rules and return
// a new ChangeEvent.
func NewChangeEvent(rules Rules, event *etcd.Event, err error) *ChangeEvent {
	rule := findEtcdRule(path.Dir(string(event.Kv.Key)), rules)
	return &ChangeEvent{
		Rule:    rule,
		Group:   rule.Group,
		Key:     path.Base(string(event.Kv.Key)),
		Value:   string(event.Kv.Value),
		Deleted: event.Type.String() == "DELETE",
		Err:     nil,
	}
}
