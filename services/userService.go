package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/JayJosh846/donationPlatform/database"
	"github.com/JayJosh846/donationPlatform/models"
	helper "github.com/JayJosh846/donationPlatform/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserService interface {
	CreateUser(*models.User) error
	GetUserByID(string) (*models.User, error)
	GetUser(*string) (*models.User, error)
	GetUserCount() (int64, error)
	GetAdmin(*string) (*models.User, error)
	UpdateUserBalance(*models.User, int, string) error
	CreateEmailVerification(*models.User, string) (*models.Otp, error)
	UpdateUserEmailPhone(string, string, string) error
	UpdateUserEmailStatus(string) error
	UpdateUserPicture(string, string) error
	UpdateUserKYCStatus(string, string) error
	GetUserKycByID(string) (*models.KYC, error)
	GetUserSocialsByID(string) (*models.Social, error)
	GetUserDonationsByID(string) ([]*models.Donation, error)
	CreateSocial(*models.Social) error
	GetAllUsers() ([]*models.User, error)
	GetAllKycUsers() ([]*models.User, error)
}

type UserServiceImpl struct {
	userCollection *mongo.Collection
	ctx            context.Context
}

func Constructor(userCollection *mongo.Collection, ctx context.Context) UserService {
	return &UserServiceImpl{
		userCollection: userCollection,
		ctx:            ctx,
	}
}

var OTPCollection *mongo.Collection = database.GetUserCollection(database.Client, "Otps")
var KycCollection *mongo.Collection = database.GetUserCollection(database.Client, "Kycs")
var SocialCollection *mongo.Collection = database.GetUserCollection(database.Client, "Socials")
var DonationCollection *mongo.Collection = database.GetUserCollection(database.Client, "Donations")

func (u *UserServiceImpl) CreateUser(user *models.User) error {
	_, err := u.userCollection.InsertOne(u.ctx, user)
	return err
}

func (u *UserServiceImpl) GetUser(email *string) (*models.User, error) {
	var user *models.User
	query := bson.M{"email": email}
	err := u.userCollection.FindOne(u.ctx, query).Decode(&user)
	return user, err
}

func (u *UserServiceImpl) GetUserByID(id string) (*models.User, error) {
	var user *models.User
	query := bson.M{"user_id": id}
	err := u.userCollection.FindOne(u.ctx, query).Decode(&user)
	return user, err
}

func (u *UserServiceImpl) GetUserCount() (int64, error) {
	query := bson.M{}
	count, err := u.userCollection.CountDocuments(u.ctx, query)
	return count, err
}

func (u *UserServiceImpl) GetAdmin(email *string) (*models.User, error) {
	var user *models.User
	query := bson.M{"email": email, "role": "admin"}
	err := u.userCollection.FindOne(u.ctx, query).Decode(&user)
	return user, err
}

func (u *UserServiceImpl) UpdateUserBalance(user *models.User, amount int, updateType string) error {
	switch updateType {
	case "add":
		user.Balance = user.Balance + amount
		filter := bson.D{primitive.E{Key: "email", Value: user.Email}}
		update := bson.D{
			primitive.E{
				Key: "$set",
				Value: bson.D{
					primitive.E{Key: "balance", Value: user.Balance},
				},
			},
		}
		result, _ := u.userCollection.UpdateOne(u.ctx, filter, update)
		if result.MatchedCount != 1 {
			return errors.New("no matched document found for update")
		}
		return nil

	case "subtract":
		user.Balance = user.Balance - amount
		filter := bson.D{primitive.E{Key: "email", Value: user.Email}}
		update := bson.D{
			primitive.E{
				Key: "$set",
				Value: bson.D{
					primitive.E{Key: "balance", Value: user.Balance},
				},
			},
		}
		result, _ := u.userCollection.UpdateOne(u.ctx, filter, update)
		if result.MatchedCount != 1 {
			return errors.New("no matched document found for update")
		}
		return nil
	default:
		return errors.New("unsupported update type")
	}
}

func (u *UserServiceImpl) CreateEmailVerification(user *models.User, email string) (*models.Otp, error) {
	var otp models.Otp
	verificationCode := helper.GenerateVerificationCode()
	expirationTime := time.Now().Add(30 * time.Minute)

	otp.User_ID = user.User_ID
	otp.Token = verificationCode
	otp.Expires_At = expirationTime
	OTPCollection.InsertOne(u.ctx, otp)

	err := sendVerificationEmail(*user.Username, email, verificationCode)
	if err != nil {
		fmt.Println("Error sending verification email:", err)
		return nil, err
	}

	return &otp, nil
}

func (u *UserServiceImpl) UpdateUserEmailPhone(id, email, phone string) error {
	filter := bson.M{"user_id": id}

	update := bson.M{
		"$set": bson.M{
			// "email_verified": true,
			"email": email,
			"phone": phone,
		},
	}
	result, _ := u.userCollection.UpdateOne(u.ctx, filter, update)
	if result.MatchedCount != 1 {
		return errors.New("no matched document found for update")
	}
	return nil
}

func (u *UserServiceImpl) UpdateUserEmailStatus(id string) error {
	filter := bson.D{primitive.E{Key: "user_id", Value: id}}
	update := bson.D{
		primitive.E{
			Key: "$set",
			Value: bson.D{
				primitive.E{Key: "email_verified", Value: true},
			},
		},
	}
	result, _ := u.userCollection.UpdateOne(u.ctx, filter, update)
	if result.MatchedCount != 1 {
		return errors.New("no matched document found for update")
	}
	return nil
}

func (u *UserServiceImpl) UpdateUserPicture(id, picture string) error {
	filter := bson.D{primitive.E{Key: "user_id", Value: id}}
	update := bson.D{
		primitive.E{
			Key: "$set",
			Value: bson.D{
				primitive.E{Key: "profile_picture", Value: picture},
			},
		},
	}
	result, _ := u.userCollection.UpdateOne(u.ctx, filter, update)
	if result.MatchedCount != 1 {
		return errors.New("no matched document found for update")
	}
	return nil
}

func (u *UserServiceImpl) UpdateUserKYCStatus(id, document string) error {
	filter := bson.M{"user_id": id}

	update := bson.M{
		"$set": bson.M{
			"identification": document,
			"kyc_status":     true,
			"id_upload":      true,
		},
	}
	result, _ := u.userCollection.UpdateOne(u.ctx, filter, update)
	if result.MatchedCount != 1 {
		return errors.New("no matched document found for update")
	}
	return nil
}

func (u *UserServiceImpl) GetUserKycByID(id string) (*models.KYC, error) {
	var kyc *models.KYC
	query := bson.M{"user_id": id}
	err := KycCollection.FindOne(u.ctx, query).Decode(&kyc)
	return kyc, err
}

func (u *UserServiceImpl) GetUserSocialsByID(id string) (*models.Social, error) {
	var social *models.Social
	query := bson.M{"user_id": id}
	err := SocialCollection.FindOne(u.ctx, query).Decode(&social)
	return social, err
}

func (u *UserServiceImpl) GetUserDonationsByID(id string) ([]*models.Donation, error) {
	var donations []*models.Donation
	query := bson.M{"user_id": id}
	cursor, err := DonationCollection.Find(u.ctx, query)
	defer cursor.Close(u.ctx)

	for cursor.Next(u.ctx) {
		var donation *models.Donation
		if err := cursor.Decode(&donation); err != nil {
			return nil, err
		}
		donations = append(donations, donation)
	}
	return donations, err
}

func (u *UserServiceImpl) CreateSocial(social *models.Social) error {
	_, err := SocialCollection.InsertOne(u.ctx, social)
	return err
}

func (u *UserServiceImpl) GetAllUsers() ([]*models.User, error) {
	var users []*models.User
	query := bson.M{"role": "user"}
	cursor, err := u.userCollection.Find(u.ctx, query)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(u.ctx)

	for cursor.Next(u.ctx) {
		var user *models.User
		if err := cursor.Decode(&user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, err
}

func (u *UserServiceImpl) GetAllKycUsers() ([]*models.User, error) {
	var users []*models.User
	query := bson.M{
		"role":       "user",
		"kyc_status": true,
	}
	cursor, err := u.userCollection.Find(u.ctx, query)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(u.ctx)

	for cursor.Next(u.ctx) {
		var user *models.User
		if err := cursor.Decode(&user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, err
}
