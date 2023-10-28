package services

import (
	"context"

	"github.com/JayJosh846/donationPlatform/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type BankService interface {
	AddBank(*models.Bank) error
	GetUserBankByID(id string) (*models.Bank, error)
}

type BankServiceImpl struct {
	bankCollection *mongo.Collection
	ctx            context.Context
}

func BankConstructor(bankCollection *mongo.Collection, ctx context.Context) BankService {
	return &BankServiceImpl{
		bankCollection: bankCollection,
		ctx:            ctx,
	}
}

func (b *BankServiceImpl) AddBank(bank *models.Bank) error {
	_, err := b.bankCollection.InsertOne(b.ctx, bank)
	return err
}

func (b *BankServiceImpl) GetUserBankByID(id string) (*models.Bank, error) {
	var bank *models.Bank
	query := bson.M{"user_id": id}
	err := b.bankCollection.FindOne(b.ctx, query).Decode(&bank)
	return bank, err
}
