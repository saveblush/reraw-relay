package policies

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	"github.com/saveblush/reraw-relay/models"
)

func TestEventVerifying(t *testing.T) {
	rawEvents := []string{
		`
			{
				"id": "266f5bc338392bd5404d8f5553ac653d61d0217ad345f711a9b9d60c4d692015",
				"pubkey": "f1e6db4c8ffad88a44f763946fec9885d794a49343ae4823c4a000706a3697e7",
				"created_at": 1740805537,
				"kind": 1,
				"tags": [],
				"content": "TestEventParsingAndVerifying\nl1\nl2",
				"sig": "712c89a2feeb2d6e93c746fc7fa42ce1b7c502509ffd8e3b650ff2fde4680f9e69fab3ad10c9843260752ecf5402c0bf45f8464ae9d552a5a53bd6f41c8bcaa6"
			}		
		`,
		`
			{
				"id": "2f0d220509502d12ae68c019b8e12bd35b46eabf0d5c14ac77e0169bf1bde65f",
				"pubkey": "f1e6db4c8ffad88a44f763946fec9885d794a49343ae4823c4a000706a3697e7",
				"created_at": 1740805537,
				"kind": 1,
				"tags": [
					[
						"e", "eee_1"
					],
					[
						"p", "ppp_2"
					]
				],
				"content": "TestEventParsingAndVerifying\nt1\nt2\nt3",
				"sig": "9770943d37d7cb24e0e030065ba9431f521c0dc402c1793c52439fb867d7c1bb82b0db245f7694685fd0b1b89e62e83fd0b39c2cda8bd6282f4a1af83d6cc04e"
			}		
		`,
		`
			{
				"id": "97bf8e1c465fd5207da097b19554cb51c6eaa2fb15b8a51b356e6852e11a1710",
				"pubkey": "f1e6db4c8ffad88a44f763946fec9885d794a49343ae4823c4a000706a3697e7",
				"created_at": 1740805537,
				"kind": 7,
				"tags": [],
				"content": "🚀",
				"sig": "5b27e6690a144409877fbce4743e098ebe10b161c1aeab747105bdceab83e29a5e67bb6d1273e83cdf91613394f227ebff309813555e239f5a924cca26c46bc6"
			}		
		`,
	}

	for _, req := range rawEvents {
		var evt models.Event
		err := json.Unmarshal([]byte(req), &evt)
		assert.NoError(t, err)

		assert.Equal(t, evt.ID, evt.GetID())

		ok, err := evt.VerifySignature()
		assert.NoError(t, err)
		assert.True(t, ok, "signature verification failed when it should have succeeded")
	}
}
