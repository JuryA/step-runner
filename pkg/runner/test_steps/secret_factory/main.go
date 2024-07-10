package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"os"
)

var (
	outputFile     = flag.String("output_file", "", "file to output secrets to")
	secretOverride = flag.String("secret_override", "", "set to output the secret to a predetermined value")
	charset        = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	secretLength   = 15
)

func main() {
	flag.Parse()

	if *outputFile == "" {
		panic("--output_file is required")
	}

	secret := secretOverride

	if secret == nil || *secret == "" {
		randBytes := make([]byte, secretLength)
		_, err := rand.Read(randBytes)

		if err != nil {
			panic(err)
		}

		for i, b := range randBytes {
			randBytes[i] = charset[int(b)%len(charset)]
		}

		*secret = string(randBytes)
	}

	if err := os.WriteFile(*outputFile, []byte(fmt.Sprintf(`secret="%s"`, *secret)), 0640); err != nil {
		panic(err)
	}
}
