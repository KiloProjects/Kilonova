package kilonova

import "time"

type DonationSource string

const (
	DonationSourceUnknown DonationSource = ""
	DonationSourceBMAC    DonationSource = "buymeacoffee"
	DonationSourcePaypal  DonationSource = "paypal"
	DonationSourceStripe  DonationSource = "stripe"
	DonationSourceOther   DonationSource = "other"
)

type DonationType string

const (
	DonationTimeUnknown DonationType = ""
	DonationTypeOneTime DonationType = "onetime"
	DonationTypeMonthly DonationType = "monthly"
	DonationTypeYearly  DonationType = "yearly"
)

type Donation struct {
	ID        int       `json:"id"`
	DonatedAt time.Time `json:"donated_at"`

	User     *UserBrief `json:"user"`
	Amount   float64    `json:"amount"`
	Currency string     `json:"currency"`

	Source DonationSource `json:"source"`
	Type   DonationType   `json:"type"`

	RealName string `json:"real_name"`

	TransactionID string     `json:"transaction_id"`
	CancelledAt   *time.Time `json:"cancelled_at"`
}
