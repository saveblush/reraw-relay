package eventstore

import "github.com/saveblush/reraw-relay/models"

type Request struct {
	NostrFilter *models.Filter
	DoCount     bool
	NoLimit     bool
}
