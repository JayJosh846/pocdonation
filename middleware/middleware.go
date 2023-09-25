package middleware

import (
	// "log"
	"net/http"

	token "github.com/JayJosh846/donationPlatform/utils"

	"github.com/gin-gonic/gin"
)

type User struct {
	Id    string
	Email string
}

func Authentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		ClientToken := c.Request.Header.Get("token")
		if ClientToken == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No Authorization Header Provided"})
			c.Abort()
			return
		}
		claims, err := token.ValidateToken(ClientToken)
		if err != "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			c.Abort()
			return
		}
		user := User{
			Id:    claims.Id,
			Email: claims.Email,
		}
		c.Set("user", user)
		c.Next()
	}
}
