package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/go-playground/validator/v10"
	"github.com/matcornic/hermes/v2"
	"log"
	"net/http"
	"strconv"
)

type Request struct {
	Name        string `json:"name" validate:"required"`
	DOB         string `json:"dob" validate:"required"`
	Email       string `json:"email" validate:"required,email"`
	Placement   string `json:"placement" validate:"required"`
	Size        int    `json:"size" validate:"required,gte=0"`
	Description string `json:"description" validate:"required"`
	Comments    string `json:"comments"`
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
			Name: "HeartPoke",
			Link: "https://heartpoke.co.uk",
			Logo: "https://heartpoke.s3.eu-west-2.amazonaws.com/logo.png",
		},
	}

}

func ReturnErrorToUser(error error, status int) (events.APIGatewayProxyResponse, error) {
	log.Println(error.Error())

	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Headers:    map[string]string{"Content-Type": "text/plain"},
		Body:       error.Error(),
	}, nil
}

func GenerateAdminEmail(request Request) hermes.Email {
	return hermes.Email{
		Body: hermes.Body{
			Name: "Katenka",
			Intros: []string{
				"New booking request",
			},
			Dictionary: []hermes.Entry{
				{Key: "Name", Value: request.Name},
				{Key: "DOB", Value: request.DOB},
				{Key: "Email", Value: request.Email},
				{Key: "Placement", Value: request.Placement},
				{Key: "Size", Value: strconv.Itoa(request.Size)},
				{Key: "Description", Value: request.Description},
				{Key: "Comments", Value: request.Comments},
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

func SendEmail(email hermes.Email, subject string, replyTo string) error {
	emailBody, err := h.GenerateHTML(email)

	if err != nil {
		return err
	}

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
			ToAddresses: []*string{aws.String("zduderman@gmail.com")},
		},
		Source:           aws.String("no-reply@heartpoke.co.uk"),
		ReplyToAddresses: []*string{aws.String(replyTo)},
	}

	_, err = emailClient.SendEmail(emailParams)

	if err != nil {
		return err
	}

	return nil
}

func Handler(request Request) (events.APIGatewayProxyResponse, error) {
	log.Println(request)
	err := validate.Struct(request)

	if err != nil {
		return ReturnErrorToUser(err, http.StatusBadRequest)
	}

	adminEmail := GenerateAdminEmail(request)
	err = SendEmail(adminEmail, "New booking", request.Email)

	if err != nil {
		return ReturnErrorToUser(err, http.StatusInternalServerError)
	}

	customerEmail := GenerateCustomerEmail(request.Name)
	err = SendEmail(customerEmail, "Booking received", "no-reply@heartpoke.co.uk")

	if err != nil {
		return ReturnErrorToUser(err, http.StatusInternalServerError)
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(Handler)
}
