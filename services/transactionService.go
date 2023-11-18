package services

import (
	"context"
	"errors"

	"github.com/JayJosh846/donationPlatform/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type TransactionService interface {
	CreateTransaction(*models.Transaction) error
	GetUserTransactionsByID(string) ([]*models.Transaction, error)
	GetTransactionByID(primitive.ObjectID) (*models.Transaction, error)
	GetTransactionCount() (int64, error)
	GetSuccessfulTransactionCount() (int64, error)
	GetFailureTransactionCount() (int64, error)
	GetTransactions() ([]*models.Transaction, error)
	UpdateTransactionStatus(*string) error
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

func (u *TransactionServiceImpl) GetTransactionByID(id primitive.ObjectID) (*models.Transaction, error) {
	var transaction *models.Transaction
	query := bson.M{"_id": id}
	err := u.transactionCollection.FindOne(u.ctx, query).Decode(&transaction)
	return transaction, err
}

func (u *TransactionServiceImpl) GetUserTransactionsByID(id string) ([]*models.Transaction, error) {
	var transactions []*models.Transaction
	query := bson.M{"user_id": id}
	cursor, err := u.transactionCollection.Find(u.ctx, query)
	defer cursor.Close(u.ctx)

	for cursor.Next(u.ctx) {
		var transaction *models.Transaction
		if err := cursor.Decode(&transaction); err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}
	return transactions, err
}

func (u *TransactionServiceImpl) GetTransactions() ([]*models.Transaction, error) {
	var transactions []*models.Transaction
	query := bson.M{}
	cursor, err := u.transactionCollection.Find(u.ctx, query)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(u.ctx)

	for cursor.Next(u.ctx) {
		var transaction *models.Transaction
		if err := cursor.Decode(&transaction); err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}
	return transactions, err
}

func (u *TransactionServiceImpl) GetTransactionCount() (int64, error) {
	query := bson.M{}
	count, err := u.transactionCollection.CountDocuments(u.ctx, query)
	return count, err
}

func (u *TransactionServiceImpl) GetSuccessfulTransactionCount() (int64, error) {
	query := bson.M{"status": "complete"}
	count, err := u.transactionCollection.CountDocuments(u.ctx, query)
	return count, err
}

func (u *TransactionServiceImpl) GetFailureTransactionCount() (int64, error) {
	query := bson.M{"status": "failed"}
	count, err := u.transactionCollection.CountDocuments(u.ctx, query)
	return count, err
}
func (u *TransactionServiceImpl) UpdateTransactionStatus(transaction *string) error {
	filter := bson.D{primitive.E{Key: "reference", Value: transaction}}
	update := bson.D{
		primitive.E{
			Key: "$set",
			Value: bson.D{
				primitive.E{Key: "status", Value: "complete"},
			},
		},
	}
	result, _ := u.transactionCollection.UpdateOne(u.ctx, filter, update)
	if result.MatchedCount != 1 {
		return errors.New("no matched document found for update")
	}
	return nil
}
