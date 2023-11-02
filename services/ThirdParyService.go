package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type BvnRequest struct {
	Bvn  string `json:"bvn"`
	Type string `json:"type"`
}

func LookUpBVN(bvn string) (string, error) {
	secKey := os.Getenv("CHECKID_SEC_KEY")
	token := secKey
	typeBody := "lookup"
	url := "https://sandbox.checkid.ng/api/v1/identity/bvn"
	method := "POST"
	bvnRequest := BvnRequest{
		Bvn:  bvn,
		Type: typeBody,
	}
	requestBodyJSON, err := json.Marshal(bvnRequest)
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
	fmt.Println("response1", string(body))

	return string(body), nil
}

func VerifyBVN(bvn string) (string, error) {
	secKey := os.Getenv("CHECKID_SEC_KEY")
	token := secKey
	typeBody := "validate"
	url := "https://sandbox.checkid.ng/api/v1/identity/bvn"
	method := "POST"
	bvnRequest := BvnRequest{
		Bvn:  bvn,
		Type: typeBody,
	}
	requestBodyJSON, err := json.Marshal(bvnRequest)
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
	return string(body), nil
}
