# 🧩 Microservices Blog Platform — Complete Implementation Guide

> **A practical, event-driven microservices demo** showcasing service autonomy, polyglot architecture, Kafka messaging, and real-world design patterns.

---

## ⚡ Quick Summary

**What You'll Build:**
A blog platform with 5 microservices that communicate via Kafka events and REST APIs. Users can register (with OAuth), create posts, receive real-time notifications, and get AI-generated summaries.

**What You'll Learn:**

- ✅ Service boundaries & data ownership
- ✅ Event-driven architecture with Kafka
- ✅ Data denormalization & eventual consistency
- ✅ Polyglot persistence (SQL + NoSQL)
- ✅ API Gateway pattern with JWT auth
- ✅ Redis caching strategies
- ✅ Third-party integrations (Firebase, OAuth)
- ✅ Docker containerization

**Tech Stack:**

```
Services:    ASP.NET Core, NestJS (x3), FastAPI
Messaging:   Apache Kafka
Databases:   PostgreSQL, MongoDB (flexible placement)
Cache:       Redis
Push:        Firebase Cloud Messaging
Infra:       Docker Compose
```

**Architecture at a Glance:**

```
Frontend → API Gateway → [Auth, Blog, AI, Notification] → Kafka → Event Consumers
                              ↓        ↓                              ↓
                          PostgreSQL  PostgreSQL/MongoDB        MongoDB/PostgreSQL
                                      + Redis Cache
```

**Key Pattern Demonstrated:**
When a user updates their name, Auth Service publishes `user.updated` to Kafka. Blog Service consumes this event and updates denormalized author names in all posts. This is **eventual consistency** in action—services stay decoupled while data propagates asynchronously.

---

## 🎯 Project Overview

A modern blog platform built with microservices architecture to demonstrate:

- **Service Autonomy** — Each service owns its data and business logic
- **Event-Driven Architecture** — Kafka for async communication and decoupling
- **Polyglot Persistence** — PostgreSQL + MongoDB + Redis (strategic database selection)
- **Data Denormalization** — Strategic data duplication for service independence
- **Polyglot Services** — ASP.NET Core + NestJS + FastAPI (3 languages, 2 frameworks)
- **Third-Party Integration** — Firebase for push notifications, OAuth providers
- **Production Patterns** — API Gateway, JWT auth, caching, idempotency

**Core Philosophy:** Each service is a mini-application with clear boundaries. Services communicate via events (Kafka) and HTTP, never by sharing databases.

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Frontend (React/Next.js)                │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
              ┌──────────────────────┐
              │   API Gateway        │  ◄── JWT Validation
              │   (NestJS)           │      Request Routing
              │                      │      Rate Limiting
              └──────────┬───────────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
        ▼                ▼                ▼
┌──────────────┐  ┌─────────────┐  ┌──────────────┐
│ Auth Service │  │Blog Service │  │  AI Service  │
│  (ASP.NET)   │  │  (NestJS)   │  │  (FastAPI)   │
│              │  │             │  │              │
│ • Users      │  │ • Posts     │  │ • Summarize  │
│ • OAuth      │  │ • Comments  │  │ • Title Gen  │
│ • JWT Issue  │  │ • Likes     │  │              │
└──────┬───────┘  └──────┬──────┘  └──────────────┘
       │                 │
       │                 │
   PostgreSQL     PostgreSQL/MongoDB      Redis
   (auth_db)         (blog_db)           (cache)
       │                 │                   
       └────────┬────────┘                
                │
                ▼
         ┌─────────────┐
         │    Kafka    │  ◄── Event Bus
         │  Topics:    │      • user.registered
         │             │      • user.updated
         └──────┬──────┘      • post.created
                │             • post.liked
                ▼
    ┌─────────────────────┐
    │ Notification Service│
    │     (NestJS)        │
    │                     │
    │ • Kafka Consumer    │
    │ • Firebase Push     │
    │ • Event Handler     │
    └─────────────────────┘
              │
     MongoDB/PostgreSQL
    (notifications_db)
```

**Database Flexibility:**
You can choose to use MongoDB for either:

- **Blog Service** (posts as documents, flexible schema for different content types)
- **Auth Service** (user profiles with flexible attributes)
- **Notification Service** (append-only notifications, different event types)

The README keeps this abstract so you can decide based on what you want to learn.

---

## 📦 Service Boundaries & Responsibilities

### 1️⃣ Auth & User Service (ASP.NET Core)

**Owns:** User accounts, authentication, authorization

**Tech Stack:**

- ASP.NET Core 8 + Identity
- PostgreSQL (or MongoDB for flexible user profiles)
- OAuth 2.0 (Google, GitHub providers)

**Database Schema (PostgreSQL):**

```sql
Users {
  id: UUID PRIMARY KEY,
  email: VARCHAR UNIQUE,
  password_hash: VARCHAR,
  name: VARCHAR,
  avatar_url: VARCHAR,
  created_at: TIMESTAMP
}

ExternalLogins {
  user_id: UUID,
  provider: VARCHAR,  -- 'google', 'github'
  provider_user_id: VARCHAR,
  UNIQUE(provider, provider_user_id)
}
```

**Alternative: MongoDB Schema:**

```javascript
{
  _id: ObjectId,
  email: "user@example.com",
  passwordHash: "...",
  name: "Alice",
  avatar: "https://...",
  preferences: {
    theme: "dark",
    notifications: true,
    // Flexible schema!
  },
  externalLogins: [
    { provider: "google", providerId: "123" }
  ],
  createdAt: ISODate("2026-01-14T10:30:00Z")
}
```

**Key Endpoints:**

- `POST /auth/register` → Create user, emit `user.registered`
- `POST /auth/login` → Return JWT
- `GET /auth/google` → OAuth flow redirect
- `GET /users/{id}` → User profile (called by other services)
- `PATCH /users/{id}` → Update profile, emit `user.updated`

**Events Published:**

```json
{
  "type": "user.registered",
  "payload": {
    "userId": "uuid",
    "email": "user@example.com",
    "name": "Alice",
    "avatar": "https://..."
  }
}

{
  "type": "user.updated",
  "payload": {
    "userId": "uuid",
    "name": "Alice Smith",
    "avatar": "https://new-url"
  }
}
```

---

### 2️⃣ Blog Service (NestJS)

**Owns:** Posts, comments, likes, user feeds

**Tech Stack:**

- NestJS (TypeScript)
- PostgreSQL or MongoDB
- Redis (caching)
- Kafka producer & consumer

**Database Schema - Option A (PostgreSQL - Denormalized):**

```sql
Posts {
  id: UUID PRIMARY KEY,
  title: VARCHAR,
  content: TEXT,
  author_id: UUID,           -- Reference only
  author_name: VARCHAR,      -- DENORMALIZED from User
  author_avatar: VARCHAR,    -- DENORMALIZED from User
  summary: TEXT,             -- Generated by AI Service
  created_at: TIMESTAMP,
  updated_at: TIMESTAMP
}

Comments {
  id: UUID PRIMARY KEY,
  post_id: UUID,
  author_id: UUID,
  author_name: VARCHAR,      -- DENORMALIZED
  content: TEXT,
  created_at: TIMESTAMP
}

Likes {
  post_id: UUID,
  user_id: UUID,
  created_at: TIMESTAMP,
  PRIMARY KEY (post_id, user_id)
}
```

**Database Schema - Option B (MongoDB):**

```javascript
// Posts collection
{
  _id: ObjectId,
  title: "My Blog Post",
  content: "Full content here...",
  author: {
    id: "user-uuid",
    name: "Alice",           // Denormalized
    avatar: "https://..."    // Denormalized
  },
  summary: "AI-generated summary",
  comments: [               // Embedded or referenced
    {
      id: ObjectId,
      author: { id: "...", name: "Bob" },
      content: "Great post!",
      createdAt: ISODate()
    }
  ],
  likes: ["user-id-1", "user-id-2"],
  createdAt: ISODate(),
  updatedAt: ISODate()
}
```

**Why Denormalization?**

- Blog Service can display posts WITHOUT calling Auth Service
- Faster reads (no cross-service JOINs)
- Service remains operational even if Auth Service is down
- Trade-off: Must handle eventual consistency

**Key Endpoints:**

- `GET /posts` → List posts (cached in Redis)
- `POST /posts` → Create post
- `GET /posts/{id}` → Single post with comments
- `POST /posts/{id}/comments` → Add comment
- `POST /posts/{id}/like` → Like post, emit `post.liked`

**Creating a Post (Data Flow):**

```typescript
async createPost(userId: string, dto: CreatePostDto) {
  // 1. Fetch user data from Auth Service
  const user = await this.authClient.get(`/users/${userId}`);
  
  // 2. Store post with denormalized user data
  const post = await this.postRepo.save({
    title: dto.title,
    content: dto.content,
    authorId: userId,
    authorName: user.name,      // Copy from Auth
    authorAvatar: user.avatar   // Copy from Auth
  });
  
  // 3. Publish event to Kafka
  await this.kafka.publish('post.created', {
    postId: post.id,
    authorId: userId,
    title: post.title
  });
  
  // 4. Invalidate cache
  await this.redis.del('posts:popular');
  
  return post;
}
```

**Handling User Updates (Eventual Consistency):**

```typescript
@OnEvent('user.updated')  // Kafka consumer
async onUserUpdated(event: UserUpdatedEvent) {
  // Update all posts by this user
  await this.postRepo.update(
    { authorId: event.userId },
    { 
      authorName: event.name,
      authorAvatar: event.avatar 
    }
  );
  
  // Update all comments by this user
  await this.commentRepo.update(
    { authorId: event.userId },
    { authorName: event.name }
  );
  
  // Invalidate related caches
  await this.redis.del(`user:${event.userId}:posts`);
}
```

**Redis Caching Strategy:**

```
Keys:
  posts:popular          → List of top 10 posts (TTL: 5min)
  posts:feed:{userId}    → User's personalized feed (TTL: 2min)
  posts:{postId}         → Single post cache (TTL: 1min)

Invalidation:
  - On new post: delete posts:popular
  - On new comment: delete posts:{postId}
  - Background job: refresh feeds every 5min
```

**Events Published:**

- `post.created` → New post published
- `post.liked` → Someone liked a post

**Events Consumed:**

- `user.updated` → Update denormalized user data

---

### 3️⃣ Notification Service (NestJS)

**Owns:** User notifications, push notifications

**Tech Stack:**

- NestJS (TypeScript)
- MongoDB or PostgreSQL
- Firebase Cloud Messaging (push notifications)
- Kafka consumer

**Why NestJS for Notifications?**

- ✅ Built-in Kafka microservices support
- ✅ Strong typing for event schemas
- ✅ Dependency injection for Firebase client
- ✅ Consistent with Blog Service architecture
- ✅ Excellent for event-driven patterns

**Database Schema - Option A (MongoDB - Recommended):**

```javascript
// Notifications collection
{
  _id: ObjectId,
  userId: "uuid",
  type: "post_liked" | "post_created" | "welcome",
  title: "New Like!",
  message: "Someone liked your post 'Hello World'",
  metadata: {           // Flexible schema per event type
    postId: "uuid",
    actorId: "uuid",
    actorName: "Bob"
  },
  read: false,
  createdAt: ISODate("2026-01-14T10:30:00Z")
}

// Index for fast queries
db.notifications.createIndex({ userId: 1, createdAt: -1 })
db.notifications.createIndex({ userId: 1, read: 1 })
```

**Database Schema - Option B (PostgreSQL):**

```sql
Notifications {
  id: UUID PRIMARY KEY,
  user_id: UUID,
  type: VARCHAR,
  title: VARCHAR,
  message: TEXT,
  metadata: JSONB,      -- Flexible metadata
  read: BOOLEAN DEFAULT false,
  created_at: TIMESTAMP
}

-- Indexes
CREATE INDEX idx_notifications_user ON Notifications(user_id, created_at DESC);
CREATE INDEX idx_notifications_unread ON Notifications(user_id, read);
```

**Key Endpoints:**

```typescript
@Controller('notifications')
export class NotificationsController {
  @Get()
  async getUserNotifications(
    @Query('userId') userId: string,
    @Query('page') page = 1,
    @Query('limit') limit = 20
  ) {
    return this.notificationService.findByUser(userId, page, limit);
  }

  @Get('unread-count')
  async getUnreadCount(@Query('userId') userId: string) {
    return this.notificationService.countUnread(userId);
  }

  @Patch(':id/read')
  async markAsRead(@Param('id') id: string) {
    return this.notificationService.markAsRead(id);
  }
}
```

**Event Handling (Kafka Consumer):**

```typescript
import { Controller } from '@nestjs/common';
import { EventPattern, Payload } from '@nestjs/microservices';
import { FirebaseService } from './firebase.service';

@Controller()
export class NotificationEventController {
  constructor(
    private readonly notificationService: NotificationService,
    private readonly firebaseService: FirebaseService
  ) {}

  @EventPattern('post.liked')
  async handlePostLiked(@Payload() event: PostLikedEvent) {
    // 1. Create notification in database
    const notification = await this.notificationService.create({
      userId: event.postAuthorId,
      type: 'post_liked',
      title: 'New Like!',
      message: `${event.likerName} liked your post "${event.postTitle}"`,
      metadata: {
        postId: event.postId,
        actorId: event.likerId,
        actorName: event.likerName
      }
    });

    // 2. Send push notification via Firebase
    await this.firebaseService.sendPushNotification(
      event.postAuthorId,
      'New Like!',
      `${event.likerName} liked your post`
    );
  }

  @EventPattern('post.created')
  async handlePostCreated(@Payload() event: PostCreatedEvent) {
    // Notify followers (if follows feature exists)
    // For now, just log
    console.log('New post created:', event.postId);
  }

  @EventPattern('user.registered')
  async handleUserRegistered(@Payload() event: UserRegisteredEvent) {
    // Send welcome notification
    await this.notificationService.create({
      userId: event.userId,
      type: 'welcome',
      title: 'Welcome!',
      message: 'Welcome to our platform. Start creating posts!',
      metadata: {}
    });

    await this.firebaseService.sendPushNotification(
      event.userId,
      'Welcome!',
      'Thanks for joining our platform'
    );
  }
}
```

**Firebase Integration:**

```typescript
import { Injectable } from '@nestjs/common';
import * as admin from 'firebase-admin';

@Injectable()
export class FirebaseService {
  constructor() {
    admin.initializeApp({
      credential: admin.credential.cert({
        projectId: process.env.FIREBASE_PROJECT_ID,
        privateKey: process.env.FIREBASE_PRIVATE_KEY,
        clientEmail: process.env.FIREBASE_CLIENT_EMAIL
      })
    });
  }

  async sendPushNotification(
    userId: string,
    title: string,
    body: string
  ): Promise<string> {
    // Get user's device token (stored during login/registration)
    const token = await this.getUserDeviceToken(userId);
    
    if (!token) {
      console.log('No device token for user:', userId);
      return null;
    }

    const message: admin.messaging.Message = {
      notification: { title, body },
      token
    };

    try {
      const response = await admin.messaging().send(message);
      console.log('Push notification sent:', response);
      return response;
    } catch (error) {
      console.error('Error sending push notification:', error);
      throw error;
    }
  }

  private async getUserDeviceToken(userId: string): Promise<string> {
    // Fetch from database where you store device tokens
    // Stored when user logs in from frontend
    return 'device-token-here';
  }
}
```

**Why Keep a Notification Service (Not Just Firebase)?**

1. ✅ In-app notification center (like Twitter/Reddit)
2. ✅ Notification history & read/unread status
3. ✅ Works even when user is offline
4. ✅ Demonstrates event-driven architecture
5. ✅ Shows MongoDB/PostgreSQL usage
6. ✅ Displays third-party API integration

**Events Consumed:**

- `user.registered` → Send welcome notification
- `post.created` → Notify followers (future feature)
- `post.liked` → Notify post author

---

### 4️⃣ AI Service (FastAPI)

**Owns:** AI/ML operations (summarization, title generation)

**Tech Stack:**

- FastAPI (Python)
- OpenAI API or local LLM
- No database (stateless)

**Key Endpoints:**

```python
from fastapi import FastAPI
from pydantic import BaseModel
import openai

app = FastAPI()

class SummarizeRequest(BaseModel):
    content: str

class TitleRequest(BaseModel):
    content: str

@app.post("/ai/summarize")
async def summarize(request: SummarizeRequest):
    """Generate a summary of blog post content"""
    response = await openai.ChatCompletion.acreate(
        model="gpt-3.5-turbo",
        messages=[{
            "role": "user",
            "content": f"Summarize this blog post in 2-3 sentences:\n\n{request.content}"
        }]
    )
    
    summary = response.choices[0].message.content
    return {"summary": summary}

@app.post("/ai/suggest-title")
async def suggest_title(request: TitleRequest):
    """Suggest a catchy title for blog content"""
    response = await openai.ChatCompletion.acreate(
        model="gpt-3.5-turbo",
        messages=[{
            "role": "user",
            "content": f"Generate a catchy blog title (max 60 chars):\n\n{request.content[:500]}"
        }]
    )
    
    title = response.choices[0].message.content.strip('"')
    return {"title": title}
```

**Integration Pattern (Option 1 - Synchronous):**

```typescript
// Blog Service calls AI Service when post is created
async createPost(dto: CreatePostDto) {
  const post = await this.postRepo.save(dto);
  
  // Call AI Service to generate summary
  try {
    const { summary } = await this.httpClient.post(
      'http://ai-service/ai/summarize',
      { content: dto.content }
    );
    await this.postRepo.update(post.id, { summary });
  } catch (error) {
    console.log('AI summarization failed, continuing without it');
  }
  
  return post;
}
```

**Integration Pattern (Option 2 - Async via Kafka - Advanced):**

```python
# AI Service listens to post.created events
from aiokafka import AIOKafkaConsumer

@app.on_event("startup")
async def start_kafka_consumer():
    consumer = AIOKafkaConsumer(
        'post.created',
        bootstrap_servers='localhost:9092',
        group_id='ai-service'
    )
    
    async for message in consumer:
        event = json.loads(message.value)
        await process_post_created(event)

async def process_post_created(event):
    # Generate summary
    summary = await generate_summary(event['content'])
    
    # Call Blog Service to update post
    await httpx.post(
        f'http://blog-service/posts/{event["postId"]}',
        json={'summary': summary},
        headers={'X-Internal-Service': 'ai-service'}
    )
```

---

### 5️⃣ API Gateway (NestJS)

**Owns:** Request routing, authentication, rate limiting

**No database**

**Responsibilities:**

- Validate JWT tokens (issued by Auth Service)
- Route requests to appropriate services
- Rate limiting (Redis-backed)
- Request logging & correlation IDs
- Error normalization

**Implementation:**

```typescript
import { Injectable, CanActivate, ExecutionContext } from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';

@Injectable()
export class JwtAuthGuard implements CanActivate {
  constructor(private jwtService: JwtService) {}

  async canActivate(context: ExecutionContext): Promise<boolean> {
    const request = context.switchToHttp().getRequest();
    const authHeader = request.headers.authorization;
    
    if (!authHeader) {
      throw new UnauthorizedException('No token provided');
    }

    const token = authHeader.split(' ')[1];
    
    try {
      const payload = this.jwtService.verify(token);
      request.user = payload; // { userId, email, name }
      return true;
    } catch (error) {
      throw new UnauthorizedException('Invalid token');
    }
  }
}

// Route configuration in main.ts
app.use('/api/auth', proxy('http://auth-service:3001'));
app.use('/api/posts', jwtGuard, proxy('http://blog-service:3002'));
app.use('/api/notifications', jwtGuard, proxy('http://notification-service:3003'));
app.use('/api/ai', proxy('http://ai-service:8000'));
```

---

## 🔄 Event-Driven Flows (Step by Step)

### Flow 1: User Registration

```
1. User submits registration form
   ↓
2. Frontend → POST /api/auth/register → API Gateway
   ↓
3. Gateway → Auth Service
   ↓
4. Auth Service:
   - Creates user in PostgreSQL/MongoDB
   - Issues JWT
   - Publishes to Kafka: { type: "user.registered", userId: "..." }
   ↓
5. Notification Service (Kafka consumer):
   - Receives user.registered event
   - Creates welcome notification in DB
   - Sends Firebase push: "Welcome to our platform!"
   ↓
6. Response → JWT returned to frontend
```

### Flow 2: Creating a Post

```
1. User writes post and clicks "Publish"
   ↓
2. Frontend → POST /api/posts (with JWT) → Gateway
   ↓
3. Gateway validates JWT, extracts userId
   ↓
4. Blog Service:
   - Calls Auth Service: GET /users/{userId}
   - Gets { name: "Alice", avatar: "url" }
   - Saves post with denormalized data:
     {
       title: "...",
       content: "...",
       authorId: userId,
       authorName: "Alice",     ← denormalized
       authorAvatar: "url"      ← denormalized
     }
   - Calls AI Service: POST /ai/summarize
   - Updates post with summary
   - Publishes to Kafka: { type: "post.created", postId: "...", authorId: "..." }
   - Invalidates Redis cache: del posts:popular
   ↓
5. Response → Post returned to frontend
```

### Flow 3: User Updates Profile

```
1. User changes name from "Alice" to "Alice Smith"
   ↓
2. Frontend → PATCH /api/users/{id} → Gateway → Auth Service
   ↓
3. Auth Service:
   - Updates Users table
   - Publishes to Kafka: {
       type: "user.updated",
       userId: "...",
       name: "Alice Smith",
       avatar: "new-url"
     }
   ↓
4. Blog Service (Kafka consumer):
   - Updates Posts table: SET authorName='Alice Smith' WHERE authorId='...'
   - Updates Comments table: SET authorName='Alice Smith' WHERE authorId='...'
   - Invalidates related caches
   ↓
5. Eventually consistent: All posts now show updated name
   (Takes a few seconds, but that's OK!)
```

### Flow 4: Liking a Post

```
1. User clicks "Like" button
   ↓
2. Frontend → POST /api/posts/{id}/like → Gateway → Blog Service
   ↓
3. Blog Service:
   - Inserts into Likes table
   - Fetches post author info
   - Publishes to Kafka: {
       type: "post.liked",
       postId: "...",
       postAuthorId: "...",
       likerId: userId,
       likerName: "Bob"
     }
   ↓
4. Notification Service (Kafka consumer):
   - Creates notification: "Bob liked your post"
   - Sends Firebase push to post author
   ↓
5. Post author receives real-time notification!
```

---

## 🔧 Technology Stack Summary

| Component | Technology | Why |
|-----------|------------|-----|
| **Auth Service** | ASP.NET Core 8 + Identity | Built-in OAuth, mature enterprise auth |
| **Blog Service** | NestJS (TypeScript) | Structured, DI, great for business logic |
| **Notification** | NestJS (TypeScript) | Built-in Kafka support, consistent architecture |
| **AI Service** | FastAPI (Python) | Python ecosystem for AI/ML, async-first |
| **Gateway** | NestJS | JWT validation, consistent with other services |
| **Message Broker** | Apache Kafka | Industry standard, event sourcing, scalable |
| **Caching** | Redis | Fast, TTL support, distributed caching |
| **DB (Auth)** | PostgreSQL or MongoDB | SQL for relations OR NoSQL for flexibility |
| **DB (Blog)** | PostgreSQL or MongoDB | Your choice based on learning goals |
| **DB (Notifications)** | MongoDB or PostgreSQL | Append-only pattern OR relational |
| **Push Notifications** | Firebase Cloud Messaging | Free, reliable, multi-platform |
| **Containerization** | Docker + Docker Compose | All infra runs locally |

**Database Placement Decision Matrix:**

| Scenario | Auth DB | Blog DB | Notification DB |
|----------|---------|---------|-----------------|
| **Learn NoSQL basics** | PostgreSQL | **MongoDB** | PostgreSQL |
| **NoSQL for flexibility** | **MongoDB** | PostgreSQL | PostgreSQL |
| **NoSQL for events** | PostgreSQL | PostgreSQL | **MongoDB** |
| **All SQL** | PostgreSQL | PostgreSQL | PostgreSQL |

Pick one based on what you want to emphasize in your learning!

---

## 🐳 Development & Deployment

### Local Development Setup

All services and infrastructure run locally using Docker Compose.

**docker-compose.yml:**

```yaml
version: '3.8'

services:
  # Databases
  postgres:
    image: postgres:15
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data
      - ./init-dbs.sql:/docker-entrypoint-initdb.d/init.sql

  mongodb:
    image: mongo:6
    ports:
      - "27017:27017"
    environment:
      MONGO_INITDB_ROOT_USERNAME: admin
      MONGO_INITDB_ROOT_PASSWORD: admin
    volumes:
      - mongodb-data:/data/db

  # Cache
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    command: redis-server --appendonly yes
    volumes:
      - redis-data:/data

  # Messaging
  zookeeper:
    image: confluentinc/cp-zookeeper:latest
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
      ZOOKEEPER_TICK_TIME: 2000
    ports:
      - "2181:2181"

  kafka:
    image: confluentinc/cp-kafka:latest
    depends_on:
      - zookeeper
    ports:
      - "9092:9092"
      - "9093:9093"
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092,PLAINTEXT_INTERNAL://kafka:9093
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: PLAINTEXT:PLAINTEXT,PLAINTEXT_INTERNAL:PLAINTEXT
      KAFKA_INTER_BROKER_LISTENER_NAME: PLAINTEXT_INTERNAL
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_AUTO_CREATE_TOPICS_ENABLE: 'true'

  # Optional: Kafka UI for debugging
  kafka-ui:
    image: provectuslabs/kafka-ui:latest
    depends_on:
      - kafka
    ports:
      - "8080:8080"
    environment:
      KAFKA_CLUSTERS_0_NAME: local
      KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS: kafka:9093

volumes:
  postgres-data:
  mongodb-data:
  redis-data:
```

**init-dbs.sql (Create multiple databases in PostgreSQL):**

```sql
CREATE DATABASE auth_db;
CREATE DATABASE blog_db;
CREATE DATABASE notification_db;
```

**Start infrastructure:**

```bash
# Start all infra
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f kafka

# Stop all
docker-compose down
```

**Create Kafka topics (optional, auto-create is enabled):**

```bash
docker exec -it <kafka-container> kafka-topics --create \
  --bootstrap-server localhost:9092 \
  --topic user.registered \
  --partitions 3 \
  --replication-factor 1

# Repeat for: user.updated, post.created, post.liked
```

**Test Kafka:**

```bash
# Produce a test message
docker exec -it <kafka-container> kafka-console-producer \
  --bootstrap-server localhost:9092 \
  --topic post.created

# Consume messages
docker exec -it <kafka-container> kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic post.created \
  --from-beginning
```

---

### Production Deployment (Single VPS)

**Deployment Strategy:** All services run on a single VPS using Docker Compose. This is cost-effective, simple to maintain, and perfectly suitable for a portfolio/demo project.

**Why Single VPS?**

- ✅ Cost-effective: $0-6/month (vs $50+ for separate services)
- ✅ Still demonstrates microservices: Each service runs in isolated containers
- ✅ Easy to deploy: One `docker-compose up` command
- ✅ Easy to demo: Single IP address
- ✅ Realistic: Even production systems use this for small-scale services

#### Recommended VPS Providers

| Provider | Cost | Specs | Notes |
|----------|------|-------|-------|
| **Oracle Cloud Free Tier** | **FREE** | 2 CPUs, 12GB RAM | Forever free, ARM-based |
| **DigitalOcean** | $6/month | 1 CPU, 1GB RAM | Simple, reliable, good docs |
| **Hetzner Cloud** | €4/month | 1 CPU, 2GB RAM | Cheapest, excellent performance |
| **Linode** | $5/month | 1 CPU, 1GB RAM | Reliable alternative |

**Minimum Requirements:**

- 1 CPU, 2GB RAM (for all services + databases)
- 20GB disk space
- Ubuntu 22.04 or newer

---

#### Production Docker Compose

**docker-compose.prod.yml:**

```yaml
version: '3.8'

services:
  # API Gateway
  gateway:
    build: ./services/gateway
    container_name: gateway
    ports:
      - "3000:3000"
    environment:
      - NODE_ENV=production
      - JWT_SECRET=${JWT_SECRET}
      - AUTH_SERVICE_URL=http://auth-service:3001
      - BLOG_SERVICE_URL=http://blog-service:3002
      - NOTIFICATION_SERVICE_URL=http://notification-service:3003
      - AI_SERVICE_URL=http://ai-service:8000
    depends_on:
      - auth-service
      - blog-service
    restart: unless-stopped
    networks:
      - microservices-network

  # Auth Service
  auth-service:
    build: ./services/auth-service
    container_name: auth-service
    environment:
      - ASPNETCORE_ENVIRONMENT=Production
      - DATABASE_URL=Host=postgres;Port=5432;Database=auth_db;Username=postgres;Password=${DB_PASSWORD}
      - KAFKA_BROKERS=kafka:9093
      - JWT_SECRET=${JWT_SECRET}
      - JWT_EXPIRATION=900
      - GOOGLE_CLIENT_ID=${GOOGLE_CLIENT_ID}
      - GOOGLE_CLIENT_SECRET=${GOOGLE_CLIENT_SECRET}
    depends_on:
      - postgres
      - kafka
    restart: unless-stopped
    networks:
      - microservices-network

  # Blog Service
  blog-service:
    build: ./services/blog-service
    container_name: blog-service
    environment:
      - NODE_ENV=production
      - DATABASE_URL=postgresql://postgres:${DB_PASSWORD}@postgres:5432/blog_db
      - REDIS_URL=redis://redis:6379
      - KAFKA_BROKERS=kafka:9093
      - AUTH_SERVICE_URL=http://auth-service:3001
      - AI_SERVICE_URL=http://ai-service:8000
    depends_on:
      - postgres
      - redis
      - kafka
    restart: unless-stopped
    networks:
      - microservices-network

  # Notification Service
  notification-service:
    build: ./services/notification-service
    container_name: notification-service
    environment:
      - NODE_ENV=production
      - MONGODB_URL=mongodb://mongodb:27017/notifications
      - KAFKA_BROKERS=kafka:9093
      - FIREBASE_PROJECT_ID=${FIREBASE_PROJECT_ID}
      - FIREBASE_PRIVATE_KEY=${FIREBASE_PRIVATE_KEY}
      - FIREBASE_CLIENT_EMAIL=${FIREBASE_CLIENT_EMAIL}
    depends_on:
      - mongodb
      - kafka
    restart: unless-stopped
    networks:
      - microservices-network

  # AI Service
  ai-service:
    build: ./services/ai-service
    container_name: ai-service
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - ENVIRONMENT=production
    restart: unless-stopped
    networks:
      - microservices-network

  # Frontend
  frontend:
    build: 
      context: ./frontend
      args:
        - REACT_APP_API_URL=${FRONTEND_API_URL}
    container_name: frontend
    ports:
      - "3001:80"
    restart: unless-stopped
    networks:
      - microservices-network

  # PostgreSQL
  postgres:
    image: postgres:15-alpine
    container_name: postgres
    environment:
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_USER: postgres
    volumes:
      - postgres-data:/var/lib/postgresql/data
      - ./init-dbs.sql:/docker-entrypoint-initdb.d/init.sql
    restart: unless-stopped
    networks:
      - microservices-network

  # MongoDB
  mongodb:
    image: mongo:6-jammy
    container_name: mongodb
    volumes:
      - mongodb-data:/data/db
    restart: unless-stopped
    networks:
      - microservices-network

  # Redis
  redis:
    image: redis:7-alpine
    container_name: redis
    command: redis-server --appendonly yes
    volumes:
      - redis-data:/data
    restart: unless-stopped
    networks:
      - microservices-network

  # Zookeeper
  zookeeper:
    image: confluentinc/cp-zookeeper:latest
    container_name: zookeeper
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
      ZOOKEEPER_TICK_TIME: 2000
    restart: unless-stopped
    networks:
      - microservices-network

  # Kafka
  kafka:
    image: confluentinc/cp-kafka:latest
    container_name: kafka
    depends_on:
      - zookeeper
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT_INTERNAL://kafka:9093
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: PLAINTEXT_INTERNAL:PLAINTEXT
      KAFKA_INTER_BROKER_LISTENER_NAME: PLAINTEXT_INTERNAL
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_AUTO_CREATE_TOPICS_ENABLE: 'true'
    restart: unless-stopped
    networks:
      - microservices-network

volumes:
  postgres-data:
  mongodb-data:
  redis-data:

networks:
  microservices-network:
    driver: bridge
```

---

#### VPS Setup & Deployment Guide

**Option 1: DigitalOcean (Using Your Existing Droplet)**

```bash
# 1. SSH into your DigitalOcean droplet
ssh root@your-droplet-ip

# 2. Install Docker (if not already installed)
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
exit  # Log out and back in for group changes

# 3. Clone your repository
git clone https://github.com/yourusername/microservices-blog.git
cd microservices-blog

# 4. Create production environment file
cat > .env << EOF
# Database
DB_PASSWORD=$(openssl rand -base64 32)

# JWT
JWT_SECRET=$(openssl rand -base64 64)

# OAuth (optional initially)
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret

# Firebase
FIREBASE_PROJECT_ID=your-firebase-project
FIREBASE_PRIVATE_KEY="your-firebase-key"
FIREBASE_CLIENT_EMAIL=your-firebase-email

# OpenAI
OPENAI_API_KEY=sk-your-openai-key

# Frontend
FRONTEND_API_URL=http://your-domain.com
EOF

# 5. Deploy all services
docker-compose -f docker-compose.prod.yml up -d --build

# 6. Check status
docker-compose -f docker-compose.prod.yml ps

# 7. View logs
docker-compose -f docker-compose.prod.yml logs -f
```

**Option 2: Oracle Cloud Free Tier (FREE Forever)**

```bash
# 1. Create Oracle Cloud account
# Visit: https://cloud.oracle.com/
# Sign up for free tier (no credit card needed after trial)

# 2. Create VM Instance
# - Image: Ubuntu 22.04 (ARM or x86)
# - Shape: VM.Standard.A1.Flex (ARM, 2 CPUs, 12GB RAM - FREE!)
# - Boot volume: 50GB

# 3. Configure firewall in Oracle Console
# VCN → Security Lists → Default Security List
# Add Ingress Rules:
# - Port 22 (SSH)
# - Port 80 (HTTP)
# - Port 443 (HTTPS)
# - Port 3000 (API Gateway - temporary)

# 4. SSH into your instance
ssh ubuntu@your-oracle-vm-ip

# 5. Install Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker ubuntu
exit  # Log out and back in

# 6. SSH back in and clone repo
ssh ubuntu@your-oracle-vm-ip
git clone https://github.com/yourusername/microservices-blog.git
cd microservices-blog

# 7. Create .env file (same as DigitalOcean above)

# 8. Deploy
docker-compose -f docker-compose.prod.yml up -d --build

# 9. Open firewall on Ubuntu (Oracle blocks by default)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 3000/tcp
sudo ufw enable
```

---

#### Nginx Reverse Proxy Setup (Optional but Recommended)

Instead of exposing Gateway on port 3000, use Nginx for cleaner URLs and HTTPS.

```bash
# Install Nginx
sudo apt update
sudo apt install nginx certbot python3-certbot-nginx -y

# Create Nginx configuration
sudo nano /etc/nginx/sites-available/microservices-blog
```

**Nginx config:**

```nginx
server {
    listen 80;
    server_name yourdomain.com www.yourdomain.com;

    # API Gateway
    location /api {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }

    # Frontend
    location / {
        proxy_pass http://localhost:3001;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
```

**Enable and test:**

```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/microservices-blog /etc/nginx/sites-enabled/

# Test configuration
sudo nginx -t

# Restart Nginx
sudo systemctl restart nginx

# Get FREE SSL certificate (HTTPS)
sudo certbot --nginx -d yourdomain.com -d www.yourdomain.com

# Auto-renewal is set up automatically!
```

Now your app is accessible at: `https://yourdomain.com` 🎉

---

#### Domain Setup (Optional)

**Get a cheap domain:**

- **Namecheap**: `.tech`, `.xyz` for $1-3/year
- **Porkbun**: Good prices, free WHOIS privacy
- **Cloudflare Registrar**: At-cost pricing (cheapest)

**Point domain to VPS:**

```
A Record: @ → your-vps-ip
A Record: www → your-vps-ip
```

Wait 5-60 minutes for DNS propagation, then run Certbot (see above).

---

#### CI/CD Pipeline (GitHub Actions → VPS)

Automatically deploy when you push to `main` branch.

**.github/workflows/deploy.yml:**

```yaml
name: Deploy to VPS

on:
  push:
    branches: [main]
  workflow_dispatch:  # Allow manual triggers

jobs:
  deploy:
    runs-on: ubuntu-latest
    
    steps:
      - name: Deploy via SSH
        uses: appleboy/ssh-action@v1.0.0
        with:
          host: ${{ secrets.VPS_HOST }}
          username: ${{ secrets.VPS_USER }}
          key: ${{ secrets.SSH_PRIVATE_KEY }}
          script: |
            cd /home/${{ secrets.VPS_USER }}/microservices-blog
            
            # Pull latest code
            git pull origin main
            
            # Rebuild and restart services
            docker-compose -f docker-compose.prod.yml up -d --build
            
            # Clean up old images
            docker system prune -f
            
            # Show status
            docker-compose -f docker-compose.prod.yml ps
```

**Setup GitHub Secrets:**

1. Generate SSH key on your VPS:

```bash
ssh-keygen -t ed25519 -C "github-actions"
cat ~/.ssh/id_ed25519.pub >> ~/.ssh/authorized_keys
cat ~/.ssh/id_ed25519  # Copy this
```

1. Add to GitHub: Repository → Settings → Secrets → Actions
   - `VPS_HOST`: Your VPS IP address
   - `VPS_USER`: `root` or `ubuntu`
   - `SSH_PRIVATE_KEY`: The private key from above

2. Push to `main` → Auto-deploys! 🚀

---

#### Monitoring & Maintenance

**View logs:**

```bash
# All services
docker-compose -f docker-compose.prod.yml logs -f

# Specific service
docker-compose -f docker-compose.prod.yml logs -f blog-service

# Last 100 lines
docker-compose -f docker-compose.prod.yml logs --tail=100
```

**Check resource usage:**

```bash
# Container stats
docker stats

# Disk usage
docker system df

# Clean up
docker system prune -a  # Remove unused images/containers
```

**Update services:**

```bash
cd microservices-blog
git pull origin main
docker-compose -f docker-compose.prod.yml up -d --build
```

**Backup databases:**

```bash
# PostgreSQL backup
docker exec postgres pg_dumpall -U postgres > backup.sql

# MongoDB backup
docker exec mongodb mongodump --out /tmp/backup
docker cp mongodb:/tmp/backup ./mongodb-backup

# Upload to cloud storage (optional)
# aws s3 cp backup.sql s3://your-bucket/
```

---

#### Troubleshooting

**Services won't start:**

```bash
# Check logs
docker-compose -f docker-compose.prod.yml logs

# Check specific service
docker logs <container-name>

# Restart all services
docker-compose -f docker-compose.prod.yml restart

# Rebuild from scratch
docker-compose -f docker-compose.prod.yml down -v
docker-compose -f docker-compose.prod.yml up -d --build
```

**Out of memory:**

```bash
# Check memory usage
free -h
docker stats

# Add swap (if needed)
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

**Port conflicts:**

```bash
# Check what's using a port
sudo lsof -i :3000

# Kill process
sudo kill -9 <PID>
```

---

#### Cost Breakdown

**Option 1: Oracle Free Tier**

- VPS: **$0** (forever free)
- Domain: $1-3/year
- **Total: ~$3/year**

**Option 2: DigitalOcean**

- VPS: $6/month = $72/year
- Domain: $1-3/year
- **Total: ~$75/year**

**Option 3: Hetzner (Cheapest Paid)**

- VPS: €4/month = ~$50/year
- Domain: $1-3/year
- **Total: ~$53/year**

**Recommendation:** Start with Oracle Free Tier or use your existing DigitalOcean droplet.

---

**Test Kafka:**

```bash
# Produce a test message
docker exec -it <kafka-container> kafka-console-producer \
  --bootstrap-server localhost:9092 \
  --topic post.created

# Consume messages
docker exec -it <kafka-container> kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic post.created \
  --from-beginning
```

---

## 📂 Project Structure (Monorepo)

**This project uses a MONOREPO approach** — all services in one Git repository for simplicity, easier collaboration, and portfolio presentation.

```
microservices-blog/                    # Single Git repository
├── .git/
├── .gitignore
├── README.md
├── ARCHITECTURE.md
├── .env.example
│
├── docker-compose.yml                 # Local development
├── docker-compose.prod.yml            # Production deployment
├── init-dbs.sql
│
├── .github/
│   └── workflows/
│       └── deploy.yml                 # CI/CD to VPS
│
├── services/
│   ├── gateway/                   # NestJS
│   │   ├── src/
│   │   │   ├── auth/
│   │   │   │   └── jwt-auth.guard.ts
│   │   │   ├── proxy/
│   │   │   └── main.ts
│   │   ├── Dockerfile
│   │   └── package.json
│   │
│   ├── auth-service/              # ASP.NET Core
│   │   ├── Controllers/
│   │   │   ├── AuthController.cs
│   │   │   └── UsersController.cs
│   │   ├── Services/
│   │   │   ├── AuthService.cs
│   │   │   └── KafkaProducerService.cs
│   │   ├── Models/
│   │   │   └── User.cs
│   │   ├── Data/
│   │   │   └── ApplicationDbContext.cs
│   │   ├── Dockerfile
│   │   ├── Program.cs
│   │   └── appsettings.json
│   │
│   ├── blog-service/              # NestJS
│   │   ├── src/
│   │   │   ├── posts/
│   │   │   │   ├── posts.controller.ts
│   │   │   │   ├── posts.service.ts
│   │   │   │   └── entities/
│   │   │   ├── comments/
│   │   │   ├── kafka/
│   │   │   │   ├── kafka.module.ts
│   │   │   │   ├── kafka-producer.service.ts
│   │   │   │   └── kafka-consumer.service.ts
│   │   │   ├── cache/
│   │   │   │   └── redis.service.ts
│   │   │   ├── http/
│   │   │   │   └── auth-client.service.ts
│   │   │   └── main.ts
│   │   ├── Dockerfile
│   │   └── package.json
│   │
│   ├── notification-service/      # NestJS
│   │   ├── src/
│   │   │   ├── notifications/
│   │   │   │   ├── notifications.controller.ts
│   │   │   │   ├── notifications.service.ts
│   │   │   │   └── schemas/
│   │   │   ├── events/
│   │   │   │   └── notification-event.controller.ts
│   │   │   ├── firebase/
│   │   │   │   └── firebase.service.ts
│   │   │   ├── kafka/
│   │   │   └── main.ts
│   │   ├── Dockerfile
│   │   └── package.json
│   │
│   ├── ai-service/                # FastAPI
│   │   ├── app/
│   │   │   ├── main.py
│   │   │   ├── routers/
│   │   │   │   └── ai_router.py
│   │   │   ├── services/
│   │   │   │   └── llm_service.py
│   │   │   └── models/
│   │   │       └── schemas.py
│   │   ├── Dockerfile
│   │   └── requirements.txt
│   │
│   └── shared/                    # Shared types (optional)
│       ├── events/
│       │   └── event.types.ts
│       └── dtos/
│
├── frontend/                      # React/Next.js
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   ├── services/
│   │   │   └── api.service.ts
│   │   └── App.tsx
│   ├── public/
│   └── package.json
│
└── docs/
    ├── architecture.md
    ├── data-flows.md
    └── api-documentation.md
```

---

## 🎯 Implementation Roadmap

### Phase 1: Foundation (Week 1)

- [ ] Set up Docker Compose (Postgres, Mongo, Redis, Kafka)
- [ ] Create Kafka topics
- [ ] Scaffold all service projects (NestJS, ASP.NET, FastAPI)
- [ ] Set up shared event type definitions

### Phase 2: Auth Service (Week 1-2)

- [ ] User registration & login (local auth)
- [ ] JWT issuance with claims (userId, email, name)
- [ ] Google OAuth integration (optional)
- [ ] Kafka producer: publish `user.registered`
- [ ] User profile endpoint: `GET /users/{id}`
- [ ] Update profile: `PATCH /users/{id}` → publish `user.updated`

### Phase 3: Blog Service Core (Week 2)

- [ ] Post CRUD endpoints
- [ ] HTTP client to call Auth Service for user data
- [ ] Denormalize user data in Posts table
- [ ] Kafka producer: publish `post.created`
- [ ] Redis caching for `GET /posts` endpoint

### Phase 4: Event-Driven Updates (Week 2-3)

- [ ] Blog Service: Kafka consumer for `user.updated`
- [ ] Update denormalized data when user profile changes
- [ ] Test eventual consistency (change name, verify posts update)

### Phase 5: Notification Service (Week 3)

- [ ] Set up NestJS microservices with Kafka
- [ ] Kafka consumer for `user.registered`, `post.liked`, `post.created`
- [ ] Store notifications in MongoDB/PostgreSQL
- [ ] Firebase setup & device token storage
- [ ] Send push notifications via Firebase
- [ ] Notification endpoints: get, mark read, unread count

### Phase 6: API Gateway (Week 3)

- [ ] NestJS setup with JWT validation middleware
- [ ] Route proxying to all services
- [ ] Rate limiting (Redis-backed)
- [ ] Error handling & logging
- [ ] Correlation ID propagation

### Phase 7: AI Service (Week 4)

- [ ] FastAPI setup
- [ ] OpenAI API integration (or local LLM)
- [ ] `/ai/summarize` endpoint
- [ ] `/ai/suggest-title` endpoint
- [ ] Blog Service integration (call AI on post creation)

### Phase 8: Comments & Likes (Week 4)

- [ ] Comments CRUD in Blog Service
- [ ] Likes functionality
- [ ] Publish `post.liked` to Kafka
- [ ] Notification Service: handle `post.liked` event

### Phase 9: Frontend (Week 4)

- [ ] Login/Register pages (with Google OAuth button)
- [ ] Post creation form (with AI title suggestion)
- [ ] Feed view (shows cached posts)
- [ ] Post detail page (with comments)
- [ ] Notification center (list + unread count)
- [ ] Real-time notification badge (poll endpoint)

### Phase 10: Polish (Week 5)

- [ ] Add proper error handling
- [ ] Add loading states in frontend
- [ ] Performance testing (stress test Kafka consumers)
- [ ] Documentation (architecture diagram, API docs)
- [ ] Demo script (5-10 min walkthrough)

---

## 🧪 Testing Strategy

**Unit Tests:**

- Service logic (post creation, user registration)
- Kafka event serialization/deserialization
- Redis caching logic

**Integration Tests:**

- Auth flow end-to-end (register → login → JWT validation)
- Post creation with denormalized data
- Event publishing and consumption
- Firebase push notification mocking

**End-to-End Tests:**

- Complete user journey: register → create post → receive notification
- Profile update → eventual consistency verification

**Manual Testing Checklist:**

- [ ] Register user → welcome notification appears
- [ ] Change user name → all posts update (wait a few seconds)
- [ ] Create post → AI summary generated
- [ ] Like post → post author receives notification
- [ ] Redis cache hit rate (check logs)

**Tools:**

- xUnit (.NET), Jest (TypeScript), pytest (Python)
- Testcontainers for Kafka/DB in tests
- Postman/Insomnia for API testing
- Kafka UI for event debugging

---

## 📊 Monitoring & Observability

**Structured Logging (All Services):**

```typescript
logger.info('Post created', {
  service: 'blog-service',
  postId: post.id,
  authorId: userId,
  timestamp: Date.now(),
  correlationId: req.headers['x-correlation-id']
});
```

**Health Checks:**
Each service exposes `/health`:

```json
{
  "status": "healthy",
  "database": "connected",
  "kafka": "connected",
  "redis": "connected",
  "uptime": 3600
}
```

**Correlation IDs:**
Gateway generates correlation ID and propagates it:

```typescript
const correlationId = req.headers['x-correlation-id'] || uuidv4();
req.headers['x-correlation-id'] = correlationId;
```

**Metrics to Track (Optional):**

- Request latency per endpoint
- Kafka consumer lag
- Cache hit/miss rate
- Event processing time
- Database query performance

**Tools (If Time Permits):**

- Prometheus for metrics collection
- Grafana for dashboards
- Jaeger for distributed tracing

---

## 🔐 Security Considerations

**Authentication & Authorization:**

- JWT with short expiration (15min) + refresh tokens
- Password hashing with bcrypt (cost factor 12+)
- OAuth 2.0 for third-party providers
- HTTPS in production (Let's Encrypt)

**API Security:**

- Rate limiting per IP/user (Redis-backed)
- Input validation on all endpoints
- SQL injection prevention (ORMs with parameterized queries)
- XSS prevention (sanitize user-generated content)
- CORS configuration (whitelist frontend domain)

**Service-to-Service Communication:**

- Internal services trust Gateway's user header (after JWT validation)
- Optional: mTLS for service-to-service encryption
- Optional: API keys for internal service calls

**Secrets Management:**

- Environment variables for sensitive config
- `.env` files excluded from git
- Firebase service account key as env var (not in code)
- Database passwords in env vars

**Frontend Security:**

- Store JWT in memory (not localStorage)
- HttpOnly cookies for refresh tokens
- CSP headers to prevent XSS
- Rate limit login attempts (prevent brute force)

---

## 🎓 Key Learning Outcomes

After completing this project, you'll deeply understand:

✅ **Service Boundaries & Ownership**

- How to decompose a monolith into autonomous services
- When to split vs when to keep together
- Database-per-service pattern

✅ **Data Management in Microservices**

- Strategic data denormalization (duplicate vs reference)
- Handling eventual consistency (it's OK for data to be stale for seconds!)
- Trade-offs between consistency and availability

✅ **Event-Driven Architecture**

- Kafka topics, partitions, consumer groups
- Event schema design (JSON payloads)
- Idempotent event processing (using eventId)
- At-least-once delivery semantics

✅ **Polyglot Persistence**

- PostgreSQL for relational data (auth, structured posts)
- MongoDB for flexible/append-only data (notifications)
- Redis for caching (TTL-based invalidation)

✅ **API Gateway Pattern**

- Centralized auth validation (JWT verification)
- Request routing and proxying
- Rate limiting and logging
- Error normalization

✅ **Caching Strategies**

- Cache-aside pattern
- TTL-based invalidation
- Write-through invalidation
- When to cache vs when not to

✅ **Third-Party Integration**

- Firebase Cloud Messaging for push notifications
- OAuth providers (Google, GitHub)
- External AI APIs (OpenAI)

✅ **Distributed Systems Concepts**

- Async communication patterns
- Correlation IDs for request tracing
- Health checks and observability
- Handling partial failures

---

## 📝 Interview Talking Points

**"Walk me through your microservices architecture"**

> "I built an event-driven blog platform with 5 microservices communicating via Kafka and REST. The architecture demonstrates service autonomy, polyglot persistence, and eventual consistency.
>
> Each service owns its domain: Auth handles users and JWT issuance, Blog manages posts with Redis caching, Notifications consumes Kafka events to send real-time alerts via Firebase, AI provides summarization, and Gateway centralizes auth validation.
>
> The key design decision was data denormalization. Since services can't share databases, I duplicate user data (name, avatar) in the posts table. When users update their profile, Auth publishes a `user.updated` event to Kafka, and Blog's consumer updates all related posts. This gives us eventual consistency—data might be stale for a few seconds, but services remain autonomous and resilient."

**"How do you handle data consistency across services?"**

> "I use eventual consistency with Kafka events. For example, when a user changes their name:
>
> 1. Auth Service updates the Users table
> 2. Publishes `user.updated` event to Kafka
> 3. Blog Service's consumer receives the event
> 4. Updates the denormalized `authorName` in all posts
>
> For a few seconds, some posts might show the old name, but that's acceptable for this use case. The trade-off is service independence—Blog can display posts even if Auth is down, because it has all the data it needs locally.
>
> For critical consistency (like financial transactions), I'd use the Saga pattern with compensating transactions, but for user profiles, eventual consistency is sufficient."

**"Why did you choose these specific databases?"**

> "I chose databases based on access patterns:
>
> - **PostgreSQL for Auth/Blog**: Relational data with ACID guarantees (user relationships, post comments)
> - **MongoDB for Notifications**: Append-only pattern with flexible schema—different notification types have different metadata
> - **Redis for caching**: Sub-millisecond reads for popular posts, TTL-based expiration
>
> This demonstrates polyglot persistence—choosing the right tool for the job rather than forcing all data into one database type."

**"How does your caching strategy work?"**

> "I use Redis with a cache-aside pattern and TTL-based invalidation:
>
> - `posts:popular` caches top 10 posts (TTL: 5min)
> - On new post creation, I invalidate this key
> - This reduced database load by 70% for the feed endpoint in testing
>
> The key insight is balancing freshness vs performance. For a blog, showing 5-minute-old data is fine, but for stock prices, you'd need a different strategy."

**"What happens if Kafka goes down?"**

> "Services degrade gracefully:
>
> - **Auth/Blog**: Continue working normally (they publish events but don't depend on consumers)
> - **Notifications**: Stop consuming events, but existing notifications still work
>
> When Kafka recovers, consumers resume from their last offset (consumer group tracking), processing missed events. This is at-least-once delivery—notifications might duplicate if a consumer crashes mid-processing, which is why I'd add idempotency checks (eventId deduplication) in production."

---

## 🚀 Extensions & Next Steps

**If you have more time, add:**

**Level 1 (Easy):**

- [ ] User follows/followers feature
- [ ] Post categories & filtering
- [ ] Pagination for all list endpoints
- [ ] Dark mode in frontend

**Level 2 (Medium):**

- [ ] Full-text search (Elasticsearch)
- [ ] Image upload (S3 + CDN)
- [ ] WebSocket for real-time notifications
- [ ] Email notifications (SendGrid)
- [ ] Rate limiting per user tier (free vs premium)

**Level 3 (Advanced):**

- [ ] Distributed tracing (Jaeger/Zipkin)
- [ ] Circuit breakers (Polly in .NET, Hystrix pattern)
- [ ] API versioning (/v1/posts, /v2/posts)
- [ ] GraphQL BFF layer (Apollo Federation)
- [ ] Saga pattern for complex workflows
- [ ] CQRS for read/write separation
- [ ] Event sourcing for audit log

---

## 📚 Resources & References

**Microservices Patterns:**

- [Microservices.io](https://microservices.io/) — Pattern catalog
- "Building Microservices" by Sam Newman
- "Microservices Patterns" by Chris Richardson

**Kafka:**

- [Kafka in 100 Seconds](https://www.youtube.com/watch?v=uvb00oaa3k8)
- [Confluent Kafka Tutorials](https://developer.confluent.io/)
- [KafkaJS Documentation](https://kafka.js.org/)

**Event-Driven Architecture:**

- [Event Sourcing - Martin Fowler](https://martinfowler.com/eaaDev/EventSourcing.html)
- [Saga Pattern Explained](https://microservices.io/patterns/data/saga.html)

**NestJS Microservices:**

- [NestJS Microservices](https://docs.nestjs.com/microservices/basics)
- [NestJS Kafka Transport](https://docs.nestjs.com/microservices/kafka)

**Firebase:**

- [Firebase Cloud Messaging](https://firebase.google.com/docs/cloud-messaging)
- [FCM HTTP v1 API](https://firebase.google.com/docs/reference/fcm/rest/v1/projects.messages)

**ASP.NET Identity:**

- [ASP.NET Core Identity](https://learn.microsoft.com/en-us/aspnet/core/security/authentication/identity)
- [External OAuth Providers](https://learn.microsoft.com/en-us/aspnet/core/security/authentication/social/)

---

## 📋 Implementation Checklist

**Infrastructure:**

- [ ] Docker Compose with all services running
- [ ] Kafka topics created
- [ ] Databases initialized with schemas
- [ ] Redis accessible from all services

**Auth Service (ASP.NET):**

- [ ] User registration with password hashing
- [ ] JWT issuance with claims
- [ ] Google OAuth integration
- [ ] User profile endpoints
- [ ] Kafka producer for user events
- [ ] Health check endpoint

**Blog Service (NestJS):**

- [ ] Post CRUD endpoints
- [ ] HTTP client to Auth Service
- [ ] Data denormalization (author info)
- [ ] Redis caching implemented
- [ ] Kafka producer for post events
- [ ] Kafka consumer for user updates
- [ ] Health check endpoint

**Notification Service (NestJS):**

- [ ] Kafka consumer setup
- [ ] Database integration (MongoDB/PostgreSQL)
- [ ] Firebase initialization
- [ ] Event handlers for all event types
- [ ] Notification endpoints (get, read, count)
- [ ] Push notification sending
- [ ] Health check endpoint

**AI Service (FastAPI):**

- [ ] OpenAI API integration
- [ ] Summarize endpoint
- [ ] Title suggestion endpoint
- [ ] Error handling for API failures
- [ ] Health check endpoint

**API Gateway (NestJS):**

- [ ] JWT validation middleware
- [ ] Route proxying to all services
- [ ] Rate limiting (Redis)
- [ ] Correlation ID generation
- [ ] Error normalization
- [ ] Health check aggregation

**Frontend (React/Next.js):**

- [ ] Login/Register pages
- [ ] OAuth flow integration
- [ ] Post creation with AI suggestions
- [ ] Feed with caching
- [ ] Post detail with comments
- [ ] Notification center
- [ ] Real-time notification badge

**Testing:**

- [ ] Unit tests for core logic
- [ ] Integration tests for event flows
- [ ] E2E test for full user journey
- [ ] Performance testing (Kafka throughput)

**Documentation:**

- [ ] Architecture diagram (draw.io or Mermaid)
- [ ] API documentation (Swagger/OpenAPI)
- [ ] Event schema documentation
- [ ] README with setup instructions
- [ ] Demo script (5-10 min walkthrough)

---

## ✅ Final Summary

This project is a **complete, production-inspired microservices platform** that demonstrates:

1. **Service Decomposition** — 5 autonomous services with clear boundaries
2. **Event-Driven Communication** — Kafka for async, decoupled messaging
3. **Data Denormalization** — Strategic duplication for service independence
4. **Eventual Consistency** — Handling async data propagation
5. **Polyglot Stack** — ASP.NET, NestJS, FastAPI (3 frameworks, 2 languages)
6. **Polyglot Persistence** — PostgreSQL + MongoDB + Redis (right tool for the job)
7. **API Gateway Pattern** — Centralized auth and routing
8. **Third-Party Integration** — Firebase, OAuth providers
9. **Caching Strategy** — Redis for performance optimization
10. **Production Patterns** — Health checks, logging, error handling

**What makes this impressive:**

- ✅ Real-world patterns (not toy examples)
- ✅ Demonstrates trade-offs (consistency vs availability)
- ✅ Shows event-driven thinking
- ✅ Touches on distributed systems challenges
- ✅ Portfolio-ready and interview-ready

**Time Investment:** 4-5 weeks (30-40 hours total)

**Result:** A complete, working system that demonstrates deep understanding of microservices architecture—something you can confidently discuss in interviews and showcase to employers.

---

**Ready to start building?** Follow the roadmap, tackle one phase at a time, and don't hesitate to simplify if you're short on time. The key is **finishing a working system** with good documentation.

Good luck! 🚀

---

**Quick Start Command:**

```bash
# Clone your repo
git clone <your-repo-url>
cd microservices-blog

# Start infrastructure
docker-compose up -d

# Verify everything is running
docker-compose ps

# Check Kafka topics (should auto-create)
docker exec -it <kafka-container> kafka-topics --list --bootstrap-server localhost:9092

# Start building services one by one!
```
