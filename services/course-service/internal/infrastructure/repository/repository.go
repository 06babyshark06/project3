package repository

import (
	"context"

	database "github.com/06babyshark06/JQKStudy/services/course-service/internal/databases"
	"github.com/06babyshark06/JQKStudy/services/course-service/internal/domain"
	"gorm.io/gorm"
)

type courseRepository struct{}

func NewCourseRepository() domain.CourseRepository {
	return &courseRepository{}
}

func (r *courseRepository) CreateCourse(ctx context.Context, tx *gorm.DB, course *domain.CourseModel) (*domain.CourseModel, error) {
	if err := tx.WithContext(ctx).Create(course).Error; err != nil {
		return nil, err
	}
	return course, nil
}

func (r *courseRepository) GetCourseByID(ctx context.Context, courseID int64) (*domain.CourseModel, error) {
	var course domain.CourseModel
	if err := database.DB.WithContext(ctx).First(&course, courseID).Error; err != nil {
		return nil, err
	}
	return &course, nil
}

func (r *courseRepository) GetCourses(ctx context.Context, search string, minPrice, maxPrice float64, sortBy string, page, limit int, instructorID int64) ([]*domain.CourseModel, int64, error) {
	var courses []*domain.CourseModel
	var total int64

	query := database.DB.WithContext(ctx).Model(&domain.CourseModel{})

	if instructorID > 0 {
        query = query.Where("instructor_id = ?", instructorID)
    } else {
        query = query.Where("is_published = ?", true)
    }

	if search != "" {
		query = query.Where("title ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if minPrice > 0 {
		query = query.Where("price >= ?", minPrice)
	}
	if maxPrice > 0 {
		query = query.Where("price <= ?", maxPrice)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	switch sortBy {
	case "price_asc":
		query = query.Order("price ASC")
	case "price_desc":
		query = query.Order("price DESC")
	default:
		query = query.Order("created_at DESC")
	}

	offset := (page - 1) * limit
	err := query.Limit(limit).Offset(offset).Find(&courses).Error

	return courses, total, err
}

func (r *courseRepository) GetCourseDetails(ctx context.Context, courseID int64) (*domain.CourseModel, error) {
	var course domain.CourseModel
	err := database.DB.WithContext(ctx).
		Preload("Sections", func(db *gorm.DB) *gorm.DB {
			return db.Order("section_models.order_index ASC")
		}).
		Preload("Sections.Lessons", func(db *gorm.DB) *gorm.DB {
			return db.Order("lesson_models.order_index ASC")
		}).
		First(&course, courseID).Error

	if err != nil {
		return nil, err
	}
	return &course, nil
}

func (r *courseRepository) GetEnrolledCourses(ctx context.Context, userID int64) ([]*domain.CourseModel, error) {
	var courses []*domain.CourseModel
	err := database.DB.WithContext(ctx).
		Joins("JOIN enrollment_models e ON e.course_id = course_models.id").
		Where("e.user_id = ?", userID).
		Find(&courses).Error
	if err != nil {
		return nil, err
	}
	return courses, nil
}

func (r *courseRepository) CreateSection(ctx context.Context, tx *gorm.DB, section *domain.SectionModel) (*domain.SectionModel, error) {
	if err := tx.WithContext(ctx).Create(section).Error; err != nil {
		return nil, err
	}
	return section, nil
}

func (r *courseRepository) CreateLesson(ctx context.Context, tx *gorm.DB, lesson *domain.LessonModel) (*domain.LessonModel, error) {
	if err := tx.WithContext(ctx).Create(lesson).Error; err != nil {
		return nil, err
	}
	return lesson, nil
}

func (r *courseRepository) GetLessonType(ctx context.Context, typeName string) (*domain.LessonTypeModel, error) {
	var lessonType domain.LessonTypeModel
	if err := database.DB.WithContext(ctx).Where("type = ?", typeName).First(&lessonType).Error; err != nil {
		return nil, err
	}
	return &lessonType, nil
}

func (r *courseRepository) CreateEnrollment(ctx context.Context, tx *gorm.DB, enrollment *domain.EnrollmentModel) error {
	if err := tx.WithContext(ctx).Create(enrollment).Error; err != nil {
		return err
	}
	return nil
}

func (r *courseRepository) GetEnrollment(ctx context.Context, userID int64, courseID int64) (*domain.EnrollmentModel, error) {
	var enrollment domain.EnrollmentModel
	err := database.DB.WithContext(ctx).
		Where("user_id = ? AND course_id = ?", userID, courseID).
		First(&enrollment).Error
	if err != nil {
		return nil, err
	}
	return &enrollment, nil
}

func (r *courseRepository) CreateLessonProgress(ctx context.Context, tx *gorm.DB, progress *domain.LessonProgressModel) error {
	if err := tx.WithContext(ctx).
		FirstOrCreate(progress, domain.LessonProgressModel{
			UserID:   progress.UserID,
			LessonID: progress.LessonID,
		}).Error; err != nil {
		return err
	}
	return nil
}

func (r *courseRepository) GetLessonProgress(ctx context.Context, userID int64, lessonID int64) (*domain.LessonProgressModel, error) {
	var progress domain.LessonProgressModel
	err := database.DB.WithContext(ctx).
		Where("user_id = ? AND lesson_id = ?", userID, lessonID).
		First(&progress).Error
	if err != nil {
		return nil, err
	}
	return &progress, nil
}

func (r *courseRepository) GetCompletedLessonIDs(ctx context.Context, userID int64, courseID int64) (map[int64]bool, error) {
	type Result struct {
		LessonID int64
	}
	var results []Result

	err := database.DB.WithContext(ctx).
		Table("lesson_progress_models lp").
		Select("lp.lesson_id").
		Joins("JOIN lesson_models l ON l.id = lp.lesson_id").
		Joins("JOIN section_models s ON s.id = l.section_id").
		Where("lp.user_id = ? AND s.course_id = ?", userID, courseID).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	completedMap := make(map[int64]bool)
	for _, res := range results {
		completedMap[res.LessonID] = true
	}
	return completedMap, nil
}

func (r *courseRepository) UpdateCourse(ctx context.Context, tx *gorm.DB, courseID int64, updates map[string]interface{}) error {
    if err := tx.WithContext(ctx).Model(&domain.CourseModel{}).Where("id = ?", courseID).Updates(updates).Error; err != nil {
        return err
    }
    return nil
}

func (r *courseRepository) UpdateSection(ctx context.Context, tx *gorm.DB, sectionID int64, title string) error {
    if err := tx.WithContext(ctx).Model(&domain.SectionModel{}).Where("id = ?", sectionID).Update("title", title).Error; err != nil {
        return err
    }
    return nil
}

func (r *courseRepository) DeleteSection(ctx context.Context, tx *gorm.DB, sectionID int64) error {
	var lessonIDs []int64
	if err := tx.WithContext(ctx).
		Model(&domain.LessonModel{}).
		Where("section_id = ?", sectionID).
		Pluck("id", &lessonIDs).Error; err != nil {
		return err
	}

	if len(lessonIDs) > 0 {
		if err := tx.WithContext(ctx).
			Where("lesson_id IN ?", lessonIDs).
			Delete(&domain.LessonProgressModel{}).Error; err != nil {
			return err
		}

		if err := tx.WithContext(ctx).
			Where("id IN ?", lessonIDs).
			Delete(&domain.LessonModel{}).Error; err != nil {
			return err
		}
	}

	if err := tx.WithContext(ctx).
		Where("id = ?", sectionID).
		Delete(&domain.SectionModel{}).Error; err != nil {
		return err
	}

	return nil
}

func (r *courseRepository) DeleteLesson(ctx context.Context, tx *gorm.DB, lessonID int64) error {
    if err := tx.WithContext(ctx).Delete(&domain.LessonModel{}, lessonID).Error; err != nil {
        return err
    }
    return nil
}

func (r *courseRepository) UpdateLesson(ctx context.Context, tx *gorm.DB, lessonID int64, updates map[string]interface{}) error {
    return tx.WithContext(ctx).Model(&domain.LessonModel{}).Where("id = ?", lessonID).Updates(updates).Error
}

func (r *courseRepository) UpdateCourseStatus(ctx context.Context, tx *gorm.DB, courseID int64, isPublished bool) error {
    return tx.WithContext(ctx).Model(&domain.CourseModel{}).Where("id = ?", courseID).Update("is_published", isPublished).Error
}

func (r *courseRepository) CountCourses(ctx context.Context) (int64, error) {
    var count int64
    if err := database.DB.WithContext(ctx).Model(&domain.CourseModel{}).Count(&count).Error; err != nil {
        return 0, err
    }
    return count, nil
}
func (r *courseRepository) DeleteCourse(ctx context.Context, tx *gorm.DB, courseID int64) error {
	if err := tx.WithContext(ctx).Where("course_id = ?", courseID).Delete(&domain.EnrollmentModel{}).Error; err != nil {
		return err
	}

	var sectionIDs []int64
	tx.WithContext(ctx).Model(&domain.SectionModel{}).Where("course_id = ?", courseID).Pluck("id", &sectionIDs)

	if len(sectionIDs) > 0 {
		var lessonIDs []int64
		tx.WithContext(ctx).Model(&domain.LessonModel{}).Where("section_id IN ?", sectionIDs).Pluck("id", &lessonIDs)

		if len(lessonIDs) > 0 {
			tx.WithContext(ctx).Where("lesson_id IN ?", lessonIDs).Delete(&domain.LessonProgressModel{})
			tx.WithContext(ctx).Where("id IN ?", lessonIDs).Delete(&domain.LessonModel{})
		}

		tx.WithContext(ctx).Where("id IN ?", sectionIDs).Delete(&domain.SectionModel{})
	}

	if err := tx.WithContext(ctx).Delete(&domain.CourseModel{}, courseID).Error; err != nil {
		return err
	}
	return nil
}