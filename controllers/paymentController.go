package controllers

import (
	"encoding/json"
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
}
type PaymentRequest struct {
	Amount int `json:"amount"`
}

type PaymentResponse struct {
	Status  bool                `json:"status"`
	Message string              `json:"message"`
	Data    PaymentResponseData `json:"data"`
}

type PaymentResponseData struct {
	Reference   string `json:"reference"`
	CheckoutURL string `json:"checkout_url"`
}

func PaymentConstructor(paymentService services.PaymentService) PaymentController {
	return PaymentController{
		PaymentService: paymentService,
	}
}

func (pc *PaymentController) Payin(c *gin.Context) {
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
	userStruct, ok := user.(middleware.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not a valid struct"})
	}
	foundUser, err := pc.PaymentService.PaymentGetUser(userStruct.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user details"})
		return
	}
	var (
		paymentRequest  PaymentRequest
		paymentResponse PaymentResponse
	)
	if err := c.ShouldBindJSON(&paymentRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	amount := paymentRequest.Amount

	payIn, err := pc.PaymentService.Payin(amount, userQueryID, *foundUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	e := json.Unmarshal([]byte(payIn), &paymentResponse)
	if e != nil {
		log.Println("Error:", e)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"error":         false,
		"response code": 200,
		"message":       "Payment link generated successfully",
		"data":          paymentResponse,
	})
}

func (pc *PaymentController) PaymentRoute(rg *gin.RouterGroup) {
	userRoute := rg.Group("/payment")
	userRoute.POST("/payin", middleware.Authentication, pc.Payin)
	// userRoute.POST("/create", uc.CreateUser)
}
