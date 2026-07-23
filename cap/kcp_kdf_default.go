//go:build !fips

package cap

import (
	"crypto/sha1"
	"hash"
)

func kcpKDFHash() func() hash.Hash {
	return sha1.New
}
