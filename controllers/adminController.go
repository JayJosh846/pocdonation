package controllers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/JayJosh846/donationPlatform/middleware"
	"github.com/JayJosh846/donationPlatform/models"
	"github.com/JayJosh846/donationPlatform/services"
	"go.mongodb.org/mongo-driver/bson/primitive"

	generate "github.com/JayJosh846/donationPlatform/utils"
	"github.com/gin-gonic/gin"
)

type AdminController struct {
	UserService        services.UserService
	TransactionService services.TransactionService
	DonationService    services.DonationService
}

func AdminConstructor(
	userService services.UserService,
	transactionService services.TransactionService,
	donationService services.DonationService,
) AdminController {
	return AdminController{
		UserService:        userService,
		TransactionService: transactionService,
		DonationService:    donationService,
	}
}

func (ac *AdminController) AdminLogin(ctx *gin.Context) {
	var _, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	var user models.User
	// var founduser models.User
	if err := ctx.BindJSON(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	foundUser, err := ac.UserService.GetAdmin(user.Email)
	defer cancel()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "Account not an admin or does not exist",
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

func (ac *AdminController) GetAllUsersCount(c *gin.Context) {
	var _, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	_, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	userCount, err := ac.UserService.GetUserCount()
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "Something went wrong while retrieving user count",
			"data":          "",
		})
		return
	}
	c.JSON(http.StatusFound, gin.H{
		"error":         false,
		"response code": 302,
		"message":       "User count retrieved successfully",
		"data":          userCount,
	})

}

func (ac *AdminController) GetAllTransactions(c *gin.Context) {
	var _, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	_, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	transactions, err := ac.TransactionService.GetTransactions()
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "Something went wrong while retrieving transactions",
			"data":          "",
		})
		return
	}
	c.JSON(http.StatusFound, gin.H{
		"error":         false,
		"response code": 302,
		"message":       "Transactions retrieved successfully",
		"data":          transactions,
	})

}

func (ac *AdminController) GetTransactionsById(c *gin.Context) {
	var _, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	_, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	id := c.Query("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "An error occured"})
		return
	}
	transaction, err := ac.TransactionService.GetTransactionByID(objectID)
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "Something went wrong while retrieving transaction",
			"data":          "",
		})
		return
	}
	c.JSON(http.StatusFound, gin.H{
		"error":         false,
		"response code": 302,
		"message":       "Transactions retrieved successfully",
		"data":          transaction,
	})

}

func (ac *AdminController) GetSuccessTransactionsCount(c *gin.Context) {
	var _, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	_, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	successfulTransactionCount, err := ac.TransactionService.GetSuccessfulTransactionCount()
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "Something went wrong while retrieving successful transaction count",
			"data":          "",
		})
		return
	}
	c.JSON(http.StatusFound, gin.H{
		"error":         false,
		"response code": 302,
		"message":       "Successful transaction count retrieved successfully",
		"data":          successfulTransactionCount,
	})

}

func (ac *AdminController) GetFailureTransactionsCount(c *gin.Context) {
	var _, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	_, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	failureTransactionCount, err := ac.TransactionService.GetFailureTransactionCount()
	defer cancel()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         true,
			"response code": 400,
			"message":       "Something went wrong while retrieving failure transaction count",
			"data":          "",
		})
		return
	}
	c.JSON(http.StatusFound, gin.H{
		"error":         false,
		"response code": 302,
		"message":       "Failure transaction count retrieved successfully",
		"data":          failureTransactionCount,
	})

}

func (ac *AdminController) AdminRoute(rg *gin.RouterGroup) {
	adminRoute := rg.Group("/admin")
	adminRoute.POST("/login", ac.AdminLogin)
	adminRoute.GET("/transactions",
		middleware.Authentication,
		ac.GetAllTransactions,
	)
	adminRoute.GET("/transaction",
		middleware.Authentication,
		ac.GetTransactionsById,
	)
	adminRoute.GET("/no-of-users",
		middleware.Authentication,
		ac.GetAllUsersCount,
	)
	adminRoute.GET("/successful-transaction",
		middleware.Authentication,
		ac.GetSuccessTransactionsCount,
	)
	adminRoute.GET("/failure-transaction",
		middleware.Authentication,
		ac.GetFailureTransactionsCount,
	)
}
