package services

import (
	"context"

	"github.com/JayJosh846/donationPlatform/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type TransactionService interface {
	CreateTransaction(*models.Transaction) error
	GetTransaction(*string) (*models.Transaction, error)
}

type TransactionServiceImpl struct {
	transactionCollection *mongo.Collection
	ctx                   context.Context
}

func TransactionConstructor(transactionCollection *mongo.Collection, ctx context.Context) TransactionService {
	return &TransactionServiceImpl{
		transactionCollection: transactionCollection,
		ctx:                   ctx,
	}
}

func (u *TransactionServiceImpl) CreateTransaction(transaction *models.Transaction) error {
	_, err := u.transactionCollection.InsertOne(u.ctx, transaction)
	return err
}

func (u *TransactionServiceImpl) GetTransaction(reference *string) (*models.Transaction, error) {
	var transaction *models.Transaction
	query := bson.M{"reference": reference}
	err := u.transactionCollection.FindOne(u.ctx, query).Decode(&transaction)
	return transaction, err
}
