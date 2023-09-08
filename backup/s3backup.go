package backup

import (
	"bytes"
	"encoding/json"
	"time"
    "os"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
    "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/honeycombio/refinery/types"
)

type S3Backup struct {
	Bucket    string
	S3        *s3.S3
	events    []*types.Event
	lastSaved time.Time
}

func NewS3Backup(bucket string) *S3Backup {
    awsAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
    awsSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
    awsConfig := &aws.Config{
        Region:      aws.String("us-east-1"), // TODO ADD to CONFIG
        Credentials: credentials.NewStaticCredentials(awsAccessKey, awsSecretKey, ""),
        // Other configuration settings
    }
    sess := session.Must(session.NewSession(awsConfig))
    
	sb := &S3Backup{
		Bucket:    bucket,
		S3:        s3.New(sess),
		lastSaved: time.Now(),
	}
	go sb.PeriodicFlush()
	return sb
}

func (s *S3Backup) Save(event *types.Event) error {
	s.events = append(s.events, event)

	if len(s.events) >= MaxBufferSize {
		return s.flushEventsToS3()
	}

	return nil
}

func (s *S3Backup) PeriodicFlush() {
	for {
		time.Sleep(FlushInterval)

		if len(s.events) > 0 && time.Since(s.lastSaved) >= FlushInterval {
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
