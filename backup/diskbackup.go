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
	events    []*types.SaveAbleEvent
	lastSaved time.Time
    FlushInterval time.Duration
    MaxBufferSize int
}

func NewDiskBackup(dir string, flushinterval time.Duration, maxbuffersize int) *DiskBackup {
	db := &DiskBackup{Dir: dir, lastSaved: time.Now(), FlushInterval: flushinterval, MaxBufferSize: maxbuffersize}
	go db.PeriodicFlush()
	return db
}

func (d *DiskBackup) Save(event *types.Event) error {
	d.events = append(d.events, event.ConvertToSaveAbleEvent())

	if len(d.events) >= d.MaxBufferSize {
		return d.flushEventsToFile()
	}

	return nil
}

func (d *DiskBackup) PeriodicFlush() {
	for {
		time.Sleep(d.FlushInterval)

		if len(d.events) > 0 && time.Since(d.lastSaved) >= d.FlushInterval {
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
