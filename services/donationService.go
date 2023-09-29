package services

import (
	"context"

	"github.com/JayJosh846/donationPlatform/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type DonationService interface {
	CreateDonation(*models.Donation) error
	GetDonations(*string) (*models.Donation, error)
}

type DonationServiceImpl struct {
	donationsCollection *mongo.Collection
	ctx                 context.Context
}

func DonationConstructor(donationsCollection *mongo.Collection, ctx context.Context) DonationService {
	return &DonationServiceImpl{
		donationsCollection: donationsCollection,
		ctx:                 ctx,
	}
}

func (u *DonationServiceImpl) CreateDonation(donations *models.Donation) error {
	_, err := u.donationsCollection.InsertOne(u.ctx, donations)
	return err
}

func (u *DonationServiceImpl) GetDonations(reference *string) (*models.Donation, error) {
	var donations *models.Donation
	query := bson.M{"transaction_reference": reference}
	err := u.donationsCollection.FindOne(u.ctx, query).Decode(&donations)
	return donations, err
}
