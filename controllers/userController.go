package controllers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/JayJosh846/donationPlatform/database"
	"github.com/JayJosh846/donationPlatform/models"
	"github.com/JayJosh846/donationPlatform/services"
	generate "github.com/JayJosh846/donationPlatform/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var UserCollection *mongo.Collection = database.GetUserCollection(database.Client, "Users")
var Validate = validator.New()

func HashPassword(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		log.Panic(err)
	}
	return string(bytes)
}

func VerifyPassword(userpassword string, givenpassword string) (bool, string) {
	err := bcrypt.CompareHashAndPassword([]byte(givenpassword), []byte(userpassword))
	valid := true
	msg := ""
	if err != nil {
		msg = "Login Or Passowrd is Incorerct"
		valid = false
	}
	return valid, msg
}

type UserController struct {
	UserService services.UserService
}

func Constructor(userService services.UserService) UserController {
	return UserController{
		UserService: userService,
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
	password := HashPassword(*user.Password)
	user.Password = &password

	user.ID = primitive.NewObjectID()
	user.User_ID = user.ID.Hex()
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
	ctx.JSON(200, "")
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
	// userRoute.POST("/create", uc.CreateUser)
}
