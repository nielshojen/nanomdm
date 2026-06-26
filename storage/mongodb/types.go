package mongodb

import "time"

type deviceDocument struct {
	ID             string    `bson:"_id"`
	IdentityCert   []byte    `bson:"identity_cert,omitempty"`
	SerialNumber   string    `bson:"serial_number,omitempty"`
	UnlockToken    []byte    `bson:"unlock_token,omitempty"`
	UnlockTokenAt  time.Time `bson:"unlock_token_at,omitempty"`
	Authenticate   []byte    `bson:"authenticate,omitempty"`
	AuthenticateAt time.Time `bson:"authenticate_at,omitempty"`
	TokenUpdate    []byte    `bson:"token_update,omitempty"`
	TokenUpdateAt  time.Time `bson:"token_update_at,omitempty"`
	BootstrapToken []byte    `bson:"bootstrap_token,omitempty"`
	BootstrapAt    time.Time `bson:"bootstrap_token_at,omitempty"`
	CreatedAt      time.Time `bson:"created_at,omitempty"`
	UpdatedAt      time.Time `bson:"updated_at,omitempty"`
}

type userDocument struct {
	ID                       string    `bson:"_id"`
	DeviceID                 string    `bson:"device_id"`
	UserShortName            string    `bson:"user_short_name,omitempty"`
	UserLongName             string    `bson:"user_long_name,omitempty"`
	TokenUpdate              []byte    `bson:"token_update,omitempty"`
	TokenUpdateAt            time.Time `bson:"token_update_at,omitempty"`
	UserAuthenticate         []byte    `bson:"user_authenticate,omitempty"`
	UserAuthenticateAt       time.Time `bson:"user_authenticate_at,omitempty"`
	UserAuthenticateDigest   []byte    `bson:"user_authenticate_digest,omitempty"`
	UserAuthenticateDigestAt time.Time `bson:"user_authenticate_digest_at,omitempty"`
	CreatedAt                time.Time `bson:"created_at,omitempty"`
	UpdatedAt                time.Time `bson:"updated_at,omitempty"`
}

type enrollmentDocument struct {
	ID               string    `bson:"_id"`
	DeviceID         string    `bson:"device_id"`
	UserID           string    `bson:"user_id,omitempty"`
	Type             string    `bson:"type"`
	Topic            string    `bson:"topic"`
	PushMagic        string    `bson:"push_magic"`
	TokenHex         string    `bson:"token_hex"`
	Enabled          bool      `bson:"enabled"`
	TokenUpdateTally int       `bson:"token_update_tally"`
	LastSeenAt       time.Time `bson:"last_seen_at"`
	CreatedAt        time.Time `bson:"created_at,omitempty"`
	UpdatedAt        time.Time `bson:"updated_at,omitempty"`
}

type commandDocument struct {
	ID          string    `bson:"_id"`
	RequestType string    `bson:"request_type"`
	Command     []byte    `bson:"command"`
	CreatedAt   time.Time `bson:"created_at,omitempty"`
	UpdatedAt   time.Time `bson:"updated_at,omitempty"`
}

type commandResultDocument struct {
	ID           string    `bson:"_id"`
	EnrollmentID string    `bson:"id"`
	CommandUUID  string    `bson:"command_uuid"`
	Status       string    `bson:"status"`
	Result       []byte    `bson:"result"`
	NotNowAt     time.Time `bson:"not_now_at,omitempty"`
	NotNowTally  int       `bson:"not_now_tally,omitempty"`
	CreatedAt    time.Time `bson:"created_at,omitempty"`
	UpdatedAt    time.Time `bson:"updated_at,omitempty"`
}

type queueDocument struct {
	ID           string    `bson:"_id"`
	EnrollmentID string    `bson:"id"`
	CommandUUID  string    `bson:"command_uuid"`
	Active       bool      `bson:"active"`
	Priority     int       `bson:"priority"`
	CreatedAt    time.Time `bson:"created_at,omitempty"`
	UpdatedAt    time.Time `bson:"updated_at,omitempty"`
}

type pushCertDocument struct {
	ID         string    `bson:"_id"`
	CertPEM    []byte    `bson:"cert_pem"`
	KeyPEM     []byte    `bson:"key_pem"`
	StaleToken int       `bson:"stale_token"`
	CreatedAt  time.Time `bson:"created_at,omitempty"`
	UpdatedAt  time.Time `bson:"updated_at,omitempty"`
}

type certAuthDocument struct {
	ID           string    `bson:"_id"`
	EnrollmentID string    `bson:"id"`
	SHA256       string    `bson:"sha256"`
	CreatedAt    time.Time `bson:"created_at,omitempty"`
	UpdatedAt    time.Time `bson:"updated_at,omitempty"`
}
