package models

type RelayInformationDocument struct {
	Name          string                   `json:"name"`
	Description   string                   `json:"description"`
	Pubkey        string                   `json:"pubkey"`
	Contact       string                   `json:"contact"`
	SupportedNIPs []int                    `json:"supported_nips"`
	Software      string                   `json:"software"`
	Version       string                   `json:"version"`
	Limitation    *RelayLimitationDocument `json:"limitation,omitempty"`
	Icon          string                   `json:"icon"`
}

type RelayLimitationDocument struct {
	MaxMessageLength int  `json:"max_message_length,omitempty"`
	MaxSubscriptions int  `json:"max_subscriptions,omitempty"`
	MaxFilters       int  `json:"max_filters,omitempty"`
	MaxLimit         int  `json:"max_limit,omitempty"`
	MaxSubidLength   int  `json:"max_subid_length,omitempty"`
	MaxEventTags     int  `json:"max_event_tags,omitempty"`
	MaxContentLength int  `json:"max_content_length,omitempty"`
	MinPowDifficulty int  `json:"min_pow_difficulty,omitempty"`
	AuthRequired     bool `json:"auth_required"`
	PaymentRequired  bool `json:"payment_required"`
	RestrictedWrites bool `json:"restricted_writes"`
}
