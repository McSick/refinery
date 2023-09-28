package backup

import (
	"time"

	"github.com/honeycombio/refinery/config"
	"github.com/honeycombio/refinery/types"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type BackupType interface {
	flush(events []*types.SaveAbleEvent) error
}
type Backup struct {
	events        []*types.SaveAbleEvent
	lastSaved     time.Time
	FlushInterval time.Duration
	MaxBufferSize int
	backuptype    BackupType
}

func (b *Backup) Save(event *types.Event) error {
	b.events = append(b.events, event.ConvertToSaveAbleEvent())

	if len(b.events) >= b.MaxBufferSize {
		b.lastSaved = time.Now()
		err := b.backuptype.flush(b.events)
		if err != nil {
			//todo
		} else {
			b.events = nil
		}
		return err
	}

	return nil
}

func (b *Backup) PeriodicFlush() {
	for {
		time.Sleep(b.FlushInterval)

		if len(b.events) > 0 && time.Since(b.lastSaved) >= b.FlushInterval {
			b.lastSaved = time.Now()
			err := b.backuptype.flush(b.events)
			if err != nil {
				//todo
			} else {
				b.events = nil
			}
		}
	}
}

func NewBackup(c config.Config) *Backup {
	var backuptype BackupType
	switch c.GetBackupType() {
	case "s3":
		{
			awsConfig := &aws.Config{
				Region:      aws.String(c.GetBackupAWSRegion()),
				Credentials: credentials.NewStaticCredentials(c.GetBackupAWSAccessKeyID(), c.GetBackupAWSSecretAccessKey(), ""),
			}
			sess := session.Must(session.NewSession(awsConfig))

			backuptype = &S3Backup{
				Bucket: c.GetBackupBucket(),
				S3:     s3.New(sess),
			}
		}
	case "disk":
		{
			backuptype = &DiskBackup{
				Dir: c.GetBackupDir(),
			}
		}
	default:
		return nil
	}
	b := &Backup{
		lastSaved:     time.Now(),
		FlushInterval: c.GetBackupFlushInterval(),
		MaxBufferSize: c.GetBackupMaxBufferSize(),
		backuptype:    backuptype,
	}
	go b.PeriodicFlush()
	return b

}
