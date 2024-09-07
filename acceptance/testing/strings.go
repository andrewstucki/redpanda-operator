package testing

import "golang.org/x/exp/rand"

var letters = "0123456789abcdefghijklmnopqrstuvwxyz"

func randomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func AddSuffix(s string) string {
	return s + "-" + randomString(10)
}
