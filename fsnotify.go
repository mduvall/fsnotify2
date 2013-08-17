// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package fsnotify implements filesystem notification.
package fsnotify2

import "fmt"
import "os"
import "path/filepath"

const (
	CREATE = 1 << iota
	MODIFY
	DELETE
	RENAME
	FILE_WRITE

	ALL_FLAGS = MODIFY | DELETE | RENAME | CREATE | FILE_WRITE
)

// Purge events from interal chan to external chan if passes filter
func (w *Watcher) purgeEvents() {
	for ev := range w.internalEvent {
		sendEvent := false
		w.fsnmut.Lock()
		fsnFlags := w.fsnFlags[ev.Name]
		w.fsnmut.Unlock()

		if (fsnFlags&CREATE == CREATE) && ev.IsCreate() {
			sendEvent = true
		}

		if (fsnFlags&MODIFY == MODIFY) && ev.IsModify() {
			sendEvent = true
		}

		if (fsnFlags&DELETE == DELETE) && ev.IsDelete() {
			sendEvent = true
		}

		if (fsnFlags&RENAME == RENAME) && ev.IsRename() {
			sendEvent = true
		}

		if (fsnFlags&FILE_WRITE == FILE_WRITE) && ev.IsFileWrite() {
			sendEvent = true
		}

		if sendEvent {
			w.Event <- ev
		}

		// If there's no file, then no more events for user
		// BSD must keep watch for internal use (watches DELETEs to keep track
		// what files exist for create events)
		if ev.IsDelete() {
			w.fsnmut.Lock()
			delete(w.fsnFlags, ev.Name)
			w.fsnmut.Unlock()
		}
	}

	close(w.Event)
}

// Watch a given file path
func (w *Watcher) Watch(path string) error {
	w.fsnmut.Lock()
	w.fsnFlags[path] = ALL_FLAGS
	w.fsnmut.Unlock()
	return w.watchAllEvents(path)
}

// Watch all files in a given directory
func (w *Watcher) WatchAll(path string) error {
	watchFunc := func(path string, fi os.FileInfo, err error) error {
		return w.Watch(path)
	}

	return filepath.Walk(path, watchFunc)
}

// Watch all files in a given directory for a set of notifications
func (w *Watcher) WatchAllFlags(path string, flags uint32) error {
	w.fsnmut.Lock()
	w.fsnFlags[path] = flags
	w.fsnmut.Unlock()

	watchFunc := func(path string, fi os.FileInfo, err error) error {
		return w.Watch(path)
	}

	return filepath.Walk(path, watchFunc)
}

// Watch a given file path for a particular set of notifications (MODIFY etc.)
func (w *Watcher) WatchFlags(path string, flags uint32) error {
	w.fsnmut.Lock()
	w.fsnFlags[path] = flags
	w.fsnmut.Unlock()

	return w.watchAllEvents(path)
}

// Remove a watch on a file
func (w *Watcher) RemoveWatch(path string) error {
	w.fsnmut.Lock()
	delete(w.fsnFlags, path)
	w.fsnmut.Unlock()

	return w.removeWatch(path)
}

// String formats the event e in the form
// "filename: DELETE|MODIFY|..."
func (e *FileEvent) String() string {
	var events string = ""

	if e.IsCreate() {
		events += "|" + "CREATE"
	}

	if e.IsDelete() {
		events += "|" + "DELETE"
	}

	if e.IsModify() {
		events += "|" + "MODIFY"
	}

	if e.IsFileWrite() {
		events += "|" + "FILE_WRITE"
	}

	if e.IsRename() {
		events += "|" + "RENAME"
	}

	if len(events) > 0 {
		events = events[1:]
	}

	return fmt.Sprintf("%q: %s", e.Name, events)
}
