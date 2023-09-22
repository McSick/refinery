package backup

import (
	"github.com/honeycombio/refinery/config"
	"github.com/honeycombio/refinery/types"
)

// const MaxBufferSize = 100
// const FlushInterval = time.Second //Make this configurable and minute?

type Backup interface {
	Save(event *types.Event) error
}

// TODO instead of location, backup settings
func NewBackup(c config.Config) Backup {
	switch c.GetBackupType() {
	case "s3":
		return NewS3Backup(c.GetBackupBucket(), c.GetBackupFlushInterval(), c.GetBackupMaxBufferSize(),
			c.GetBackupAWSAccessKeyID(), c.GetBackupAWSSecretAccessKey(), c.GetBackupAWSRegion())
	case "disk":
		return NewDiskBackup(c.GetBackupDir(), c.GetBackupFlushInterval(), c.GetBackupMaxBufferSize())
	default:
		return nil
	}
}
