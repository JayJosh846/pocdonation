package controllers

import (
	"log"
	"net/http"

	"github.com/JayJosh846/donationPlatform/middleware"
	"github.com/JayJosh846/donationPlatform/services"

	"github.com/gin-gonic/gin"
)

type PaymentError struct {
	Error        bool
	ResponseCode int
	Message      string
	Data         string
}

type PaymentController struct {
	PaymentService services.PaymentService
	UserService    services.UserService
}
type PaymentRequest struct {
	Amount *int `json:"amount"`
}

func PaymentConstructor(paymentService services.PaymentService) PaymentController {
	return PaymentController{
		PaymentService: paymentService,
	}
}

func (uc *PaymentController) Payin(c *gin.Context) {
	userQueryID := c.Query("userID")
	if userQueryID == "" {
		log.Println("user id is empty")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user id is empty"})
		return
	}
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	log.Println("userVal", user)

	userStruct, ok := user.(middleware.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not a valid struct"})
	}

	// var userString *string
	// if userVal, ok := user.(string); ok {
	// 	log.Println("userVal", userVal)
	// 	log.Println("&userVal", &userVal)
	// 	userString = &userVal
	// 	log.Println("userString", userString)

	// }
	// log.Println("user", userString)
	foundUser, err := uc.UserService.GetUser(&userStruct.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user details"})
		return
	}
	var paymentRequest PaymentRequest
	if err := c.ShouldBindJSON(&paymentRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	amount := paymentRequest.Amount

	uc.PaymentService.Payin(amount, &userQueryID, foundUser)
	c.JSON(http.StatusOK, foundUser)

}

func (uc *PaymentController) PaymentRoute(rg *gin.RouterGroup) {
	userRoute := rg.Group("/payment")
	userRoute.POST("/payin", middleware.Authentication(), uc.Payin)
	// userRoute.POST("/create", uc.CreateUser)
}
