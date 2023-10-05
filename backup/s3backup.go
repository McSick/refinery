package backup

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/honeycombio/refinery/types"
)

type S3Backup struct {
	Bucket string
	S3     s3iface.S3API
}

func (s *S3Backup) flush(events []*types.SaveAbleEvent) error {
	data, err := json.Marshal(events)
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
	events = nil

	return err
}
