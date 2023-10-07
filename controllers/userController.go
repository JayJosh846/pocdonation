package controllers

import (
	"context"
	"fmt"
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
)

var UserCollection *mongo.Collection = database.GetUserCollection(database.Client, "Users")
var Validate = validator.New()

type UserController struct {
	UserService     services.UserService
	DonationService services.DonationService
}

func Constructor(
	userService services.UserService,
	donationService services.DonationService,
) UserController {
	return UserController{
		UserService:     userService,
		DonationService: donationService,
	}
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
	user.Link = link
	user.Role = "user"
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

func (uc *UserController) GetUser(ctx *gin.Context) {
	ctx.JSON(200, "")
}

func (uc *UserController) GetAllUsers(ctx *gin.Context) {
	ctx.JSON(200, "")
}

func (uc *UserController) UserRoutes(rg *gin.RouterGroup) {
	userRoute := rg.Group("/users")
	userRoute.POST("/sign-up", uc.Signup)
	userRoute.POST("/login", uc.Login)
	userRoute.POST("/donate",
		middleware.Authentication,
		uc.Donation,
	)

	// userRoute.POST("/create", uc.CreateUser)
}
