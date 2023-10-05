package backup

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/honeycombio/refinery/types"
	"github.com/stretchr/testify/assert"
)

type mockS3 struct {
	s3iface.S3API
	PutObjectInvoked bool
	LastKey          string
	LastBody         io.ReadSeeker
}

func (m *mockS3) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	m.PutObjectInvoked = true
	m.LastKey = *input.Key
	m.LastBody = input.Body
	return &s3.PutObjectOutput{}, nil
}

func TestS3Backup(t *testing.T) {
	t.Run("CreateS3Backup", func(t *testing.T) {
		bucket := "test-bucket"
		s3Client := &mockS3{}
		sb := &Backup{
			lastSaved:     time.Now(),
			FlushInterval: time.Second,
			MaxBufferSize: 10,
			backuptype: &S3Backup{
				Bucket: bucket,
				S3:     s3Client,
			},
		}

		assert.NotNil(t, sb)
	})

	t.Run("SaveEventFlush", func(t *testing.T) {
		bucket := "test-bucket"
		s3Client := &mockS3{}
		sb := &Backup{
			lastSaved:     time.Now(),
			FlushInterval: time.Second,
			MaxBufferSize: 10,
			backuptype: &S3Backup{
				Bucket: bucket,
				S3:     s3Client,
			},
		}
		for i := 0; i < sb.MaxBufferSize; i++ {
			sb.Save(&types.Event{})
		}
		assert.True(t, s3Client.PutObjectInvoked)
	})

	t.Run("PeriodicFlush", func(t *testing.T) {
		bucket := "test-bucket"
		s3Client := &mockS3{}
		sb := &Backup{
			lastSaved:     time.Now(),
			FlushInterval: time.Second,
			MaxBufferSize: 10,
			backuptype: &S3Backup{
				Bucket: bucket,
				S3:     s3Client,
			},
		}
		go sb.PeriodicFlush() // Explicitly start the goroutine in the test
		sb.Save(&types.Event{})
		time.Sleep(sb.FlushInterval + time.Second)
		assert.True(t, s3Client.PutObjectInvoked)
	})

	t.Run("EventSerialization", func(t *testing.T) {
		bucket := "test-bucket"
		s3Client := &mockS3{}
		sb := &Backup{
			lastSaved:     time.Now(),
			FlushInterval: time.Second,
			MaxBufferSize: 10,
			backuptype: &S3Backup{
				Bucket: bucket,
				S3:     s3Client,
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

		sb.Save(event)
		sb.backuptype.flush(sb.events)

		// Read the body of the put object input
		data, readErr := io.ReadAll(s3Client.LastBody)
		assert.NoError(t, readErr)

		var loadedEvents []*types.SaveAbleEvent
		err := json.Unmarshal(data, &loadedEvents)
		assert.NoError(t, err)
		assert.NotEmpty(t, loadedEvents)

		// Assuming the S3 body contains only one event
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
