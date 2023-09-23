package services

import (
	"context"

	"github.com/JayJosh846/donationPlatform/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserService interface {
	CreateUser(*models.User) error
	GetUser(*string) (*models.User, error)
	GetAllUsers() ([]*models.User, error)
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

func (u *UserServiceImpl) GetAllUsers() ([]*models.User, error) {
	return nil, nil
}
