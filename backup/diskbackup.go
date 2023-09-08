package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/honeycombio/refinery/types"
)

type DiskBackup struct {
	Dir       string
	events    []*types.Event
	lastSaved time.Time
}

func NewDiskBackup(dir string) *DiskBackup {
	db := &DiskBackup{Dir: dir, lastSaved: time.Now()}
	go db.PeriodicFlush()
	return db
}

func (d *DiskBackup) Save(event *types.Event) error {
	d.events = append(d.events, event)

	if len(d.events) >= MaxBufferSize {
		return d.flushEventsToFile()
	}

	return nil
}

func (d *DiskBackup) PeriodicFlush() {
	for {
		time.Sleep(FlushInterval)

		if len(d.events) > 0 && time.Since(d.lastSaved) >= FlushInterval {
			d.flushEventsToFile()
		}
	}
}

func (d *DiskBackup) flushEventsToFile() error {
	data, err := json.Marshal(d.events)
	if err != nil {
		return err
	}

	// Generate a filename based on the current timestamp
	timestamp := time.Now().Format("2006-01-02T15-04-05.999999Z07-00")
	filename := timestamp + ".json"
	fullPath := filepath.Join(d.Dir, filename)
	fmt.Println("Saving to", fullPath)

	// Reset the events slice and update lastSaved time
	d.events = nil
	d.lastSaved = time.Now()

	return os.WriteFile(fullPath, data, 0644) 
}
