package args

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type SourceFlag int64

const (
	FromArgv SourceFlag = 1 << iota
	FromDefault
	FromMap
	FromEnv
)

var StoreTimeout = time.Second * 5

// The interface used to interact with all data stores
type Store interface {
	// Get retrieves a value from the store.
	Get(ctx context.Context, key Key) (Value, error)

	// List retrieves all keys and values under a provided key.
	List(ctx context.Context, key Key) ([]Value, error)

	// Set the provided value to the key.
	Set(ctx context.Context, key Key, value Value) error

	// Monitors store for changes to key, and provides a ChangeEvent when modifications are made
	Watch(ctx context.Context, root string) (<-chan ChangeEvent, error)

	// Closes any connections or open files and cancels all watches
	Close()

	// TODO: Add Apply() and Merge() to this interface
}

// The key used by value stores to retrieve and set values
type Key struct {
	Group string
	Name  string
}

func (s Key) String() string {
	if s.Group != "" && s.Name != "" {
		return strings.Join([]string{s.Group, s.Name}, ".")
	}
	if s.Group != "" {
		return s.Group
	}
	if s.Name != "" {
		return s.Name
	}
	return ""
}

// A ChangeEvent is a representation of an key=value update, delete or expire. Args attempts to match
// a rule to the change and includes the matched rule in the ChangeEvent. If args is unable to match
// a with this change, then ChangeEvent.Rule will be nil
type ChangeEvent struct {
	Key     Key
	Value   Value
	Deleted bool
	Err     error
	Rule    *Rule
}

// Create an empty StringStore
func NewStringStore() Store {
	return make(StringStore, 0)
}

// Implements the Store interface
type StringStore map[Key]Value

// Get retrieves a value from the store. Returns nil if key doesn't exist
func (s StringStore) Get(ctx context.Context, key Key) (Value, error) {
	value, ok := s[key]
	if ok {
		return value, nil
	}
	return StringValue{}, &NotFoundErr{fmt.Sprintf("No such key: %+v", key)}
}

// List retrieves all values listed under the provided key, Returns nil if key doesn't exist
func (s StringStore) List(ctx context.Context, key Key) ([]Value, error) {
	var results []Value
	for _, v := range s {
		results = append(results, v)
	}
	return results, nil
}

// Set the provided key and value in the store
func (s StringStore) Set(ctx context.Context, key Key, value Value) error {
	s[key] = value
	return nil
}

// Not Implemented for value store
func (s StringStore) Watch(ctx context.Context, root string) (<-chan ChangeEvent, error) {
	return nil, nil
}

// Return the root key used to store keys in the store
func (s StringStore) GetRootKey() string {
	return ""
}

// Does nothing
func (s StringStore) Close() {
}
