package http

import (
	"github.com/OmarrGhorab/payment-service/internal/application/payment"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/paymob"
	"github.com/OmarrGhorab/payment-service/internal/interfaces/http/dto"
	"github.com/OmarrGhorab/payment-service/internal/interfaces/http/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type PaymentHandler struct {
	paymentService *payment.Service
	authClient     *authclient.Client
}

func NewPaymentHandler(svc *payment.Service, auth *authclient.Client) *PaymentHandler {
	return &PaymentHandler{
		paymentService: svc,
		authClient:     auth,
	}
}

func (h *PaymentHandler) RegisterRoutes(router fiber.Router) {
	payments := router.Group("/payments", middleware.Authenticate(h.authClient))

	payments.Post("/create", h.CreatePayment)
	payments.Get("/:id/status", h.GetStatus)
}

func (h *PaymentHandler) CreatePayment(c *fiber.Ctx) error {
	var req dto.CreatePaymentRequest
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}

	userIDStr := c.Locals("userId").(string)
	userID, _ := uuid.Parse(userIDStr)
	courseID, _ := uuid.Parse(req.CourseID)

	paymentURL, orderID, err := h.paymentService.CreatePayment(c.Context(), payment.CreatePaymentOptions{
		UserID:        userID,
		CourseID:      courseID,
		PaymentMethod: req.PaymentMethod,
		PhoneNumber:   req.PhoneNumber,
		BillingData: paymob.BillingData{
			FirstName:   req.FirstName,
			LastName:    req.LastName,
			Email:       req.Email,
			PhoneNumber: req.PhoneNumber,
		},
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": dto.CreatePaymentResponse{
			PaymentURL:     paymentURL,
			PaymentOrderID: orderID.String(),
		},
	})
}

func (h *PaymentHandler) GetStatus(c *fiber.Ctx) error {
	userIDStr := c.Locals("userId").(string)
	userID, _ := uuid.Parse(userIDStr)
	orderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid order ID",
		})
	}

	order, err := h.paymentService.GetOrderStatus(c.Context(), userID, orderID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": dto.PaymentStatusResponse{
			OrderID:   order.ID.String(),
			UserID:    order.UserID.String(),
			CourseID:  order.CourseID.String(),
			Amount:    order.AmountCents,
			Currency:  order.Currency,
			Status:    string(order.Status),
			CreatedAt: order.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	})
}
