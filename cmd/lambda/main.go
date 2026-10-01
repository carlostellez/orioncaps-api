// Entrada para AWS Lambda detras de API Gateway (REST, payload v1).
package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"

	"orioncaps-api/internal/config"
	"orioncaps-api/internal/contact"
	"orioncaps-api/internal/email"
	"orioncaps-api/internal/server"
)

func main() {
	cfg := config.Load()
	mailer := contact.NewMailer(email.NewMailgun(cfg), cfg)
	lambda.Start(httpadapter.New(server.New(cfg, mailer)).ProxyWithContext)
}
