package handlers

import (
	"exam-system/middleware"
	"exam-system/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type ResultHandler struct {
	resultService *services.ResultService
	logger        *logrus.Logger
}

func NewResultHandler(resultService *services.ResultService, logger *logrus.Logger) *ResultHandler {
	return &ResultHandler{
		resultService: resultService,
		logger:        logger,
	}
}

// GetResults returns a paginated list of results
// @Summary Get results list
// @Description Get a paginated list of exam results (admin sees all, users see their own)
// @Tags results
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Param exam_id query int false "Filter by exam ID"
// @Success 200 {object} services.ResultListResponse "Results list"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/results [get]
func (h *ResultHandler) GetResults(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		middleware.StructuredErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// Parse exam_id filter
	var examID *uint
	if examIDStr := c.Query("exam_id"); examIDStr != "" {
		if id, err := strconv.ParseUint(examIDStr, 10, 32); err == nil {
			examIDUint := uint(id)
			examID = &examIDUint
		}
	}

	isAdmin := middleware.IsAdmin(c)

	results, err := h.resultService.GetResults(page, pageSize, userID, examID, isAdmin)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":    userID,
			"page":       page,
			"page_size":  pageSize,
			"exam_id":    examID,
			"is_admin":   isAdmin,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to get results")

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "RESULTS_FETCH_FAILED", "Failed to get results", nil)
		return
	}

	c.JSON(http.StatusOK, results)
}

// GetResult returns a specific result by ID
// @Summary Get result by ID
// @Description Get a specific exam result by its ID
// @Tags results
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Result ID"
// @Success 200 {object} map[string]interface{} "Result details"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Result not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/results/{id} [get]
func (h *ResultHandler) GetResult(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		middleware.StructuredErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	resultIDStr := c.Param("id")
	resultID, err := strconv.ParseUint(resultIDStr, 10, 32)
	if err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_RESULT_ID", "Invalid result ID", nil)
		return
	}

	isAdmin := middleware.IsAdmin(c)

	result, err := h.resultService.GetResult(uint(resultID), userID, isAdmin)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"result_id":  resultID,
			"user_id":    userID,
			"is_admin":   isAdmin,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to get result")

		if err.Error() == "result not found" {
			middleware.StructuredErrorResponse(c, http.StatusNotFound, "RESULT_NOT_FOUND", "Result not found", nil)
			return
		}

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "RESULT_FETCH_FAILED", "Failed to get result", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"result": result.ToResponse(true, isAdmin), // Include answers, correct answers for admin
	})
}

// GetUserResults returns results for the current user
// @Summary Get current user's results
// @Description Get exam results for the currently authenticated user
// @Tags results
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} services.ResultListResponse "User's results"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/results/my-results [get]
func (h *ResultHandler) GetUserResults(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		middleware.StructuredErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	results, err := h.resultService.GetUserResults(userID, page, pageSize)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":    userID,
			"page":       page,
			"page_size":  pageSize,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to get user results")

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "USER_RESULTS_FETCH_FAILED", "Failed to get user results", nil)
		return
	}

	c.JSON(http.StatusOK, results)
}

// GetExamResults returns results for a specific exam (admin only)
// @Summary Get exam results
// @Description Get all results for a specific exam (admin only)
// @Tags results
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param exam_id path int true "Exam ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} services.ResultListResponse "Exam results"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/results/exam/{exam_id} [get]
func (h *ResultHandler) GetExamResults(c *gin.Context) {
	examIDStr := c.Param("exam_id")
	examID, err := strconv.ParseUint(examIDStr, 10, 32)
	if err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_EXAM_ID", "Invalid exam ID", nil)
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	results, err := h.resultService.GetExamResults(uint(examID), page, pageSize)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"exam_id":    examID,
			"page":       page,
			"page_size":  pageSize,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to get exam results")

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "EXAM_RESULTS_FETCH_FAILED", "Failed to get exam results", nil)
		return
	}

	c.JSON(http.StatusOK, results)
}

// GetStatistics returns comprehensive statistics (admin only)
// @Summary Get statistics
// @Description Get comprehensive exam, user, and question statistics (admin only)
// @Tags results
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} services.StatisticsResponse "Statistics data"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/results/statistics [get]
func (h *ResultHandler) GetStatistics(c *gin.Context) {
	statistics, err := h.resultService.GetStatistics()
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to get statistics")

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "STATISTICS_FETCH_FAILED", "Failed to get statistics", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"statistics": statistics,
	})
}

// GetUserStatistics returns statistics for the current user
// @Summary Get current user's statistics
// @Description Get exam statistics for the currently authenticated user
// @Tags results
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "User statistics"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/results/my-statistics [get]
func (h *ResultHandler) GetUserStatistics(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		middleware.StructuredErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	// Get user's results to calculate statistics
	results, err := h.resultService.GetUserResults(userID, 1, 1000) // Get all results
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":    userID,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to get user results for statistics")

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "USER_STATISTICS_FETCH_FAILED", "Failed to get user statistics", nil)
		return
	}

	// Calculate user statistics
	totalExams := len(results.Results)
	passedExams := 0
	totalScore := 0.0
	totalTimeSpent := 0
	highestScore := 0.0
	lowestScore := 100.0

	for _, result := range results.Results {
		if result.Passed {
			passedExams++
		}
		totalScore += result.Score
		totalTimeSpent += result.Duration

		if result.Score > highestScore {
			highestScore = result.Score
		}
		if result.Score < lowestScore {
			lowestScore = result.Score
		}
	}

	var averageScore float64
	var passRate float64

	if totalExams > 0 {
		averageScore = totalScore / float64(totalExams)
		passRate = float64(passedExams) / float64(totalExams) * 100
	}

	if totalExams == 0 {
		lowestScore = 0
	}

	statistics := gin.H{
		"total_exams":      totalExams,
		"passed_exams":     passedExams,
		"failed_exams":     totalExams - passedExams,
		"pass_rate":        passRate,
		"average_score":    averageScore,
		"highest_score":    highestScore,
		"lowest_score":     lowestScore,
		"total_time_spent": totalTimeSpent,
	}

	c.JSON(http.StatusOK, gin.H{
		"statistics": statistics,
	})
}

