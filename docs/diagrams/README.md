# Hướng Dẫn Sử Dụng Class Diagrams với Astah

## Tổng Quan

Thư mục `docs/diagrams/` chứa các file PlantUML (`.puml`) mô tả class diagrams cho từng service trong backend:

| File | Mô tả |
|------|-------|
| `user-service.puml` | User, Role, Class, ClassMember models |
| `course-service.puml` | Course, Section, Lesson, Enrollment models |
| `exam-service.puml` | Topic, Question, Exam, Submission models |
| `notification-service.puml` | Notification, Template models |
| `api-gateway.puml` | Handlers, gRPC Clients, Middleware |

---

## Cách Import vào Astah

### Phương pháp 1: Copy & Paste (Đơn giản nhất)

> [!NOTE]
> Astah phiên bản thường **không hỗ trợ import PlantUML trực tiếp**. Bạn cần tạo thủ công trong Astah dựa trên nội dung file `.puml`.

1. Mở file `.puml` bằng VS Code hoặc Notepad
2. Xem cấu trúc các class và relationships
3. Trong Astah, tạo Class Diagram mới
4. Thêm các classes và relationships theo thông tin trong file

### Phương pháp 2: Sử dụng PlantUML Online để Preview

1. Truy cập [PlantUML Online Editor](https://www.plantuml.com/plantuml/uml/)
2. Copy nội dung file `.puml` vào editor
3. Xem diagram được render
4. Tham khảo diagram để tạo trong Astah

### Phương pháp 3: Sử dụng VS Code Extension

1. Cài extension **PlantUML** trong VS Code
2. Mở file `.puml`
3. Nhấn `Alt + D` để preview diagram
4. Export ra PNG/SVG nếu cần

---

## Cách Tạo Class Diagram Thủ Công trong Astah

### Bước 1: Tạo Project Mới
- File → New Project → UML

### Bước 2: Tạo Class Diagram
- Diagram → Class Diagram → New

### Bước 3: Thêm Classes
- Toolbox → Class → Kéo vào diagram
- Double-click để thêm attributes và methods

### Bước 4: Thêm Relationships
- **Association**: Class A → Class B (quan hệ thông thường)
- **Composition**: Class A ◆→ Class B (chứa và sở hữu)
- **Dependency**: Class A ⋯→ Class B (phụ thuộc)
- **Realization**: Class A ⋯▷ Interface (implement interface)

---

## Mẹo Khi Tạo Trong Astah

### Quy ước màu sắc (tuỳ chọn)
- **Domain Models**: Màu xanh lá
- **Interfaces**: Màu vàng
- **Handlers/Controllers**: Màu xanh dương

### Tổ chức packages
- Tạo package cho mỗi layer: Domain, Repository, Service, Handler

---

## Xuất File .asta

Sau khi tạo xong trong Astah:
1. File → Save As
2. Chọn định dạng `.asta`
3. Đặt tên phù hợp (ví dụ: `backend-class-diagrams.asta`)

---

## Tài Liệu Tham Khảo

- [Astah User Guide](https://astah.net/support/astah-professional)
- [PlantUML Language Reference](https://plantuml.com/class-diagram)
- [UML Class Diagram Tutorial](https://www.visual-paradigm.com/guide/uml-unified-modeling-language/uml-class-diagram-tutorial/)
