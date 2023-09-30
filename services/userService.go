package services

import (
	"context"
	"errors"

	"github.com/JayJosh846/donationPlatform/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserService interface {
	CreateUser(*models.User) error
	GetUser(*string) (*models.User, error)
	GetAllUsers() ([]*models.User, error)
	UpdateUserBalance(*models.User, int) error
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

func (u *UserServiceImpl) UpdateUserBalance(user *models.User, amount int) error {
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
}

func (u *UserServiceImpl) GetAllUsers() ([]*models.User, error) {
	return nil, nil
}
