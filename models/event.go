package models

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2/schnorr"
)

const (
	MaxUint16 = 65535
	MaxUint32 = 4294967295
)

type Event struct {
	ID         string     `json:"id" gorm:"primaryKey;type:varchar(64)"`
	CreatedAt  Timestamp  `json:"created_at" gorm:"type:integer"`
	Pubkey     string     `json:"pubkey" gorm:"type:varchar(64)"`
	Kind       int        `json:"kind" gorm:"type:integer"`
	Content    string     `json:"content"`
	Tags       Tags       `json:"tags" gorm:"type:jsonb"`
	Sig        string     `json:"sig"`
	Tagvalues  []string   `json:"-" gorm:"-"`
	Expiration *Timestamp `json:"-" gorm:"type:integer"`
	UpdatedIP  *string    `json:"-"`
	UpdatedAt  *Timestamp `json:"-" gorm:"type:integer"`
	DeletedAt  *Timestamp `json:"-" gorm:"type:integer"`
}

func (Event) TableName() string {
	return "events"
}

// GetID get event id
func (evt *Event) GetID() string {
	hash := evt.hash()
	return hex.EncodeToString(hash[:])
}

// VerifySignature verify signature
func (evt *Event) VerifySignature() (bool, error) {
	// check pubkey
	pk, err := hex.DecodeString(evt.Pubkey)
	if err != nil {
		return false, fmt.Errorf("decoding pubkey error: %s", err)
	}

	pubkey, err := schnorr.ParsePubKey(pk)
	if err != nil {
		return false, fmt.Errorf("parse pubkey error: %s", err)
	}

	// check signature
	s, err := hex.DecodeString(evt.Sig)
	if err != nil {
		return false, fmt.Errorf("decoding signature error: %s", err)
	}

	sig, err := schnorr.ParseSignature(s)
	if err != nil {
		return false, fmt.Errorf("parse signature error: %s", err)
	}

	// verify
	hash := evt.hash()
	verified := sig.Verify(hash[:], pubkey)

	return verified, nil
}

// Serialize serialize event data
func (evt *Event) Serialize() string {
	return fmt.Sprintf(
		`[0,"%s",%d,%d,%s,"%s"]`,
		evt.Pubkey,
		evt.CreatedAt,
		evt.Kind,
		evt.Tags.Serialize(),
		escapeSpecialChars(evt.Content),
	)
}

func (evt *Event) hash() [32]byte {
	return sha256.Sum256([]byte(evt.Serialize()))
}

func escapeSpecialChars(s string) string {
	s = strings.ReplaceAll(s, "\"", `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	s = strings.ReplaceAll(s, "\b", `\b`)
	s = strings.ReplaceAll(s, "\f", `\f`)

	return s
}

/*func escapeSpecialChars(s string) string {
	var result []byte
	for i := 0; i < len(s); i++ {
		v := s[i]
		switch {
		case v == 0x22:
			result = append(result, []byte{'\\', '"'}...)
		case v == 0x5C:
			result = append(result, []byte{'\\', '\\'}...)
		case v == 0x0A:
			result = append(result, []byte{'\\', 'n'}...)
		case v == 0x0D:
			result = append(result, []byte{'\\', 'r'}...)
		case v == 0x09:
			result = append(result, []byte{'\\', 't'}...)
		case v == 0x08:
			result = append(result, []byte{'\\', 'b'}...)
		case v == 0x0c:
			result = append(result, []byte{'\\', 'f'}...)
		default:
			result = append(result, v)
		}
	}

	return string(result)
}*/
