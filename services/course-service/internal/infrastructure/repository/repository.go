package repository

import (
	"context"

	// THAY ĐỔI: Trỏ đến database của course-service
	database "github.com/06babyshark06/JQKStudy/services/course-service/internal/databases"
	// THAY ĐỔI: Trỏ đến domain của course-service
	"github.com/06babyshark06/JQKStudy/services/course-service/internal/domain"
	"gorm.io/gorm"
)

// THAY ĐỔI: Tên struct
type courseRepository struct{}

// THAY ĐỔI: Tên hàm
func NewCourseRepository() domain.CourseRepository {
	return &courseRepository{}
}

// =================================================================
// Course
// =================================================================

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
        // Nếu là giảng viên xem khóa học của mình -> Lấy tất cả (cả nháp) của giảng viên đó
        query = query.Where("instructor_id = ?", instructorID)
    } else {
        // Nếu là khách xem chung -> Chỉ lấy khóa học đã xuất bản
        query = query.Where("is_published = ?", true)
    }

	// 2. Áp dụng bộ lọc Tìm kiếm (Case-insensitive)
	if search != "" {
        // Postgres dùng ILIKE, MySQL dùng LIKE
		query = query.Where("title ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 3. Áp dụng bộ lọc Giá
	if minPrice > 0 {
		query = query.Where("price >= ?", minPrice)
	}
	if maxPrice > 0 {
		query = query.Where("price <= ?", maxPrice)
	}

	// 4. Đếm tổng số lượng (trước khi phân trang)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 5. Sắp xếp
	switch sortBy {
	case "price_asc":
		query = query.Order("price ASC")
	case "price_desc":
		query = query.Order("price DESC")
	default:
		query = query.Order("created_at DESC") // Mặc định mới nhất
	}

	// 6. Phân trang
	offset := (page - 1) * limit
	err := query.Limit(limit).Offset(offset).Find(&courses).Error

	return courses, total, err
}

func (r *courseRepository) GetCourseDetails(ctx context.Context, courseID int64) (*domain.CourseModel, error) {
	var course domain.CourseModel
	// Preload lồng nhau để lấy Sections và Lessons
	err := database.DB.WithContext(ctx).
		Preload("Sections", func(db *gorm.DB) *gorm.DB {
			return db.Order("section_models.order_index ASC") // Sắp xếp sections
		}).
		Preload("Sections.Lessons", func(db *gorm.DB) *gorm.DB {
			return db.Order("lesson_models.order_index ASC") // Sắp xếp lessons
		}).
		First(&course, courseID).Error

	if err != nil {
		return nil, err
	}
	return &course, nil
}

func (r *courseRepository) GetEnrolledCourses(ctx context.Context, userID int64) ([]*domain.CourseModel, error) {
	var courses []*domain.CourseModel
	// Dùng Join để tìm các khóa học mà user đã đăng ký
	err := database.DB.WithContext(ctx).
		Joins("JOIN enrollment_models e ON e.course_id = course_models.id").
		Where("e.user_id = ?", userID).
		Find(&courses).Error
	if err != nil {
		return nil, err
	}
	return courses, nil
}

// =================================================================
// Section
// =================================================================

func (r *courseRepository) CreateSection(ctx context.Context, tx *gorm.DB, section *domain.SectionModel) (*domain.SectionModel, error) {
	if err := tx.WithContext(ctx).Create(section).Error; err != nil {
		return nil, err
	}
	return section, nil
}

// =================================================================
// Lesson
// =================================================================

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

// =================================================================
// Enrollment
// =================================================================

func (r *courseRepository) CreateEnrollment(ctx context.Context, tx *gorm.DB, enrollment *domain.EnrollmentModel) error {
	if err := tx.WithContext(ctx).Create(enrollment).Error; err != nil {
		return err
	}
	return nil
}

func (r *courseRepository) GetEnrollment(ctx context.Context, userID int64, courseID int64) (*domain.EnrollmentModel, error) {
	var enrollment domain.EnrollmentModel
	// Tìm bằng khóa chính kết hợp
	err := database.DB.WithContext(ctx).
		Where("user_id = ? AND course_id = ?", userID, courseID).
		First(&enrollment).Error
	if err != nil {
		return nil, err
	}
	return &enrollment, nil
}

// =================================================================
// Progress
// =================================================================

func (r *courseRepository) CreateLessonProgress(ctx context.Context, tx *gorm.DB, progress *domain.LessonProgressModel) error {
	// Dùng FirstOrCreate để tránh lỗi "duplicate key" nếu đã hoàn thành rồi
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
	// Struct tạm để nhận kết quả
	type Result struct {
		LessonID int64
	}
	var results []Result

	// Join 3 bảng để tìm các lesson đã hoàn thành (progress)
	// của một user (user_id)
	// thuộc một khóa học (course_id)
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

	// Chuyển sang map để tra cứu nhanh (O(1))
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

// Implement UpdateSection
func (r *courseRepository) UpdateSection(ctx context.Context, tx *gorm.DB, sectionID int64, title string) error {
    if err := tx.WithContext(ctx).Model(&domain.SectionModel{}).Where("id = ?", sectionID).Update("title", title).Error; err != nil {
        return err
    }
    return nil
}

// Implement DeleteSection
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

// Implement DeleteLesson
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
	// 1. Xóa Enrollment (Học viên đăng ký)
	if err := tx.WithContext(ctx).Where("course_id = ?", courseID).Delete(&domain.EnrollmentModel{}).Error; err != nil {
		return err
	}

	// 2. Lấy danh sách Section để xóa Lesson bên trong
	var sectionIDs []int64
	tx.WithContext(ctx).Model(&domain.SectionModel{}).Where("course_id = ?", courseID).Pluck("id", &sectionIDs)

	if len(sectionIDs) > 0 {
		// Gọi hàm DeleteSection (mà chúng ta đã viết trước đó) cho từng Section
		// Hoặc viết query xóa gộp (để đơn giản, ta xóa gộp ở đây):
		
		// 2a. Lấy Lesson IDs
		var lessonIDs []int64
		tx.WithContext(ctx).Model(&domain.LessonModel{}).Where("section_id IN ?", sectionIDs).Pluck("id", &lessonIDs)

		if len(lessonIDs) > 0 {
			// 2b. Xóa Progress
			tx.WithContext(ctx).Where("lesson_id IN ?", lessonIDs).Delete(&domain.LessonProgressModel{})
			// 2c. Xóa Lessons
			tx.WithContext(ctx).Where("id IN ?", lessonIDs).Delete(&domain.LessonModel{})
		}
		// 2d. Xóa Sections
		tx.WithContext(ctx).Where("id IN ?", sectionIDs).Delete(&domain.SectionModel{})
	}

	// 3. Cuối cùng xóa Course
	if err := tx.WithContext(ctx).Delete(&domain.CourseModel{}, courseID).Error; err != nil {
		return err
	}
	return nil
}