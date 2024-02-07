package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/nats-io/jwt"
	"github.com/nats-io/nkeys"
)

// SplitAndTrimEmpty slices s into all subslices separated by sep and returns a
// slice of the string s with all leading and trailing Unicode code points
// contained in cutset removed. If sep is empty, SplitAndTrim splits after each
// UTF-8 sequence. First part is equivalent to strings.SplitN with a count of
// -1.  also filter out empty strings, only return non-empty strings.
func SplitAndTrimEmpty(s, sep, cutset string) []string {
	if s == "" {
		return []string{}
	}

	spl := strings.Split(s, sep)
	nonEmptyStrings := make([]string, 0, len(spl))

	for i := 0; i < len(spl); i++ {
		element := strings.Trim(spl[i], cutset)
		if element != "" {
			nonEmptyStrings = append(nonEmptyStrings, element)
		}
	}

	return nonEmptyStrings
}

func setDefault(field string, value string) {
	if os.Getenv(field) == "" {
		os.Setenv(field, value)
	}
}

// CreateUser creates NATS user NKey and JWT from given account seed NKey.
func CreateUser(seed string) (*string, *string, error) {
	accountSeed := []byte(seed)

	accountKeys, err := nkeys.FromSeed(accountSeed)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get account key from seed: %w", err)
	}

	accountPubKey, err := accountKeys.PublicKey()
	if err != nil {
		return nil, nil, fmt.Errorf("error getting public key: %w", err)
	}

	userKeys, err := nkeys.CreateUser()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create account key: %w", err)
	}

	userSeed, err := userKeys.Seed()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get seed: %w", err)
	}
	nkey := string(userSeed)

	userPubKey, err := userKeys.PublicKey()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot get user's public key: %w", err)
	}

	claims := jwt.NewUserClaims(userPubKey)
	claims.Issuer = accountPubKey
	jwt, err := claims.Encode(accountKeys)
	if err != nil {
		return nil, nil, fmt.Errorf("error encoding token to jwt: %w", err)
	}

	return &nkey, &jwt, nil
}
