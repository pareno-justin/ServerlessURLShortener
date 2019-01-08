package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

type newRecord struct {
	ID      string `json:"ID"`
	LongURL string `json:"LongURL"`
}

func handleRequest(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc := dynamodb.New(session.New())

	//Unmarshal request Body into a map
	var longURLMap map[string]string
	err := json.Unmarshal([]byte(req.Body), &longURLMap)
	if err != nil {
		return respond500(err), err
	}

	shortCode, err := getUniqueCode(svc)

	sendToDynamo := newRecord{ID: shortCode, LongURL: longURLMap["LongURL"]}
	err = writeToDynamo(sendToDynamo, svc)
	if err != nil {
		return respond500(err), err
	}

	respBody := json.RawMessage(fmt.Sprintf(`{"ShortURL": "%s%s%s"}`, os.Getenv("URL"), "/", shortCode))

	return respond200(string(respBody)), nil
}

func writeToDynamo(jsonIN newRecord, svc *dynamodb.DynamoDB) error {
	av, err := dynamodbattribute.MarshalMap(jsonIN)
	if err != nil {
		log.Print(err)
	}

	_, err = svc.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("TABLE_NAME")),
		Item:      av,
	})
	if err != nil {
		log.Print(err)
	}
	return err
}

func main() {
	lambda.Start(handleRequest)
}

func getUniqueCode(svc *dynamodb.DynamoDB) (string, error) {
	var code string
	unique := false

	for unique == false {
		code = randStringBytes(8)

		request, err := dynamodbattribute.MarshalMap(map[string]string{"ID": code})
		if err != nil {
			log.Print(err)
			return "", err
		}

		input := &dynamodb.GetItemInput{
			Key:       request,
			TableName: aws.String(os.Getenv("TABLE_NAME")),
		}

		getResult, err := svc.GetItem(input)
		if err != nil {
			log.Print(err)
			return "", err
		}

		var result map[string]interface{}
		err = dynamodbattribute.UnmarshalMap(getResult.Item, &result)
		if err != nil {
			log.Print(err)
			return "", err
		}

		if result["LongURL"] == nil {
			unique = true
		}
	}

	return code, nil
}

func respond500(err error) events.APIGatewayProxyResponse {
	var resp events.APIGatewayProxyResponse
	resp.Body = fmt.Sprintf("%v", err)
	resp.StatusCode = 500
	resp.Headers = make(map[string]string)
	resp.Headers["Access-Control-Allow-Origin"] = "*"
	resp.Headers["Content-Type"] = "text/plain"
	return resp
}

func respond200(respbody string) events.APIGatewayProxyResponse {
	var resp events.APIGatewayProxyResponse
	resp.Body = respbody
	resp.StatusCode = 200
	resp.Headers = make(map[string]string)
	resp.Headers["Access-Control-Allow-Origin"] = "*"
	resp.Headers["Content-Type"] = "application/json"
	return resp
}
