package controllers

import (
	"encoding/json"
	"fmt"
	"strconv"

	// "io"
	"log"
	"net/http"

	"github.com/JayJosh846/donationPlatform/models"
	"github.com/JayJosh846/donationPlatform/services"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/gin-gonic/gin"
)

type PaymentError struct {
	Error        bool
	ResponseCode int
	Message      string
	Data         string
}

type PaymentController struct {
	PaymentService     services.PaymentService
	UserService        services.UserService
	TransactionService services.TransactionService
	DonationService    services.DonationService
}
type PaymentRequest struct {
	Email  string `json:"email" validate:"required"`
	Amount string `json:"amount" validate:"required"`
}

type PaymentResponse struct {
	Status  bool                `json:"status"`
	Message string              `json:"message"`
	Data    PaymentResponseData `json:"data"`
}

type PaymentResponseData struct {
	AccessCode       string `json:"access_code"`
	Reference        string `json:"reference"`
	AuthorizationURL string `json:"authorization_url"`
}

func PaymentConstructor(
	paymentService services.PaymentService,
	userService services.UserService,
	transactionService services.TransactionService,
	donationService services.DonationService,

) PaymentController {
	return PaymentController{
		PaymentService:     paymentService,
		UserService:        userService,
		TransactionService: transactionService,
		DonationService:    donationService,
	}
}

var ValidatePaymentBody = validator.New()

func (pc *PaymentController) Payin(c *gin.Context) {
	var (
		paymentRequest  PaymentRequest
		paymentResponse PaymentResponse
		createTrans     models.Transaction
		// createDonation  models.Donation
	)
	if err := c.ShouldBindJSON(&paymentRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	validationErr := ValidatePaymentBody.Struct(paymentRequest)
	if validationErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 500,
			"message":       "Enter the required fields",
			"data":          "",
		})
		return
	}
	foundUser, err := pc.PaymentService.PaymentGetUser(&paymentRequest.Email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":         true,
			"response code": 404,
			"message":       "Failed to retrieve user details",
			"data":          "",
		})
		return
	}
	amount := paymentRequest.Amount
	// Parse the string to an integer
	amountInt, err := strconv.Atoi(amount)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	// Multiply by 100
	newAmountInt := amountInt * 100
	// Convert the integer back to a string
	newAmountStr := strconv.Itoa(newAmountInt)

	payIn, err := pc.PaymentService.Payin(newAmountStr, *foundUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	e := json.Unmarshal([]byte(payIn), &paymentResponse)
	if e != nil {
		log.Println("Error:", e)
		return
	}
	createTrans.ID = primitive.NewObjectID()
	createTrans.Reference = &paymentResponse.Data.Reference
	createTrans.Amount = paymentRequest.Amount
	createTrans.Status = "pending"
	createErr := pc.TransactionService.CreateTransaction(&createTrans)
	if createErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       createErr.Error(),
			"data":          "",
		})
		return
	}
	// createDonation.ID = primitive.NewObjectID()
	// createDonation.User_ID = foundUser.User_ID
	// createDonation.Transaction_Reference = &paymentResponse.Data.Reference
	// createDonation.Amount = paymentRequest.Amount
	// createDonation.Status = "pending"
	// createDonErr := pc.DonationService.CreateDonation(&createDonation)
	// if createDonErr != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{
	// 		"error":         true,
	// 		"response code": 400,
	// 		"message":       createDonErr.Error(),
	// 		"data":          "",
	// 	})
	// 	return
	// }
	c.JSON(http.StatusFound, gin.H{
		"error":         false,
		"response code": 200,
		"message":       "Payment link generated successfully",
		"data":          paymentResponse,
	})
}

func (pc *PaymentController) ConfirmWebhook(c *gin.Context) {
	var eventData map[string]interface{}
	if err := c.ShouldBindJSON(&eventData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	jsonData, err := json.Marshal(eventData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	verifyRes, verifyErr := pc.PaymentService.VerifyDeposit(jsonData)
	if verifyErr != nil {
		fmt.Println("verifyErr", verifyErr)
	}
	paidUser, err := pc.UserService.GetUser(&verifyRes.Data.Customer.Email)
	if err != nil {
		fmt.Println("err", err)
	}
	newAmount := verifyRes.Data.Amount / 100
	updateErr := pc.UserService.UpdateUserBalance(paidUser, newAmount)
	if updateErr != nil {
		fmt.Println("updateErr", updateErr)
	}
	updateTransErr := pc.TransactionService.UpdateTransactionStatus(&verifyRes.Data.Reference)
	if updateTransErr != nil {
		fmt.Println("updateTransErr", updateTransErr)
	}
	updateDonationsErr := pc.DonationService.UpdateDonationStatus(&verifyRes.Data.Reference)
	if updateDonationsErr != nil {
		fmt.Println("updateDonationsErr", updateDonationsErr)
	}
	c.JSON(http.StatusOK, gin.H{"message": "success"})

}

// func (pc *PaymentController) Payin(c *gin.Context) {
// 	userQueryID := c.Query("userID")
// 	if userQueryID == "" {
// 		log.Println("user id is empty")
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "user id is empty"})
// 		return
// 	}
// 	user, exists := c.Get("user")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
// 		return
// 	}
// 	userStruct, ok := user.(middleware.User)
// 	if !ok {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not a valid struct"})
// 	}
// 	foundUser, err := pc.PaymentService.PaymentGetUser(userStruct.Email)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user details"})
// 		return
// 	}
// 	var (
// 		paymentRequest  PaymentRequest
// 		paymentResponse PaymentResponse
// 	)
// 	if err := c.ShouldBindJSON(&paymentRequest); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}
// 	validationErr := ValidatePaymentBody.Struct(paymentRequest)
// 	if validationErr != nil {
// 		fmt.Println(validationErr)
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error":         true,
// 			"response code": 400,
// 			"message":       "Enter the required fields",
// 			"data":          "",
// 		})
// 		return
// 	}
// 	amount := paymentRequest.Amount

// 	payIn, err := pc.PaymentService.Payin(amount, userQueryID, *foundUser)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 	}
// 	e := json.Unmarshal([]byte(payIn), &paymentResponse)
// 	if e != nil {
// 		log.Println("Error:", e)
// 		return
// 	}
// 	c.JSON(http.StatusFound, gin.H{
// 		"error":         false,
// 		"response code": 302,
// 		"message":       "Payment link generated successfully",
// 		"data":          paymentResponse,
// 	})
// }

func (pc *PaymentController) PaymentRoute(rg *gin.RouterGroup) {
	userRoute := rg.Group("/payment")
	userRoute.POST("/payin",
		// middleware.Authentication,
		pc.Payin)
	userRoute.POST("/confirmation",
		// middleware.PaystackWebhook(),
		pc.ConfirmWebhook)
	// userRoute.POST("/create", uc.CreateUser)
}
