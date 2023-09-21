package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID            primitive.ObjectID `json:"_id" bson:"_id"`
	User_ID       string             `json:"user_id"`
	Fullname      *string            `json:"full_name" validate:"required,min=2,max=30"`
	Email         *string            `json:"email" validate:"required"`
	Phone         *string            `json:"phone" validate:"required"`
	DOB           *string            `json:"dob" validate:"required"`
	Gender        *string            `json:"gender" validate:"required"`
	Password      *string            `json:"password" validate:"required,min=6"`
	Country       *string            `json:"country" validate:"required"`
	Bio           *string            `json:"bio"`
	Username      string             `json:"username"`
	Balance       string             `json:"balance"`
	Token         *string            `json:"token"`
	Refresh_Token *string            `json:"refresh_token"`
	Created_At    time.Time          `json:"created_at"`
	Updated_At    time.Time          `json:"updated_at"`
	Transactions  []Transaction      `json:"transaction" bson:"transaction"`
	Banks         []Bank             `json:"bank" bson:"bank"`
	Socials       Socials            `json:"socials" bson:"socials"`
}

type Transaction struct {
	ID         primitive.ObjectID `json:"_id" bson:"_id"`
	Amount     *string            `json:"amount" bson:"amount"`
	Created_At time.Time          `json:"created_at"`
	Updated_At time.Time          `json:"updated_at"`
}

type Bank struct {
	ID             primitive.ObjectID `json:"_id" bson:"_id"`
	Account_Number *string            `json:"account_number" bson:"account_number"`
	Account_Name   *string            `json:"account_name" bson:"account_name"`
	Bank_Name      *string            `json:"bank_name" bson:"bank_name"`
	Bvn            string             `json:"bvn" bson:"bvn"`
	Created_At     time.Time          `json:"created_at"`
	Updated_At     time.Time          `json:"updated_at"`
}

type Socials struct {
	ID        primitive.ObjectID `json:"_id" bson:"_id"`
	Twitter   string             `json:"twitter" bson:"twitter"`
	Instagram string             `json:"instagram" bson:"instagram"`
	Facebook  string             `json:"facebook" bson:"facebook"`
	LinkedIn  string             `json:"linkedin" bson:"linkedin"`
}
