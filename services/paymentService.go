package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/JayJosh846/donationPlatform/models"
	// "go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type PaymentService interface {
	Payin(amount string, user models.User) (string, error)
	PaymentGetUser(*string) (*models.User, error)
	VerifyDeposit(eventData []byte) (WebhookPayload, error)
}

type PaymentServiceImpl struct {
	paymentCollection *mongo.Collection
	ctx               context.Context
}

type PaymentRequest struct {
	Amount string `json:"amount"`
	Email  string `json:"email"`
	// RedirectURL    string   `json:"redirect_url"`
	Currency  string   `json:"currency"`
	Reference string   `json:"reference"`
	Channels  []string `json:"channels"`
	// DefaultChannel string   `json:"default_channel"`
	// Customer       struct {
	// 	Name  *string `json:"name"`
	// 	Email *string `json:"email"`
	// } `json:"customer"`
	// NotificationURL string `json:"notification_url"`
	// Metadata        map[string]string `json:"metadata"`
}

type WebhookPayload struct {
	Event string `json:"event"`
	Data  struct {
		Reference string `json:"reference"`
		Currency  string `json:"currency"`
		Amount    int    `json:"amount"`
		Channel   string `json:"channel"`
		// Fee               string `json:"fee"`
		Status   string `json:"status"`
		Customer struct {
			Email string `json:"email"`
		} `json:"customer"`
		// Payment_method    string `json:"payment_method"`
		// Payment_reference string `json:"payment_reference"`
	} `json:"data"`
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

func (u *PaymentServiceImpl) PaymentGetUser(email *string) (*models.User, error) {
	var user *models.User
	query := bson.M{"email": email}
	err := u.paymentCollection.FindOne(u.ctx, query).Decode(&user)
	return user, err
}

func (u *PaymentServiceImpl) Payin(amount string, user models.User) (string, error) {
	// u.
	// var paymentRequest PaymentRequest
	secKey := os.Getenv("PAYSTACK_SEC_KEY")
	token := secKey
	url := "https://api.paystack.co/transaction/initialize"
	method := "POST"

	fmt.Println("user from paymentservice", *user.Email)

	reference := generateTransactionReference()
	paymentRequest := PaymentRequest{
		Amount:    amount,
		Email:     *user.Email,
		Currency:  "NGN",
		Reference: reference,
		Channels:  []string{"card", "bank", "ussd", "mobile_money", "qr", "bank_transfer"},
		// DefaultChannel: "card",
		// Customer: struct {
		// 	Name  *string `json:"name"`
		// 	Email *string `json:"email"`
		// }{
		// 	Name:  user.Fullname,
		// 	Email: user.Email,
		// },
		// NotificationURL: "https://2764-41-217-1-238.ngrok-free.app/api/v1/payment/confirmation",
	}
	requestBodyJSON, err := json.Marshal(paymentRequest)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return "", err
	}
	bodyReader := bytes.NewReader([]byte(requestBodyJSON))
	client := &http.Client{}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return "", err

	}
	fmt.Println("ctx", u.ctx)
	return string(body), nil
}

func (u *PaymentServiceImpl) VerifyDeposit(eventData []byte) (WebhookPayload, error) {

	var webhookPayload WebhookPayload
	if err := json.Unmarshal(eventData, &webhookPayload); err != nil {
		fmt.Println("Error:", err)
		return WebhookPayload{}, err
	}
	fmt.Println("Payload:", webhookPayload)
	return webhookPayload, nil
}
