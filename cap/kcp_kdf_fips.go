//go:build fips

package cap

import (
	"crypto/sha256"
	"hash"
)

func kcpKDFHash() func() hash.Hash {
	return sha256.New
}
