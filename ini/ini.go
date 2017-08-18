package ini

import (
	"context"

	"errors"

	"github.com/go-ini/ini"
	"github.com/thrawn01/args"
	"gopkg.in/fsnotify.v1"
)

// Key Value Pairs found in this section will be considered as if they had no section. You can overide this section name
// by including a list of sections in the FromINI() call.
const DefaultSection string = ""

type Backend struct {
	fsWatch  *fsnotify.Watcher
	done     chan struct{}
	cfg      *ini.File
	fileName string
}

func NewBackendFromFile(fileName string) (*Backend, error) {
	content, err := args.LoadFile(fileName)
	if err != nil {
		return nil, err
	}
	return NewBackend(content, fileName)
}

func NewBackend(input []byte, fileName string) (*Backend, error) {
	cfg, err := ini.Load(input)
	if err != nil {
		return nil, err
	}
	return &Backend{cfg: cfg, fileName: fileName}, nil
}

func (s *Backend) Get(ctx context.Context, key args.Key) (args.Pair, error) {
	group, err := s.cfg.GetSection(key.Group)
	if err != nil {
		return args.Pair{}, err
	}

	result, err := group.GetKey(key.Name)
	if err != nil {
		return args.Pair{}, err
	}

	return args.Pair{
		Key:   key,
		Value: result.Value(),
	}, nil
}

func (s *Backend) List(ctx context.Context, key args.Key) ([]args.Pair, error) {
	group, err := s.cfg.GetSection(key.Group)
	if err != nil {
		return []args.Pair{}, err
	}

	var results []args.Pair
	for _, item := range group.KeyStrings() {
		value, err := s.Get(ctx, args.Key{Name: item, Group: key.Group})
		if err != nil {
			// Shouldn't happen, but will return the error anyway
			return results, err
		}
		results = append(results, value)
	}
	return results, nil
}

func (s *Backend) Set(ctx context.Context, key args.Key, value string) error {
	return errors.New("Set() now allowed on ini files")
}

func (s *Backend) GetRootKey() string {
	return s.fileName
}
