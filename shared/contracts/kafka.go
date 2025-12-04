package contracts

type UserRegisteredEvent struct {
	UserID   int64  `json:"user_id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

type ExamSubmittedEvent struct {
	UserID       int64   `json:"user_id"`
	ExamID       int64   `json:"exam_id"`
	SubmissionID int64   `json:"submission_id"`
	ExamTitle    string  `json:"exam_title"`
	Score        float64 `json:"score"`
	Email        string  `json:"email"`
	FullName     string  `json:"full_name"`
}

type CourseEnrolledEvent struct {
	UserID      int64  `json:"user_id"`
	CourseID    int64  `json:"course_id"`
	CourseTitle string `json:"course_title"`
	Email       string `json:"email"`
	FullName    string `json:"full_name"`
}