package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/JayJosh846/donationPlatform/models"
	// "go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type PaymentService interface {
	Payin(amount *int, id *string, user *models.User) error
}

type PaymentServiceImpl struct {
	paymentCollection *mongo.Collection
	ctx               context.Context
}

type PaymentRequest struct {
	Amount         int      `json:"amount"`
	RedirectURL    string   `json:"redirect_url"`
	Currency       string   `json:"currency"`
	Reference      string   `json:"reference"`
	Narration      string   `json:"narration"`
	Channels       []string `json:"channels"`
	DefaultChannel string   `json:"default_channel"`
	Customer       struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"customer"`
	NotificationURL string            `json:"notification_url"`
	Metadata        map[string]string `json:"metadata"`
}

func PaymentConstructor(paymentCollection *mongo.Collection, ctx context.Context) PaymentService {
	return &PaymentServiceImpl{
		paymentCollection: paymentCollection,
		ctx:               ctx,
	}
}

func generateTransactionReference() string {
	// Generate a random identifier.
	identifier := rand.Intn(1000000) // Change the range as needed.

	// Get the current timestamp.
	currentTime := time.Now()

	// Format the timestamp and combine it with the identifier.
	transactionReference := currentTime.Format("20060102150405") + fmt.Sprintf("%06d", identifier)

	return transactionReference
}

func (u *PaymentServiceImpl) Payin(amount *int, id *string, user *models.User) error {
	// u.
	// var paymentRequest PaymentRequest
	url := "https://api.korapay.com/merchant/api/v1/charges/initialize"
	method := "POST"

	reference := generateTransactionReference()
	paymentRequest := PaymentRequest{
		Amount:         *amount,
		RedirectURL:    "https://korapay.com",
		Currency:       "NGN",
		Reference:      reference,
		Channels:       []string{"card", "bank_transfer", "pay_with_bank", "mobile_money"},
		DefaultChannel: "card",
		Customer: struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}{
			Name:  *user.Fullname,
			Email: *user.Email,
		},
	}
	requestBodyJSON, err := json.Marshal(paymentRequest)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return err
	}
	bodyReader := bytes.NewReader([]byte(requestBodyJSON))
	client := &http.Client{}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		fmt.Println(err)
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(u.ctx)
	fmt.Println(string(body))
	return nil
}
