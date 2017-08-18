package ini

import (
	"context"
	"sync"
	"time"

	"github.com/go-ini/ini"
	"github.com/thrawn01/args"
	fsnotify "gopkg.in/fsnotify.v1"
)

var watchInterval = time.Second

func (s *Backend) Watch(ctx context.Context, path string) (<-chan args.ChangeEvent, error) {
	var isRunning sync.WaitGroup
	var err error

	s.fsWatch, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if err := s.fsWatch.Add(path); err != nil {
		return nil, err
	}

	// Check for write events at this interval
	changeChan := make(chan args.ChangeEvent, 2)
	tick := time.Tick(watchInterval)
	s.done = make(chan struct{}, 1)
	once := sync.Once{}

	isRunning.Add(1)
	go func() {
		var lastWriteEvent *fsnotify.Event
		var checkFile *fsnotify.Event
		for {
			once.Do(func() { isRunning.Done() }) // Notify we are watching
			select {
			case event := <-s.fsWatch.Events:
				//fmt.Printf("Event %s\n", event.String())
				// If it was a write event
				if event.Op&fsnotify.Write == fsnotify.Write {
					lastWriteEvent = &event
				}
				// VIM apparently renames a file before writing
				if event.Op&fsnotify.Rename == fsnotify.Rename {
					checkFile = &event
				}
				// If we see a Remove event, This is probably ConfigMap updating the config symlink
				if event.Op&fsnotify.Remove == fsnotify.Remove {
					checkFile = &event
				}
			case <-tick:
				// If the file was renamed or removed; maybe it re-appears after our duration?
				if checkFile != nil {
					// Since the file was removed, we must
					// re-register the file to be watched
					s.fsWatch.Remove(checkFile.Name)
					if err := s.fsWatch.Add(checkFile.Name); err != nil {
						// Nothing left to watch
						changeChan <- args.ChangeEvent{Err: err}
						return
					}
					lastWriteEvent = checkFile
					checkFile = nil
					continue
				}

				// No events during this interval
				if lastWriteEvent == nil {
					continue
				}

				cfg, err := loadINI(path)
				if err != nil {
					changeChan <- args.ChangeEvent{Err: err}
					continue
				}

				// Send along all the changes we discovered
				for _, change := range s.DiffINI(cfg) {
					changeChan <- change
				}

				// Reset the last event
				lastWriteEvent = nil

			case <-ctx.Done():
				return

			case <-s.done:
				return
			}
		}
	}()

	// Wait until the go-routine is running before we return, this ensures we
	// pickup any file changes after we leave this function
	isRunning.Wait()

	return changeChan, nil
}

func (s *Backend) Close() {
	if s.done != nil {
		close(s.done)
	}
	s.fsWatch.Close()
}

func loadINI(path string) (*ini.File, error) {
	// Load the ini file
	content, err := args.LoadFile(path)
	if err != nil {
		return nil, err
	}

	cfg, err := ini.Load(content)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// diff the ini file currently loaded with the ini.File provided
func (s *Backend) DiffINI(cfg *ini.File) []args.ChangeEvent {
	var results []args.ChangeEvent

	// Find new and changed items
	for _, section := range cfg.Sections() {
		// Translate the default section back to empty string
		group := section.Name()
		if group == ini.DEFAULT_SECTION {
			group = ""
		}

		for _, key := range section.Keys() {
			pair, err := s.Get(context.Background(), args.Key{Name: key.Name(), Group: group})
			if err != nil {
				// Generate a new value event
				results = append(results, args.ChangeEvent{
					Key:     args.Key{Name: key.Name(), Group: group},
					Value:   key.Value(),
					Deleted: false,
				})
				continue
			}
			if pair.Value != key.Value() {
				// Generate changed value event
				results = append(results, args.ChangeEvent{
					Key:     pair.Key,
					Value:   key.Value(),
					Deleted: false,
				})
			}
		}
	}

	// Find deleted items
	for _, section := range s.cfg.Sections() {
		for _, key := range section.Keys() {
			if _, err := cfg.Section(section.Name()).GetKey(key.Name()); err != nil {
				// Translate the default section back to empty string
				group := section.Name()
				if group == ini.DEFAULT_SECTION {
					group = ""
				}

				// Generate changed value event
				results = append(results, args.ChangeEvent{
					Key:     args.Key{Name: key.Name(), Group: group},
					Value:   key.Value(),
					Deleted: true,
				})
			}
		}
	}
	return results
}
