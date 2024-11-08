package eventstore

import "github.com/nbd-wtf/go-nostr"

type Request struct {
	NostrFilter *nostr.Filter
	DoCount     bool
	NoLimit     bool
}
