package orgunit

import (
	"crypto/rand"
	"errors"
	"regexp"
	"strings"
)

const dhis2UIDLength = 11

var dhis2UIDPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]{10}$`)

const dhis2UIDFirstChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
const dhis2UIDChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func normalizeUID(uid string) string {
	return strings.TrimSpace(uid)
}

func ensureDHIS2UID(uid string) (string, error) {
	normalized := normalizeUID(uid)
	if normalized == "" {
		return generateDHIS2UID()
	}
	if !dhis2UIDPattern.MatchString(normalized) {
		return "", errors.New("uid must be DHIS2-style: 11 characters, starting with a letter")
	}
	return normalized, nil
}

func generateDHIS2UID() (string, error) {
	bytes := make([]byte, dhis2UIDLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	uid := make([]byte, dhis2UIDLength)
	uid[0] = dhis2UIDFirstChars[int(bytes[0])%len(dhis2UIDFirstChars)]
	for i := 1; i < dhis2UIDLength; i++ {
		uid[i] = dhis2UIDChars[int(bytes[i])%len(dhis2UIDChars)]
	}
	return string(uid), nil
}
