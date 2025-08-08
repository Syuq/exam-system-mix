package handlers

import (
	"exam-system/middleware"
	"exam-system/models"
	"exam-system/services"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type QuestionHandler struct {
	questionService *services.QuestionService
	logger          *logrus.Logger
}

func NewQuestionHandler(questionService *services.QuestionService, logger *logrus.Logger) *QuestionHandler {
	return &QuestionHandler{
		questionService: questionService,
		logger:          logger,
	}
}

// GetQuestions returns a paginated list of questions
// @Summary Get questions list
// @Description Get a paginated list of questions with optional filtering
// @Tags questions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Param tags query string false "Comma-separated list of tags"
// @Param difficulty query string false "Question difficulty (easy, medium, hard)"
// @Param type query string false "Question type (multiple_choice, true_false)"
// @Param search query string false "Search term"
// @Param is_active query bool false "Filter by active status"
// @Success 200 {object} services.QuestionListResponse "Questions list"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/questions [get]
func (h *QuestionHandler) GetQuestions(c *gin.Context) {
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

	// Parse filters
	filter := services.QuestionFilter{
		Search: c.Query("search"),
	}

	if tagsStr := c.Query("tags"); tagsStr != "" {
		filter.Tags = strings.Split(tagsStr, ",")
		// Trim whitespace from tags
		for i, tag := range filter.Tags {
			filter.Tags[i] = strings.TrimSpace(tag)
		}
	}

	if difficulty := c.Query("difficulty"); difficulty != "" {
		filter.Difficulty = models.QuestionDifficulty(difficulty)
	}

	if questionType := c.Query("type"); questionType != "" {
		filter.Type = models.QuestionType(questionType)
	}

	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		if isActive, err := strconv.ParseBool(isActiveStr); err == nil {
			filter.IsActive = &isActive
		}
	}

	questions, err := h.questionService.GetQuestions(page, pageSize, filter)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"page":       page,
			"page_size":  pageSize,
			"filter":     filter,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to get questions")

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "QUESTIONS_FETCH_FAILED", "Failed to get questions", nil)
		return
	}

	c.JSON(http.StatusOK, questions)
}

// GetQuestion returns a specific question by ID
// @Summary Get question by ID
// @Description Get a specific question by its ID
// @Tags questions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Question ID"
// @Success 200 {object} map[string]interface{} "Question details"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Question not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/questions/{id} [get]
func (h *QuestionHandler) GetQuestion(c *gin.Context) {
	questionIDStr := c.Param("id")
	questionID, err := strconv.ParseUint(questionIDStr, 10, 32)
	if err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_QUESTION_ID", "Invalid question ID", nil)
		return
	}

	// Check if user is admin to include correct answers
	isAdmin := middleware.IsAdmin(c)

	question, err := h.questionService.GetQuestion(uint(questionID), isAdmin)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"question_id": questionID,
			"request_id":  middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to get question")

		if err.Error() == "question not found" {
			middleware.StructuredErrorResponse(c, http.StatusNotFound, "QUESTION_NOT_FOUND", "Question not found", nil)
			return
		}

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "QUESTION_FETCH_FAILED", "Failed to get question", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"question": question.ToResponse(isAdmin),
	})
}

// CreateQuestion creates a new question (admin only)
// @Summary Create question
// @Description Create a new question (admin only)
// @Tags questions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body services.CreateQuestionRequest true "Question data"
// @Success 201 {object} map[string]interface{} "Question created successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/questions [post]
func (h *QuestionHandler) CreateQuestion(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		middleware.StructuredErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	var req services.CreateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request data", err.Error())
		return
	}

	question, err := h.questionService.CreateQuestion(req, userID)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":    userID,
			"title":      req.Title,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to create question")

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "QUESTION_CREATE_FAILED", "Failed to create question", nil)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"question_id": question.ID,
		"title":       question.Title,
		"user_id":     userID,
		"request_id":  middleware.GetRequestID(c),
	}).Info("Question created successfully")

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Question created successfully",
		"question": question.ToResponse(true), // Include correct answers for admin
	})
}

// UpdateQuestion updates a specific question (admin only)
// @Summary Update question
// @Description Update a specific question (admin only)
// @Tags questions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Question ID"
// @Param request body services.UpdateQuestionRequest true "Question update data"
// @Success 200 {object} map[string]interface{} "Question updated successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 404 {object} map[string]interface{} "Question not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/questions/{id} [put]
func (h *QuestionHandler) UpdateQuestion(c *gin.Context) {
	questionIDStr := c.Param("id")
	questionID, err := strconv.ParseUint(questionIDStr, 10, 32)
	if err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_QUESTION_ID", "Invalid question ID", nil)
		return
	}

	var req services.UpdateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request data", err.Error())
		return
	}

	question, err := h.questionService.UpdateQuestion(uint(questionID), req)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"question_id": questionID,
			"title":       req.Title,
			"request_id":  middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to update question")

		if err.Error() == "question not found" {
			middleware.StructuredErrorResponse(c, http.StatusNotFound, "QUESTION_NOT_FOUND", "Question not found", nil)
			return
		}

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "QUESTION_UPDATE_FAILED", "Failed to update question", nil)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"question_id": questionID,
		"title":       question.Title,
		"request_id":  middleware.GetRequestID(c),
	}).Info("Question updated successfully")

	c.JSON(http.StatusOK, gin.H{
		"message":  "Question updated successfully",
		"question": question.ToResponse(true), // Include correct answers for admin
	})
}

// DeleteQuestion deletes a specific question (admin only)
// @Summary Delete question
// @Description Delete a specific question (admin only)
// @Tags questions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Question ID"
// @Success 200 {object} map[string]interface{} "Question deleted successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 404 {object} map[string]interface{} "Question not found"
// @Failure 409 {object} map[string]interface{} "Question is used in active exams"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/questions/{id} [delete]
func (h *QuestionHandler) DeleteQuestion(c *gin.Context) {
	questionIDStr := c.Param("id")
	questionID, err := strconv.ParseUint(questionIDStr, 10, 32)
	if err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_QUESTION_ID", "Invalid question ID", nil)
		return
	}

	if err := h.questionService.DeleteQuestion(uint(questionID)); err != nil {
		h.logger.WithFields(logrus.Fields{
			"question_id": questionID,
			"request_id":  middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to delete question")

		if err.Error() == "question not found" {
			middleware.StructuredErrorResponse(c, http.StatusNotFound, "QUESTION_NOT_FOUND", "Question not found", nil)
			return
		}

		if strings.Contains(err.Error(), "used in active or draft exams") {
			middleware.StructuredErrorResponse(c, http.StatusConflict, "QUESTION_IN_USE", "Cannot delete question as it is used in active or draft exams", nil)
			return
		}

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "QUESTION_DELETE_FAILED", "Failed to delete question", nil)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"question_id": questionID,
		"request_id":  middleware.GetRequestID(c),
	}).Info("Question deleted successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Question deleted successfully",
	})
}

// GetTags returns all available question tags
// @Summary Get question tags
// @Description Get all available question tags
// @Tags questions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Tags list"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/questions/tags [get]
func (h *QuestionHandler) GetTags(c *gin.Context) {
	tags, err := h.questionService.GetAllTags()
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to get question tags")

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "TAGS_FETCH_FAILED", "Failed to get question tags", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tags": tags,
	})
}

