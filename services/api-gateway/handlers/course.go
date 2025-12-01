package handlers

import (
	"net/http"
	"strconv"

	grpcclients "github.com/06babyshark06/JQKStudy/services/api-gateway/grpc_clients"
	"github.com/06babyshark06/JQKStudy/shared/contracts"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/course"
	"github.com/gin-gonic/gin"
)

type CourseHandler struct {
	courseClient pb.CourseServiceClient
}

func NewCourseHandler(client *grpcclients.CourseServiceClient) *CourseHandler {
	return &CourseHandler{courseClient: client.Client}
}

func (h *CourseHandler) GetCourses(c *gin.Context) {
    search := c.Query("search")
    sortBy := c.Query("sort")
    minPrice, _ := strconv.ParseFloat(c.Query("min_price"), 64)
    maxPrice, _ := strconv.ParseFloat(c.Query("max_price"), 64)
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "9"))

    req := &pb.GetCoursesRequest{
        Page:     int32(page),
        Limit:    int32(limit),
        Search:   search,
        SortBy:   sortBy,
        MinPrice: minPrice,
        MaxPrice: maxPrice,
    }

    resp, err := h.courseClient.GetCourses(c.Request.Context(), req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *CourseHandler) CreateCourse(c *gin.Context) {
	var req pb.CreateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	req.InstructorId = userID

	resp, err := h.courseClient.CreateCourse(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, contracts.APIResponse{Data: resp})
}

func (h *CourseHandler) GetCourseDetails(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	req := &pb.GetCourseDetailsRequest{
		CourseId: courseID,
		UserId:   userID,
	}

	resp, err := h.courseClient.GetCourseDetails(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *CourseHandler) EnrollCourse(c *gin.Context) {
	var req pb.EnrollCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	req.UserId = userID

	resp, err := h.courseClient.EnrollCourse(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *CourseHandler) GetMyCourses(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	req := &pb.GetMyCoursesRequest{UserId: userID}
	resp, err := h.courseClient.GetMyCourses(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *CourseHandler) CreateSection(c *gin.Context) {
	var req pb.CreateSectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.courseClient.CreateSection(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, contracts.APIResponse{Data: resp})
}

func (h *CourseHandler) CreateLesson(c *gin.Context) {
	var req pb.CreateLessonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.courseClient.CreateLesson(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, contracts.APIResponse{Data: resp})
}

func (h *CourseHandler) GetUploadURL(c *gin.Context) {
	var req pb.GetUploadURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.courseClient.GetUploadURL(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *CourseHandler) MarkLessonCompleted(c *gin.Context) {
	var req pb.MarkLessonCompletedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	req.UserId = userID // Gán UserID

	resp, err := h.courseClient.MarkLessonCompleted(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *CourseHandler) UpdateCourse(c *gin.Context) {
    idStr := c.Param("id")
    courseID, _ := strconv.ParseInt(idStr, 10, 64)

    var req pb.UpdateCourseRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    req.CourseId = courseID // Gán ID từ URL

    resp, err := h.courseClient.UpdateCourse(c.Request.Context(), &req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

// Update Section
func (h *CourseHandler) UpdateSection(c *gin.Context) {
    idStr := c.Param("id")
    sectionID, _ := strconv.ParseInt(idStr, 10, 64)
    
    var req struct { Title string `json:"title"` }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    resp, err := h.courseClient.UpdateSection(c.Request.Context(), &pb.UpdateSectionRequest{
        SectionId: sectionID,
        Title:     req.Title,
    })
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

// Delete Section
func (h *CourseHandler) DeleteSection(c *gin.Context) {
    idStr := c.Param("id")
    sectionID, _ := strconv.ParseInt(idStr, 10, 64)

    resp, err := h.courseClient.DeleteSection(c.Request.Context(), &pb.DeleteSectionRequest{SectionId: sectionID})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

// Delete Lesson
func (h *CourseHandler) DeleteLesson(c *gin.Context) {
    idStr := c.Param("id")
    lessonID, _ := strconv.ParseInt(idStr, 10, 64)

    resp, err := h.courseClient.DeleteLesson(c.Request.Context(), &pb.DeleteLessonRequest{LessonId: lessonID})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *CourseHandler) UpdateLesson(c *gin.Context) {
	idStr := c.Param("id")
	lessonID, _ := strconv.ParseInt(idStr, 10, 64)

	var req pb.UpdateLessonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.LessonId = lessonID // Gán ID từ URL

	resp, err := h.courseClient.UpdateLesson(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *CourseHandler) GetInstructorCourses(c *gin.Context) {
    userID, err := getUserIDFromContext(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
        return
    }

    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100")) // Lấy nhiều để hiển thị hết

    req := &pb.GetCoursesRequest{
        Page:         int32(page),
        Limit:        int32(limit),
        InstructorId: userID, 
    }

    resp, err := h.courseClient.GetCourses(c.Request.Context(), req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *CourseHandler) PublishCourse(c *gin.Context) {
    idStr := c.Param("id")
    courseID, _ := strconv.ParseInt(idStr, 10, 64)
    
    var req struct { IsPublished bool `json:"is_published"` }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    resp, err := h.courseClient.PublishCourse(c.Request.Context(), &pb.PublishCourseRequest{
        CourseId:    courseID,
        IsPublished: req.IsPublished,
    })
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *CourseHandler) DeleteCourse(c *gin.Context) {
	idStr := c.Param("id")
	courseID, _ := strconv.ParseInt(idStr, 10, 64)
	
	_, err := h.courseClient.DeleteCourse(c.Request.Context(), &pb.DeleteCourseRequest{CourseId: courseID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: gin.H{"message": "Course deleted"}})
}