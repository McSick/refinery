package backup

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/honeycombio/refinery/types"
	"github.com/stretchr/testify/assert"
)

func TestDiskBackup(t *testing.T) {

	t.Run("CreateDiskBackup", func(t *testing.T) {
		dir, err := os.MkdirTemp("", "backup")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(dir) // Cleanup after the test
		db := &Backup{
			lastSaved:     time.Now(),
			FlushInterval: time.Second,
			MaxBufferSize: 10,
			backuptype: &DiskBackup{
				Dir: dir,
			},
		}
		assert.NotNil(t, db)
	})

	t.Run("SaveEventFlush", func(t *testing.T) {
		dir, err := os.MkdirTemp("", "backup")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(dir) // Cleanup after the test

		db := &Backup{
			lastSaved:     time.Now(),
			FlushInterval: time.Second,
			MaxBufferSize: 10,
			backuptype: &DiskBackup{
				Dir: dir,
			},
		}
		for i := 0; i < db.MaxBufferSize; i++ {
			db.Save(&types.Event{})
		}
		// Check if a file exists in the directory after buffer is filled.
		files, err := os.ReadDir(dir)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(files))
	})

	t.Run("PeriodicFlush", func(t *testing.T) {
		dir, err := os.MkdirTemp("", "backup")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(dir) // Cleanup after the test

		db := &Backup{
			lastSaved:     time.Now(),
			FlushInterval: time.Millisecond,
			MaxBufferSize: 10,
			backuptype: &DiskBackup{
				Dir: dir,
			},
		}
		go db.PeriodicFlush()
		db.Save(&types.Event{})
		// Wait for a time greater than the FlushInterval
		time.Sleep(db.FlushInterval + time.Second)
		files, err := os.ReadDir(dir)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(files))
	})

	t.Run("EventSerialization", func(t *testing.T) {
		dir, err := os.MkdirTemp("", "backup")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(dir) // Cleanup after the test
		db := &Backup{
			lastSaved:     time.Now(),
			FlushInterval: time.Second,
			MaxBufferSize: 10,
			backuptype: &DiskBackup{
				Dir: dir,
			},
		}
		event := &types.Event{
			Context:     context.Background(),
			APIHost:     "https://api.honeycomb.io",
			APIKey:      "test-api-key-1234",
			Dataset:     "test-dataset",
			Environment: "development",
			SampleRate:  1,
			Timestamp:   time.Now(),
			Data: map[string]interface{}{
				"key1": "value1",
				"key2": 12345,
			},
		}

		db.Save(event)
		db.backuptype.flush(db.events)

		files, err := os.ReadDir(dir)
		assert.NoError(t, err)
		assert.NotEmpty(t, files)

		data, err := os.ReadFile(filepath.Join(dir, files[0].Name()))
		assert.NoError(t, err)

		var loadedEvents []*types.SaveAbleEvent
		err = json.Unmarshal(data, &loadedEvents)
		assert.NoError(t, err)
		assert.NotEmpty(t, loadedEvents)

		// Assuming that the file contains only one event
		loadedEvent := loadedEvents[0]

		// Assert the contents of the Data field
		assert.Contains(t, loadedEvent.Data, "key1")
		assert.Equal(t, "value1", loadedEvent.Data["key1"])

		assert.Contains(t, loadedEvent.Data, "key2")
		assert.Equal(t, 12345.0, loadedEvent.Data["key2"])

		// Assert other fields of the event
		assert.Equal(t, "https://api.honeycomb.io", loadedEvent.APIHost)
		assert.Equal(t, "test-api-key-1234", loadedEvent.APIKey)
		assert.Equal(t, "test-dataset", loadedEvent.Dataset)
		assert.Equal(t, "development", loadedEvent.Environment)
		assert.Equal(t, uint(1), loadedEvent.SampleRate)
	})
}
