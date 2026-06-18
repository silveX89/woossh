package plugin

import (
	"strings"
)

// TrustLevel beschreibt die Vertrauensstufe einer Plugin-Quelle.
type TrustLevel int

const (
	TrustOfficial  TrustLevel = iota // github.com/silveX89/woossh-*
	TrustCommunity                   // github.com/* (anderer User)
	TrustUnknown                     // alles andere
)

func (t TrustLevel) String() string {
	switch t {
	case TrustOfficial:
		return "official"
	case TrustCommunity:
		return "community"
	default:
		return "unknown"
	}
}

// OfficialPrefix ist das Prefix für offizielle woossh-Plugins.
const OfficialPrefix = "github.com/silveX89/woossh-"

// TrustLevelForURL ermittelt die Vertrauensstufe einer Repository-URL.
func TrustLevelForURL(repoURL string) TrustLevel {
	clean := strings.TrimPrefix(repoURL, "https://")
	clean = strings.TrimPrefix(clean, "http://")

	if strings.HasPrefix(clean, OfficialPrefix) {
		return TrustOfficial
	}
	if strings.HasPrefix(clean, "github.com/") {
		return TrustCommunity
	}
	return TrustUnknown
}