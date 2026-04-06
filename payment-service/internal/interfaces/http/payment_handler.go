package http

import (
	"github.com/OmarrGhorab/payment-service/internal/application/payment"
	paymentDomain "github.com/OmarrGhorab/payment-service/internal/domain/payment"
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
	// Public routes
	router.Get("/payments/:id/status", h.GetStatus)
	router.Get("/payments/status", h.GetStatus)

	// Authenticated routes
	payments := router.Group("/payments", middleware.Authenticate(h.authClient))
	payments.Post("/create", h.CreatePayment)
	payments.Get("/methods", h.ListPaymentMethods)
}

func (h *PaymentHandler) CreatePayment(c *fiber.Ctx) error {
	var req dto.CreatePaymentRequest
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}

	userIDStr := c.Locals("userId").(string)
	userID, _ := uuid.Parse(userIDStr)
	courseID, _ := uuid.Parse(req.CourseID)

	// 1. Check live database/service first (Highest priority)
	isEnrolled, isPaid, err := h.paymentService.GetEnrollmentStatus(c.Context(), userIDStr, req.CourseID)
	if err == nil && isEnrolled && isPaid {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false,
			"error":   "You are already enrolled/paid for this course",
		})
	}

	// 2. Check for idempotency key (Medium priority)
	idempotencyKey := c.Get("Idempotency-Key")
	if idempotencyKey != "" {
		if cached, _ := h.paymentService.GetIdempotentResponse(c.Context(), userIDStr, req.CourseID, idempotencyKey); cached != nil {
			return c.JSON(fiber.Map{
				"success": true,
				"data":    cached,
			})
		}
	}


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
			Street:      h.defaultIfEmpty(req.Street, "N/A"),
			Building:    h.defaultIfEmpty(req.Building, "N/A"),
			Floor:       h.defaultIfEmpty(req.Floor, "N/A"),
			Apartment:   h.defaultIfEmpty(req.Apartment, "N/A"),
			City:        h.defaultIfEmpty(req.City, "N/A"),
			State:       h.defaultIfEmpty(req.State, "N/A"),
			Country:     h.defaultIfEmpty(req.Country, "Egypt"),
		},
		SaveCard: req.SaveCard,
	})


	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	response := dto.CreatePaymentResponse{
		PaymentURL:     paymentURL,
		PaymentOrderID: orderID.String(),
	}

	// Cache for idempotency if key was provided
	if idempotencyKey != "" {
		h.paymentService.CacheIdempotentResponse(c.Context(), userIDStr, req.CourseID, idempotencyKey, response)
	}


	return c.JSON(fiber.Map{
		"success": true,
		"data":    response,
	})
}

func (h *PaymentHandler) GetStatus(c *fiber.Ctx) error {
	idParam := c.Params("id")
	if idParam == "" {
		idParam = c.Query("id")
	}

	// Try to get userID from locals if authenticated
	var userID uuid.UUID
	if val := c.Locals("userId"); val != nil {
		if id, ok := val.(string); ok {
			userID, _ = uuid.Parse(id)
		}
	}

	// 1. Try to parse as UUID (Our internal ID)
	orderID, err := uuid.Parse(idParam)
	var order *paymentDomain.PaymentOrder
	
	if err == nil {
		// Found UUID, fetch by internal ID
		order, err = h.paymentService.GetOrderStatus(c.Context(), userID, orderID)
	} else {
		// Not a UUID, check if it's a numeric Paymob ID (either 'id' or 'order' param)
		paymobOrderID := c.Query("order")
		if paymobOrderID == "" {
			paymobOrderID = idParam // Fallback to the main ID param
		}
		
		// If it's numeric, we skip the UUID check and fetch from service by Paymob ID
		order, err = h.paymentService.GetOrderByPaymobID(c.Context(), paymobOrderID)
	}

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	// Get first course ID from order items for backward compatibility
	courseID := ""
	if len(order.Items) > 0 {
		courseID = order.Items[0].CourseID.String()
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": dto.PaymentStatusResponse{
			OrderID:   order.ID.String(),
			UserID:    order.UserID.String(),
			CourseID:  courseID,
			Amount:    order.AmountCents,
			Currency:  order.Currency,
			Status:    string(order.Status),
			CreatedAt: order.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	})
}

func (h *PaymentHandler) defaultIfEmpty(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

func (h *PaymentHandler) ListPaymentMethods(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, _ := uuid.Parse(userIDStr)

	methods, err := h.paymentService.GetUserPaymentMethods(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    methods,
	})
}

