package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type urlID struct {
	ID string `json:"ID"`
}

type longURLResult struct {
	LongURL string `json:"LongURL"`
}

func handleRequest(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	longURL, err := getFromDynamo(req.PathParameters["param"])
	if err != nil {
		log.Print(err)
		return respond500(err), nil
	} else if longURL == "" {
		log.Print("no such code")
		return respond400("error: no such short code"), nil
	}

	return respond302(longURL), nil
}

func main() {
	lambda.Start(handleRequest)
}

func getFromDynamo(shortcode string) (string, error) {
	svc := dynamodb.New(session.New())

	request, err := dynamodbattribute.MarshalMap(map[string]string{"ID": shortcode})
	if err != nil {
		log.Print(fmt.Sprintf("failed to DynamoDB marshal Record, %v", err))
		return "", err
	}

	input := &dynamodb.GetItemInput{
		Key:       request,
		TableName: aws.String(os.Getenv("TABLE_NAME")),
	}

	result, err := svc.GetItem(input)
	if err != nil {
		return "", err
	}

	var resultURL map[string]string
	err = dynamodbattribute.UnmarshalMap(result.Item, &resultURL)
	if err != nil {
		return "", err
	}

	return resultURL["LongURL"], nil
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

func respond302(respbody string) events.APIGatewayProxyResponse {
	var resp events.APIGatewayProxyResponse
	resp.Body = respbody
	resp.StatusCode = 302
	resp.Headers = make(map[string]string)
	resp.Headers["Access-Control-Allow-Origin"] = "*"
	resp.Headers["Location"] = respbody
	return resp
}

func respond400(respbody string) events.APIGatewayProxyResponse {
	var resp events.APIGatewayProxyResponse
	resp.StatusCode = 400
	resp.Headers = make(map[string]string)
	resp.Headers["Access-Control-Allow-Origin"] = "*"
	resp.Headers["Content-Type"] = "text/html"
	resp.Body = respbody
	return resp
}
