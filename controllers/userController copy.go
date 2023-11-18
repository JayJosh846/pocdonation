package controllers

import (
	"context"
	// "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"net/url"

	"time"

	"cloud.google.com/go/storage"
	"github.com/JayJosh846/donationPlatform/database"
	"github.com/JayJosh846/donationPlatform/middleware"
	"github.com/JayJosh846/donationPlatform/models"
	"github.com/JayJosh846/donationPlatform/services"
	generate "github.com/JayJosh846/donationPlatform/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"google.golang.org/api/option"
	"google.golang.org/appengine"
)

var UserCollection *mongo.Collection = database.GetUserCollection(database.Client, "Users")
var BankCollection *mongo.Collection = database.GetUserCollection(database.Client, "Banks")
var OtpCollection *mongo.Collection = database.GetUserCollection(database.Client, "Otps")
var KycCollection *mongo.Collection = database.GetUserCollection(database.Client, "Kycs")
var SocialCollection *mongo.Collection = database.GetUserCollection(database.Client, "Socials")

var db = database.GetDBInstance(database.Client)
var Validate = validator.New()

type UserController struct {
	UserService        services.UserService
	TransactionService services.TransactionService
	DonationService    services.DonationService
	BankService        services.BankService
	PaymentService     services.PaymentService
}

func Constructor(
	userService services.UserService,
	transactionService services.TransactionService,
	donationService services.DonationService,
	bankService services.BankService,
	paymentService services.PaymentService,
) UserController {
	return UserController{
		UserService:        userService,
		TransactionService: transactionService,
		DonationService:    donationService,
		BankService:        bankService,
		PaymentService:     paymentService,
	}
}

type BankRequest struct {
	Account_bank   string `json:"account_bank" validate:"required"`
	Account_number string `json:"account_number" validate:"required"`
}

type AccountVerificationResponse struct {
	Status  bool                            `json:"status"`
	Message string                          `json:"message"`
	Data    AccountVerificationResponseData `json:"data"`
}

type AccountVerificationResponseData struct {
	AccountNumber string `json:"account_number"`
	AccountName   string `json:"account_name"`
	BankId        int    `json:"bank_id"`
}

type EmailVerificationRequest struct {
	Phone string `json:"phone"`
	Email string `json:"email"`
}

type VerificationCodeRequest struct {
	Code string `json:"code"`
}

type SelfieRequest struct {
	Pic string `json:"pic"`
}

type BVNRequest struct {
	Bvn string `json:"bvn"`
}

type LookUpBVNResponse struct {
	Status int `json:"status"`
}

type VerifyBVNResponse struct {
	Status  int                     `json:"status"`
	Message string                  `json:"message"`
	Data    VerifyBVNResponseEntity `json:"data"`
}

type VerifyBVNResponseEntity struct {
	Entity VerifyBVNResponseBvn `json:"entity"`
}

type VerifyBVNResponseBvn struct {
	Bvn VerifyBVNResponseData `json:"bvn"`
}

type VerifyBVNResponseData struct {
	Status bool `json:"status"`
}

type KycFileTypeRequest struct {
	Document_Type string `json:"document_type"`
	Document      string `json:"document"`
}

type UserProfile struct {
	User_ID         string             `json:"user_id"`
	Fullname        *string            `json:"full_name"`
	Email           *string            `json:"email"`
	Bio             *string            `json:"bio"`
	Username        *string            `json:"username"`
	Balance         int                `json:"balance"`
	Profile_Picture *string            `json:"profile_picture"`
	Identification  *string            `json:"identification"`
	Social          *models.Social     `json:"socials"`
	Donation        []*models.Donation `json:"donations"`
}

type Usernames struct {
	User_ID         string  `json:"user_id"`
	Username        *string `json:"username"`
	Profile_Picture *string `json:"profile_picture"`
}

func (uc *UserController) Signup(ctx *gin.Context) {
	var ct, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	var user models.User
	if err := ctx.BindJSON(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "err.Error()"})
		return
	}
	validationErr := Validate.Struct(user)
	if validationErr != nil {
		fmt.Println(validationErr)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       validationErr.Error(),
			"data":          "",
		})
		return
	}

	count, err := UserCollection.CountDocuments(ct, bson.M{"email": user.Email})
	if err != nil {
		log.Panic(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	if count > 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "User already exists",
			"data":          "",
		})
		return
	}
	count, err = UserCollection.CountDocuments(ct, bson.M{"phone": user.Phone})
	defer cancel()
	if err != nil {
		log.Panic(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	if count > 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "Phone number is already in use",
			"data":          "",
		})
		return
	}
	password := generate.HashPassword(*user.Password)
	user.Password = &password
	userName, err := generate.ExtractUsernameFromEmail(*user.Email)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	baseURL := "https://donation-platform.netlify.app/user/"
	link := fmt.Sprintf("%s%s", baseURL, userName)

	user.ID = primitive.NewObjectID()
	user.User_ID = user.ID.Hex()
	user.Username = &userName
	user.Link = &link
	user.Role = "user"
	user.Email_Verified = false
	user.Selfie_Upload = false
	user.Bvn_Verified = false
	user.ID_Upload = false
	user.Kyc_Status = false
	user.Created_At, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	user.Updated_At, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	token, refreshtoken, _ := generate.TokenGenerator(user.User_ID, *user.Email)
	user.Token = &token
	user.Refresh_Token = &refreshtoken
	user.Transactions = make([]models.Transaction, 0)

	createErr := uc.UserService.CreateUser(&user)
	if createErr != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"message": createErr.Error()})
		return
	}

	defer cancel()
	ctx.JSON(http.StatusCreated, gin.H{
		"error":         false,
		"response code": 201,
		"message":       "Account created successfully",
		"data":          "",
	})
}

func (uc *UserController) Login(ctx *gin.Context) {
	var _, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	var user models.User
	if err := ctx.BindJSON(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	foundUser, err := uc.UserService.GetUser(user.Email)

	defer cancel()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "Account does not exist",
			"data":          err,
		})
		return
	}

	var bank models.Bank
	query := bson.M{"user_id": foundUser.User_ID}
	er := BankCollection.FindOne(ctx, query).Decode(&bank)
	if er != nil {
		fmt.Println("Bank details not found for user:", foundUser.User_ID)
	}
	foundUser.Banks = bank
	PasswordIsValid, msg := generate.VerifyPassword(*user.Password, *foundUser.Password)
	defer cancel()
	if !PasswordIsValid {
		ctx.JSON(http.StatusBadRequest, msg)
		fmt.Println("msg", msg)
		return
	}
	token, refreshToken, _ := generate.TokenGenerator(foundUser.User_ID, *foundUser.Email)
	defer cancel()
	generate.UpdateAllTokens(token, refreshToken, foundUser.User_ID)
	ctx.JSON(http.StatusFound, gin.H{
		"error":         false,
		"response code": 302,
		"message":       "Login successfully",
		"data":          foundUser,
	})
}

func (uc *UserController) RefreshToken(c *gin.Context) {
	// Extract refresh token from the request
	refreshToken := c.Request.Header.Get("refresh-token")
	// Validate refresh token
	claims, msg, err := generate.ValidateToken(refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": msg})
		return
	}
	// Generate new access and refresh tokens
	newToken, newRefreshToken, err := generate.TokenGenerator(claims.Id, claims.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating new tokens"})
		return
	}
	// Update tokens in the database
	generate.UpdateAllTokens(newToken, newRefreshToken, claims.Id)
	// Return the new tokens to the client
	c.JSON(http.StatusOK, gin.H{
		"error":         false,
		"response code": 200,
		"message":       "User session refreshed",
		"data": gin.H{
			"token": newToken,
			// "refresh_token": newRefreshToken,
		},
	})
}

func (uc *UserController) Donation(c *gin.Context) {
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
	var donation models.Donation
	if err := c.BindJSON(&donation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	validationErr := Validate.Struct(donation)
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
	foundUser, err := uc.UserService.GetUser(userStruct.Email)
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "User does not exist",
			"data":          "",
		})
		return
	}
	donation.ID = primitive.NewObjectID()
	donation.User_ID = foundUser.User_ID
	// donation.Amount = donation.Amount
	donation.Created_At, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	donation.Updated_At, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	createErr := uc.DonationService.CreateDonation(&donation)
	if createErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": createErr.Error()})
		return
	}
	defer cancel()
	c.JSON(http.StatusCreated, gin.H{
		"error":         false,
		"response code": 201,
		"message":       "Donation created successfully",
		"data":          "",
	})
}

func (uc *UserController) Socials(c *gin.Context) {
	var ct, cancel = context.WithTimeout(context.Background(), 100*time.Second)
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
	var social models.Social
	if err := c.BindJSON(&social); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	validationErr := Validate.Struct(social)
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
	foundUser, err := uc.UserService.GetUser(userStruct.Email)
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "User does not exist",
			"data":          "",
		})
		return
	}

	count, err := SocialCollection.CountDocuments(ct, bson.M{"user_id": foundUser.User_ID})
	if err != nil {
		log.Panic(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "User social details already exists",
			"data":          "",
		})
		return
	}
	social.ID = primitive.NewObjectID()
	social.User_ID = foundUser.User_ID
	createErr := uc.UserService.CreateSocial(&social)
	if createErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": createErr.Error()})
		return
	}
	defer cancel()
	c.JSON(http.StatusCreated, gin.H{
		"error":         false,
		"response code": 201,
		"message":       "User socials created successfully",
		"data":          "",
	})
}

func (uc *UserController) GetUserTransaction(c *gin.Context) {
	var _, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	_, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	userId := c.Query("id")
	foundUser, err := uc.TransactionService.GetUserTransactionsByID(userId)
	fmt.Println("foundUser", foundUser)
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "User does not exist",
			"data":          "",
		})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"error":         false,
		"response code": 201,
		"message":       "User transactions retrieved successfully",
		"data":          foundUser,
	})

}

func (uc *UserController) AddBank(c *gin.Context) {
	var ct, cancel = context.WithTimeout(context.Background(), 100*time.Second)
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
		bankRequest     BankRequest
		accountResponse AccountVerificationResponse
		addBank         models.Bank
	)
	if err := c.BindJSON(&bankRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	validationErr := Validate.Struct(bankRequest)
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

	foundUser, err := uc.UserService.GetUser(userStruct.Email)
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "User does not exist",
			"data":          "",
		})
		return
	}
	count, err := BankCollection.CountDocuments(ct, bson.M{"user_id": foundUser.User_ID})
	if err != nil {
		log.Panic(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "User bank details already exists",
			"data":          "",
		})
		return
	}

	bank, err := uc.PaymentService.VerifyAccountNumber(bankRequest.Account_number, bankRequest.Account_bank)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	e := json.Unmarshal([]byte(bank), &accountResponse)
	if e != nil {
		log.Println("Error:", e)
		return
	}

	addBank.ID = primitive.NewObjectID()
	addBank.User_ID = foundUser.User_ID
	addBank.Account_Number = &bankRequest.Account_number
	addBank.Account_Name = &accountResponse.Data.AccountName
	addBank.Bank_Name = &bankRequest.Account_bank

	createErr := uc.BankService.AddBank(&addBank)
	if createErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       createErr.Error(),
			"data":          "",
		})
		return
	}

	c.JSON(http.StatusFound, gin.H{
		"error":         false,
		"response code": 200,
		"message":       "User bank details added successfully",
		"data":          addBank,
	})
}

func (uc *UserController) GetBankDetails(c *gin.Context) {
	var _, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	_, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	userId := c.Query("id")
	foundBank, err := uc.BankService.GetUserBankByID(userId)
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "User bank not found",
			"data":          "",
		})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"error":         false,
		"response code": 201,
		"message":       "User bank details retrieved successfully",
		"data":          foundBank,
	})

}

func (uc *UserController) RequestEmailVerification(c *gin.Context) {
	var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
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
		emailVerificationRequest EmailVerificationRequest
		kyc                      models.KYC
	)

	if err := c.BindJSON(&emailVerificationRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	validationErr := Validate.Struct(emailVerificationRequest)
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
	foundUser, err := uc.UserService.GetUser(userStruct.Email)
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "User does not exist",
			"data":          "",
		})
		return
	}

	// count, err := UserCollection.CountDocuments(ctx, bson.M{"email": emailVerificationRequest.Email})
	// if err != nil {
	// 	log.Panic(err)
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": err})
	// 	return
	// }
	// if count > 0 {
	// 	c.JSON(http.StatusBadRequest, gin.H{
	// 		"error":         true,
	// 		"response code": 400,
	// 		"message":       "User with this email exists",
	// 		"data":          "",
	// 	})
	// 	return
	// } else if count == 0 || emailVerificationRequest.Email == *foundUser.Email {
	returnedOtp, err := uc.UserService.CreateEmailVerification(foundUser, emailVerificationRequest.Email)
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "Error sending email verification",
			"data":          "",
		})
		return
	}

	updateUserEmail := uc.UserService.UpdateUserEmailPhone(foundUser.User_ID, emailVerificationRequest.Email, emailVerificationRequest.Phone)
	if updateUserEmail != nil {
		c.JSON(http.StatusFound, gin.H{
			"error":         false,
			"response code": 200,
			"message":       "Something went wrong while updating user email",
			"data":          updateUserEmail,
		})
		return
	}
	kyc.ID = primitive.NewObjectID()
	kyc.User_ID = foundUser.User_ID
	kyc.Tier = 1
	kyc.Status = "ongoing"

	_, kycErr := KycCollection.InsertOne(ctx, kyc)
	if kycErr != nil {
		c.JSON(http.StatusFound, gin.H{
			"error":         false,
			"response code": 200,
			"message":       "Something went wrong while updating user kyc status",
			"data":          kycErr,
		})
		return
	}
	c.JSON(http.StatusFound, gin.H{
		"error":         false,
		"response code": 200,
		"message":       "OTP sent successfully",
		"data":          returnedOtp,
	})
	// }
}

func (uc *UserController) EmailVerification(c *gin.Context) {
	var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
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
		verificationCodeRequest VerificationCodeRequest
	)

	if err := c.BindJSON(&verificationCodeRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	validationErr := Validate.Struct(verificationCodeRequest)
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
	foundUser, err := uc.UserService.GetUser(userStruct.Email)
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "User does not exist",
			"data":          "",
		})
		return
	}
	if foundUser.Email_Verified {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "Email already verified",
			"data":          "",
		})
		return
	}
	var otp models.Otp
	query := bson.M{"token": verificationCodeRequest.Code}
	er := OtpCollection.FindOne(ctx, query).Decode(&otp)
	if er != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "OTP not valid",
			"data":          "",
		})
		return
	}
	if time.Now().After(otp.Expires_At) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "OTP expired",
			"data":          "",
		})
		return
	}
	updateUserEmail := uc.UserService.UpdateUserEmailStatus(foundUser.User_ID)
	if updateUserEmail != nil {
		c.JSON(http.StatusFound, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "Something went wrong while updating user email",
			"data":          updateUserEmail,
		})
		return
	}
	c.JSON(http.StatusFound, gin.H{
		"error":         false,
		"response code": 200,
		"message":       "Email Verified successfully",
		"data":          "",
	})
}

func (uc *UserController) Userselfie(c *gin.Context) {
	bucket := "poc-donation-bucket1"
	ctxAppEngine := appengine.NewContext(c.Request)
	var err error

	var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
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
		storageClient *storage.Client
	)

	storageClient, err = storage.NewClient(ctxAppEngine, option.WithCredentialsFile("poc-donation-1fd74214994c.json"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":         true,
			"response code": 500,
			"message":       err.Error(),
		})
		return
	}

	// storageClient, err = storage.NewClient(ctxAppEngine, option.WithCredentialsFile("poc-donation-176e6cbcd64b.json"))
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{
	// 		"error":         true,
	// 		"response code": 500,
	// 		"message":       err.Error(),
	// 	})
	// 	return
	// }

	file, header, err := c.Request.FormFile("image")
	if err != nil {
		fmt.Println("err", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()

	if file == nil || header == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sw := storageClient.Bucket(bucket).Object(header.Filename).NewWriter(ctxAppEngine)

	if _, err := io.Copy(sw, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":         true,
			"response code": 500,
			"message":       err.Error(),
		})
		return
	}

	if err := sw.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":         true,
			"response code": 500,
			"message":       err.Error(),
		})
		return
	}

	_, errr := url.Parse("/" + bucket + "/" + sw.Attrs().Name)
	if errr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":         true,
			"response code": 500,
			"message":       errr.Error(),
		})
		return
	}

	// Get a GridFS bucket
	// fs, err := gridfs.NewBucket(db)
	// if err != nil {
	// 	fmt.Println("err", err)
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	// 	return
	// }

	// // Create an upload stream with some options and upload the file
	// uploadStream, err := fs.OpenUploadStream(header.Filename)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	// 	return
	// }
	// defer uploadStream.Close()

	// // Copy the file data to GridFS
	// if _, err = io.Copy(uploadStream, file); err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	// 	return
	// }
	foundUser, err := uc.UserService.GetUser(userStruct.Email)
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "User does not exist",
			"data":          "",
		})
		return
	}
	filterUser := bson.D{primitive.E{Key: "user_id", Value: foundUser.User_ID}}
	updateUser := bson.D{
		primitive.E{
			Key: "$set",
			Value: bson.D{
				primitive.E{Key: "selfie_upload", Value: true},
				// primitive.E{Key: "profile_picture", Value: true},
			},
		},
	}
	resultUser, _ := UserCollection.UpdateOne(ctx, filterUser, updateUser)
	if resultUser.MatchedCount != 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "no matched document found for update",
			"data":          "",
		})
		return
	}

	filter := bson.D{primitive.E{Key: "user_id", Value: foundUser.User_ID}}
	update := bson.D{
		primitive.E{
			Key: "$set",
			Value: bson.D{
				primitive.E{Key: "kyc_image", Value: header.Filename},
			},
		},
	}
	result, _ := KycCollection.UpdateOne(ctx, filter, update)
	if result.MatchedCount != 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "no matched document found for update",
			"data":          "",
		})
		return
	}

	// updatePic := uc.UserService.UpdateUserPicture(foundUser.User_ID, selfieRequest.Pic)
	// if updatePic != nil {
	// 	c.JSON(http.StatusFound, gin.H{
	// 		"error":         false,
	// 		"response code": 200,
	// 		"message":       "Something went wrong while updating user email",
	// 		"data":          updatePic,
	// 	})
	// 	return
	// }
	c.JSON(http.StatusFound, gin.H{
		"error":         false,
		"response code": 200,
		"message":       "Profile picture updated successfully",
		"data":          "",
	})

}

func (uc *UserController) UserProfile(c *gin.Context) {
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
	foundUser, err := uc.UserService.GetUser(userStruct.Email)
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "User does not exist",
			"data":          "",
		})
		return
	}
	c.JSON(http.StatusFound, gin.H{
		"error":         false,
		"response code": 200,
		"message":       "User profile retrieved successfully",
		"data":          foundUser,
	})
}

func (uc *UserController) UserProfileNoAuth(c *gin.Context) {
	// bucket := "poc-donation-bucket1"
	// ctxAppEngine := appengine.NewContext(c.Request)

	var _, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	user := c.Query("id")
	if user == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":         true,
			"response code": 401,
			"message":       "No userId provided",
		})
		return
	}

	var (
		userProfile UserProfile
	)

	foundUser, err := uc.UserService.GetUserByID(user)
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "User does not exist",
			"data":          "",
		})
		return
	}

	// foundKyc, _ := uc.UserService.GetUserKycByID(user)
	// defer cancel()

	// fs, err := gridfs.NewBucket(db)
	// if err != nil {
	// 	fmt.Println("err", err)
	// }

	// downloadStream, err := fs.OpenDownloadStreamByName(*foundKyc.Kyc_Image)
	// if err != nil {
	// 	fmt.Println("err", err)
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Error opening download stream"})
	// 	return
	// }
	// defer downloadStream.Close()

	// Get the length of the download stream
	// fileSize := downloadStream.GetFile().Length
	// photoContent := make([]byte, fileSize)

	// // Read the photo content into the byte slice
	// if _, err := io.ReadFull(downloadStream, photoContent); err != nil {
	// 	fmt.Println("err", err)
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading photo content"})
	// 	return
	// }
	// base64Photo := base64.StdEncoding.EncodeToString(photoContent)

	// fmt.Println("photoContent", base64Photo)

	foundSocial, _ := uc.UserService.GetUserSocialsByID(user)
	defer cancel()

	foundDonations, _ := uc.UserService.GetUserDonationsByID(user)
	defer cancel()
	userProfile.Balance = foundUser.Balance
	userProfile.Bio = foundUser.Bio
	userProfile.Email = foundUser.Email
	userProfile.Fullname = foundUser.Fullname
	userProfile.Identification = foundUser.Identification
	userProfile.Profile_Picture = foundUser.Profile_Picture
	userProfile.User_ID = foundUser.User_ID
	userProfile.Username = foundUser.Username
	userProfile.Social = foundSocial
	userProfile.Donation = foundDonations
	c.JSON(http.StatusFound, gin.H{
		"error":         false,
		"response code": 200,
		"message":       "User profile retrieved successfully",
		"data":          userProfile,
	})
}

func (uc *UserController) VerifyBVN(c *gin.Context) {
	bucket := "poc-donation-bucket1"
	ctxAppEngine := appengine.NewContext(c.Request)

	var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
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
		bvnRequest    BVNRequest
		verifyResonse VerifyBVNResponse
	)

	if err := c.BindJSON(&bvnRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	validationErr := Validate.Struct(bvnRequest)
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
	// bvnLookup, err := services.LookUpBVN(bvnRequest.Bvn)
	// if err != nil {
	// 	c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	// 	return
	// }
	// e := json.Unmarshal([]byte(bvnLookup), &lookUpBvn)
	// if e != nil {
	// 	log.Println("Error:", e)
	// 	return
	// }
	// if lookUpBvn.Status != 200 {
	// 	c.JSON(http.StatusNotFound, gin.H{
	// 		"error":         true,
	// 		"response code": 400,
	// 		"message":       "Invalid BVN entered",
	// 		"data":          "",
	// 	})
	// 	return
	// }

	bvnVerify, err := services.VerifyBVN(bvnRequest.Bvn)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	er := json.Unmarshal([]byte(bvnVerify), &verifyResonse)
	if er != nil {
		log.Println("Error:", er)
		return
	}
	if !verifyResonse.Data.Entity.Bvn.Status {
		c.JSON(http.StatusNotFound, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "Could not verify BVN. Please try again",
			"data":          "",
		})
		return
	}

	storageClient, err := storage.NewClient(ctxAppEngine, option.WithCredentialsFile("poc-donation-1fd74214994c.json"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":         true,
			"response code": 500,
			"message":       err.Error(),
		})
		return
	}
	defer storageClient.Close()

	foundKyc, _ := uc.UserService.GetUserKycByID(*userStruct.Id)
	defer cancel()

	objectHandle := storageClient.Bucket(bucket).Object(*foundKyc.Kyc_Image)
	// Check if the file exists
	if _, err := objectHandle.Attrs(ctxAppEngine); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "Image not found",
		})
		return
	}

	url, err := storage.SignedURL(bucket, *foundKyc.Kyc_Image, &storage.SignedURLOptions{
		GoogleAccessID: "poc-donation-service1@poc-donation.iam.gserviceaccount.com",                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   // Replace with your Google Cloud Storage service account's email
		PrivateKey:     []byte("-----BEGIN PRIVATE KEY-----\nMIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQDnM6/2VxT7P/rA\n2Lp/bICJrRWFiBxVR2vAXkWCpj86YN3me8dnorLwBkDUByLgWTtg91Ls4WikjB2y\ncxphwk0Bg0T5oqlp+zsNPZ6XKUReB2rqg3QJ9839V4uzJSlverROiHz/3Hl/V7b7\n28lz2+R2peoTv+3GuGVKx+cMWcl9gX/GKtT1JjqnKMz5TnDMU/obfmzIC8I8Xprj\nmPnp1Dc55C3O9eVdV7L4aarEql0/LZAoG/9UlG8oz1/mgeWeQs/XFDRemVcZ3uIC\njnOjOTr4zutuEfHkb1En1NSaSvNmG8JgxcDnpsUSf9kPZLBmaajHiu4/yAtQeeCh\nJ/nmBRr1AgMBAAECggEACMb1jyyPJ1quclPIAL5lwtRHVOJt8O7dMFhj2ynkjJrQ\n0ccxMsYCdQpHu8TplgrNLkk1ZLjJ+DU5i2TDQ6LUuZH6NF/wfo2DGGWWd7ahWdB+\nRpjm9tnpgAyqyQpIIGtQHQshc7UzB5qU38rgQv2+FqMF1+oZZMnrToN4Sge+ln0Z\nmTOH8DuqRTFGCiTKpFeucFgsHm6uEuaSVJtKaqiS77yTsF4TazprirHu+WbeC7UE\nM8wVlpx1smpPLcDBfP68vZxRNY5wKHGqEWJLJf+PiI6Wn9LU/elU5XIelORx85X3\nqNKUTuL2Yp1+S7deMAlgqsRjdUoPs2MeNB7ucbKhoQKBgQD7kBVWhIYDVFfmstJg\n1yzeG9pDac3pwi2KW5cI834agLk2/i/nPF6omZDGUuXCK7RqvjFcyPTmWhZjQJpX\ndDeqRLLPo4bPv735XN/KZkUVDArGa8JPBS7YnlP3G1M8e4GcK8bnjUU8i410NC38\ndLjFrVJOPnnynWP6B2KnKkm8qwKBgQDrR6pd8yLW7TQrEYL6cGLpyKIOWQPC6amH\nj6kkNWlKya5Q2ng6FavKdie+PHw5rekaoQnFqd55SIxfa3evUlrQVwE0y5JaMZtM\nwgs194e/BNSxG63NmXTdFl0OKd8jkbXNbmK+3IOQL0szV7/5JmrYzFMmAY9vZJie\nPd9mbJlG3wKBgG22Kvgul9u/3w4oEwRVE6ZSc2BPNpSqMP5Ub4xh1S9t0FkhhnbM\np2PUhYVZBgcm1GpxREn5AoWr6HOk6ysU7mn9yBYydUsJjqrATIGTFLHXLKPYv0eD\nNSkX8/qjGiwYmTApD3hQ7k83dZumXh/qL+NWcbzaFokvBzk2G1pYYQw9AoGAPJXb\nvQ2a7xVt1ZlQzQSbs+/CK0eovExHJ21K9NP8JRICHTfktbBW6G+8lDQnGQM7f2vw\nhEHV1A1meDvIOhFO6U8+NEYnjaowf3eIQ4FWJ04rJuAlxUe63COiGr+VgidHVXsT\nWmqWRk6nYrU57gKCiQk0cBj+woR4+GaeXFWisqkCgYB6wto3zrMzeROo0D1jmRPU\nKqce7x+8ZaDlge/z3TpKCAp+ztzJybxYIJvdFkz+CPMyLGkcwTRb7QD86UuKfWQR\ncwpBmRFw41RqOnCrhmSi9QXvuBJ1RU1knQLKwoDN8Ck76rACWlXhat/KIroWFCtO\nQ3xb+767sc9b2JZI/0/1PQ==\n-----END PRIVATE KEY-----\n"), // Replace with your Google Cloud Storage service account's private key
		Method:         http.MethodGet,
		Expires:        time.Now().AddDate(100, 0, 0), // Set expiration to 100 years in the future
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":         true,
			"response code": 500,
			"message":       "Error generating signed URL",
		})
		return
	}

	foundUser, err := uc.UserService.GetUser(userStruct.Email)
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "User does not exist",
			"data":          "",
		})
		return
	}

	filterUser := bson.D{primitive.E{Key: "user_id", Value: foundUser.User_ID}}
	updateUser := bson.D{
		primitive.E{
			Key: "$set",
			Value: bson.D{
				primitive.E{Key: "bvn_verified", Value: true},
				primitive.E{Key: "profile_picture", Value: url},
			},
		},
	}
	resultUser, _ := UserCollection.UpdateOne(ctx, filterUser, updateUser)
	if resultUser.MatchedCount != 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "no matched document found for update",
			"data":          "",
		})
		return
	}

	filter := bson.D{primitive.E{Key: "user_id", Value: foundUser.User_ID}}
	update := bson.D{
		primitive.E{
			Key: "$set",
			Value: bson.D{
				primitive.E{Key: "tier", Value: 2},
			},
		},
	}
	result, _ := KycCollection.UpdateOne(ctx, filter, update)
	if result.MatchedCount != 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "no matched document found for update",
			"data":          "",
		})
		return
	}

	c.JSON(http.StatusFound, gin.H{
		"error":         false,
		"response code": 200,
		"message":       "BVN verified successfully",
		"data":          "",
	})

}

func (uc *UserController) KycFileUpload(c *gin.Context) {
	var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
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
		kycFileTypeRequest KycFileTypeRequest
		// r                  *http.Request
		// w                  http.ResponseWriter
	)

	// if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 	return
	// }

	file, header, err := c.Request.FormFile("document")
	if err != nil {
		fmt.Println("err", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()
	if file == nil || header == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get a GridFS bucket
	fs, err := gridfs.NewBucket(db)
	if err != nil {
		fmt.Println("err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create an upload stream with some options and upload the file
	uploadStream, err := fs.OpenUploadStream(header.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer uploadStream.Close()

	// Copy the file data to GridFS
	if _, err = io.Copy(uploadStream, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	foundUser, err := uc.UserService.GetUser(userStruct.Email)
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "User does not exist",
			"data":          "",
		})
		return
	}

	updateDocs := uc.UserService.UpdateUserKYCStatus(foundUser.User_ID, kycFileTypeRequest.Document_Type)
	if updateDocs != nil {
		c.JSON(http.StatusFound, gin.H{
			"error":         false,
			"response code": 200,
			"message":       "Something went wrong while updating user email",
			"data":          updateDocs,
		})
		return
	}

	filter := bson.M{"user_id": foundUser.User_ID}

	update := bson.M{
		"$set": bson.M{
			"kyc_docs": header.Filename,
			"tier":     3,
			"status":   "complete",
		},
	}
	result, _ := KycCollection.UpdateOne(ctx, filter, update)
	if result.MatchedCount != 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "no matched document found for update",
			"data":          "",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"error":         false,
		"response code": 200,
		"message":       "User profile updated successfully",
		"data":          "",
	})

}

func (uc *UserController) GetKycDetails(c *gin.Context) {
	var _, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	_, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	userId := c.Query("id")
	foundKyc, err := uc.UserService.GetUserKycByID(userId)
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "User does not exist",
			"data":          "",
		})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"error":         false,
		"response code": 201,
		"message":       "User kyc details retrieved successfully",
		"data":          foundKyc,
	})

}

func (uc *UserController) GetAllUsers(c *gin.Context) {
	var _, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	var (
	// username Username
	// user     models.User
	)
	foundUser, err := uc.UserService.GetAllKycUsers()
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "Something went wrong fetching users",
			"data":          "",
		})
		return
	}
	// username.User = foundUser
	var selectedUsers []Usernames
	for _, user := range foundUser {
		selectedUser := Usernames{
			User_ID:         user.User_ID,
			Username:        user.Username,
			Profile_Picture: user.Profile_Picture,
		}
		selectedUsers = append(selectedUsers, selectedUser)
	}
	c.JSON(http.StatusFound, gin.H{
		"error":         false,
		"response code": 200,
		"message":       "Users retrieved successfully",
		"data":          selectedUsers,
	})
}

func (uc *UserController) UserRoutes(rg *gin.RouterGroup) {
	userRoute := rg.Group("/user")
	// {
	// 	userRoute.Use(middleware.CORSMiddleware())

	userRoute.POST("/signup", uc.Signup)
	userRoute.POST("/login", uc.Login)
	userRoute.POST("/refresh-token", uc.RefreshToken)
	userRoute.POST("/donate",
		middleware.Authentication,
		uc.Donation,
	)
	userRoute.POST("/add-socials",
		middleware.Authentication,
		uc.Socials,
	)
	userRoute.GET("/transactions",
		middleware.Authentication,
		uc.GetUserTransaction,
	)
	userRoute.POST("/add-bank",
		middleware.Authentication,
		uc.AddBank,
	)
	userRoute.GET("/get-bank",
		middleware.Authentication,
		uc.GetBankDetails,
	)
	userRoute.PUT("/email-verification-request",
		middleware.Authentication,
		uc.RequestEmailVerification,
	)

	userRoute.PUT("/email-verification",
		middleware.Authentication,
		uc.EmailVerification,
	)

	userRoute.POST("/profile-picture",
		middleware.Authentication,
		uc.Userselfie,
	)

	userRoute.POST("/verify-bvn",
		middleware.Authentication,
		uc.VerifyBVN,
	)

	userRoute.POST("/file-upload",
		middleware.Authentication,
		uc.KycFileUpload,
	)
	userRoute.GET("profile",
		middleware.Authentication,
		uc.UserProfile,
	)

	userRoute.GET("profile-noauth",
		uc.UserProfileNoAuth,
	)

	userRoute.GET("/kyc-status",
		middleware.Authentication,
		uc.GetKycDetails,
	)

	userRoute.GET("/all",
		uc.GetAllUsers,
	)
	// userRoute.POST("/create", uc.CreateUser)
	// }
}
