package domain

import "time"

type ClassModel struct {
	Id          int64              `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string             `gorm:"size:255;not null" json:"name"`
	Code        string             `gorm:"size:50;uniqueIndex;not null" json:"code"`
	Description string             `gorm:"type:text" json:"description"`
	TeacherID   int64              `gorm:"not null;index" json:"teacher_id"`
	Teacher     *UserModel         `gorm:"foreignKey:TeacherID" json:"teacher"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	Members     []ClassMemberModel `gorm:"foreignKey:ClassID;constraint:OnDelete:CASCADE;" json:"members"`
}

func (ClassModel) TableName() string { return "classes" }

type ClassMemberModel struct {
	ClassID  int64      `gorm:"primaryKey" json:"class_id"`
	UserID   int64      `gorm:"primaryKey" json:"user_id"`
	Role     string     `gorm:"size:20;default:'student'" json:"role"`
	Status   string      `gorm:"size:20;default:'joined'" json:"status"`
	JoinedAt time.Time  `json:"joined_at"`

	User     UserModel   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Class    ClassModel  `gorm:"foreignKey:ClassID" json:"class,omitempty"`
}

func (ClassMemberModel) TableName() string { return "class_members" }