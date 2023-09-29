package services

import (
	"context"
	"errors"

	"github.com/JayJosh846/donationPlatform/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type DonationService interface {
	CreateDonation(*models.Donation) error
	GetDonations(*string) (*models.Donation, error)
	UpdateDonationStatus(*string) error
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

func (u *DonationServiceImpl) UpdateDonationStatus(donation *string) error {
	filter := bson.D{primitive.E{Key: "transaction_reference", Value: donation}}
	update := bson.D{
		primitive.E{
			Key: "$set",
			Value: bson.D{
				primitive.E{Key: "status", Value: "complete"},
			},
		},
	}
	result, _ := u.donationsCollection.UpdateOne(u.ctx, filter, update)
	if result.MatchedCount != 1 {
		return errors.New("no matched document found for update")
	}
	return nil
}
