package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/go-playground/validator/v10"
	"github.com/matcornic/hermes/v2"
)

type Request struct {
	Name        string   `json:"name" validate:"required"`
	DOB         string   `json:"dob" validate:"required"`
	Email       string   `json:"email" validate:"required,email"`
	Placement   string   `json:"placement" validate:"required"`
	Size        string   `json:"size" validate:"required"`
	Description string   `json:"description" validate:"required"`
	Comments    string   `json:"comments"`
	Technique   string   `json:"technique"`
	References  []string `json:"references"`
}

var emailClient *ses.SES
var h hermes.Hermes
var validate *validator.Validate

func init() {
	validate = validator.New()
	awsSession, _ := session.NewSession()
	emailClient = ses.New(awsSession, aws.NewConfig().WithRegion("eu-west-2"))
	h = hermes.Hermes{
		Product: hermes.Product{
			Name:      "HeartPoke",
			Link:      "https://heartpoke.co.uk",
			Logo:      "https://heartpoke.co.uk/logo.png",
			Copyright: fmt.Sprintf("Copyright © %d HeartPoke. All rights reserved.", time.Now().Year()),
		},
	}

}

func ReturnErrorToUser(error error, status int) (events.APIGatewayV2HTTPResponse, error) {
	log.Println(error.Error())

	return events.APIGatewayV2HTTPResponse{
		StatusCode: status,
		Headers:    map[string]string{"Content-Type": "text/plain"},
		Body:       error.Error(),
	}, nil
}

func ImageTagForSrc(src string) string {
	return fmt.Sprintf("<img src=\"%s\" width=\"400\" />", src)
}

func GenerateImagesHTML(refs []string) string {
	var b bytes.Buffer
	for _, ref := range refs {
		b.WriteString(ImageTagForSrc(ref))
	}
	return b.String()
}

func GenerateAdminEmail(request Request) hermes.Email {
	return hermes.Email{
		Body: hermes.Body{
			Name: "Katie",
			Intros: []string{
				"New booking request",
			},
			Dictionary: []hermes.Entry{
				{Key: "Name", Value: request.Name},
				{Key: "DOB", Value: request.DOB},
				{Key: "Email", Value: request.Email},
				{Key: "Placement", Value: request.Placement},
				{Key: "Size", Value: request.Size},
				{Key: "Description", Value: request.Description},
				{Key: "Comments", Value: request.Comments},
				{Key: "Technique", Value: request.Technique},
				{Key: "References", Value: "{{SUB}}"},
			},
		},
	}
}

func GenerateCustomerEmail(name string) hermes.Email {
	return hermes.Email{
		Body: hermes.Body{
			Name: name,
			Intros: []string{
				"Thanks for you booking enquiry!",
				"I've received your request and will be in touch shortly",
			},
		},
	}
}

func SendEmail(email hermes.Email, destination string, subject string, replyTo string, substitutes string) error {
	emailBody, err := h.GenerateHTML(email)

	if err != nil {
		return err
	}
	emailBody = strings.ReplaceAll(emailBody, "{{SUB}}", substitutes)

	emailText, err := h.GeneratePlainText(email)

	if err != nil {
		return err
	}

	emailParams := &ses.SendEmailInput{
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{Data: aws.String(emailBody)},
				Text: &ses.Content{Data: aws.String(emailText)},
			},
			Subject: &ses.Content{
				Data: aws.String(subject),
			},
		},
		Destination: &ses.Destination{
			ToAddresses: []*string{aws.String(destination)},
		},
		Source:           aws.String("HeartPoke <no-reply@heartpoke.co.uk>"),
		ReplyToAddresses: []*string{aws.String(replyTo)},
	}

	_, err = emailClient.SendEmail(emailParams)

	if err != nil {
		return err
	}

	return nil
}

func ParseRequestBody(req events.APIGatewayV2HTTPRequest) (request Request, err error) {
	b := []byte(req.Body)

	if req.IsBase64Encoded {
		base64Body, err := base64.StdEncoding.DecodeString(req.Body)
		if err != nil {
			return request, err
		}
		b = base64Body
	}

	err = json.Unmarshal(b, &request)

	if err != nil {
		return request, err
	}

	return request, nil
}

var corsHeaders = map[string]string{
	"Access-Control-Allow-Origin":  "*",
	"Access-Control-Allow-Methods": "OPTIONS,POST",
	"Access-Control-Allow-Headers": "Content-Type",
	"Content-Type":                 "application/json",
}

func Handler(_ context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	if req.RequestContext.HTTP.Method == "OPTIONS" {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusOK,
			Body:       "{\"status\": \"ok\"}",
			Headers:    corsHeaders,
		}, nil
	}

	request, err := ParseRequestBody(req)
	if err != nil {
		return ReturnErrorToUser(err, http.StatusBadRequest)
	}

	err = validate.Struct(request)
	if err != nil {
		return ReturnErrorToUser(err, http.StatusBadRequest)
	}

	adminEmail := GenerateAdminEmail(request)
	adminEmailSubject := fmt.Sprintf("Booking request from %s", request.Email)
	refs := GenerateImagesHTML(request.References)
	err = SendEmail(adminEmail, os.Getenv("ADMIN_EMAIL"), adminEmailSubject, request.Email, refs)

	if err != nil {
		return ReturnErrorToUser(err, http.StatusInternalServerError)
	}

	customerEmail := GenerateCustomerEmail(request.Name)
	err = SendEmail(customerEmail, request.Email, "Booking received", "no-reply@heartpoke.co.uk", "")

	if err != nil {
		return ReturnErrorToUser(err, http.StatusInternalServerError)
	}

	return events.APIGatewayV2HTTPResponse{StatusCode: http.StatusOK}, nil
}

func main() {
	lambda.Start(Handler)
}
