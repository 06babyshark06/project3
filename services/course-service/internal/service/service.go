package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	database "github.com/06babyshark06/JQKStudy/services/course-service/internal/databases"
	"github.com/06babyshark06/JQKStudy/services/course-service/internal/domain"
	"github.com/06babyshark06/JQKStudy/shared/contracts"
	"github.com/06babyshark06/JQKStudy/shared/env"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/course"
	"gorm.io/gorm"
)

type courseService struct {
	repo     domain.CourseRepository
	producer domain.EventProducer
}

func NewCourseService(repo domain.CourseRepository, producer domain.EventProducer) domain.CourseService {
	return &courseService{repo: repo, producer: producer}
}

func (s *courseService) CreateCourse(ctx context.Context, req *pb.CreateCourseRequest) (*pb.CreateCourseResponse, error) {
	var createdCourse *domain.CourseModel

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		course := &domain.CourseModel{
			Title:        req.Title,
			Description:  req.Description,
			InstructorID: req.InstructorId,
			Price:        req.Price,
			ThumbnailURL: req.ThumbnailUrl,
			IsPublished:  false, 
		}

		var err error
		createdCourse, err = s.repo.CreateCourse(ctx, tx, course)
		return err
	})

	if err != nil {
		return nil, err
	}

	return &pb.CreateCourseResponse{
		Course: &pb.Course{
			Id:           createdCourse.Id,
			Title:        createdCourse.Title,
			Description:  createdCourse.Description,
			InstructorId: createdCourse.InstructorID,
			ThumbnailUrl: createdCourse.ThumbnailURL,
			Price:        createdCourse.Price,
			IsPublished:  createdCourse.IsPublished,
		},
	}, nil
}

func (s *courseService) CreateSection(ctx context.Context, req *pb.CreateSectionRequest) (*pb.CreateSectionResponse, error) {
	var createdSection *domain.SectionModel

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		section := &domain.SectionModel{
			CourseID:   req.CourseId,
			Title:      req.Title,
			OrderIndex: int(req.OrderIndex),
		}

		var err error
		createdSection, err = s.repo.CreateSection(ctx, tx, section)
		return err
	})

	if err != nil {
		return nil, err
	}

	return &pb.CreateSectionResponse{
		Section: &pb.Section{
			Id:         createdSection.Id,
			CourseId:   createdSection.CourseID,
			Title:      createdSection.Title,
			OrderIndex: int32(createdSection.OrderIndex),
		},
	}, nil
}

func (s *courseService) CreateLesson(ctx context.Context, req *pb.CreateLessonRequest) (*pb.CreateLessonResponse, error) {
	var createdLesson *domain.LessonModel

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		lessonType, err := s.repo.GetLessonType(ctx, req.LessonType)
		if err != nil {
			if err = tx.WithContext(ctx).Where("type = ?", req.LessonType).First(&lessonType).Error; err != nil {
				return errors.New("lesson type không hợp lệ")
			}
		}

		lesson := &domain.LessonModel{
			SectionID:  req.SectionId,
			Title:      req.Title,
			TypeID:     lessonType.Id,
			ContentURL: req.ContentUrl,
			OrderIndex: int(req.OrderIndex),
		}

		createdLesson, err = s.repo.CreateLesson(ctx, tx, lesson)
		return err
	})

	if err != nil {
		return nil, err
	}

	return &pb.CreateLessonResponse{
		Lesson: &pb.Lesson{
			Id:         createdLesson.Id,
			SectionId:  createdLesson.SectionID,
			Title:      createdLesson.Title,
			LessonType: req.LessonType,
			ContentUrl: createdLesson.ContentURL,
			OrderIndex: int32(createdLesson.OrderIndex),
		},
	}, nil
}

func (s *courseService) GetCourses(ctx context.Context, req *pb.GetCoursesRequest) (*pb.GetCoursesResponse, error) {
	courses, total, err := s.repo.GetCourses(ctx, req.Search, req.MinPrice, req.MaxPrice, req.SortBy, int(req.Page), int(req.Limit), req.InstructorId)
	if err != nil {
		return nil, err
	}

	var pbCourses []*pb.Course
	for _, c := range courses {
		pbCourses = append(pbCourses, &pb.Course{
			Id:           c.Id,
			Title:        c.Title,
			Description:  c.Description,
			InstructorId: c.InstructorID,
			ThumbnailUrl: c.ThumbnailURL,
			Price:        c.Price,
			IsPublished:  c.IsPublished,
		})
	}

	return &pb.GetCoursesResponse{Courses: pbCourses, Total: total, Page: int32(req.Page), TotalPages: int32(int32(total)/int32(req.Limit))}, nil
}

func (s *courseService) GetCourseDetails(ctx context.Context, req *pb.GetCourseDetailsRequest) (*pb.GetCourseDetailsResponse, error) {
	courseModel, err := s.repo.GetCourseDetails(ctx, req.CourseId)
	if err != nil {
		return nil, err
	}

	isEnrolled := false
	completedMap := make(map[int64]bool)

	if req.UserId != 0 {
		_, err := s.repo.GetEnrollment(ctx, req.UserId, req.CourseId)
		if err == nil {
			isEnrolled = true
		}

		completedMap, _ = s.repo.GetCompletedLessonIDs(ctx, req.UserId, req.CourseId)
	}

	pbSections := []*pb.Section{}
	for _, sModel := range courseModel.Sections {
		pbLessons := []*pb.Lesson{}
		for _, lModel := range sModel.Lessons {
			pbLessons = append(pbLessons, &pb.Lesson{
				Id:          lModel.Id,
				SectionId:   lModel.SectionID,
				Title:       lModel.Title,
				LessonType:  lModel.Type.Type,
				ContentUrl:  lModel.ContentURL,
				OrderIndex:  int32(lModel.OrderIndex),
				IsCompleted: completedMap[lModel.Id],
			})
		}
		pbSections = append(pbSections, &pb.Section{
			Id:         sModel.Id,
			CourseId:   sModel.CourseID,
			Title:      sModel.Title,
			OrderIndex: int32(sModel.OrderIndex),
			Lessons:    pbLessons,
		})
	}

	return &pb.GetCourseDetailsResponse{
		Course: &pb.Course{
			Id:           courseModel.Id,
			Title:        courseModel.Title,
			Description:  courseModel.Description,
			InstructorId: courseModel.InstructorID,
			ThumbnailUrl: courseModel.ThumbnailURL,
			Price:        courseModel.Price,
			IsPublished:  courseModel.IsPublished,
		},
		Sections:   pbSections,
		IsEnrolled: isEnrolled,
	}, nil
}

func (s *courseService) EnrollCourse(ctx context.Context, req *pb.EnrollCourseRequest) (*pb.EnrollCourseResponse, error) {
	course, err := s.repo.GetCourseByID(ctx, req.CourseId)
	if err != nil {
		return nil, errors.New("không tìm thấy khóa học")
	}

	_, err = s.repo.GetEnrollment(ctx, req.UserId, req.CourseId)
	if err == nil {
		return nil, errors.New("bạn đã đăng ký khóa học này")
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		enrollment := &domain.EnrollmentModel{
			UserID:     req.UserId,
			CourseID:   req.CourseId,
			EnrolledAt: time.Now().UTC(),
		}
		return s.repo.CreateEnrollment(ctx, tx, enrollment)
	})

	if err != nil {
		return nil, err
	}

	eventPayload := contracts.CourseEnrolledEvent{
		UserID:      req.UserId,
		CourseID:    req.CourseId,
		CourseTitle: course.Title,
	}
	eventBytes, err := json.Marshal(eventPayload)

	if err != nil {
		log.Printf("LỖI: Không thể marshal sự kiện course_enrolled: %v", err)
	} else {
		key := []byte(strconv.FormatInt(req.UserId, 10))
		err = s.producer.Produce("course_events", key, eventBytes)
		if err != nil {
			log.Printf("LỖI: Không thể gửi sự kiện course_enrolled: %v", err)
		}
	}

	return &pb.EnrollCourseResponse{Success: true}, nil
}

func (s *courseService) GetMyCourses(ctx context.Context, req *pb.GetMyCoursesRequest) (*pb.GetMyCoursesResponse, error) {
	courses, err := s.repo.GetEnrolledCourses(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	var pbCourses []*pb.Course
	for _, c := range courses {
		pbCourses = append(pbCourses, &pb.Course{
			Id:           c.Id,
			Title:        c.Title,
			Description:  c.Description,
			InstructorId: c.InstructorID,
			ThumbnailUrl: c.ThumbnailURL,
			Price:        c.Price,
			IsPublished:  c.IsPublished,
		})
	}

	return &pb.GetMyCoursesResponse{Courses: pbCourses}, nil
}

func (s *courseService) MarkLessonCompleted(ctx context.Context, req *pb.MarkLessonCompletedRequest) (*pb.MarkLessonCompletedResponse, error) {
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		progress := &domain.LessonProgressModel{
			UserID:      req.UserId,
			LessonID:    req.LessonId,
			CompletedAt: time.Now().UTC(),
		}
		return s.repo.CreateLessonProgress(ctx, tx, progress)
	})

	if err != nil {
		return nil, err
	}

	return &pb.MarkLessonCompletedResponse{Success: true}, nil
}

func (s *courseService) GetUploadURL(ctx context.Context, req *pb.GetUploadURLRequest) (*pb.GetUploadURLResponse, error) {
	bucketName := env.GetString("R2_BUCKET_NAME", "")

	presignClient, err := s.createR2Client(ctx)
	if err != nil {
		return nil, errors.New("không thể tạo R2 client: " + err.Error())
	}

	fileKey := fmt.Sprintf("sections/%d/%s", req.SectionId, req.FileName)

	presignedURLRequest, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(fileKey),
		ContentType: aws.String(req.ContentType),
	}, s3.WithPresignExpires(15*time.Minute))

	if err != nil {
		return nil, errors.New("không thể tạo Presigned URL: " + err.Error())
	}

	publicDomain := env.GetString("R2_PUBLIC_DOMAIN", "")
	if publicDomain == "" {
		return nil, errors.New("R2_PUBLIC_DOMAIN chưa được cấu hình")
	}
	finalURL := fmt.Sprintf("https://%s/%s", publicDomain, fileKey)

	return &pb.GetUploadURLResponse{
		UploadUrl: presignedURLRequest.URL,
		FinalUrl:  finalURL,
	}, nil
}

func (s *courseService) createR2Client(ctx context.Context) (*s3.PresignClient, error) {
	accountID := env.GetString("R2_ACCOUNT_ID", "")
	accessKey := env.GetString("R2_ACCESS_KEY_ID", "")
	secretKey := env.GetString("R2_SECRET_ACCESS_KEY", "")
	region := "auto"
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, err
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint) 
		o.UsePathStyle = true               
	})
	presignClient := s3.NewPresignClient(s3Client)

	return presignClient, nil
}

func (s *courseService) UpdateCourse(ctx context.Context, req *pb.UpdateCourseRequest) (*pb.UpdateCourseResponse, error) {
    err := database.DB.Transaction(func(tx *gorm.DB) error {
        updates := make(map[string]interface{})
        if req.Title != "" {
            updates["title"] = req.Title
        }
        if req.Description != "" {
            updates["description"] = req.Description
        }
        if req.ThumbnailUrl != "" {
            updates["thumbnail_url"] = req.ThumbnailUrl
        }
        updates["price"] = req.Price

        return s.repo.UpdateCourse(ctx, tx, req.CourseId, updates)
    })
    if err != nil {
        return nil, err
    }
    return &pb.UpdateCourseResponse{Success: true}, nil
}

func (s *courseService) UpdateSection(ctx context.Context, req *pb.UpdateSectionRequest) (*pb.UpdateSectionResponse, error) {
    err := database.DB.Transaction(func(tx *gorm.DB) error {
        return s.repo.UpdateSection(ctx, tx, req.SectionId, req.Title)
    })
    if err != nil {
        return nil, err
    }
    return &pb.UpdateSectionResponse{Success: true}, nil
}

func (s *courseService) DeleteSection(ctx context.Context, req *pb.DeleteSectionRequest) (*pb.DeleteSectionResponse, error) {
    err := database.DB.Transaction(func(tx *gorm.DB) error {
        return s.repo.DeleteSection(ctx, tx, req.SectionId)
    })
    if err != nil {
        return nil, err
    }
    return &pb.DeleteSectionResponse{Success: true}, nil
}

func (s *courseService) DeleteLesson(ctx context.Context, req *pb.DeleteLessonRequest) (*pb.DeleteLessonResponse, error) {
    err := database.DB.Transaction(func(tx *gorm.DB) error {
        return s.repo.DeleteLesson(ctx, tx, req.LessonId)
    })
    if err != nil {
        return nil, err
    }
    return &pb.DeleteLessonResponse{Success: true}, nil
}

func (s *courseService) UpdateLesson(ctx context.Context, req *pb.UpdateLessonRequest) (*pb.UpdateLessonResponse, error) {
    err := database.DB.Transaction(func(tx *gorm.DB) error {
        updates := map[string]interface{}{
            "title": req.Title,
            "content_url": req.ContentUrl,
        }
        return s.repo.UpdateLesson(ctx, tx, req.LessonId, updates)
    })
    if err != nil { return nil, err }
    return &pb.UpdateLessonResponse{Success: true}, nil
}

func (s *courseService) PublishCourse(ctx context.Context, req *pb.PublishCourseRequest) (*pb.PublishCourseResponse, error) {
    err := database.DB.Transaction(func(tx *gorm.DB) error {
        return s.repo.UpdateCourseStatus(ctx, tx, req.CourseId, req.IsPublished)
    })
    if err != nil { return nil, err }
    return &pb.PublishCourseResponse{Success: true}, nil
}

func (s *courseService) GetCourseCount(ctx context.Context, req *pb.GetCourseCountRequest) (*pb.GetCourseCountResponse, error) {
    count, err := s.repo.CountCourses(ctx)
    if err != nil { return nil, err }
    return &pb.GetCourseCountResponse{Count: count}, nil
}

func (s *courseService) DeleteCourse(ctx context.Context, req *pb.DeleteCourseRequest) (*pb.DeleteCourseResponse, error) {
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		return s.repo.DeleteCourse(ctx, tx, req.CourseId)
	})
	if err != nil { return nil, err }
	return &pb.DeleteCourseResponse{Success: true}, nil
}