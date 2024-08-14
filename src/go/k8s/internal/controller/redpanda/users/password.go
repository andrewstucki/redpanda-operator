package users

import (
	"crypto/rand"
	"io"
	"math/big"
	"strings"
)

type passwordGenerator struct {
	reader          io.Reader
	length          int
	firstCharacters string
	alphabet        string
}

func newPasswordGenerator() *passwordGenerator {
	return &passwordGenerator{
		reader:          rand.Reader,
		length:          32,
		firstCharacters: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ",
		alphabet:        "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
	}
}

func (p *passwordGenerator) Generate() (string, error) {
	var password strings.Builder
	nextIndex := func(length int) (int, error) {
		n, err := rand.Int(p.reader, big.NewInt(int64(length)))
		if err != nil {
			return -1, err
		}
		return int(n.Int64()), nil
	}

	index, err := nextIndex(len(p.firstCharacters))
	if err != nil {
		return "", err
	}
	password.WriteByte(p.firstCharacters[index])

	for i := 0; i < p.length; i++ {
		index, err := nextIndex(len(p.alphabet))
		if err != nil {
			return "", err
		}
		password.WriteByte(p.alphabet[index])
	}

	return password.String(), nil
}
