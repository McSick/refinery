package backup

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/honeycombio/refinery/types"
)

type S3Backup struct {
	Bucket        string
	S3            s3iface.S3API
	events        []*types.SaveAbleEvent
	lastSaved     time.Time
	FlushInterval time.Duration
	MaxBufferSize int
}

func NewS3Backup(bucket string, flushinterval time.Duration, maxbuffersize int, awsAccessKey string, awsSecretKey string, awsRegion string) *S3Backup {
	awsConfig := &aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: credentials.NewStaticCredentials(awsAccessKey, awsSecretKey, ""),
	}
	sess := session.Must(session.NewSession(awsConfig))

	sb := &S3Backup{
		Bucket:        bucket,
		S3:            s3.New(sess),
		lastSaved:     time.Now(),
		FlushInterval: flushinterval,
		MaxBufferSize: maxbuffersize,
	}
	go sb.PeriodicFlush()
	return sb
}

func (s *S3Backup) Save(event *types.Event) error {
	s.events = append(s.events, event.ConvertToSaveAbleEvent())

	if len(s.events) >= s.MaxBufferSize {
		return s.flushEventsToS3()
	}

	return nil
}

func (s *S3Backup) PeriodicFlush() {
	for {
		time.Sleep(s.FlushInterval)
		if len(s.events) > 0 && time.Since(s.lastSaved) >= s.FlushInterval {
			s.flushEventsToS3()
		}
	}
}

func (s *S3Backup) flushEventsToS3() error {
	data, err := json.Marshal(s.events)
	if err != nil {
		return err
	}

	// Generate a filename based on the current timestamp
	key := time.Now().Format("2006-01-02T15-04-05.999999Z07-00") + ".json"

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	}

	_, err = s.S3.PutObject(input)

	// Reset the events slice and update lastSaved time
	s.events = nil
	s.lastSaved = time.Now()

	return err
}
