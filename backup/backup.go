package backup

import (
	"github.com/honeycombio/refinery/types"
)
type Backup interface {
	Save(event *types.Event) error
}