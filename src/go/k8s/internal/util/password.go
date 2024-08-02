package util

import (
	"crypto/rand"
	"io"
	"math/big"
	"strings"
)

// public PasswordGenerator(int length, String firstCharacterAlphabet, String alphabet) {
// 	this.length = length;
// 	this.firstCharacterAlphabet = firstCharacterAlphabet;
// 	this.alphabet = alphabet;
// }

// public static final ConfigParameter<Integer> SCRAM_SHA_PASSWORD_LENGTH = new ConfigParameter<>("STRIMZI_SCRAM_SHA_PASSWORD_LENGTH", strictlyPositive(INTEGER), "32",  CONFIG_VALUES);

type PasswordGenerator struct {
	reader          io.Reader
	length          int
	firstCharacters string
	alphabet        string
}

func NewPasswordGenerator() *PasswordGenerator {
	return &PasswordGenerator{
		reader:          rand.Reader,
		length:          32,
		firstCharacters: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ",
		alphabet:        "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
	}
}

func (p *PasswordGenerator) Generate() (string, error) {
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
