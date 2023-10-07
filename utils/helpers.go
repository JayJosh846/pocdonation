package utils

import (
	cryptorand "crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	mathrand "math/rand"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Usererror struct {
	Error        bool
	ResponseCode int
	Message      string
	Data         string
}

func HashPassword(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		log.Panic(err)
	}
	return string(bytes)
}

func VerifyPassword(userpassword string, givenpassword string) (bool, Usererror) {
	err := bcrypt.CompareHashAndPassword([]byte(givenpassword), []byte(userpassword))
	valid := true
	if err != nil {
		valid = false
		return valid,
			Usererror{
				Error:        true,
				ResponseCode: 400,
				Message:      "Invalid Password",
				Data:         "",
			}
	}
	return valid, Usererror{}
}

func GenerateTransactionReference() string {
	// Generate a random identifier.
	identifier := mathrand.Intn(1000000) // Change the range as needed.
	// Get the current timestamp.
	currentTime := time.Now()
	// Format the timestamp and combine it with the identifier.
	transactionReference := currentTime.Format("20060102150405") + fmt.Sprintf("%06d", identifier)
	return transactionReference
}

func ExtractUsernameFromEmail(email string) (string, error) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", fmt.Errorf("Invalid email format")
	}
	return parts[0], nil
}

func GenerateRandomPassword(length int) (string, error) {
	numBytes := (length * 3) / 4

	// Generate random bytes
	randomBytes := make([]byte, numBytes)
	_, err := cryptorand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	password := base64.URLEncoding.EncodeToString(randomBytes)
	password = password[:length]

	return password, nil
}
