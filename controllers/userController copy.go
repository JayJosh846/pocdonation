package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

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
)

var UserCollection *mongo.Collection = database.GetUserCollection(database.Client, "Users")
var BankCollection *mongo.Collection = database.GetUserCollection(database.Client, "Banks")
var OtpCollection *mongo.Collection = database.GetUserCollection(database.Client, "Otps")
var KycCollection *mongo.Collection = database.GetUserCollection(database.Client, "Kycs")
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
	user.Kyc_Status = false
	user.Created_At, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	user.Updated_At, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	token, refreshtoken, _ := generate.TokenGenerator(user.User_ID, *user.Email)
	user.Token = &token
	user.Refresh_Token = &refreshtoken
	user.Transactions = make([]models.Transaction, 0)
	user.Banks = make([]models.Bank, 0)

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
	// var founduser models.User
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
			"data":          "",
		})
		return
	}
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
			"message":       "User does not exist",
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
			"error":         false,
			"response code": 200,
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

	// var (
	// 	selfieRequest SelfieRequest
	// )

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

func (uc *UserController) VerifyBVN(c *gin.Context) {
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

func (uc *UserController) UserRoutes(rg *gin.RouterGroup) {
	userRoute := rg.Group("/user")
	// {
	// 	userRoute.Use(middleware.CORSMiddleware())

	userRoute.POST("/signup", uc.Signup)
	userRoute.POST("/login", uc.Login)
	userRoute.POST("/donate",
		middleware.Authentication,
		uc.Donation,
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

	userRoute.GET("/kyc-status",
		middleware.Authentication,
		uc.GetKycDetails,
	)

	// userRoute.POST("/create", uc.CreateUser)
	// }
}
