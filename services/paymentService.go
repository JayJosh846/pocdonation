package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/JayJosh846/donationPlatform/models"
	helper "github.com/JayJosh846/donationPlatform/utils"

	// "go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type PaymentService interface {
	Payin(amount string, user models.User) (string, error)
	PaymentGetUser(*string) (*models.User, error)
	VerifyDeposit(eventData []byte) (WebhookPayload, error)
	GetBanks() (string, error)
	VerifyAccountNumber(string, string) (string, error)
	TransferRecipientCreation(string, string, string) (string, error)
	InitiateTransfer(int, string) (string, error)
}

type PaymentServiceImpl struct {
	paymentCollection *mongo.Collection
	ctx               context.Context
}

func PaymentConstructor(paymentCollection *mongo.Collection, ctx context.Context) PaymentService {
	return &PaymentServiceImpl{
		paymentCollection: paymentCollection,
		ctx:               ctx,
	}
}

type PaymentRequest struct {
	Amount    string   `json:"amount"`
	Email     string   `json:"email"`
	Currency  string   `json:"currency"`
	Reference string   `json:"reference"`
	Channels  []string `json:"channels"`
}

type WebhookPayload struct {
	Event string `json:"event"`
	Data  struct {
		Reference string `json:"reference"`
		Currency  string `json:"currency"`
		Amount    int    `json:"amount"`
		Channel   string `json:"channel"`
		Status    string `json:"status"`
		Customer  struct {
			Email string `json:"email"`
		} `json:"customer"`
	} `json:"data"`
}

type Bank struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type RecipientCreationRequest struct {
	Type           string `json:"type"`
	Name           string `json:"name"`
	Account_number string `json:"account_number"`
	Bank_code      string `json:"bank_code"`
	Currency       string `json:"currency"`
}

type TransferRequest struct {
	Source         string `json:"source"`
	Amount         int    `json:"amount"`
	Recipient_code string `json:"recipient"`
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

	reference := helper.GenerateTransactionReference()
	paymentRequest := PaymentRequest{
		Amount:    amount,
		Email:     *user.Email,
		Currency:  "NGN",
		Reference: reference,
		Channels:  []string{"card", "bank", "ussd", "mobile_money", "qr", "bank_transfer"},
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

func (u *PaymentServiceImpl) GetBanks() (string, error) {
	secKey := os.Getenv("PAYSTACK_SEC_KEY")
	token := secKey
	url := "https://api.paystack.co/bank"
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
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
	// Read the response body
	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return "", err
	}
	return string(responseBody), nil

}

func (u *PaymentServiceImpl) VerifyAccountNumber(accountNumber string, bank string) (string, error) {
	secKey := os.Getenv("PAYSTACK_SEC_KEY")
	token := secKey
	method := "GET"
	// Retrieve the list of banks as a JSON string
	jsonString, err := u.GetBanks()
	if err != nil {
		return "", err
	}
	var bankData struct {
		Data []Bank `json:"data"`
	}
	errr := json.Unmarshal([]byte(jsonString), &bankData)
	if errr != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return "", err
	}

	// Extract the slice of Bank structs
	banks := bankData.Data
	for _, b := range banks {
		if bank == b.Name {
			url := fmt.Sprintf("https://api.paystack.co/bank/resolve?account_number=%s&bank_code=%s", accountNumber, b.Code)
			client := &http.Client{}
			req, err := http.NewRequest(method, url, nil)
			if err != nil {
				return "", err
			}
			// Add the authorization header
			req.Header.Set("Content-Type", "application/json")
			req.Header.Add("Authorization", "Bearer "+token)
			// Send the request
			resp, err := client.Do(req)
			if err != nil {
				return "", err
			}
			defer resp.Body.Close()
			responseBody, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("Error reading response:", err)
				return "", err
			}
			fmt.Println("responseBody", string(responseBody))
			return string(responseBody), nil
		}
	}

	return "", fmt.Errorf("Bank not found in the bank list")
}

func (u *PaymentServiceImpl) TransferRecipientCreation(username string, accountNumber string, bank string) (string, error) {
	secKey := os.Getenv("PAYSTACK_SEC_KEY")
	token := secKey
	url := "https://api.paystack.co/transferrecipient"
	method := "POST"
	// Retrieve the list of banks as a JSON string
	jsonString, err := u.GetBanks()
	if err != nil {
		return "", err
	}
	var bankData struct {
		Data []Bank `json:"data"`
	}
	errr := json.Unmarshal([]byte(jsonString), &bankData)
	if errr != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return "", err
	}

	// Extract the slice of Bank structs
	banks := bankData.Data
	for _, b := range banks {
		if bank == b.Name {
			transferRecipient := RecipientCreationRequest{
				Type:           "nuban",
				Name:           username,
				Account_number: accountNumber,
				Bank_code:      b.Code,
				Currency:       "NGN",
			}
			requestBodyJSON, err := json.Marshal(transferRecipient)
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
	}
	return "", fmt.Errorf("Something went wrong while creating reciept. Please try again")

}

func (u *PaymentServiceImpl) InitiateTransfer(amount int, recipientCode string) (string, error) {
	secKey := os.Getenv("PAYSTACK_SEC_KEY")
	token := secKey
	url := "https://api.paystack.co/transfer"
	method := "POST"

	transferRequest := TransferRequest{
		Source:         "balance",
		Amount:         amount,
		Recipient_code: recipientCode,
	}
	requestBodyJSON, err := json.Marshal(transferRequest)
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
