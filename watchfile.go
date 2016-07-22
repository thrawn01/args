package args

import (
	"time"

	"sync"

	"gopkg.in/fsnotify.v1"
)

func WatchFile(path string, interval time.Duration, callBack func()) (WatchCancelFunc, error) {
	var isRunning sync.WaitGroup
	fsWatch, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	fsWatch.Add(path)

	// Check for write events at this interval
	tick := time.Tick(interval)
	done := make(chan struct{}, 1)
	once := sync.Once{}

	isRunning.Add(1)
	go func() {
		var lastWriteEvent *fsnotify.Event
		for {
			once.Do(func() { isRunning.Done() }) // Notify we are watching
			select {
			case event := <-fsWatch.Events:
				// If it was a write event
				if event.Op&fsnotify.Write == fsnotify.Write {
					lastWriteEvent = &event
				}
				// If we see a Remove event, This is probably ConfigMap updating the config symlink
				if event.Op&fsnotify.Remove == fsnotify.Remove {
					// Since the symlink was removed, we must
					// re-register the file to be watched
					fsWatch.Remove(event.Name)
					fsWatch.Add(event.Name)
					lastWriteEvent = &event
				}
			case <-tick:
				// No events during this interval
				if lastWriteEvent == nil {
					continue
				}
				// Execute the callback
				callBack()
				// Reset the last event
				lastWriteEvent = nil
			case <-done:
				close(done)
				return
			}
		}
	}()

	// Wait until the go-routine is running before we return, this ensures we
	// pickup any file changes after we leave this function
	isRunning.Wait()

	return func() {
		done <- struct{}{}
		fsWatch.Close()
	}, err
}
