package http

import (
	cartApp "github.com/OmarrGhorab/payment-service/internal/application/cart"
	"github.com/OmarrGhorab/payment-service/internal/application/payment"
	cartDomain "github.com/OmarrGhorab/payment-service/internal/domain/cart"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/paymob"
	"github.com/OmarrGhorab/payment-service/internal/interfaces/http/dto"
	"github.com/OmarrGhorab/payment-service/internal/interfaces/http/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type CartHandler struct {
	cartService    *cartApp.Service
	paymentService *payment.Service
	authClient     *authclient.Client
}

func NewCartHandler(cartSvc *cartApp.Service, paymentSvc *payment.Service, auth *authclient.Client) *CartHandler {
	return &CartHandler{
		cartService:    cartSvc,
		paymentService: paymentSvc,
		authClient:     auth,
	}
}

func (h *CartHandler) RegisterRoutes(router fiber.Router) {
	cart := router.Group("/cart", middleware.Authenticate(h.authClient))

	cart.Post("/add", h.AddToCart)
	cart.Post("/remove", h.RemoveFromCart)
	cart.Get("/", h.GetCart)
	cart.Delete("/clear", h.ClearCart)
	cart.Post("/checkout", h.CheckoutCart)
}

func (h *CartHandler) AddToCart(c *fiber.Ctx) error {
	var req dto.AddToCartRequest
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}

	userIDStr := c.Locals("userId").(string)
	userID, _ := uuid.Parse(userIDStr)
	courseID, _ := uuid.Parse(req.CourseID)

	billingType := cartDomain.BillingTypeOneTime
	if req.BillingType == "MONTHLY" {
		billingType = cartDomain.BillingTypeMonthly
	}

	err := h.cartService.AddToCart(c.Context(), cartApp.AddToCartOptions{
		UserID:      userID,
		CourseID:    courseID,
		BillingType: billingType,
	})

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Course added to cart",
	})
}

func (h *CartHandler) RemoveFromCart(c *fiber.Ctx) error {
	var req dto.RemoveFromCartRequest
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}

	userIDStr := c.Locals("userId").(string)
	userID, _ := uuid.Parse(userIDStr)
	courseID, _ := uuid.Parse(req.CourseID)

	err := h.cartService.RemoveFromCart(c.Context(), userID, courseID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Course removed from cart",
	})
}

func (h *CartHandler) GetCart(c *fiber.Ctx) error {
	userIDStr := c.Locals("userId").(string)
	userID, _ := uuid.Parse(userIDStr)

	cart, err := h.cartService.GetCart(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Cart not found",
		})
	}

	total, currency, _ := h.cartService.GetCartTotal(c.Context(), userID)

	var items []dto.CartItemResponse
	for _, item := range cart.Items {
		items = append(items, dto.CartItemResponse{
			ID:          item.ID.String(),
			CourseID:    item.CourseID.String(),
			BillingType: string(item.BillingType),
			PriceCents:  item.PriceCents,
			Currency:    item.Currency,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": dto.CartResponse{
			ID:         cart.ID.String(),
			Items:      items,
			TotalCents: total,
			Currency:   currency,
		},
	})
}

func (h *CartHandler) ClearCart(c *fiber.Ctx) error {
	userIDStr := c.Locals("userId").(string)
	userID, _ := uuid.Parse(userIDStr)

	err := h.cartService.ClearCart(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Cart cleared",
	})
}

func (h *CartHandler) CheckoutCart(c *fiber.Ctx) error {
	var req dto.CheckoutCartRequest
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}

	userIDStr := c.Locals("userId").(string)
	userID, _ := uuid.Parse(userIDStr)

	pmID := uuid.Nil
	if req.PaymentMethodID != "" {
		pmID, _ = uuid.Parse(req.PaymentMethodID)
	}

	paymentURL, orderID, err := h.paymentService.CheckoutCart(c.Context(), payment.CheckoutCartOptions{
		UserID:          userID,
		PaymentMethod:   req.PaymentMethod,
		PaymentMethodID: pmID,
		PhoneNumber:     req.PhoneNumber,
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

	return c.JSON(fiber.Map{
		"success": true,
		"data": dto.CreatePaymentResponse{
			PaymentURL:     paymentURL,
			PaymentOrderID: orderID.String(),
		},
	})
}

func (h *CartHandler) defaultIfEmpty(val, def string) string {
	if val == "" {
		return def
	}
	return val
}
