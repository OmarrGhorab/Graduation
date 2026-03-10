package http

import (
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/persistence/postgres"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http/dto"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// TeacherHandler handles teacher-related HTTP requests
type TeacherHandler struct {
	ratingRepo *postgres.TeacherRatingRepository
	authClient *authclient.Client
}

func NewTeacherHandler(ratingRepo *postgres.TeacherRatingRepository, authClient *authclient.Client) *TeacherHandler {
	return &TeacherHandler{
		ratingRepo: ratingRepo,
		authClient: authClient,
	}
}

func (h *TeacherHandler) RegisterRoutes(router fiber.Router) {
	teachers := router.Group("/teachers")
	teachers.Get("/top-rated", h.GetTopRatedTeachers)
	teachers.Get("/:id/rating", h.GetTeacherRating)
}

// GetTopRatedTeachers godoc
// @Summary Get top-rated teachers
// @Tags teachers
// @Produce json
// @Param limit query int false "Limit" default(10)
// @Param minRating query float64 false "Minimum rating" default(4.0)
// @Success 200 {array} dto.TeacherRatingResponse
// @Router /api/v1/teachers/top-rated [get]
func (h *TeacherHandler) GetTopRatedTeachers(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 10)
	if limit < 1 || limit > 100 {
		limit = 10
	}

	minRating := c.QueryFloat("minRating", 4.0)
	if minRating < 0 || minRating > 5 {
		minRating = 4.0
	}

	// Get top-rated teachers from database
	var ratings []postgres.TeacherAvgRating
	err := h.ratingRepo.GetTopRatedTeachers(c.Context(), limit, minRating, &ratings)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to fetch top-rated teachers",
		})
	}

	// Fetch teacher info for each rating
	var responses []dto.TeacherRatingResponse
	for _, rating := range ratings {
		response := dto.TeacherRatingResponse{
			TeacherID:    rating.TeacherID,
			AverageRating: rating.AvgRating,
			TotalRatings: rating.TotalRatings,
		}

		// Fetch teacher info from auth service
		if h.authClient != nil {
			userInfo, err := h.authClient.GetUserInfo(c.Context(), rating.TeacherID.String())
			if err == nil && userInfo != nil {
				response.TeacherName = userInfo.Name
				response.TeacherProfileImg = userInfo.ProfileImg
			}
		}

		responses = append(responses, response)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responses,
	})
}

// GetTeacherRating godoc
// @Summary Get rating for a specific teacher
// @Tags teachers
// @Produce json
// @Param id path string true "Teacher ID"
// @Success 200 {object} dto.TeacherRatingResponse
// @Router /api/v1/teachers/{id}/rating [get]
func (h *TeacherHandler) GetTeacherRating(c *fiber.Ctx) error {
	teacherID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid teacher ID",
		})
	}

	// Get teacher rating
	rating, err := h.ratingRepo.GetTeacherAvgRating(c.Context(), teacherID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to fetch teacher rating",
		})
	}

	if rating == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Teacher rating not found",
		})
	}

	response := dto.TeacherRatingResponse{
		TeacherID:     rating.TeacherID,
		AverageRating: rating.AvgRating,
		TotalRatings:  rating.TotalRatings,
	}

	// Fetch teacher info from auth service
	if h.authClient != nil {
		userInfo, err := h.authClient.GetUserInfo(c.Context(), teacherID.String())
		if err == nil && userInfo != nil {
			response.TeacherName = userInfo.Name
			response.TeacherProfileImg = userInfo.ProfileImg
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    response,
	})
}
