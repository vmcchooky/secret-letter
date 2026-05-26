# SRS — secret-letter / Secret Letter

**Version:** 1.1  
**Project name:** secret-letter / Secret Letter  
**Owner:** Quorix Việt Nam  
**Document type:** Software Requirements Specification  
**Primary goal:** Cho phép người dùng tạo và chia sẻ một nội dung bí mật qua đường link dùng một lần, với trải nghiệm đọc thư dạng phong thư điện ảnh: mở thư, đọc nội dung, đóng lại, phong thư bốc cháy và biến mất.

---

## 1. Mục tiêu dự án

### 1.1. Mục tiêu chính

Dự án **secret-letter / Secret Letter** là một ứng dụng web cho phép người dùng gửi nội dung nhạy cảm hoặc riêng tư thông qua một đường link chỉ có thể mở một lần.

Điểm khác biệt của dự án không chỉ nằm ở tính năng “one-time secret”, mà còn ở trải nghiệm thị giác: người nhận mở link sẽ thấy một phong thư bí mật, nhấn vào để mở, đọc nội dung bên trong, sau đó đóng lại để phong thư và lá thư tự bốc cháy rồi biến mất.

Ứng dụng cần đạt hai mục tiêu song song:

1. **Bảo mật thực tế:** nội dung bí mật chỉ được đọc một lần, không lưu thừa, không bị cache, không bị log.
2. **Trải nghiệm cảm xúc:** người đọc có cảm giác như đang mở một lá thư thật, rồi chứng kiến bí mật biến mất như tro tàn.

### 1.2. Định vị sản phẩm

Secret Letter không nên được thiết kế như một form nhập liệu khô khan. Sản phẩm cần giống một nghi thức số: người gửi trao đi một bí mật, người nhận mở nó một lần, và hệ thống đóng lại khoảnh khắc đó bằng hiệu ứng tự hủy giàu cảm xúc.

Trọng tâm thiết kế:

- Tối giản nhưng có chiều sâu.
- Điện ảnh nhưng không nặng nề.
- Đẹp nhưng không làm chậm thao tác đọc.
- Bảo mật thật, không chỉ là hiệu ứng trình diễn.

---

## 2. Phạm vi hệ thống

### 2.1. Trong phạm vi

Hệ thống cần hỗ trợ:

- Tạo secret message.
- Sinh link chia sẻ dùng một lần.
- Người nhận mở link để đọc nội dung.
- Secret bị đánh dấu đã sử dụng hoặc bị xóa sau lần đọc đầu tiên.
- Trang đọc thư có UI phong thư điện ảnh:
  - Phong thư xuất hiện.
  - Nhấn để mở.
  - Lá thư trượt ra.
  - Nội dung hiện lên.
  - Nhấn để đóng.
  - Phong thư bốc cháy.
  - Secret biến mất.
- Trang trạng thái có đồ họa riêng cho:
  - Link đã đọc.
  - Link hết hạn.
  - Link không tồn tại.
  - Lỗi mạng hoặc lỗi server.
- Cấu hình thời gian hết hạn cho secret.
- Responsive trên desktop và mobile.
- Không yêu cầu người nhận đăng nhập.

### 2.2. Ngoài phạm vi ở phiên bản đầu

Phiên bản đầu chưa bắt buộc hỗ trợ:

- File attachment.
- Tài khoản người dùng.
- Lịch sử secret đã tạo.
- Chat hai chiều.
- Secret có nhiều người nhận.
- End-to-end encryption hoàn chỉnh phía client.
- Realtime notification.
- 3D realtime bằng WebGL/Three.js.

Các tính năng này có thể được đưa vào roadmap sau.

---

## 3. Đối tượng sử dụng

### 3.1. Sender — người gửi

Người gửi là người tạo nội dung bí mật. Họ cần:

- Nhập nội dung muốn gửi.
- Chọn thời gian hết hạn nếu có.
- Nhấn tạo link.
- Copy link và gửi qua Messenger, Zalo, email, Slack hoặc kênh khác.

### 3.2. Receiver — người nhận

Người nhận là người mở link. Họ cần:

- Truy cập link.
- Xem phong thư.
- Nhấn mở thư.
- Đọc nội dung.
- Đóng thư để hoàn tất quá trình tự hủy.

### 3.3. System administrator — quản trị hệ thống

Quản trị viên cần:

- Cấu hình giới hạn kích thước nội dung.
- Cấu hình TTL mặc định.
- Theo dõi log hệ thống không chứa secret.
- Kiểm tra lỗi server, database, rate limit, abuse.

---

## 4. Giả định và ràng buộc

### 4.1. Giả định

- Người gửi tự chịu trách nhiệm về nội dung họ nhập.
- Người gửi chia sẻ link qua nền tảng bên ngoài hệ thống.
- Người nhận có trình duyệt hiện đại hỗ trợ JavaScript.
- Hệ thống được triển khai qua HTTPS.
- Secret có thể bị xem bởi bất kỳ ai có link, trừ khi thêm cơ chế password ở phiên bản sau.

### 4.2. Ràng buộc

- Không được lưu secret dạng plain text lâu dài nếu có thể tránh.
- Không được log nội dung secret.
- Không được cache response chứa secret.
- Link sau khi mở thành công phải không thể đọc lại.
- UI animation không được làm ảnh hưởng nghiêm trọng đến khả năng đọc nội dung.
- Mobile phải có fallback nhẹ hơn nếu thiết bị yếu.
- Nếu animation asset không tải được, hệ thống vẫn phải cho phép đọc secret bằng fallback UI.

---

## 5. Tổng quan chức năng

Hệ thống gồm ba luồng chính:

### 5.1. Luồng tạo secret

Người gửi nhập nội dung → chọn tùy chọn → tạo secret → nhận link chia sẻ.

### 5.2. Luồng đọc secret

Người nhận mở link → hệ thống kiểm tra token → nếu hợp lệ, trả nội dung một lần → frontend hiển thị phong thư → người nhận mở thư → đọc nội dung.

### 5.3. Luồng tự hủy

Sau khi secret đã được cấp cho người nhận, backend đánh dấu secret là đã dùng. Khi người nhận đóng thư, frontend phát animation phong thư cháy và biến mất. Nếu mở lại link, hệ thống hiển thị trạng thái đã biến mất.

### 5.4. Luồng trạng thái đặc biệt

Nếu secret không còn khả dụng, hệ thống không nên chỉ hiển thị một dòng lỗi khô khan. Mỗi trạng thái cần có một hình ảnh/animation riêng:

- **Consumed:** phong thư chỉ còn tro.
- **Expired:** phong thư cũ, bụi phủ, niêm phong đã phai màu.
- **Not found:** nền trống, chỉ còn một mảnh giấy rách hoặc dấu sáp vỡ.
- **Network error:** phong thư bị kẹt trong bóng tối hoặc ánh sáng chập chờn.

---

## 6. Functional Requirements

## 6.1. Tạo secret

### FR-001 — Nhập nội dung secret

Hệ thống phải cung cấp giao diện để người gửi nhập nội dung bí mật.

**Thông tin đầu vào:**

- Secret message.
- Optional expiration time.
- Optional display mode.
- Optional password, nếu hỗ trợ ở phiên bản sau.

**Quy tắc:**

- Nội dung không được rỗng.
- Nội dung phải nằm trong giới hạn kích thước.
- Mặc định giới hạn đề xuất: 10.000 ký tự.
- Nếu vượt giới hạn, hệ thống phải báo lỗi rõ ràng.

---

### FR-002 — Tạo token chia sẻ

Sau khi người gửi nhấn tạo secret, hệ thống phải sinh một token ngẫu nhiên, khó đoán.

**Yêu cầu:**

- Token phải có entropy đủ mạnh.
- Token không được tuần tự.
- Token không được để lộ ID nội bộ của database.
- Token nên được hash trước khi lưu trong database.

Ví dụ:

```txt
https://domain.com/s/AbC9xYzSecretToken
```

Nhưng trong database chỉ nên lưu hash của token, không lưu token gốc.

---

### FR-003 — Lưu secret

Hệ thống phải lưu secret cùng metadata cần thiết.

Metadata đề xuất:

```txt
id
token_hash
encrypted_content
status
created_at
expires_at
consumed_at
max_views
view_count
burn_after_read
theme
```

Trạng thái secret:

```txt
active
consumed
expired
deleted
```

---

### FR-004 — Trả link cho người gửi

Sau khi tạo thành công, hệ thống phải trả về link để người gửi copy.

Giao diện cần có:

- Link readonly.
- Nút Copy.
- Thông báo copy thành công.
- Cảnh báo rằng link chỉ đọc được một lần.
- Hiển thị thời gian hết hạn nếu có.

---

## 6.2. Mở secret

### FR-005 — Kiểm tra token

Khi người nhận mở link, hệ thống phải kiểm tra token.

Các trường hợp:

| Trạng thái | Hành vi |
|---|---|
| Token hợp lệ | Cho phép đọc |
| Token không tồn tại | Hiển thị lỗi không tìm thấy |
| Token đã dùng | Hiển thị secret đã biến mất |
| Token hết hạn | Hiển thị secret đã hết hạn |
| Token lỗi định dạng | Hiển thị lỗi không hợp lệ |

---

### FR-006 — Đọc secret một lần

Khi token hợp lệ, hệ thống phải trả nội dung secret cho frontend và ngay lập tức đánh dấu secret là đã sử dụng.

**Yêu cầu quan trọng:**

- Secret phải bị consume ở backend khi nội dung được trả về thành công.
- Nếu người dùng refresh sau đó, link không được trả nội dung lần nữa.
- Nếu hai request mở cùng lúc, chỉ một request được nhận secret.
- Cần xử lý race condition bằng transaction hoặc atomic update.

Ví dụ logic:

```sql
UPDATE secrets
SET status = 'consumed', consumed_at = NOW(), view_count = view_count + 1
WHERE token_hash = $1
AND status = 'active'
AND expires_at > NOW()
RETURNING encrypted_content;
```

Nếu không có row nào được trả về, nghĩa là secret không còn hợp lệ.

---

### FR-007 — Không cache secret

Response chứa nội dung secret phải có header chống cache.

Headers đề xuất:

```http
Cache-Control: no-store, no-cache, must-revalidate, private
Pragma: no-cache
Expires: 0
Referrer-Policy: no-referrer
X-Content-Type-Options: nosniff
```

---

## 6.3. Trải nghiệm phong thư

### FR-008 — Hiển thị scene phong thư

Sau khi nội dung được tải thành công, frontend phải hiển thị scene phong thư ở giữa màn hình.

Scene gồm:

- Background tối hoặc gradient điện ảnh.
- Phong thư nằm giữa.
- Ánh sáng mềm.
- Hiệu ứng hover/tap nhẹ.
- Dòng hướng dẫn ngắn.

Ví dụ text:

```txt
A secret letter is waiting for you.
Tap to open.
```

Bản tiếng Việt:

```txt
Một lá thư bí mật đang chờ bạn.
Chạm để mở.
```

---

### FR-009 — Mở phong thư

Khi người nhận nhấn vào phong thư, hệ thống phải phát animation mở thư.

Animation gồm:

- Phong thư rung nhẹ.
- Nắp phong thư mở ra.
- Lá thư trượt lên.
- Nội dung hiện dần.
- Background sáng nhẹ hơn để tập trung vào lá thư.

Trạng thái chuyển từ:

```txt
sealed -> opening -> revealed
```

---

### FR-010 — Hiển thị nội dung thư

Khi thư mở xong, nội dung secret phải hiển thị rõ ràng, dễ đọc.

Yêu cầu:

- Text contrast cao.
- Font dễ đọc.
- Hỗ trợ xuống dòng.
- Hỗ trợ copy nếu cấu hình cho phép.
- Không tự động đóng khi người dùng chưa đọc xong.
- Có nút hoặc hành động rõ ràng để đóng thư.

Gợi ý nút:

```txt
Close & Burn
```

Hoặc tiếng Việt:

```txt
Đóng thư & thiêu hủy
```

---

### FR-011 — Đóng thư

Khi người dùng nhấn đóng thư, hệ thống phải phát animation đóng phong thư.

Animation gồm:

- Nội dung mờ dần.
- Lá thư trượt lại vào phong thư.
- Nắp phong thư đóng lại.
- Scene chuyển sang trạng thái chuẩn bị cháy.

Trạng thái:

```txt
revealed -> closing -> burning
```

---

### FR-012 — Hiệu ứng cháy và biến mất

Sau khi phong thư đóng lại, hệ thống phải phát hiệu ứng tự hủy.

Animation gồm:

- Mép phong thư bắt lửa.
- Lửa lan dần.
- Khói nhẹ xuất hiện.
- Tro hoặc hạt giấy bay ra.
- Phong thư mờ dần, co lại hoặc tan biến.
- Scene kết thúc bằng thông báo secret đã biến mất.

Trạng thái:

```txt
burning -> vanished
```

Thông báo sau cùng:

```txt
This secret has vanished.
```

Hoặc:

```txt
Bí mật này đã biến mất.
```

---

### FR-013 — Không phát lại secret sau khi cháy

Sau khi animation cháy hoàn tất, frontend phải xóa secret khỏi state.

Yêu cầu:

- Clear biến chứa nội dung secret.
- Không lưu vào localStorage.
- Không lưu vào sessionStorage.
- Không expose qua URL.
- Không giữ trong global debug object.

---

## 6.4. Trạng thái lỗi và trạng thái đặc biệt

### FR-014 — Link đã được đọc

Nếu link đã được đọc, hệ thống hiển thị màn hình “secret đã biến mất”.

**Visual direction:**

- Nền tối, yên tĩnh.
- Một đống tro nhỏ ở giữa.
- Vài hạt tro bay chậm rồi tắt.
- Có thể còn một mảnh sáp seal nứt vỡ.
- Không hiển thị phong thư nguyên vẹn để tránh gây hiểu nhầm rằng vẫn mở được.

Nội dung gợi ý:

```txt
This secret has already vanished.
It could only be opened once.
```

Tiếng Việt:

```txt
Bí mật này đã biến mất.
Nó chỉ có thể được mở một lần.
```

---

### FR-015 — Link hết hạn

Nếu link hết hạn, hệ thống hiển thị trạng thái expired.

**Visual direction:**

- Phong thư cũ nằm trên mặt bàn tối.
- Bụi phủ nhẹ.
- Seal sáp bị phai màu hoặc nứt.
- Ánh sáng lạnh hơn trạng thái active.
- Không có lửa, vì thư không được mở; nó chết vì thời gian, không chết vì thiêu hủy.

Nội dung gợi ý:

```txt
This secret has expired.
The letter was never opened in time.
```

Tiếng Việt:

```txt
Bí mật này đã hết hạn.
Lá thư đã không được mở kịp thời.
```

---

### FR-016 — Link không tồn tại

Nếu token không tồn tại hoặc sai, hệ thống hiển thị lỗi nhẹ nhàng.

**Visual direction:**

- Không hiển thị phong thư hoàn chỉnh.
- Có thể hiển thị mảnh giấy rách, dấu sáp vỡ hoặc chiếc bàn trống.
- Tone màu trung tính, ít bi kịch hơn trạng thái consumed/expired.

Nội dung gợi ý:

```txt
No secret letter was found here.
The link may be broken or invalid.
```

Tiếng Việt:

```txt
Không tìm thấy lá thư bí mật nào ở đây.
Liên kết có thể đã sai hoặc không còn hợp lệ.
```

---

### FR-017 — Lỗi mạng hoặc server

Nếu không thể kết nối tới server hoặc API lỗi tạm thời, hệ thống hiển thị trạng thái lỗi có thể thử lại.

**Visual direction:**

- Phong thư bị kẹt trong bóng tối.
- Ánh sáng chập chờn nhẹ.
- Không dùng hiệu ứng cháy vì secret chưa được xác nhận là đã mất.

Nội dung gợi ý:

```txt
We could not open this secret letter.
Please check your connection and try again.
```

Tiếng Việt:

```txt
Không thể mở lá thư bí mật này.
Hãy kiểm tra kết nối và thử lại.
```

---

## 7. UI/UX Requirements

## 7.1. Phong cách thiết kế

Giao diện nên theo phong cách:

- Cinematic.
- Tối giản.
- Sang.
- Có cảm giác bí mật.
- Không cyberpunk nặng.
- Không quá neon.
- Không giống game rẻ tiền.

Từ khóa định hướng:

```txt
dark cinematic
realistic paper texture
soft warm light
mysterious
elegant
glass and fire
ritual-like interaction
```

---

## 7.2. Visual design

### Background

- Nền tối, gradient đen/xanh đậm/nâu trầm.
- Có ánh sáng mềm phía sau phong thư.
- Có vignette nhẹ.
- Không dùng background quá rối.

### Envelope

- Phong thư có texture giấy.
- Có shadow mềm.
- Có chiều sâu giả 3D.
- Có thể có seal nhỏ hoặc logo Quorix.

### Letter

- Lá thư màu giấy ngà.
- Bo góc nhẹ.
- Có texture giấy rất nhẹ.
- Nội dung rõ ràng.
- Không để hiệu ứng làm khó đọc.

### Fire

- Lửa không nên quá hoạt hình.
- Nên dùng overlay video hoặc particle.
- Có glow ánh cam.
- Có khói nhẹ.
- Có tro bay.

---

## 7.3. Motion design

Animation phải có timeline rõ ràng:

| Giai đoạn | Thời lượng đề xuất |
|---|---:|
| Envelope idle | Loop nhẹ |
| Opening | 800–1400ms |
| Letter reveal | 700–1200ms |
| Text reveal | 400–1000ms |
| Closing | 600–1000ms |
| Burning | 1800–3500ms |
| Vanish | 500–900ms |

Tổng thời gian từ đóng thư đến biến mất nên khoảng 2.5–4 giây.

---

## 7.4. Mobile UX

Trên mobile:

- Phong thư phải vừa màn hình.
- Text không bị quá nhỏ.
- Nút đóng thư phải dễ bấm.
- Animation cháy có thể giảm particle để tối ưu hiệu năng.
- Không phụ thuộc hover.

---

## 7.5. Accessibility

Hệ thống cần hỗ trợ:

- `prefers-reduced-motion`.
- Keyboard navigation.
- Focus visible.
- Text contrast đạt mức đọc được.
- Alt text hoặc ARIA label cho phong thư.
- Không chỉ dùng màu để truyền đạt trạng thái.

Nếu người dùng bật reduced motion:

- Không phát animation cháy phức tạp.
- Dùng fade-out đơn giản.
- Vẫn hiển thị thông báo secret đã biến mất.

---

## 8. Cinematic Graphics Requirements

Phần này bổ sung riêng cho yêu cầu đồ họa đẹp, để tránh trường hợp UI chỉ đẹp ở màn hình active nhưng các trạng thái khác lại sơ sài.

### CGR-001 — Active unopened letter scene

Trạng thái thư hợp lệ nhưng chưa mở cần có cảm giác mời gọi.

Visual:

- Phong thư nguyên vẹn.
- Ánh sáng ấm nhẹ phía sau.
- Seal hoặc biểu tượng nhỏ.
- Shadow mềm bên dưới.
- Animation idle rất nhẹ.

Mood:

```txt
curious
quiet
intimate
premium
```

---

### CGR-002 — Revealed letter scene

Trạng thái đang đọc thư cần ưu tiên khả năng đọc.

Visual:

- Lá thư nổi lên phía trước.
- Background giảm tương phản.
- Fire, smoke, particle không xuất hiện ở trạng thái này.
- Text rõ, không bị che bởi texture.

Mood:

```txt
focused
private
calm
```

---

### CGR-003 — Burn scene

Trạng thái cháy cần là điểm nhấn cảm xúc của sản phẩm.

Visual:

- Mép phong thư/lá thư phát sáng trước khi cháy.
- Flame overlay có alpha.
- Smoke nhẹ, không quá dày.
- Ash particle bay lên hoặc rơi xuống.
- Phong thư biến mất dần bằng opacity, mask hoặc fragment effect.

Không nên:

- Dùng lửa hoạt hình quá giả.
- Lạm dụng neon.
- Làm cháy quá lâu gây sốt ruột.
- Che thông báo cuối bằng khói quá dày.

---

### CGR-004 — Consumed scene

Trạng thái link đã đọc cần cho người dùng hiểu rằng secret thật sự không còn.

Visual:

- Tro tàn ở giữa màn hình.
- Một vài hạt ember tắt dần.
- Có thể có dấu seal vỡ hoặc góc giấy cháy còn sót lại.
- Không có nút “open”.

Message:

```txt
This secret has already vanished.
```

---

### CGR-005 — Expired scene

Trạng thái hết hạn cần khác với trạng thái đã đọc.

Ý nghĩa thị giác:

- **Consumed:** bí mật đã được mở và thiêu hủy.
- **Expired:** bí mật chưa được mở, nhưng đã bị thời gian chôn vùi.

Visual:

- Phong thư cũ, còn nguyên nhưng xỉn màu.
- Bụi, vết thời gian, giấy hơi cong.
- Seal nứt nhẹ.
- Ánh sáng lạnh, tĩnh.
- Không có lửa.

Message:

```txt
This secret has expired.
The letter was never opened in time.
```

---

### CGR-006 — Not found scene

Trạng thái không tìm thấy nên trung tính, không quá bi thương.

Visual:

- Bàn trống.
- Một mảnh giấy nhỏ hoặc dấu sáp vỡ.
- Ánh sáng yếu.
- Không có thư hoàn chỉnh.

Message:

```txt
No secret letter was found here.
```

---

### CGR-007 — Error scene

Trạng thái lỗi mạng/server nên cho phép thử lại.

Visual:

- Phong thư mờ trong bóng tối.
- Ánh sáng chập chờn nhẹ.
- Không dùng cháy/tro vì trạng thái secret chưa được xác định.

Actions:

- Retry.
- Back to create page.

---

### CGR-008 — Asset quality requirements

Các asset đồ họa cần đạt:

- Texture giấy tối thiểu 2x resolution cho màn hình retina.
- Fire/smoke overlay nên có nền trong suốt nếu dùng WebM alpha.
- Particle không vượt quá số lượng gây lag trên mobile.
- Tất cả asset phải có fallback nếu tải lỗi.
- Asset nên được preload theo giai đoạn, không tải toàn bộ ngay từ đầu nếu quá nặng.

---

## 9. Security Requirements

### SR-001 — HTTPS bắt buộc

Ứng dụng phải chạy qua HTTPS ở production.

---

### SR-002 — Không log secret

Server không được log:

- Nội dung secret.
- Decrypted content.
- Full request body nếu có chứa secret.
- Full URL nếu token nằm trong path và log có thể bị lộ.

---

### SR-003 — Token bảo mật

Token phải:

- Đủ dài.
- Sinh bằng CSPRNG.
- Không đoán được.
- Không lưu raw token nếu không cần thiết.

---

### SR-004 — Hash token trong database

Database nên lưu:

```txt
token_hash
```

Không nên lưu:

```txt
raw_token
```

Khi người dùng mở link, server hash token nhận được rồi so sánh với `token_hash`.

---

### SR-005 — Encrypt secret at rest

Secret nên được mã hóa trước khi lưu.

Tối thiểu:

- AES-GCM hoặc cơ chế encryption tương đương.
- Key lấy từ environment variable hoặc secret manager.
- Không hard-code encryption key.

---

### SR-006 — Consume atomic

Việc đọc và đánh dấu secret đã dùng phải là một thao tác atomic.

Mục tiêu: tránh trường hợp hai người mở link cùng lúc và cả hai đều đọc được.

---

### SR-007 — Rate limiting

Hệ thống cần rate limit:

- Endpoint tạo secret.
- Endpoint đọc secret.
- Theo IP hoặc fingerprint nhẹ.

Ví dụ:

```txt
Create secret: 20 requests / 10 minutes / IP
Read secret: 60 requests / 10 minutes / IP
```

---

### SR-008 — Input sanitization

Nếu secret hiển thị dạng plain text, frontend phải escape HTML.

Không được render trực tiếp bằng `dangerouslySetInnerHTML`, trừ khi có sanitizer nghiêm ngặt.

---

### SR-009 — No third-party leak

Trang đọc secret không nên tải script analytics bên thứ ba có thể thấy URL token.

Nếu dùng analytics:

- Không gửi token.
- Không gửi secret.
- Mask URL path.
- Tốt nhất là tắt analytics ở trang `/s/:token`.

---

## 10. Non-Functional Requirements

## 10.1. Performance

- First load dưới 3 giây trên mạng ổn định.
- Animation phải đạt gần 60 FPS trên desktop hiện đại.
- Mobile yếu có fallback reduced effects.
- Asset animation cần được tối ưu.

Giới hạn đề xuất:

```txt
Initial JS bundle: < 300KB gzip nếu có thể
Fire video/effect asset: < 2MB
Main scene load: < 1.5s sau khi HTML tải xong
```

---

## 10.2. Reliability

- Secret không được mất trước khi người nhận mở, trừ khi hết hạn.
- Secret không được đọc lại sau khi consumed.
- Server phải xử lý được refresh, back, retry.
- Nếu animation lỗi, nội dung vẫn phải đọc được sau khi token hợp lệ.

---

## 10.3. Compatibility

Hỗ trợ:

- Chrome latest.
- Edge latest.
- Firefox latest.
- Safari latest.
- Mobile Chrome.
- Mobile Safari.

Không bắt buộc hỗ trợ Internet Explorer.

---

## 10.4. Maintainability

Code frontend cần tách rõ:

```txt
components/
  SecretEnvelopeScene
  Envelope
  Letter
  BurnEffect
  AshParticles
  SecretStatusScreen

hooks/
  useSecret
  useEnvelopeTimeline

api/
  secretApi
```

Không viết toàn bộ animation dồn vào một file khổng lồ.

---

## 11. API Requirements

## 11.1. Create secret

### Endpoint

```http
POST /api/secrets
```

### Request

```json
{
  "content": "This is my secret message.",
  "expiresInMinutes": 1440,
  "burnAfterRead": true,
  "theme": "classic-letter"
}
```

### Response success

```json
{
  "success": true,
  "url": "https://domain.com/s/example-token",
  "expiresAt": "2026-05-24T10:00:00.000Z"
}
```

### Response error

```json
{
  "success": false,
  "error": {
    "code": "CONTENT_TOO_LONG",
    "message": "Secret content is too long."
  }
}
```

---

## 11.2. Read secret

### Endpoint

```http
POST /api/secrets/:token/open
```

Dùng `POST` thay vì `GET` để giảm rủi ro cache và tránh prefetch ngoài ý muốn.

### Response success

```json
{
  "success": true,
  "secret": {
    "content": "This is my secret message.",
    "createdAt": "2026-05-23T10:00:00.000Z",
    "expiresAt": "2026-05-24T10:00:00.000Z",
    "theme": "classic-letter"
  }
}
```

### Response already consumed

```json
{
  "success": false,
  "error": {
    "code": "SECRET_CONSUMED",
    "message": "This secret has already vanished."
  }
}
```

### Response expired

```json
{
  "success": false,
  "error": {
    "code": "SECRET_EXPIRED",
    "message": "This secret has expired."
  }
}
```

### Response not found

```json
{
  "success": false,
  "error": {
    "code": "SECRET_NOT_FOUND",
    "message": "No secret letter was found here."
  }
}
```

---

## 12. Database Requirements

Bảng đề xuất:

```sql
CREATE TABLE secrets (
  id UUID PRIMARY KEY,
  token_hash TEXT NOT NULL UNIQUE,
  encrypted_content TEXT NOT NULL,
  encryption_iv TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMP NOT NULL,
  consumed_at TIMESTAMP NULL,
  burn_after_read BOOLEAN NOT NULL DEFAULT TRUE,
  theme TEXT NOT NULL DEFAULT 'classic-letter',
  view_count INTEGER NOT NULL DEFAULT 0
);
```

Index đề xuất:

```sql
CREATE INDEX idx_secrets_token_hash ON secrets(token_hash);
CREATE INDEX idx_secrets_expires_at ON secrets(expires_at);
CREATE INDEX idx_secrets_status ON secrets(status);
```

---

## 13. Frontend State Machine

Trạng thái frontend:

```ts
type SecretSceneState =
  | "loading"
  | "error"
  | "expired"
  | "consumed"
  | "not-found"
  | "sealed"
  | "opening"
  | "revealed"
  | "closing"
  | "burning"
  | "vanished";
```

Luồng chính:

```txt
loading
  -> sealed
  -> opening
  -> revealed
  -> closing
  -> burning
  -> vanished
```

Luồng lỗi:

```txt
loading -> expired
loading -> consumed
loading -> not-found
loading -> error
```

---

## 14. Animation Requirements

### 14.1. Envelope idle

- Phong thư scale nhẹ 1.00 → 1.015.
- Shadow thở nhẹ.
- Glow rất nhẹ.
- Loop chậm, không gây mỏi mắt.

---

### 14.2. Open animation

Timeline đề xuất:

```txt
0ms     Envelope lifts slightly
200ms   Top flap rotates open
500ms   Letter starts sliding up
900ms   Letter fully visible
1100ms  Content fades/types in
```

---

### 14.3. Close animation

Timeline đề xuất:

```txt
0ms     Content fades out
300ms   Letter slides down
700ms   Envelope flap closes
1000ms  Fire ignition starts
```

---

### 14.4. Burn animation

Timeline đề xuất:

```txt
0ms      Edge glow appears
400ms    Fire spreads across envelope
1000ms   Smoke appears
1600ms   Paper darkens and fragments
2400ms   Envelope dissolves
3000ms   Ash particles fade
3500ms   Final message appears
```

---

## 15. Error Handling

### 15.1. Network error

Nếu API lỗi mạng:

```txt
We could not open this secret letter.
Please check your connection and try again.
```

Không được tự mark secret là đọc nếu server chưa trả content thành công.

---

### 15.2. Animation error

Nếu animation asset không tải được:

- Vẫn hiển thị secret bằng UI fallback.
- Dùng card đơn giản.
- Sau khi người dùng đóng, fade out thay vì burn.

---

### 15.3. Expired during open

Nếu secret hết hạn trước khi người dùng mở:

- Không trả content.
- Hiển thị expired screen.

---

## 16. Acceptance Criteria

### AC-001 — Tạo secret thành công

Given người gửi nhập nội dung hợp lệ  
When nhấn Create Secret  
Then hệ thống trả về link dùng một lần.

---

### AC-002 — Mở secret lần đầu

Given link hợp lệ và chưa được đọc  
When người nhận mở link  
Then hệ thống hiển thị phong thư  
And người nhận có thể mở thư  
And đọc được nội dung.

---

### AC-003 — Không đọc lại được

Given secret đã được mở một lần  
When người dùng truy cập lại link  
Then hệ thống không trả nội dung  
And hiển thị thông báo secret đã biến mất.

---

### AC-004 — Tự hủy UI

Given người nhận đang xem nội dung thư  
When nhấn Close & Burn  
Then lá thư đóng lại  
And phong thư bốc cháy  
And biến mất  
And nội dung bị xóa khỏi frontend state.

---

### AC-005 — Secret hết hạn

Given secret đã quá hạn  
When người dùng mở link  
Then hệ thống hiển thị expired screen  
And không trả nội dung secret  
And visual thể hiện thư đã cũ, bụi phủ hoặc seal phai màu.

---

### AC-006 — Reduced motion

Given người dùng bật reduced motion trên hệ điều hành  
When mở secret  
Then hệ thống dùng animation tối giản  
And vẫn cho đọc nội dung bình thường.

---

### AC-007 — Race condition

Given hai client mở cùng một secret gần như đồng thời  
When cả hai gửi request open  
Then chỉ một client nhận được nội dung  
And client còn lại nhận trạng thái consumed hoặc unavailable.

---

### AC-008 — Distinct status visuals

Given secret ở các trạng thái khác nhau  
When người dùng truy cập link  
Then hệ thống phải hiển thị visual khác nhau cho active, consumed, expired, not-found và error.

---

## 17. Suggested Implementation Plan

### Phase 1 — Core secret system

- Tạo API create secret.
- Tạo API open secret.
- Lưu token hash.
- Encrypt content.
- Consume one-time bằng atomic update.
- Tạo màn hình status: consumed, expired, not found.

### Phase 2 — Basic envelope UI

- Dựng trang `/s/:token`.
- Tạo phong thư bằng HTML/CSS/SVG.
- Tạo state machine.
- Thêm animation mở/đóng bằng GSAP.
- Hiển thị nội dung thư.

### Phase 3 — Cinematic status screens

- Tạo visual cho consumed: tro tàn.
- Tạo visual cho expired: thư cũ, bụi phủ, seal nứt.
- Tạo visual cho not-found: giấy rách hoặc bàn trống.
- Tạo visual cho error: phong thư kẹt trong bóng tối.

### Phase 4 — Cinematic burn effect

- Thêm fire overlay.
- Thêm smoke/ash particles.
- Thêm burn mask.
- Thêm final vanish message.
- Thêm reduced-motion fallback.

### Phase 5 — Polish & production hardening

- Rate limit.
- Security headers.
- No-cache headers.
- Logging policy.
- Mobile optimization.
- Error fallback.
- Performance audit.

---

## 18. Recommended Tech Stack

### Frontend

```txt
Next.js hoặc React + Vite
TypeScript
GSAP
CSS Modules hoặc Tailwind CSS
Canvas particle effect
Optional: Rive / Lottie
```

### Backend

```txt
Node.js / NestJS / Express
hoặc Go / Fiber / Gin
```

### Database

```txt
PostgreSQL cho production
SQLite cho local demo
Redis optional cho TTL nhanh
```

### Deployment

```txt
Docker
Caddy hoặc Nginx reverse proxy
HTTPS bắt buộc
```

---

## 19. Future Enhancements

Các tính năng có thể thêm sau:

- Password-protected secret.
- Client-side encryption.
- Secret dạng file.
- QR code secret.
- Custom letter theme.
- Custom burn style.
- Sender chọn theme: classic, dark, royal, cyber, parchment.
- Expire after time hoặc after read.
- Self-destruct countdown.
- Anonymous reply.
- Audit event không chứa nội dung secret.
- Browser screenshot warning, dù không thể chặn hoàn toàn.
- Fake opened decoy mode cho bảo mật nâng cao.
- AI-assisted letter styling, chỉ áp dụng lên hình thức, không gửi nội dung secret ra bên thứ ba nếu chưa có consent rõ ràng.

---

## 20. Ghi chú thiết kế quan trọng

Điểm cần giữ trong đầu: **animation không phải bảo mật**.

Phong thư cháy chỉ là phần nghi lễ thị giác. Bảo mật thật nằm ở:

- Token khó đoán.
- Secret chỉ trả một lần.
- Atomic consume.
- Không cache.
- Không log.
- Không lưu plain text.
- Xóa state frontend sau khi đọc.

Nếu làm đúng, Secret Letter sẽ không chỉ là một clone của các one-time secret app khác. Nó sẽ có linh hồn riêng: một bí mật được gửi đi như một lá thư, được đọc một lần, rồi tắt lịm như ngọn lửa cuối cùng trên mép giấy.
