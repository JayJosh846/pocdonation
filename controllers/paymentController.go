package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	// "io"
	"log"
	"net/http"

	"github.com/JayJosh846/donationPlatform/middleware"
	"github.com/JayJosh846/donationPlatform/models"
	"github.com/JayJosh846/donationPlatform/services"
	helper "github.com/JayJosh846/donationPlatform/utils"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
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
	BankService        services.BankService
}

func PaymentConstructor(
	paymentService services.PaymentService,
	userService services.UserService,
	transactionService services.TransactionService,
	donationService services.DonationService,
	bankService services.BankService,

) PaymentController {
	return PaymentController{
		PaymentService:     paymentService,
		UserService:        userService,
		TransactionService: transactionService,
		DonationService:    donationService,
		BankService:        bankService,
	}
}

type PaymentRequest struct {
	Email       string `json:"email" validate:"required"`
	Donor_Email string `json:"donor_email" validate:"required"`
	Amount      string `json:"amount" validate:"required"`
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

type BankResponse struct {
	Status  bool               `json:"status"`
	Message string             `json:"message"`
	Data    []BankResponseData `json:"data"`
}

type BankResponseData struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	Code string `json:"code"`
}

type PayoutRequest struct {
	Amount int `json:"amount" validate:"required"`
}

type TransferRecipientResponse struct {
	Status  bool                          `json:"status"`
	Message string                        `json:"message"`
	Data    TransferRecipientResponseData `json:"data"`
}

type TransferRecipientResponseData struct {
	Active         bool   `json:"active"`
	Recipient_code string `json:"recipient_code"`
}
type Transfer struct {
	Status bool `json:"status"`
}

var ValidatePaymentBody = validator.New()

func (pc *PaymentController) Payin(c *gin.Context) {
	var ct, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	var (
		paymentRequest  PaymentRequest
		paymentResponse PaymentResponse
		createTrans     models.Transaction
		createDonor     models.User
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
	createTrans.Donor_Email = &paymentRequest.Donor_Email
	createTrans.User_ID = foundUser.User_ID
	createTrans.User_Full_name = foundUser.Fullname
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
	count, err := UserCollection.CountDocuments(ct, bson.M{"email": paymentRequest.Donor_Email})
	if err != nil {
		log.Panic(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	if count > 0 {
		c.JSON(http.StatusFound, gin.H{
			"error":         false,
			"response code": 200,
			"message":       "Payment link generated successfully",
			"data":          paymentResponse,
		})
		return
	}

	userName, err := helper.ExtractUsernameFromEmail(paymentRequest.Donor_Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	password, ranPassErr := helper.GenerateRandomPassword(12)
	if ranPassErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ranPassErr})
		return
	}
	hashedPassword := helper.HashPassword(password)

	createDonor.Password = &hashedPassword
	createDonor.ID = primitive.NewObjectID()
	createDonor.User_ID = createDonor.ID.Hex()
	createDonor.Email = &paymentRequest.Donor_Email
	createDonor.Username = &userName
	createDonor.Role = "donor"
	createDonor.Created_At, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	createDonor.Updated_At, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	token, refreshtoken, _ := helper.TokenGenerator(createDonor.User_ID, paymentRequest.Email)
	createDonor.Token = &token
	createDonor.Refresh_Token = &refreshtoken

	createDonorErr := pc.UserService.CreateUser(&createDonor)
	if createDonorErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": createErr.Error()})
		return
	}

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
	updateErr := pc.UserService.UpdateUserBalance(paidUser, newAmount, "add")
	if updateErr != nil {
		fmt.Println("updateErr", updateErr)
	}
	updateTransErr := pc.TransactionService.UpdateTransactionStatus(&verifyRes.Data.Reference)
	if updateTransErr != nil {
		fmt.Println("updateTransErr", updateTransErr)
	}

}

func (pc *PaymentController) GetBanks(c *gin.Context) {
	var _, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	var bankResponse BankResponse
	banks, err := pc.PaymentService.GetBanks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	e := json.Unmarshal([]byte(banks), &bankResponse)
	if e != nil {
		log.Println("Error:", e)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"error":         false,
		"response code": 200,
		"message":       "List of banks generated successfully",
		"data":          bankResponse,
	})
	return
}

func (pc *PaymentController) Payout(c *gin.Context) {
	var _, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	userStruct, ok := user.(middleware.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not a valid struct"})
	}
	var (
		payout            PayoutRequest
		transferRecipient TransferRecipientResponse
		transfer          Transfer
	)
	if err := c.BindJSON(&payout); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	validationErr := Validate.Struct(payout)
	if validationErr != nil {
		fmt.Println(validationErr)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       validationErr.Error(),
			"data":          "",
		})
		return
	}
	foundBank, err := pc.BankService.GetUserBankByID(*userStruct.Id)
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "User's bank does not exist",
			"data":          "",
		})
		return
	}
	foundUser, err := pc.PaymentService.PaymentGetUser(userStruct.Email)
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "Account does not exist",
			"data":          "",
		})
		return
	}
	if foundUser.Balance < payout.Amount {
		c.JSON(http.StatusForbidden, gin.H{
			"error":         true,
			"response code": 403,
			"message":       "You cannot withdraw an amount greater than your balance",
			"data":          "",
		})
		return
	}
	transferReciept, err := pc.PaymentService.TransferRecipientCreation(
		*foundUser.Fullname,
		*foundBank.Account_Number,
		*foundBank.Bank_Name,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	e := json.Unmarshal([]byte(transferReciept), &transferRecipient)
	if e != nil {
		log.Println("Error:", e)
		return
	}
	fmt.Println("tranferRecipient", transferRecipient)
	if transferRecipient.Status && transferRecipient.Data.Active == true {
		newAmount := payout.Amount * 100
		transferResponse, err := pc.PaymentService.InitiateTransfer(newAmount, transferRecipient.Data.Recipient_code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		e := json.Unmarshal([]byte(transferResponse), &transfer)
		if e != nil {
			log.Println("Error:", e)
			return
		}
		fmt.Println("transfer", transfer)

		if !transfer.Status {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":         true,
				"response code": 400,
				"message":       nil,
				"data":          "Failed to complete transaction. Please try again.",
			})
			return
		}
		updateErr := pc.UserService.UpdateUserBalance(foundUser, payout.Amount, "subtract")
		if updateErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":         true,
				"response code": 400,
				"message":       updateErr.Error(),
				"data":          "Failed to update user balance. Please try again.",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"error":         false,
			"response code": 200,
			"message":       "Withdrawal operation successful",
			"data":          "",
		})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{
		"error":         true,
		"response code": 403,
		"message":       nil,
		"data":          "Something went wrong. Please try again.",
	})
	return
}

func (pc *PaymentController) PaymentRoute(rg *gin.RouterGroup) {
	paymentRoute := rg.Group("/payment")
	// {
	// 	paymentRoute.Use(middleware.CORSMiddleware())

	paymentRoute.POST("/payin", pc.Payin)
	paymentRoute.GET("/banks", pc.GetBanks)
	paymentRoute.POST("/confirmation",
		// middleware.PaystackWebhook(),
		pc.ConfirmWebhook)
	paymentRoute.POST("/payouts",
		middleware.Authentication,
		pc.Payout,
	)
	// }
}
