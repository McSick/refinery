package backup

import (
    "fmt"
	"os"
	"encoding/json"
	"path/filepath"
	"time"
    "github.com/honeycombio/refinery/types"
)

type DiskBackup struct {
    Dir string
}

func NewDiskBackup(dir string) *DiskBackup {
    return &DiskBackup{Dir: dir}
}

func (d *DiskBackup) Save(event *types.Event) error {
    // Convert event to JSON (or any other format)
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }
    
	// Generate a filename based on the current timestamp
	timestamp := time.Now().Format("2006-01-02T15-04-05.999999Z07-00")
	filename := timestamp + ".json"
    fullPath := filepath.Join(d.Dir, filename)
    fmt.Println("Saving to", fullPath)
    return os.WriteFile(fullPath, data, 0644)  // Adjust permissions as needed
}
