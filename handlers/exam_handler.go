package handlers

import (
	"exam-system/middleware"
	"exam-system/services"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type ExamHandler struct {
	examService *services.ExamService
	logger      *logrus.Logger
}

func NewExamHandler(examService *services.ExamService, logger *logrus.Logger) *ExamHandler {
	return &ExamHandler{
		examService: examService,
		logger:      logger,
	}
}

// GetExams returns a paginated list of exams
// @Summary Get exams list
// @Description Get a paginated list of exams (admin sees all, users see assigned exams)
// @Tags exams
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} services.ExamListResponse "Exams list"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/exams [get]
func (h *ExamHandler) GetExams(c *gin.Context) {
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

	isAdmin := middleware.IsAdmin(c)

	exams, err := h.examService.GetExams(page, pageSize, userID, isAdmin)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":    userID,
			"page":       page,
			"page_size":  pageSize,
			"is_admin":   isAdmin,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to get exams")

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "EXAMS_FETCH_FAILED", "Failed to get exams", nil)
		return
	}

	c.JSON(http.StatusOK, exams)
}

// GetExam returns a specific exam by ID
// @Summary Get exam by ID
// @Description Get a specific exam by its ID
// @Tags exams
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Exam ID"
// @Success 200 {object} map[string]interface{} "Exam details"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Exam not assigned to user"
// @Failure 404 {object} map[string]interface{} "Exam not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/exams/{id} [get]
func (h *ExamHandler) GetExam(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		middleware.StructuredErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	examIDStr := c.Param("id")
	examID, err := strconv.ParseUint(examIDStr, 10, 32)
	if err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_EXAM_ID", "Invalid exam ID", nil)
		return
	}

	isAdmin := middleware.IsAdmin(c)

	exam, userExam, err := h.examService.GetExam(uint(examID), userID, isAdmin)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"exam_id":    examID,
			"user_id":    userID,
			"is_admin":   isAdmin,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to get exam")

		if err.Error() == "exam not found" {
			middleware.StructuredErrorResponse(c, http.StatusNotFound, "EXAM_NOT_FOUND", "Exam not found", nil)
			return
		}

		if err.Error() == "exam not assigned to user" {
			middleware.StructuredErrorResponse(c, http.StatusForbidden, "EXAM_NOT_ASSIGNED", "Exam not assigned to user", nil)
			return
		}

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "EXAM_FETCH_FAILED", "Failed to get exam", nil)
		return
	}

	// Include questions for admin or if user has started the exam
	includeQuestions := isAdmin || (userExam != nil && userExam.Status == "started")

	c.JSON(http.StatusOK, gin.H{
		"exam": exam.ToResponse(includeQuestions, userExam),
	})
}

// CreateExam creates a new exam (admin only)
// @Summary Create exam
// @Description Create a new exam (admin only)
// @Tags exams
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body services.CreateExamRequest true "Exam data"
// @Success 201 {object} map[string]interface{} "Exam created successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/exams [post]
func (h *ExamHandler) CreateExam(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		middleware.StructuredErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	var req services.CreateExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request data", err.Error())
		return
	}

	exam, err := h.examService.CreateExam(req, userID)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":    userID,
			"title":      req.Title,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to create exam")

		if strings.Contains(err.Error(), "invalid or inactive") {
			middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_QUESTIONS", "Some questions are invalid or inactive", nil)
			return
		}

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "EXAM_CREATE_FAILED", "Failed to create exam", nil)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"exam_id":    exam.ID,
		"title":      exam.Title,
		"user_id":    userID,
		"request_id": middleware.GetRequestID(c),
	}).Info("Exam created successfully")

	c.JSON(http.StatusCreated, gin.H{
		"message": "Exam created successfully",
		"exam":    exam.ToResponse(true, nil), // Include questions for admin
	})
}

// UpdateExam updates a specific exam (admin only)
// @Summary Update exam
// @Description Update a specific exam (admin only)
// @Tags exams
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Exam ID"
// @Param request body services.UpdateExamRequest true "Exam update data"
// @Success 200 {object} map[string]interface{} "Exam updated successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 404 {object} map[string]interface{} "Exam not found"
// @Failure 409 {object} map[string]interface{} "Cannot update completed exam"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/exams/{id} [put]
func (h *ExamHandler) UpdateExam(c *gin.Context) {
	examIDStr := c.Param("id")
	examID, err := strconv.ParseUint(examIDStr, 10, 32)
	if err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_EXAM_ID", "Invalid exam ID", nil)
		return
	}

	var req services.UpdateExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request data", err.Error())
		return
	}

	exam, err := h.examService.UpdateExam(uint(examID), req)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"exam_id":    examID,
			"title":      req.Title,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to update exam")

		if err.Error() == "exam not found" {
			middleware.StructuredErrorResponse(c, http.StatusNotFound, "EXAM_NOT_FOUND", "Exam not found", nil)
			return
		}

		if err.Error() == "cannot update completed exam" {
			middleware.StructuredErrorResponse(c, http.StatusConflict, "EXAM_COMPLETED", "Cannot update completed exam", nil)
			return
		}

		if strings.Contains(err.Error(), "invalid or inactive") {
			middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_QUESTIONS", "Some questions are invalid or inactive", nil)
			return
		}

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "EXAM_UPDATE_FAILED", "Failed to update exam", nil)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"exam_id":    examID,
		"title":      exam.Title,
		"request_id": middleware.GetRequestID(c),
	}).Info("Exam updated successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Exam updated successfully",
		"exam":    exam.ToResponse(true, nil), // Include questions for admin
	})
}

// DeleteExam deletes a specific exam (admin only)
// @Summary Delete exam
// @Description Delete a specific exam (admin only)
// @Tags exams
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Exam ID"
// @Success 200 {object} map[string]interface{} "Exam deleted successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 404 {object} map[string]interface{} "Exam not found"
// @Failure 409 {object} map[string]interface{} "Cannot delete exam with existing results"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/exams/{id} [delete]
func (h *ExamHandler) DeleteExam(c *gin.Context) {
	examIDStr := c.Param("id")
	examID, err := strconv.ParseUint(examIDStr, 10, 32)
	if err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_EXAM_ID", "Invalid exam ID", nil)
		return
	}

	if err := h.examService.DeleteExam(uint(examID)); err != nil {
		h.logger.WithFields(logrus.Fields{
			"exam_id":    examID,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to delete exam")

		if err.Error() == "exam not found" {
			middleware.StructuredErrorResponse(c, http.StatusNotFound, "EXAM_NOT_FOUND", "Exam not found", nil)
			return
		}

		if strings.Contains(err.Error(), "existing results") {
			middleware.StructuredErrorResponse(c, http.StatusConflict, "EXAM_HAS_RESULTS", "Cannot delete exam with existing results", nil)
			return
		}

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "EXAM_DELETE_FAILED", "Failed to delete exam", nil)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"exam_id":    examID,
		"request_id": middleware.GetRequestID(c),
	}).Info("Exam deleted successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Exam deleted successfully",
	})
}

// AssignExam assigns an exam to users (admin only)
// @Summary Assign exam to users
// @Description Assign an exam to specific users (admin only)
// @Tags exams
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Exam ID"
// @Param request body services.AssignExamRequest true "Assignment data"
// @Success 200 {object} map[string]interface{} "Exam assigned successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 404 {object} map[string]interface{} "Exam not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/exams/{id}/assign [post]
func (h *ExamHandler) AssignExam(c *gin.Context) {
	examIDStr := c.Param("id")
	examID, err := strconv.ParseUint(examIDStr, 10, 32)
	if err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_EXAM_ID", "Invalid exam ID", nil)
		return
	}

	var req services.AssignExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request data", err.Error())
		return
	}

	if err := h.examService.AssignExam(uint(examID), req); err != nil {
		h.logger.WithFields(logrus.Fields{
			"exam_id":    examID,
			"user_ids":   req.UserIDs,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to assign exam")

		if err.Error() == "exam not found or inactive" {
			middleware.StructuredErrorResponse(c, http.StatusNotFound, "EXAM_NOT_FOUND", "Exam not found or inactive", nil)
			return
		}

		if strings.Contains(err.Error(), "invalid or inactive") {
			middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_USERS", "Some users are invalid or inactive", nil)
			return
		}

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "EXAM_ASSIGN_FAILED", "Failed to assign exam", nil)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"exam_id":    examID,
		"user_ids":   req.UserIDs,
		"request_id": middleware.GetRequestID(c),
	}).Info("Exam assigned successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Exam assigned successfully",
	})
}

// StartExam starts an exam for the current user
// @Summary Start exam
// @Description Start an assigned exam for the current user
// @Tags exams
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Exam ID"
// @Success 200 {object} services.StartExamResponse "Exam started successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Exam cannot be started"
// @Failure 404 {object} map[string]interface{} "Exam not assigned to user"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/exams/{id}/start [post]
func (h *ExamHandler) StartExam(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		middleware.StructuredErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	examIDStr := c.Param("id")
	examID, err := strconv.ParseUint(examIDStr, 10, 32)
	if err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_EXAM_ID", "Invalid exam ID", nil)
		return
	}

	response, err := h.examService.StartExam(uint(examID), userID)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"exam_id":    examID,
			"user_id":    userID,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to start exam")

		if err.Error() == "exam not assigned to user" {
			middleware.StructuredErrorResponse(c, http.StatusNotFound, "EXAM_NOT_ASSIGNED", "Exam not assigned to user", nil)
			return
		}

		if err.Error() == "exam cannot be started" {
			middleware.StructuredErrorResponse(c, http.StatusForbidden, "EXAM_CANNOT_START", "Exam cannot be started", nil)
			return
		}

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "EXAM_START_FAILED", "Failed to start exam", nil)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"exam_id":    examID,
		"user_id":    userID,
		"request_id": middleware.GetRequestID(c),
	}).Info("Exam started successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Exam started successfully",
		"data":    response,
	})
}

// SubmitExam submits an exam for the current user
// @Summary Submit exam
// @Description Submit answers for an exam
// @Tags exams
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Exam ID"
// @Param request body services.SubmitExamRequest true "Exam answers"
// @Success 200 {object} map[string]interface{} "Exam submitted successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Exam cannot be submitted"
// @Failure 404 {object} map[string]interface{} "Exam not assigned to user"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/exams/{id}/submit [post]
func (h *ExamHandler) SubmitExam(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		middleware.StructuredErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	examIDStr := c.Param("id")
	examID, err := strconv.ParseUint(examIDStr, 10, 32)
	if err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_EXAM_ID", "Invalid exam ID", nil)
		return
	}

	var req services.SubmitExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request data", err.Error())
		return
	}

	result, err := h.examService.SubmitExam(uint(examID), userID, req)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"exam_id":    examID,
			"user_id":    userID,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to submit exam")

		if err.Error() == "exam not assigned to user" {
			middleware.StructuredErrorResponse(c, http.StatusNotFound, "EXAM_NOT_ASSIGNED", "Exam not assigned to user", nil)
			return
		}

		if err.Error() == "exam cannot be submitted" {
			middleware.StructuredErrorResponse(c, http.StatusForbidden, "EXAM_CANNOT_SUBMIT", "Exam cannot be submitted", nil)
			return
		}

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "EXAM_SUBMIT_FAILED", "Failed to submit exam", nil)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"exam_id":    examID,
		"user_id":    userID,
		"score":      result.Score,
		"passed":     result.Passed,
		"request_id": middleware.GetRequestID(c),
	}).Info("Exam submitted successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Exam submitted successfully",
		"result":  result.ToResponse(true, false), // Include answers but not correct answers for user
	})
}

