package main

import (
	"context"
	"log"

	"github.com/JayJosh846/donationPlatform/controllers"
	"github.com/JayJosh846/donationPlatform/database"
	"github.com/JayJosh846/donationPlatform/services"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	// "go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	server       *gin.Engine
	us           services.UserService
	ps           services.PaymentService
	ds           services.DonationService
	ts           services.TransactionService
	bs           services.BankService
	uc           controllers.UserController
	pc           controllers.PaymentController
	ac           controllers.AdminController
	ctx          context.Context
	userc        *mongo.Collection
	paymentc     *mongo.Collection
	transactionc *mongo.Collection
	donationc    *mongo.Collection
	bankc        *mongo.Collection
)

func init() {
	ctx = context.TODO()

	userc = database.GetUserCollection(database.Client, "Users")
	paymentc = database.GetUserCollection(database.Client, "Users")
	transactionc = database.GetUserCollection(database.Client, "Transactions")
	donationc = database.GetUserCollection(database.Client, "Donations")
	bankc = database.GetUserCollection(database.Client, "Banks")
	us = services.Constructor(userc, ctx)
	ps = services.PaymentConstructor(paymentc, ctx)
	ts = services.TransactionConstructor(transactionc, ctx)
	ds = services.DonationConstructor(donationc, ctx)
	bs = services.BankConstructor(bankc, ctx)
	uc = controllers.Constructor(us, ts, ds, bs, ps)
	pc = controllers.PaymentConstructor(ps, us, ts, ds, bs)
	ac = controllers.AdminConstructor(us, ts, ds)

	server = gin.Default()
}

func main() {
	client := database.ConnectToMongoDB()
	defer database.CloseMongoDBConnection(client)

	server = gin.Default()
	// Define CORS configuration with specific allowed origins
	corsConfig := cors.DefaultConfig()

	// Allow specific origins
	corsConfig.AllowAllOrigins = true

	// To be able to send tokens to the server.
	corsConfig.AllowCredentials = true

	// OPTIONS method for ReactJS
	corsConfig.AddAllowMethods("OPTIONS")

	// Register the middleware
	server.Use(cors.New(corsConfig))

	basepath := server.Group("/api/v1")
	uc.UserRoutes(basepath)
	ac.AdminRoute(basepath)
	pc.PaymentRoute(basepath)

	log.Fatal(server.Run(":9000"))
}
