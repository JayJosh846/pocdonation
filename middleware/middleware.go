package middleware

import (
	// "log"
	"net/http"

	token "github.com/JayJosh846/donationPlatform/utils"

	"github.com/gin-gonic/gin"
)

type User struct {
	Id    *string
	Email *string
}

func Authentication(c *gin.Context) {
	ClientToken := c.Request.Header.Get("token")
	if ClientToken == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No Authorization Header Provided"})
		c.Abort()
		return
	}
	claims, msg, err := token.ValidateToken(ClientToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":         true,
			"response code": 401,
			"message":       msg,
			"data":          "",
		})
		c.Abort()
		return
	}
	user := User{
		Id:    &claims.Id,
		Email: &claims.Email,
	}

	c.Set("user", user)
	c.Next()

}
