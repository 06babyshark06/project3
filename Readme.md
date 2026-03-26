# 📚 JQK Study - Hệ Thống Thi Trắc Nghiệm Trực Tuyến

<div align="center">

![Go Version](https://img.shields.io/badge/Go-1.25.3-00ADD8?style=for-the-badge&logo=go)
![Next.js](https://img.shields.io/badge/Next.js-16.0.1-black?style=for-the-badge&logo=next.js)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?style=for-the-badge&logo=postgresql)
![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=for-the-badge&logo=redis)
![Kubernetes](https://img.shields.io/badge/Kubernetes-Ready-326CE5?style=for-the-badge&logo=kubernetes)

**Nền tảng học tập và thi trắc nghiệm trực tuyến hiện đại với kiến trúc Microservices**

[Tính năng](#-tính-năng-chính) • [Kiến trúc](#-kiến-trúc-hệ-thống) • [Cài đặt](#-cài-đặt) • [API](#-api-documentation)

</div>

---

## 📋 Mục Lục

- [Giới thiệu](#-giới-thiệu)
- [Tính năng chính](#-tính-năng-chính)
- [Công nghệ sử dụng](#-công-nghệ-sử-dụng)
- [Kiến trúc hệ thống](#-kiến-trúc-hệ-thống)
- [Cấu trúc cơ sở dữ liệu](#-cấu-trúc-cơ-sở-dữ-liệu)
- [Cấu trúc thư mục](#-cấu-trúc-thư-mục)
- [Cài đặt](#-cài-đặt)
- [API Documentation](#-api-documentation)
- [Bảo mật](#-bảo-mật)
- [Đóng góp](#-đóng-góp)

---

## 🎯 Giới Thiệu

**JQK Study** là một hệ thống thi trắc nghiệm trực tuyến được phát triển với mục tiêu cung cấp nền tảng học tập và kiểm tra kiến thức toàn diện. Hệ thống cho phép giảng viên tạo và quản lý các bài thi, câu hỏi, khóa học, trong khi học viên có thể tham gia thi, học tập và theo dõi tiến độ.

### Mục tiêu
- ✅ Xây dựng nền tảng thi trắc nghiệm trực tuyến hiệu quả và dễ sử dụng
- ✅ Hỗ trợ quản lý ngân hàng câu hỏi linh hoạt
- ✅ Cung cấp khả năng tạo và quản lý khóa học trực tuyến
- ✅ Hỗ trợ quản lý lớp học và thành viên
- ✅ Đảm bảo tính bảo mật và tin cậy trong quá trình thi

### Đối tượng sử dụng

| Vai trò | Mô tả | Quyền hạn |
|---------|-------|-----------|
| 👨‍💼 **Admin** | Quản trị viên hệ thống | Toàn quyền: quản lý users, xem thống kê, xóa courses |
| 👨‍🏫 **Instructor** | Giảng viên | Tạo/quản lý courses, exams, questions, classes |
| 👨‍🎓 **Student** | Học viên | Tham gia lớp, đăng ký khóa học, làm bài thi |

---

## ⭐ Tính Năng Chính

### 📝 Quản lý Bài thi
- Tạo và quản lý ngân hàng câu hỏi theo chủ đề (Topic) và phần (Section)
- Hỗ trợ nhiều loại câu hỏi: trắc nghiệm một đáp án, nhiều đáp án
- Import câu hỏi hàng loạt từ file Excel
- Tự động tạo đề thi từ ngân hàng câu hỏi theo cấu hình
- Xáo trộn câu hỏi và đáp án
- Đặt thời gian thi, số lần thi, mật khẩu bảo vệ

### 📊 Giám sát & Thống kê
- Theo dõi vi phạm trong quá trình thi (chuyển tab, copy/paste)
- Thống kê chi tiết kết quả thi
- Xuất kết quả ra file Excel
- Dashboard thống kê tổng quan

### 📚 Quản lý Khóa học
- Tạo khóa học với nhiều sections và lessons
- Upload video/tài liệu học tập
- Theo dõi tiến độ học của học viên

### 👥 Quản lý Lớp học
- Tạo lớp học với mã code tham gia
- Gán đề thi cho lớp học
- Quản lý thành viên lớp

---

## 🛠 Công Nghệ Sử Dụng

### Backend

| Thành phần | Công nghệ | Phiên bản |
|------------|-----------|-----------|
| Ngôn ngữ | Go (Golang) | 1.25.3 |
| Kiến trúc | Microservices | - |
| Giao tiếp Services | gRPC + Protocol Buffers | - |
| Web Framework | Gin | 1.11.0 |
| ORM | GORM | 1.31.0 |
| Database | PostgreSQL | 16 |
| Caching | Redis | 7 |
| Authentication | JWT | - |
| Cloud Storage | AWS S3 / Cloudflare R2 | - |
| Message Queue | Apache Kafka | - |
| File Processing | Excelize | - |

### Frontend

| Thành phần | Công nghệ | Phiên bản |
|------------|-----------|-----------|
| Framework | Next.js | 16.0.1 |
| Language | TypeScript | 5.x |
| UI Library | React | 19.2.0 |
| Styling | TailwindCSS | 4.x |
| UI Components | Radix UI + shadcn/ui | - |
| Form | React Hook Form + Zod | - |
| HTTP Client | Axios | - |
| Rich Text Editor | TipTap | - |
| Charts | Recharts | - |
| Notifications | Sonner | - |

### Infrastructure

| Thành phần | Công nghệ |
|------------|-----------|
| Containerization | Docker |
| Orchestration | Kubernetes |
| Local Development | Tilt |

---

## 🏗 Kiến Trúc Hệ Thống

### Sơ đồ tổng quan

```
                                    ┌─────────────────┐
                                    │   Frontend      │
                                    │   (Next.js)     │
                                    │   :4001         │
                                    └────────┬────────┘
                                             │
                                             │ REST API
                                             ▼
                                    ┌─────────────────┐
                                    │  API Gateway    │
                                    │    (Gin)        │
                                    │   :8081         │
                                    └────────┬────────┘
                                             │
                     ┌───────────────────────┼───────────────────────┐
                     │                       │                       │
                     ▼                       ▼                       ▼
            ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
            │  User Service   │    │  Exam Service   │    │ Course Service  │
            │   (gRPC)        │    │   (gRPC)        │    │   (gRPC)        │
            │   :50051        │    │   :50052        │    │   :50053        │
            └────────┬────────┘    └────────┬────────┘    └────────┬────────┘
                     │                       │                       │
                     └───────────────────────┼───────────────────────┘
                                             │
                                             ▼
                     ┌───────────────────────┴───────────────────────┐
                     │                                               │
                     ▼                                               ▼
            ┌─────────────────┐                             ┌─────────────────┐
            │   PostgreSQL    │                             │     Redis       │
            │    :5432        │                             │    :6379        │
            └─────────────────┘                             └─────────────────┘
```

### Mô tả Services

| Service | Port | Mô tả |
|---------|------|-------|
| **API Gateway** | 8081 | Điểm vào duy nhất, routing, authentication, CORS |
| **User Service** | 50051 | Quản lý users, authentication, classes |
| **Exam Service** | 50052 | Quản lý câu hỏi, đề thi, làm bài, chấm điểm |
| **Course Service** | 50053 | Quản lý khóa học, lessons, tiến độ học |
| **Notification Service** | 50054 | Consume events từ Kafka, gửi thông báo |

---

## 💾 Cấu Trúc Cơ Sở Dữ Liệu

### User Service Database

```
┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
│     ROLES       │       │     USERS       │       │    CLASSES      │
├─────────────────┤       ├─────────────────┤       ├─────────────────┤
│ id (PK)         │◄──────│ role_id (FK)    │       │ id (PK)         │
│ name            │       │ id (PK)         │──────►│ teacher_id (FK) │
└─────────────────┘       │ full_name       │       │ name            │
                          │ email           │       │ code            │
                          │ password        │       │ description     │
                          └────────┬────────┘       └────────┬────────┘
                                   │                         │
                                   │    ┌─────────────────┐  │
                                   └───►│ CLASS_MEMBERS   │◄─┘
                                        ├─────────────────┤
                                        │ class_id (FK)   │
                                        │ user_id (FK)    │
                                        │ role            │
                                        │ joined_at       │
                                        └─────────────────┘
```

### Exam Service Database

```
┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
│    TOPICS       │       │   SECTIONS      │       │   QUESTIONS     │
├─────────────────┤       ├─────────────────┤       ├─────────────────┤
│ id (PK)         │◄──────│ topic_id (FK)   │◄──────│ section_id (FK) │
│ name            │       │ id (PK)         │       │ id (PK)         │
│ description     │       │ name            │       │ content         │
│ creator_id      │       │ description     │       │ question_type   │
└─────────────────┘       └─────────────────┘       │ difficulty      │
                                                    │ explanation     │
                                                    └────────┬────────┘
                                                             │
┌─────────────────┐       ┌─────────────────┐       ┌────────┴────────┐
│     EXAMS       │       │ EXAM_QUESTIONS  │       │    CHOICES      │
├─────────────────┤       ├─────────────────┤       ├─────────────────┤
│ id (PK)         │◄──────│ exam_id (FK)    │       │ question_id(FK) │
│ title           │       │ question_id(FK) │──────►│ id (PK)         │
│ description     │       │ sequence        │       │ content         │
│ duration_minutes│       └─────────────────┘       │ is_correct      │
│ max_attempts    │                                 └─────────────────┘
│ status          │
└────────┬────────┘
         │
         │       ┌─────────────────┐       ┌─────────────────┐
         └──────►│ EXAM_SUBMISSIONS│◄──────│  USER_ANSWERS   │
                 ├─────────────────┤       ├─────────────────┤
                 │ exam_id (FK)    │       │ submission_id   │
                 │ user_id         │       │ question_id     │
                 │ score           │       │ chosen_choice_id│
                 │ submitted_at    │       │ is_correct      │
                 └─────────────────┘       └─────────────────┘
```

---

## 📁 Cấu Trúc Thư Mục

```
project3/
├── 📂 frontend/                    # Next.js Frontend
│   ├── 📂 app/                     # App Router
│   │   ├── 📂 admin/               # Trang quản trị
│   │   │   ├── 📂 dashboard/       # Dashboard Admin
│   │   │   ├── 📂 users/           # Quản lý users
│   │   │   ├── 📂 exams/           # Quản lý đề thi
│   │   │   ├── 📂 questions/       # Quản lý câu hỏi
│   │   │   ├── 📂 courses/         # Quản lý khóa học
│   │   │   └── 📂 classes/         # Quản lý lớp học
│   │   ├── 📂 dashboard/           # Dashboard học viên/giảng viên
│   │   ├── 📂 courses/             # Danh sách khóa học
│   │   ├── 📂 exams/               # Danh sách & làm bài thi
│   │   ├── 📂 login/               # Đăng nhập
│   │   └── 📂 register/            # Đăng ký
│   ├── 📂 components/              # React Components
│   │   ├── 📂 ui/                  # UI Components (shadcn)
│   │   └── ...                     # Feature components
│   ├── 📂 contexts/                # React Contexts
│   └── 📂 lib/                     # Utilities
│
├── 📂 services/                    # Backend Microservices
│   ├── 📂 api-gateway/             # API Gateway (Gin)
│   │   ├── 📂 handlers/            # HTTP Handlers
│   │   ├── 📂 middleware/          # JWT, Auth middleware
│   │   ├── 📂 grpc_clients/        # gRPC client connections
│   │   └── main.go
│   ├── 📂 user-service/            # User Service
│   │   ├── 📂 cmd/                 # Entry point
│   │   └── 📂 internal/
│   │       ├── 📂 domain/          # Domain models
│   │       ├── 📂 service/         # Business logic
│   │       ├── 📂 databases/       # Repository layer
│   │       └── 📂 infrastructure/  # gRPC server, Kafka
│   ├── 📂 exam-service/            # Exam Service
│   ├── 📂 course-service/          # Course Service
│   └── 📂 notification-service/    # Notification Service
│
├── 📂 proto/                       # Protocol Buffer definitions
│   ├── exam.proto
│   ├── user.proto
│   └── course.proto
│
├── 📂 shared/                      # Shared code
│   ├── 📂 contracts/               # Common contracts
│   ├── 📂 env/                     # Environment utilities
│   ├── 📂 proto/                   # Generated proto code
│   └── 📂 util/                    # Utilities
│
├── 📂 infra/                       # Infrastructure
│   ├── 📂 development/
│   │   ├── 📂 docker/              # Dockerfiles
│   │   └── 📂 k8s/                 # Kubernetes manifests
│   └── 📂 production/
│
├── Tiltfile                        # Tilt configuration
├── go.mod                          # Go modules
└── Makefile                        # Build commands
```

---

## 🚀 Cài Đặt

### Yêu cầu hệ thống

- **Go** >= 1.21
- **Node.js** >= 18
- **Docker** & **Docker Compose**
- **Tilt** (cho development)
- **kubectl** (cho Kubernetes)

### Cài đặt & Chạy (Development)

#### 1. Clone repository

```bash
git clone https://github.com/06babyshark06/JQKStudy.git
cd JQKStudy
```

#### 2. Khởi động services với Tilt

```bash
# Đảm bảo Docker đang chạy
tilt up
```

Tilt sẽ tự động:
- Build các Docker images
- Deploy lên local Kubernetes
- Hot-reload khi có thay đổi code

#### 3. Chạy Frontend riêng (nếu cần)

```bash
cd frontend
npm install
npm run dev
```

### Truy cập hệ thống

| Service | URL |
|---------|-----|
| Frontend | http://localhost:4001 |
| API Gateway | http://localhost:8081 |
| PostgreSQL | localhost:5432 |
| Redis | redis:6379 |

---

## 📖 API Documentation

### Authentication

```http
POST /api/v1/register
POST /api/v1/login
POST /api/v1/refresh
POST /api/v1/logout
```

### User Management

```http
GET    /api/v1/users/me           # Lấy profile
PUT    /api/v1/users/me           # Cập nhật profile
PUT    /api/v1/users/password     # Đổi mật khẩu
GET    /api/v1/users              # [Admin] Danh sách users
DELETE /api/v1/users/:id          # [Admin] Xóa user
PUT    /api/v1/users/:id/role     # [Admin] Cập nhật role
```

### Exam Management

```http
# Topics & Sections
GET    /api/v1/topics
POST   /api/v1/topics             # [Instructor]
PUT    /api/v1/topics/:id         # [Instructor]
DELETE /api/v1/topics/:id         # [Instructor]
GET    /api/v1/exam-sections
POST   /api/v1/exam-sections      # [Instructor]

# Questions
GET    /api/v1/questions          # [Instructor]
POST   /api/v1/questions          # [Instructor]
PUT    /api/v1/questions/:id      # [Instructor]
DELETE /api/v1/questions/:id      # [Instructor]
POST   /api/v1/questions/import   # [Instructor] Import Excel
GET    /api/v1/questions/export   # [Instructor] Export Excel

# Exams
GET    /api/v1/exams
GET    /api/v1/exams/:id
POST   /api/v1/exams              # [Instructor] Tạo thủ công
POST   /api/v1/exams/generate     # [Instructor] Tạo tự động
PUT    /api/v1/exams/:id          # [Instructor]
DELETE /api/v1/exams/:id          # [Instructor]
PUT    /api/v1/exams/:id/publish  # [Instructor]

# Taking Exam
POST   /api/v1/exams/:id/start    # [Student] Bắt đầu thi
POST   /api/v1/exams/save-answer  # [Student] Lưu câu trả lời
POST   /api/v1/exams/submit       # [Student] Nộp bài
GET    /api/v1/submissions/:id    # [Student] Xem kết quả

# Exam Monitoring
GET    /api/v1/exams/:id/stats        # [Instructor] Thống kê
GET    /api/v1/exams/:id/submissions  # [Instructor] DS bài nộp
GET    /api/v1/exams/:id/violations   # [Instructor] DS vi phạm
GET    /api/v1/exams/:id/export       # [Instructor] Xuất kết quả
```

### Course Management

```http
GET    /api/v1/courses
GET    /api/v1/courses/:id
POST   /api/v1/courses              # [Instructor]
PUT    /api/v1/courses/:id          # [Instructor]
PUT    /api/v1/courses/:id/publish  # [Instructor]

POST   /api/v1/sections             # [Instructor]
PUT    /api/v1/sections/:id         # [Instructor]
DELETE /api/v1/sections/:id         # [Instructor]

POST   /api/v1/lessons              # [Instructor]
PUT    /api/v1/lessons/:id          # [Instructor]
DELETE /api/v1/lessons/:id          # [Instructor]

POST   /api/v1/courses/enroll       # [Student] Đăng ký
GET    /api/v1/my-courses           # [Student] Khóa học của tôi
POST   /api/v1/lessons/complete     # [Student] Hoàn thành bài học
```

### Class Management

```http
GET    /api/v1/classes
GET    /api/v1/classes/:id
POST   /api/v1/classes              # [Instructor]
PUT    /api/v1/classes/:id          # [Instructor]
DELETE /api/v1/classes/:id          # [Instructor]
POST   /api/v1/classes/members      # [Instructor] Thêm thành viên
DELETE /api/v1/classes/members      # [Instructor] Xóa thành viên
POST   /api/v1/classes/:id/exams    # [Instructor] Gán đề thi

POST   /api/v1/classes/join         # [Student] Tham gia bằng mã
GET    /api/v1/classes/:id/exams    # [Student] Đề thi của lớp
```

---

## 🔐 Bảo Mật

### Authentication
- **JWT Token** với access token và refresh token
- Token được lưu trong Redis để quản lý session
- Hỗ trợ logout và revoke token

### Authorization
- **Role-based Access Control (RBAC)** với 3 roles
- Middleware kiểm tra quyền cho từng endpoint
- Protect routes trên cả frontend và backend

### Exam Security
- **Giám sát vi phạm**: Ghi nhận chuyển tab, copy/paste, blur window
- **Thời gian thi**: Server-side timer, tự động nộp khi hết giờ
- **Mật khẩu đề thi**: Bảo vệ bài thi bằng password
- **Phê duyệt truy cập**: Instructor duyệt trước khi cho thi
- **Xáo trộn**: Shuffle câu hỏi và đáp án

---

## 👨‍💻 Đóng Góp

1. Fork repository
2. Tạo branch mới (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Tạo Pull Request

---

## 📞 Liên Hệ

- **Author**: JQK Team
- **Email**: [an8112004@gmail.com](mailto:an8112004@gmail.com)
- **GitHub**: [https://github.com/06babyshark06/JQKStudy](https://github.com/06babyshark06/JQKStudy)

---

## 📄 License

Distributed under the MIT License. See `LICENSE` for more information.

---

<div align="center">

**⭐ Nếu thấy hữu ích, hãy star repo này nhé! ⭐**

*Built with ❤️ by JQK Team*

</div>
