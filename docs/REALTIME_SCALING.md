# Realtime chat scale ngang, độ trễ thấp

Mục tiêu: có thể tăng số instance API mà không làm mất realtime khi hai user được NGINX đưa vào hai server khác nhau; tin nhắn vẫn hiện gần như tức thì, có thứ tự hợp lý trong một room và không mất sau reconnect.

Tài liệu này dùng tên `API-1`, `API-2` cho các instance chạy cùng source Go.

## Nguyên tắc nền tảng

1. WebSocket connection là **local state**: nó sống trong memory của đúng API instance đang giữ TCP connection.
2. Không có API instance nào là owner toàn cục của một user/room. State dùng chung nằm ngoài process: MySQL cho dữ liệu bền vững, Redis cho presence và transport realtime.
3. Message chỉ là “đã gửi thành công” sau khi được ghi bền vững vào MySQL. Redis Pub/Sub chỉ mang event để hiển thị realtime.
4. Client phải có khả năng sync lại từ MySQL sau reconnect. Không phụ thuộc Redis Pub/Sub để replay event bị bỏ lỡ.

```text
                          ┌──────────────────┐
                          │       MySQL      │
                          │ messages/rooms   │
                          └────────▲─────────┘
                                   │ transaction
┌────────┐      ┌────────┐  ┌──────┴──────┐        ┌────────┐      ┌────────┐
│Client A├─ WS ─┤ NGINX  ├─►│    API-1    │◄──────►│ Redis  │◄─────┤ API-2  │─ WS ─┤Client B│
└────────┘      └────────┘  │ local hub   │ Pub/Sub│         │ Pub/Sub│local hub│      └────────┘
                             └─────────────┘        └────────┘       └─────────┘
```

## Thành phần và trách nhiệm

| Thành phần | Giữ gì | Không được giữ |
| --- | --- | --- |
| NGINX | TLS termination, upgrade WebSocket, phân tải và drain instance | Session/socket state, routing room/user |
| API instance | Socket cục bộ; map subscriber theo room/user; xác thực, gửi/nhận event | Presence hoặc membership chung toàn hệ thống |
| Redis keys | Presence connection có TTL; typing indicator TTL; rate-limit ngắn hạn | Lịch sử message là nguồn sự thật |
| Redis Pub/Sub | Fan-out event ephemeral giữa API instances | Queue đảm bảo delivery/replay |
| MySQL | Message, room membership, cursor/sync query, `last_seen_at` | Heartbeat/typing per second |

Redis Pub/Sub có delivery at-most-once: subscriber offline lúc publish sẽ mất event. Vì vậy message phải ghi vào MySQL và reconnect phải sync từ database. Nếu một tác vụ thực sự cần retry/replay, dùng Redis Streams consumer groups hoặc một broker durable khác; đừng dùng Pub/Sub cho nó. Xem [Redis Pub/Sub](https://redis.io/docs/latest/develop/use-cases/pub-sub/) và [Redis Streams](https://redis.io/docs/latest/develop/data-types/streams/).

## NGINX

Không cần sticky session để xử lý socket cross-server. Một WebSocket đã upgrade sẽ ở nguyên backend đó cho đến lúc đóng; các event cross-server được fan-out bằng Redis. `ip_hash` chỉ cố gắng giữ client về một upstream theo IP và có thể phân tải lệch ở mobile carrier/NAT.

```nginx
upstream chat_api {
    least_conn;
    server api-1:8080 max_fails=3 fail_timeout=10s;
    server api-2:8080 max_fails=3 fail_timeout=10s;
}

server {
    listen 443 ssl http2;
    server_name api.example.com;

    location /ws {
        proxy_pass http://chat_api;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_read_timeout 90s;
        proxy_send_timeout 90s;
        proxy_buffering off;
    }

    location /api/ {
        proxy_pass http://chat_api;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

`proxy_read_timeout` phải lớn hơn heartbeat/pong deadline của API. Ví dụ server ping 25 giây và close sau 75 giây thì 90 giây là hợp lý. NGINX hỗ trợ upstream load balancing và `ip_hash` khi thực sự cần persistent routing; xem [NGINX load balancing documentation](https://nginx.org/en/docs/http/load_balancing.html).

## Thiết kế WebSocket local hub

Mỗi API instance có một hub local, chỉ chứa socket đang cắm vào chính instance đó:

```go
type Client struct {
    ID     string // connectionID
    UserID string
    Send   chan []byte
}

type Hub struct {
    byRoom map[string]map[*Client]struct{}
    byUser map[string]map[*Client]struct{}
}
```

Khi client mở room, server kiểm tra membership trong MySQL/cache rồi đăng ký socket vào `byRoom[roomID]`. Khi client rời room hoặc socket đóng, luôn remove ở cả `byRoom` và `byUser`.

Hub không gọi Redis và không ghi database. Hai public method chính:

```go
SendToRoomLocal(roomID string, payload []byte)
SendToUserLocal(userID string, payload []byte)
```

Mỗi socket chỉ có **một writer goroutine** đọc từ `Send`; tất cả fan-out chỉ enqueue vào channel bounded. Nếu queue đầy, coi socket là slow consumer, close nó và để client reconnect. Không block vòng lặp broadcast bởi một client chậm.

## Redis realtime bus

Mỗi API instance tạo một Redis Pub/Sub subscriber riêng trong lúc khởi động. Subscriber lắng nghe channel event realtime, decode event và gọi hub local.

### Channel strategy

Giai đoạn đầu (dễ vận hành):

```text
realtime:room            # chứa event của tất cả room
realtime:user            # presence/device event theo user
```

Mỗi instance nhận event rồi hub kiểm tra xem có local socket room/user đó không. Cách này đơn giản và phù hợp khi số instance/throughput còn vừa phải.

Khi throughput lớn: subscribe động theo room có local socket, với ref-count:

```text
realtime:room:<roomID>
```

- Local subscriber đầu tiên của room: instance `SUBSCRIBE realtime:room:<roomID>`.
- Local subscriber cuối cùng rời room: instance `UNSUBSCRIBE` channel đó.
- Pub/Sub subscriber cần tách riêng khỏi Redis command client; connection subscribe không dùng cho `GET`, `SET`, `XADD`.

Không subscribe một channel cho mỗi user ở quy mô rất lớn nếu không có subscription registry/ref-count; ưu tiên event room, vì đó là fan-out tự nhiên của chat.

### Envelope event

Mọi event realtime dùng envelope chung:

```json
{
  "eventId": "01J...",
  "type": "message.created",
  "roomId": "4c3e...",
  "sequence": 841,
  "occurredAt": "2026-09-29T15:30:10.123Z",
  "sourceInstanceId": "api-1-7f8c",
  "data": {
    "id": "msg-123",
    "clientMessageId": "local-uuid",
    "senderId": "user-a",
    "content": "Chào B"
  }
}
```

- `eventId`: deduplicate telemetry/subscriber khi cần.
- `sequence`: tăng đơn điệu theo room; dùng để phát hiện gap/late event, không dùng timestamp client để sort.
- `sourceInstanceId`: phục vụ observability, không phải logic business.
- `data` chỉ có field UI cần dùng. Client query message đầy đủ khi cần, không nhét full room/profile vào mỗi event.

## Luồng gửi message, tối ưu latency

```text
Client A                API-1                  MySQL          Redis        API-2               Client B
   │ WS send                 │                    │              │            │                    │
   │ clientMessageId         │                    │              │            │                    │
   │────────────────────────►│ validate quyền     │              │            │                    │
   │ optimistic render       │ INSERT + sequence  │              │            │                    │
   │                         │───────────────────►│              │            │                    │
   │                         │◄───────────────────│ commit       │            │                    │
   │ ACK(messageId, seq)     │                    │              │            │                    │
   │◄────────────────────────│ publish event      │              │            │                    │
   │                         │──────────────────────────────────►│            │                    │
   │                         │ local hub          │              │            │ local hub          │
   │                         │────────────────────────────────────────────────►│ WS event           │
   │                         │                    │              │            │───────────────────►│
```

Quy tắc bắt buộc:

1. Client A render optimistic ngay khi send; phải gắn `clientMessageId` UUID.
2. API authenticate từ socket session, kiểm tra user là member của room, validate payload/rate limit.
3. Trong một MySQL transaction: insert message, cấp `sequence` theo room và commit.
4. Chỉ sau commit, gửi ACK cho A và publish `message.created` lên Redis.
5. API-1 nhận/broadcast event local giống mọi instance khác. Không special-case theo user A; client deduplicate bằng `messageId`/`clientMessageId`.

Không đợi recipient ACK trước khi trả ACK sender. Đó là điều làm chat chậm. Nếu Redis publish thất bại sau commit, message vẫn đúng trong MySQL: log/metric lỗi và client sẽ thấy message qua sync; trong production nên dùng outbox để retry publish.

## Ordering, duplicate và idempotency

WebSocket/TCP giữ thứ tự trong một connection, nhưng cross-instance reconnect và retry có thể tạo event duplicate/late. Cần thiết kế client/server chịu được điều đó.

### Server

- Client gửi `clientMessageId` duy nhất. MySQL có unique key `(sender_id, client_message_id)`; retry trả về chính message cũ thay vì insert lần hai.
- `sequence` cấp atomically trong transaction. Có thể lock room row (`SELECT ... FOR UPDATE`) hoặc bảng counter theo room; đo contention với group rất lớn.
- Event chỉ được publish sau commit.
- Với outbox: transaction ghi `messages` + `outbox_events`; worker publish rồi đánh dấu delivered. Worker retry idempotent.

### Client

- Key UI theo `messageId`; optimistic message map thêm `clientMessageId`.
- ACK/event trùng: merge, không thêm row thứ hai.
- Nếu nhận `sequence > lastSequence + 1`: gọi sync endpoint lấy message kể từ `lastSequence`, sau đó mới render tiếp.
- Nếu event có `sequence <= lastSequence`: bỏ qua hoặc merge update (edit/delete reaction dùng version riêng).
- Sau WebSocket reconnect: gọi sync trước, rồi resume room subscriptions. Server có thể buffer event trong lúc sync hoặc client sync thêm một lần với cursor mới để tránh race.

## Reconnect và message sync API

Client giữ cursor của từng room:

```text
roomID -> lastSequence / lastMessageID
```

Sau reconnect:

1. Mở socket, authenticate và subscribe các room đang mở.
2. Gọi batch sync, ví dụ `GET /api/v1/rooms/{roomId}/messages?after_sequence=841&limit=100`.
3. Merge theo `messageId`, cập nhật `lastSequence`.
4. Nếu còn page tiếp theo, tiếp tục fetch. Khi kết thúc, UI đã hội tụ với MySQL.

Để tránh event rơi đúng giữa bước 1–3, đơn giản nhất là sync thêm lần cuối với cursor mới ngay sau khi nhận kết quả page cuối. Tối ưu hơn là server chấp nhận `resumeFromSequence` khi subscribe và trả catch-up + live event có fence/cursor, nhưng chỉ cần làm khi đã có số liệu chứng minh cần thiết.

## Presence và typing ở nhiều instance

### Presence

Mỗi WebSocket có `connectionID`. Redis giữ `presence:connection:<id>` với TTL 75 giây và user-to-connections set. Pong hợp lệ refresh TTL; close socket dọn ngay; TTL dọn crash/mất mạng.

Chuyển trạng thái chỉ phát event khi user đi từ 0 → 1 connection sống hoặc 1 → 0. Dùng Lua script/transaction để cleanup + đếm atomically. Tài liệu chi tiết: [PRESENCE.md](PRESENCE.md).

### Typing

Redis key `typing:room:<roomId>:user:<userId>` có TTL 5 giây. `typing.start` chỉ refresh TTL (throttle 2–3 giây); `typing.stop`, send message hoặc socket close thì xoá ngay. Publish `typing.changed` vào channel room; client tự hide khi `expiresAt` qua, không đợi stop event.

Typing và presence có thể mất một event, vì TTL + snapshot/next event vẫn đưa UI về đúng trạng thái. Message thì không được thiết kế theo kiểu đó.

## Reliability: outbox cho message event

MVP có thể `commit MySQL → PUBLISH Redis` trực tiếp. Rủi ro nhỏ nhưng có thật: process chết đúng sau commit, trước publish; recipient không nhận realtime event dù sync sau đó vẫn đúng.

Khi cần đảm bảo mạnh hơn, dùng transactional outbox:

```text
MySQL transaction:
  INSERT messages (...)
  INSERT outbox_events (event_id, type, room_id, payload, published_at = NULL)
commit

Publisher worker:
  lấy outbox chưa publish (claim/lock)
  PUBLISH Redis
  set published_at
```

Publisher có thể publish trùng khi crash giữa PUBLISH và update `published_at`. Đây là lý do client cần deduplicate `messageId` và consumer cần idempotent. Không cần đợi outbox worker trước khi trả ACK sender: sender đã được xác nhận bởi database commit.

## Scale theo từng giai đoạn

| Giai đoạn | Hạ tầng | Cách fan-out |
| --- | --- | --- |
| 1–2 API instances | MySQL + một Redis HA/managed | `realtime:room` global channel, mỗi instance lọc local subscribers |
| Nhiều instance / throughput tăng | Redis HA + metrics + outbox | Dynamic per-room Pub/Sub hoặc topic routing |
| Rất lớn / nhiều region | Shard room theo region/partition, durable broker | NATS/Kafka/Redis Streams cho workloads cần replay; WebSocket gateway riêng nếu cần |

Không tách microservice chỉ vì thêm instance. Bước đầu tốt nhất là stateless API replicas cùng source và shared Redis/MySQL. Tách WebSocket gateway chỉ khi socket CPU/memory hoặc network fan-out trở thành bottleneck đã đo được.

## Deploy, autoscale và graceful shutdown

1. Readiness endpoint chỉ trả ready khi MySQL và Redis sẵn sàng, đồng thời Redis subscriber đã subscribe.
2. Khi scale down/deploy, mark instance `draining`: NGINX không route handshake mới vào instance đó.
3. Instance gửi `server.draining` cho socket local; client reconnect bằng exponential backoff có jitter.
4. Đợi grace period, đóng socket còn lại, unsubscribe Redis, close client. Presence TTL là safety net nếu process bị kill.
5. Không dùng `SIGKILL` là flow deploy bình thường; sẽ làm dồn reconnect và mất Pub/Sub event không cần thiết.

Theo dõi: connected sockets/instance, socket send-queue drops, WebSocket reconnect rate, Redis publish/subscribe latency, Pub/Sub subscriber reconnect, message write latency p50/p95/p99, outbox backlog, sync gap count và number of local subscribers/room.

## Security và quota

- Xác thực token ở WebSocket upgrade; lấy `userID` từ claims, không nhận từ event payload.
- Kiểm tra membership cho `subscribe_room`, `message.send`, typing và presence snapshot.
- Giới hạn socket/user, rooms subscribed/socket, payload size, event rate và message rate.
- Bảo vệ Redis trong private network; app dùng credential/ACL tối thiểu; không expose Redis public.
- Log `instanceID`, `connectionID`, `roomID`, `eventID`; không log access token/nội dung nhạy cảm.

## Lộ trình cho codebase này

1. Hoàn thành `internal/presence/store.go` dùng Redis và test atomic multi-connection.
2. Tạo `internal/realtime/bus.go` bọc Redis Pub/Sub; startup/subscriber lifecycle ở `cmd/server/main.go`.
3. Refactor `internal/ws/hub.go` để `Client` có `UserID`, room subscription và local `SendToRoomLocal`/`SendToUserLocal`.
4. Thêm WebSocket endpoint Gin (`/ws`), JWT authentication, read/write pumps, ping/pong deadline và graceful close.
5. Dùng realtime bus cho `message.created`, presence và typing. Event inbound từ Redis chỉ fan-out local, không publish lại (tránh loop).
6. Thêm message `sequence`, `clientMessageId` unique index và endpoint sync cursor.
7. Thêm transactional outbox khi realtime event loss sau DB commit không còn chấp nhận được.
8. Thêm NGINX upstream, readiness/drain endpoint, compose replicas hoặc Kubernetes deployment/HPA.

## Checklist load test trước production

- 2 API instances: sender và recipient cố ý connect khác instance, recipient vẫn nhận message.
- Kill API-1 giữa DB commit và Pub/Sub: recipient sync được message sau reconnect.
- 1.000+ socket test: slow client không làm chậm room broadcast.
- Scale API-2 lên/xuống trong lúc chat: client reconnect và không duplicate/mất message sau sync.
- Hai client cùng resend một `clientMessageId`: chỉ có một DB message.
- Redis restart: message durable vẫn sync được, typing/presence tự hồi phục theo TTL/reconnect.
- Đo p50/p95/p99 send-to-receive trong cùng region; đặt SLO trước khi tối ưu, ví dụ p95 dưới 200 ms cho realtime event.
