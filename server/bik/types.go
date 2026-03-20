package bik

import "time"

const APContext = "https://www.w3.org/ns/activitystreams"

// AP activity types

type Activity struct {
	Context string `json:"@context"`
	Type    string `json:"type"`
	Actor   string `json:"actor"`
	Object  any    `json:"object"`
}

type NoteObject struct {
	Type         string `json:"type"` // "Note"
	ID           string `json:"id"`
	AttributedTo string `json:"attributedTo,omitempty"`
	Content      string `json:"content"`
	InReplyTo    string `json:"inReplyTo,omitempty"`
	Attachment   any    `json:"attachment,omitempty"`
}

// BIK-specific attachment types

type TimeControl struct {
	System     string  `json:"system"`               // byoyomi, fischer, absolute
	MainTime   float64 `json:"mainTime"`             // seconds
	Periods    int     `json:"periods,omitempty"`    // byoyomi
	PeriodTime float64 `json:"periodTime,omitempty"` // byoyomi
	Increment  float64 `json:"increment,omitempty"`  // fischer
}

type BikChallenge struct {
	Type        string      `json:"type"` // "BikChallenge"
	BoardSize   int         `json:"boardSize"`
	TimeControl TimeControl `json:"timeControl"`
	ColorPref   string      `json:"colorPreference"` // black, white, any
	ExpiresAt   time.Time   `json:"expiresAt"`
}

type BikGame struct {
	Type      string `json:"type"` // "BikGame"
	ID        string `json:"id"`
	Black     string `json:"black"`     // actor URI
	White     string `json:"white"`     // actor URI
	URL       string `json:"url"`       // human-readable game page
	WebSocket string `json:"websocket"` // wss:// URL
}

type BikGameResult struct {
	Type   string `json:"type"` // "BikGameResult"
	Winner string `json:"winner"`
	Result string `json:"result"` // SGF: B+R, W+3.5, etc.
	SGF    string `json:"sgf,omitempty"`
}

// Token issued by a guest server to authenticate a remote player to the host WS

type BikToken struct {
	Player string `json:"player"` // @user@domain
	Game   string `json:"game"`   // game URI on host
	Exp    int64  `json:"exp"`    // unix timestamp
}

// WebFinger response

type WebFingerResponse struct {
	Subject string          `json:"subject"`
	Links   []WebFingerLink `json:"links"`
}

type WebFingerLink struct {
	Rel  string `json:"rel"`
	Type string `json:"type,omitempty"`
	Href string `json:"href,omitempty"`
}

// JWK set for public key exposure

type JWKSet struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kty string `json:"kty"` // "OKP"
	Crv string `json:"crv"` // "Ed25519"
	X   string `json:"x"`   // base64url-encoded public key
}
