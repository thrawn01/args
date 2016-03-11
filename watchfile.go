package args

import (
	"time"

	"gopkg.in/fsnotify.v1"
)

type FileWatcher struct {
	fsNotify *fsnotify.Watcher
	interval time.Duration
	done     chan struct{}
	callback func()
}

func WatchFile(path string, interval time.Duration, callBack func()) (*FileWatcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	fsWatcher.Add(path)

	watcher := &FileWatcher{
		fsWatcher,
		interval,
		make(chan struct{}, 1),
		callBack,
	}
	go watcher.run()
	return watcher, err
}

func (self *FileWatcher) run() {
	// Check for write events at this interval
	tick := time.Tick(self.interval)

	var lastWriteEvent *fsnotify.Event
	for {
		select {
		case event := <-self.fsNotify.Events:
			// If it was a write event
			if event.Op == fsnotify.Write {
				lastWriteEvent = &event
			}
		case <-tick:
			// No events during this interval
			if lastWriteEvent == nil {
				continue
			}
			// Execute the callback
			self.callback()
			// Reset the last event
			lastWriteEvent = nil
		case <-self.done:
			goto Close
		}
	}
Close:
	close(self.done)
}

func (self *FileWatcher) Close() {
	self.done <- struct{}{}
	self.fsNotify.Close()
}
