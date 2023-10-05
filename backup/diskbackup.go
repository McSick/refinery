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
	Dir string
}

func (d *DiskBackup) flush(events []*types.SaveAbleEvent) error {
	data, err := json.Marshal(events)
	if err != nil {
		return err
	}

	// Generate a filename based on the current timestamp
	timestamp := time.Now().Format("2006-01-02T15-04-05.999999Z07-00")
	filename := timestamp + ".json"
	fullPath := filepath.Join(d.Dir, filename)
	fmt.Println("Saving to", fullPath)

	// Reset the events slice and update lastSaved time
	events = nil

	return os.WriteFile(fullPath, data, 0644)
}
