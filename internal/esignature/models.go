package esignature

// Envelope represents an e-signature envelope sent to a signer.
type Envelope struct {
	ID          int    `json:"id"`
	ContractID  string `json:"contract_id"`
	SignerEmail string `json:"signer_email"`
	Status      string `json:"status"`
	EnvelopeID  string `json:"envelope_id"`
	CreatedAt   string `json:"created_at"`
}

// SignatureResult represents the result of a simulated e-signature.
type SignatureResult struct {
	EnvelopeID  string `json:"envelope_id"`
	ContractID  string `json:"contract_id"`
	Status      string `json:"status"`
	SignerEmail string `json:"signer_email"`
	SignedAt    string `json:"signed_at,omitempty"`
}
