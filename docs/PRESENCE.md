# User presence (online/offline)

Tài liệu này mô tả cách xác định trạng thái online đáng tin cậy cho chat. Presence là **trạng thái tạm thời**, không phải dữ liệu nghiệp vụ: server là nguồn sự thật, còn client chỉ là tín hiệu.

## Mục tiêu và định nghĩa

| Trạng thái | Điều kiện |
| --- | --- |
| `online` | User còn ít nhất một WebSocket connection hợp lệ trên bất kỳ tab/thiết bị nào. |
| `offline` | Không còn connection hợp lệ. `last_seen_at` là lần cuối server xác nhận user còn online. |
| `away` (tuỳ chọn) | User online nhưng không có tương tác trong một ngưỡng, ví dụ 5 phút. Không dùng trạng thái này để quyết định gửi/không gửi message. |

Không coi một lệnh REST `setOnline` hay `logout` là nguồn duy nhất: tab có thể bị kill, app bị crash hoặc mạng bị mất nên client không kịp gọi API.

## Kiến trúc đề xuất

```text
Browser/mobile ── WebSocket ──> Chat API instance
     heartbeat                         │
                                       ├── Redis: connections + TTL (nguồn presence)
                                       ├── Redis Pub/Sub: user.presence.changed
                                       └── MySQL: last_seen_at (lưu lịch sử, không đọc realtime)
```

- Dùng WebSocket ping/pong ở protocol level. Không cần thêm message `heartbeat` JSON nếu thư viện WebSocket đã hỗ trợ ping/pong.
- Redis giữ mỗi connection riêng với TTL. TTL là cơ chế an toàn khi server không nhận được sự kiện đóng kết nối.
- MySQL chỉ lưu `last_seen_at` khi user chuyển từ online sang offline; không ghi database cho từng heartbeat.
- Khi chạy nhiều API instance, các instance cùng dùng Redis. Không dùng `map` trong memory làm nguồn presence chung.

## Cấu trúc Redis

Giả sử `userID = 42`, connection ID ngẫu nhiên là `c_01J...`:

```text
presence:connection:c_01J...   = 42      EX 75
presence:user:42               = Set(c_01J..., c_01K...)
```

Mỗi tab hoặc thiết bị có một `connectionID` mới. Redis set giúp biết user còn ít nhất một connection; key connection có TTL giúp dọn các socket chết. Set có thể còn member cũ sau khi TTL hết, vì vậy **không được** chỉ kiểm tra `SCARD` mà phải loại những member không còn key `presence:connection:<id>`.

Nên đóng gói các thao tác Redis bằng Lua script hoặc transaction để tránh race condition giữa connect/disconnect/TTL cleanup:

1. `connect(userID, connectionID)`: tạo key connection với TTL, thêm ID vào set; nếu trước đó user không có connection sống thì trả về `becameOnline=true`.
2. `refresh(connectionID)`: chỉ gia hạn TTL nếu key vẫn tồn tại và thuộc đúng user đã xác thực.
3. `disconnect(userID, connectionID)`: xoá key, xoá member khỏi set, dọn member hết hạn; nếu không còn connection sống thì trả về `becameOffline=true`.
4. `cleanup(userID)`: chạy khi kiểm tra presence hoặc định kỳ. Dọn member hết hạn rồi quyết định online/offline.

`SET`/`SADD`/kiểm tra số connection phải được thực hiện nguyên tử. Nếu không, hai tab cùng kết nối hoặc cùng đóng có thể phát sai event online/offline.

## Flow kết nối

Các giá trị khởi đầu hợp lý:

```text
WebSocket ping interval: 25 giây
Pong/read deadline:      75 giây
Redis connection TTL:    75 giây
Client reconnect:        exponential backoff 1s → 2s → 4s ... tối đa 30s, có jitter
```

1. Client mở `wss://api.example.com/ws` và gửi access token bằng cách được hệ thống hỗ trợ (ưu tiên cookie `HttpOnly` với web; header `Authorization` khi client hỗ trợ). Không ghi JWT vào URL query vì có thể bị log.
2. Server xác thực JWT **trước** khi đăng ký presence; lấy `userID` từ claim, không tin `userId` do client gửi.
3. Server tạo `connectionID`, gọi `connect`. Nếu đây là connection sống đầu tiên, publish event `presence.changed` với `status=online`.
4. Server gửi `connection.ready` cho client. Client chỉ hiện online khi đã nhận acknowledgement này.
5. Server gửi ping mỗi 25 giây và gia hạn TTL khi nhận pong (hoặc message hợp lệ từ client). Nếu hết read deadline, đóng socket; defer cleanup sẽ chạy `disconnect`.
6. Khi TCP/WebSocket đóng bình thường, handler gọi `disconnect` ngay. Nếu server/client chết đột ngột, sau tối đa TTL user mới thành offline.

`offline` có độ trễ tối đa khoảng 75 giây là đánh đổi cần thiết để không báo offline nhầm khi mạng chỉ chập chờn ngắn.

## Event contract

Client subscribe event presence cho những user có liên quan, ví dụ các thành viên của room đang mở. Không broadcast toàn bộ user của hệ thống.

```json
{
  "type": "presence.changed",
  "data": {
    "userId": "42",
    "status": "online",
    "lastSeenAt": null,
    "occurredAt": "2026-09-29T10:15:00Z"
  }
}
```

Khi offline:

```json
{
  "type": "presence.changed",
  "data": {
    "userId": "42",
    "status": "offline",
    "lastSeenAt": "2026-09-29T10:16:15Z",
    "occurredAt": "2026-09-29T10:16:15Z"
  }
}
```

`occurredAt` do server tạo. Client nên xử lý event idempotent: nếu nhận event cũ hơn trạng thái đang có thì bỏ qua. Sau reconnect, client gọi REST/GraphQL query lấy snapshot presence cho danh sách user cần hiển thị rồi tiếp tục nhận event; đừng giả định event trong lúc mất kết nối sẽ được replay.

## Typing indicator: “A, B đang soạn tin”

Typing indicator khác presence: nó áp dụng cho **một room cụ thể**, chỉ có ý nghĩa vài giây và tuyệt đối không lưu MySQL. Server không cần biết user có thật sự đang gõ hay không; nó chỉ chuyển tiếp một tín hiệu UI có TTL.

### State và event

Lưu mỗi người đang soạn bằng một key Redis có TTL (cũng có thể dùng memory nếu app chỉ có một instance):

```text
typing:room:<roomID>:user:<userID> = 1  EX 5
```

Client gửi event khi người dùng bắt đầu hoặc tiếp tục gõ:

```json
{ "type": "typing.start", "data": { "roomId": "room-123" } }
```

Server xác thực socket, kiểm tra user là member của room rồi `SET ... EX 5`. Nếu key chưa tồn tại trước đó, broadcast đến **các thành viên khác trong room**:

```json
{
  "type": "typing.changed",
  "data": {
    "roomId": "room-123",
    "userId": "42",
    "isTyping": true,
    "expiresAt": "2026-09-29T10:20:05Z"
  }
}
```

Khi user gửi message, chuyển room, đóng socket hoặc UI hết focus, client có thể gửi:

```json
{ "type": "typing.stop", "data": { "roomId": "room-123" } }
```

Server xoá key và broadcast `isTyping: false`. Đây chỉ là tối ưu để indicator biến mất sớm; tính đúng đắn đến từ TTL 5 giây. Nếu client bị kill, mất mạng, hoặc quên gửi `typing.stop`, key tự hết hạn và UI tự ẩn indicator. Không cần broadcast một event `false` đúng thời điểm TTL hết hạn nếu client đã dùng `expiresAt` để tự dọn state.

### Client behaviour

- Gửi `typing.start` ngay ở lần input đầu tiên; sau đó throttle/debounce để refresh tối đa **một lần mỗi 2–3 giây** khi vẫn gõ. Không gửi event theo từng phím.
- Khi nhận `typing.changed`, lưu `userId → expiresAt` cho room. Một timer local loại user hết hạn, kể cả không nhận được event stop.
- Không hiển thị bản thân trong danh sách người đang soạn.
- Khi gửi message thành công hoặc socket reconnect/room bị unmount, xoá local typing state và gửi `typing.stop` nếu socket còn mở.

Ví dụ tạo text cho group, với danh sách đã lọc là `names`:

| Số người đang gõ | Hiển thị |
| --- | --- |
| 0 | Không hiển thị. |
| 1 | `An đang soạn tin…` |
| 2 | `An và Bình đang soạn tin…` |
| 3 | `An, Bình và Chi đang soạn tin…` |
| 4+ | `An, Bình và 2 người khác đang soạn tin…` |

Nên giới hạn tối đa 2 tên để dòng UI không nhảy/overflow. Với room rất lớn, chỉ fan-out indicator đến những user đang mở/subscribed room đó, không gửi cho toàn bộ membership.

### Anti-abuse và edge cases

- Chỉ chấp nhận `roomId` mà user có quyền gửi message; `userId` luôn lấy từ WebSocket session, không nhận từ payload.
- Rate limit khoảng 1 event/giây/socket; event thừa chỉ refresh TTL hoặc bị bỏ qua.
- Một user mở hai tab cùng gõ: UI vẫn chỉ có một user. Key theo `roomID + userID` nên tự deduplicate; TTL được refresh bởi tab nào đang hoạt động.
- Khi message được tạo qua REST thay vì WebSocket, backend cũng nên xoá `typing:room:<roomID>:user:<userID>` và phát `isTyping:false` nếu key tồn tại.
- Message `typing.start` có thể bị đến trễ; client không kéo dài `expiresAt` quá giá trị server gửi. Event `typing.stop` cho cùng user/room phải thắng state đang có.

### Gắn vào code hiện tại

`graph/schema.graphqls` đã có subscription `typingIndicator(roomID)` và model `TypingEvent`, nhưng resolver hiện chưa được implement. Có thể dùng nó làm public contract, hoặc thống nhất toàn bộ realtime qua WebSocket event ở trên; không nên duy trì hai luồng độc lập cho cùng một indicator. Dù chọn cách nào, lớp service nên có API tương tự `StartTyping(roomID, userID)`, `StopTyping(roomID, userID)` và chỉ phát event sau khi đã kiểm tra room membership.

## Quyền riêng tư và phân quyền

- Chỉ trả presence khi requester có quyền xem user đó, tối thiểu là hai người cùng room/direct conversation; kiểm tra quyền ở cả snapshot API và event subscription.
- Cần hỗ trợ user tắt “hiển thị trạng thái online”. Khi đó không phát presence của user đó cho người khác; vẫn có thể giữ presence nội bộ để routing notification.
- Không đưa danh sách connection, IP, device ID hoặc `lastSeenAt` có độ chính xác quá cao ra client nếu không cần.
- Giới hạn số WebSocket/connection trên mỗi user và rate limit endpoint upgrade để chống abuse.

## Gắn vào codebase hiện tại

`internal/ws/hub.go` hiện chỉ giữ client theo `RoomId` trong memory. Nó phù hợp để fan-out message trong **một instance**, nhưng chưa có `UserID`, lifecycle WebSocket thực tế, hay shared storage nên không nên dùng nó làm presence store.

Thứ tự triển khai còn lại:

1. Tạo `internal/presence/store.go`: interface `Connect`, `Refresh`, `Disconnect`, `Get` và implementation Redis. Interface giúp test bằng fake store.
2. Tạo WebSocket handler, xác thực từ JWT và tạo một `connectionID` cho từng socket. Một goroutine chỉ đọc/pong, một goroutine chỉ ghi; không ghi socket đồng thời từ nhiều goroutine.
3. Trong lifecycle handler gọi `Connect` sau authenticate, `Refresh` khi pong, và luôn `defer Disconnect` ngay sau khi Connect thành công.
4. Khi transition xảy ra, publish `presence.changed` vào event bus. Với nhiều instance, thay `pkg/eventbus` in-process bằng Redis Pub/Sub (hoặc dùng cả hai: local hub nhận event từ subscriber Redis).
5. Thêm endpoint/query snapshot, ví dụ `GET /api/v1/users/presence?ids=...`, có kiểm tra quyền và batch query Redis. Đừng gọi Redis một lần cho mỗi user nếu danh sách room lớn.
6. Cập nhật fan-out room để chỉ gửi event presence tới các subscriber/room liên quan.

## Pseudocode lifecycle

```go
func serveWS(conn *websocket.Conn, userID string) {
    connectionID := uuid.NewString()
    result := presence.Connect(ctx, userID, connectionID) // Redis TTL = 75s
    if result.BecameOnline {
        publishPresence(userID, "online", nil)
    }
    defer func() {
        result := presence.Disconnect(ctx, userID, connectionID)
        if result.BecameOffline {
            now := time.Now().UTC()
            saveLastSeenAsync(userID, now)
            publishPresence(userID, "offline", &now)
        }
    }()

    for readMessagesAndPongs(conn) {
        presence.Refresh(ctx, userID, connectionID)
    }
}
```

Phần `readMessagesAndPongs` phải đặt read deadline và chỉ refresh sau một pong/message hợp lệ. Đừng refresh TTL từ một HTTP endpoint công khai vì như vậy client có thể giữ user online mà không có socket thật.

## Cases bắt buộc phải test

- Kết nối/đóng một tab: online rồi offline ngay.
- Hai tab/thiết bị: đóng một connection vẫn online; chỉ offline khi đóng connection cuối.
- Mất Wi-Fi, kill app, laptop sleep: user offline khi TTL hết, không cần gọi logout.
- Reconnect trước TTL hết: không phát chuỗi offline/online sai.
- Hai API instance: connect ở instance A, disconnect ở B hoặc instance A bị kill; trạng thái vẫn đúng sau TTL.
- Redis restart/unavailable: chọn rõ chính sách fail-safe (khuyến nghị không hiển thị `online` thay vì báo sai); log/metric lỗi và không làm crash chat socket.
- Event trùng/lệch thứ tự: UI vẫn hội tụ nhờ xử lý idempotent và snapshot sau reconnect.
- User không có quyền xem presence hoặc đã tắt hiển thị trạng thái: không nhận được event/snapshot.

## Observability

Theo dõi các metric: số socket đang mở, số user online, reconnect rate, pong timeout, thời gian từ mất heartbeat đến offline, lỗi Redis và số event online/offline. Ghi log có `user_id`, `connection_id`, `instance_id` nhưng không log access token.

## Những lỗi cần tránh

- Cập nhật `users.status` bằng REST ở login/logout và tin rằng nó luôn đúng.
- Một biến `isOnline` duy nhất cho user: sai khi nhiều tab/thiết bị.
- Dùng in-memory map khi có nhiều process/pod.
- Ghi MySQL mỗi heartbeat: tạo write load không cần thiết.
- Chỉ dựa vào callback disconnect: không bắt được crash và partition mạng.
- Broadcast presence đến tất cả user hoặc cho client tự chọn `userId` để set trạng thái.
