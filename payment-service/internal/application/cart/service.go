package cart

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/OmarrGhorab/payment-service/internal/domain/cart"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/coursesclient"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/persistence/postgres"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	repo          *postgres.CartRepository
	coursesClient *coursesclient.Client
}

func NewService(repo *postgres.CartRepository, coursesClient *coursesclient.Client) *Service {
	return &Service{
		repo:          repo,
		coursesClient: coursesClient,
	}
}

type AddToCartOptions struct {
	UserID      uuid.UUID
	CourseID    uuid.UUID
	BillingType cart.BillingType
}

func (s *Service) AddToCart(ctx context.Context, opts AddToCartOptions) error {
	// Validate course exists and get pricing
	course, err := s.coursesClient.GetCourseByID(ctx, opts.CourseID.String())
	if err != nil {
		return fmt.Errorf("failed to fetch course: %w", err)
	}

	if !course.IsPaid || course.Price <= 0 {
		return errors.New("course is free or price is invalid")
	}

	// Check if user is the owner
	if course.TeacherID == opts.UserID.String() {
		return errors.New("you cannot add your own course to cart")
	}

	// Check if already enrolled and paid
	isEnrolled, isPaid, err := s.coursesClient.CheckEnrollment(ctx, opts.UserID.String(), opts.CourseID.String())
	if err == nil && isEnrolled && isPaid {
		return errors.New("you are already enrolled and paid for this course")
	}

	// Get or create cart
	userCart, err := s.repo.GetOrCreateCart(ctx, opts.UserID)
	if err != nil {
		return fmt.Errorf("failed to get cart: %w", err)
	}

	// Add item to cart
	item := &cart.CartItem{
		CartID:      userCart.ID,
		CourseID:    opts.CourseID,
		BillingType: opts.BillingType,
		PriceCents:  int64(math.Round(course.Price * 100)),
		Currency:    course.Currency,
	}

	if err := s.repo.AddItem(ctx, item); err != nil {
		return fmt.Errorf("failed to add item to cart: %w", err)
	}

	return nil
}

func (s *Service) RemoveFromCart(ctx context.Context, userID, courseID uuid.UUID) error {
	userCart, err := s.repo.GetCartWithItems(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("cart not found")
		}
		return err
	}

	return s.repo.RemoveItem(ctx, userCart.ID, courseID)
}

func (s *Service) GetCart(ctx context.Context, userID uuid.UUID) (*cart.Cart, error) {
	return s.repo.GetCartWithItems(ctx, userID)
}

func (s *Service) ClearCart(ctx context.Context, userID uuid.UUID) error {
	userCart, err := s.repo.GetCartWithItems(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // Already empty
		}
		return err
	}

	return s.repo.ClearCart(ctx, userCart.ID)
}

func (s *Service) GetCartTotal(ctx context.Context, userID uuid.UUID) (int64, string, error) {
	userCart, err := s.repo.GetCartWithItems(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, "EGP", nil
		}
		return 0, "", err
	}

	if len(userCart.Items) == 0 {
		return 0, "EGP", nil
	}

	var total int64
	currency := userCart.Items[0].Currency

	for _, item := range userCart.Items {
		if item.Currency != currency {
			return 0, "", errors.New("mixed currencies in cart not supported")
		}
		total += item.PriceCents
	}

	return total, currency, nil
}

func (s *Service) GetCourseInfo(ctx context.Context, courseID string) (*coursesclient.CourseInfo, error) {
	return s.coursesClient.GetCourseByID(ctx, courseID)
}
