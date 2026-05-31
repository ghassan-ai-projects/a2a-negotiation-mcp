package vendorreviews

type VendorReview struct {
	ID        int    `json:"id"`
	Vendor    string `json:"vendor"`
	Rating    int    `json:"rating"`
	Comment   string `json:"comment"`
	CreatedAt string `json:"created_at"`
}
