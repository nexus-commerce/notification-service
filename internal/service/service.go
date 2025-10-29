package service

import (
	"bytes"
	"context"
	"fmt"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"html/template"
	"log"
	"time"
)

type OrderCreatedEvent struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Data      OrderData `json:"data"`
}

type OrderData struct {
	CustomerFirstName string           `json:"customer_first_name"`
	CustomerLastName  string           `json:"customer_last_name"`
	CustomerEmail     string           `json:"customer_email"`
	OrderID           int64            `json:"order_id"`
	OrderDate         time.Time        `json:"order_date"`
	PaymentMethod     string           `json:"payment_method"`
	PaymentIntentID   *string          `json:"payment_intent_id"`
	UserID            int64            `json:"user_id"`
	Items             []*OrderItemData `json:"items"`
	Amount            float64          `json:"amount"`
	ShippingAddress   string           `json:"shipping_address"`
	EstimatedDelivery time.Time        `json:"estimated_delivery"`
}

type OrderItemData struct {
	Quantity       int32        `json:"quantity"`
	Price          float64      `json:"price"`
	Sku            string       `json:"sku"`
	Name           string       `json:"name"`
	ImageURL       template.URL `json:"image_url"`
	ItemTotalPrice float64      `json:"item_total_price"`
}

type UserRegisteredEvent struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Data      UserData  `json:"data"`
}

type UserData struct {
	UserID    int64  `json:"user_id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	LoginURL  string `json:"login_url"`
}

const orderConfirmedTemplate = "order-confirmed.page.gohtml"
const userRegisteredTemplate = "user-registered.page.gohtml"

type Service struct {
	sendGridClient *sendgrid.Client
	templateCache  map[string]*template.Template
}

func New(sendGridClient *sendgrid.Client, templateCache map[string]*template.Template) *Service {
	return &Service{
		sendGridClient: sendGridClient,
		templateCache:  templateCache,
	}
}

func (s *Service) SendOrderConfirmationEmail(ctx context.Context, eventData OrderData) error {
	ts, ok := s.templateCache[orderConfirmedTemplate]
	if !ok {
		return fmt.Errorf("the template %s does not exist", orderConfirmedTemplate)
	}
	var renderedHTML bytes.Buffer
	if err := ts.Execute(&renderedHTML, eventData); err != nil {
		return err
	}

	for _, item := range eventData.Items {
		fmt.Println(item.ImageURL)
	}

	// SendGrid setup
	from := mail.NewEmail("MyEcom", "contact@my-ecom-project.dynv6.net")
	subject := fmt.Sprintf("Your Order Confirmation - #%d", eventData.OrderID)
	name := fmt.Sprintf("%s %s", eventData.CustomerFirstName, eventData.CustomerLastName)
	to := mail.NewEmail(name, eventData.CustomerEmail)
	plainTextContent := fmt.Sprintf("Thank you for your order #%d", eventData.OrderID)
	htmlContent := renderedHTML.String()

	m := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)

	response, err := s.sendGridClient.SendWithContext(ctx, m)
	if err != nil {
		return err
	}

	log.Printf("Email sent, status code %d", response.StatusCode)

	return nil
}

func (s *Service) SendWelcomeEmail(ctx context.Context, eventData UserData) error {
	ts, ok := s.templateCache[userRegisteredTemplate]
	if !ok {
		return fmt.Errorf("the template %s does not exist", userRegisteredTemplate)
	}
	var renderedHTML bytes.Buffer
	if err := ts.Execute(&renderedHTML, eventData); err != nil {
		return err
	}

	// SendGrid setup
	from := mail.NewEmail("MyEcom", "contact@my-ecom-project.dynv6.net")
	subject := "Your Registration"
	name := fmt.Sprintf("%s %s", eventData.FirstName, eventData.LastName)
	to := mail.NewEmail(name, eventData.Email)
	plainTextContent := "Thank you for registering!"
	htmlContent := renderedHTML.String()

	m := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)

	response, err := s.sendGridClient.SendWithContext(ctx, m)
	if err != nil {
		return err
	}

	log.Printf("Email sent, status code %d", response.StatusCode)

	return nil
}
