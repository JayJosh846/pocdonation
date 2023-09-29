package middleware

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"net/http"
	"os"

	// Import your utils package
	"github.com/gin-gonic/gin"
)

// PaystackWebhook is a middleware function to verify Paystack webhook signature.
func PaystackWebhook() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read the request body.
		body, err := c.GetRawData()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Error reading request body"})
			c.Abort()
			return
		}

		// Calculate the expected signature.
		secKey := os.Getenv("PAYSTACK_SEC_KEY")
		secretKey := []byte(secKey)
		h := hmac.New(sha512.New, secretKey)
		h.Write(body)
		expectedSignature := hex.EncodeToString(h.Sum(nil))

		// Get the signature from the request header.
		signature := c.GetHeader("x-paystack-signature")

		// Compare the signatures.
		if expectedSignature != signature {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Paystack signature"})
			c.Abort()
			return
		}

		// Continue with the request.
		c.Next()
	}
}
