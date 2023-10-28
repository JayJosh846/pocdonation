package services

import (
	"fmt"
	"net/smtp"
)

// Send a verification email with the code
func sendVerificationEmail(userName, email, code string) error {
	from := "jesudara.j@gmail.com" // Replace with your email
	password := "ycqphxpqaudxipsg" // Replace with your email password
	to := email
	subject := "Email Verification Code"
	body := fmt.Sprintf("Hello %s,\n\nYour verification code is: %s", userName, code)

	// Compose the email
	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: " + subject + "\n\n" +
		body

	// Set up the SMTP server configuration
	smtpServer := "smtp.gmail.com"
	smtpPort := "587"

	auth := smtp.PlainAuth("", from, password, smtpServer)

	// Connect to the server, authenticate, and send the email
	err := smtp.SendMail(smtpServer+":"+smtpPort, auth, from, []string{to}, []byte(msg))
	if err != nil {
		return err
	}

	return nil
}
