package args

import (
	"fmt"
	"path"
	"sync"
	"time"

	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

func (self *ArgParser) ParseEtcd(api etcd.KeysAPI) (*Options, error) {
	values := self.NewOptions()
	for _, rule := range self.rules {
		if rule.Etcd {
			resp, err := api.Get(context.Background(), rule.EtcdPath, nil)
			if err != nil {
				if self.log != nil {
					self.log.Printf("args.ParseEtcd(): Failed to fetch key '%s' - '%s'",
						rule.EtcdPath, err.Error())
				}
				continue
			}
			// If it's a directory and this rule is a config group
			if resp.Node.Dir {
				if rule.IsConfigGroup {
					// Retrieve all the key=values in the directory
					for _, node := range resp.Node.Nodes {
						// That are not directories
						if node.Dir {
							continue
						}
						values.Group(rule.Group).Set(path.Base(node.Key), node.Value, false)
					}
				}
				// Should we silently fail if the rule is not for a
				// config group, but the node is a directory?
				continue
			}
			values.Group(rule.Group).Set(rule.Name, resp.Node.Value, false)
		}
	}
	return values, nil
}

func (self *ArgParser) generateEtcdPathKeys() {
	for _, rule := range self.rules {
		if rule.Etcd {
			rule.EtcdPath = rule.EtcdKeyPath(self.EtcdRoot)
		}
	}
}

func (self *ArgParser) FromEtcd(client etcd.Client) (*Options, error) {
	self.generateEtcdPathKeys()
	options, err := self.ParseEtcd(etcd.NewKeysAPI(client))
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

func (self *ArgParser) WatchEtcd(client etcd.Client, callBack func(*ChangeEvent)) context.CancelFunc {
	self.generateEtcdPathKeys()

	api := etcd.NewKeysAPI(client)
	watcher := api.Watcher(self.EtcdRoot, &etcd.WatcherOptions{
		AfterIndex: 0,
		Recursive:  true,
	})
	ctx, cancel := context.WithCancel(context.Background())

	var isRunning sync.WaitGroup
	isRunning.Add(1)
	go func() {
		// Notify we are running
		isRunning.Done()
		for {
			fmt.Printf("Watching....\n")
			event, err := watcher.Next(ctx)
			if err != nil {
				if err == context.Canceled {
					fmt.Printf("return\n")
					return
				}
				fmt.Printf("WatchEtcd: %s\n", err.Error())
				self.log.Printf("WatchEtcd: %s", err.Error())
				self.Sleep()
				continue
			}
			self.attempts = 0
			callBack(NewChangeEvent(self.rules, event))
		}
	}()
	// Wait until the goroutine is running before we return, this ensures any updates
	// our application might make to etcd will be picked up by WatchEtcd()
	isRunning.Wait()
	return cancel
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
func NewChangeEvent(rules Rules, event *etcd.Response) *ChangeEvent {
	rule := findEtcdRule(path.Dir(event.Node.Key), rules)
	deleted := (event.Action == "del" || event.Action == "expire")
	return &ChangeEvent{
		Rule:    rule,
		Group:   rule.Group,
		Key:     path.Base(event.Node.Key),
		Value:   event.Node.Value,
		Deleted: deleted,
	}
}
