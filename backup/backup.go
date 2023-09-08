package backup

import (
	"github.com/honeycombio/refinery/types"
	"time"
)

const MaxBufferSize = 100        
const FlushInterval = time.Minute 

type Backup interface {
	Save(event *types.Event) error
}