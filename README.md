# Pomocore Data Pipeline Architecture

## 📋 Overview
Pomocore 앱의 사용자 활동 데이터를 실시간으로 분류, 저장, 그리고 리더보드를 업데이트하는 고성능 데이터 파이프라인입니다.

**핵심 최적화**: go 를 활용한 동시성 제어 및 비동기 처리로 90% 이상의 처리 속도 개선 (20s -> 1.5s)

## 🏗️ Architecture Pattern
**헥사고날 아키텍처 (Clean Architecture)**를 따라 구현되었습니다:
- `domains/`: 비즈니스 로직과 도메인 모델
- `infrastructure/`: 외부 시스템과의 연결 (MongoDB, Redis)
- `cmd/`: 애플리케이션 엔트리포인트

## 🔄 Data Flow

```
[Redis Stream] → [PomodoroPatternConsumer] → [MongoDB + Redis ZSet]
     ↓                    ↓                         ↓
사용자 활동 데이터    AI 분류 + 배치 처리      영구 저장 + 리더보드
```

### 상세 처리 과정:

1. **Message Ingestion**: Redis Stream에서 최대 50개 메시지를 배치로 읽음
2. **AI Classification**: 앱/제목/URL을 기반으로 카테고리 분류
3. **Batch Database Operations**: 
   - N개 중복 키 → 1번 배치 조회로 최적화
   - 새 데이터만 배치 저장
4. **Leaderboard Aggregation**: 같은 사용자+카테고리 조합을 집계
5. **Direct Redis Update**: Stream 없이 ZSet 직접 업데이트

## 📂 Key Components

### 1. Domain Layer (`domains/`)
Domain Layer는 Spring 메인 서버의 도메인과 동일하게 싱크를 유지합니다.
#### Message Types
- **PomodoroPatternClassifyMessage**: 포모도로 세션 데이터
  ```go
  type PomodoroPatternClassifyMessage struct {
      UserID            string
      App               string  // 사용 앱
      Title             string  // 창 제목
      URL               string  // 웹사이트 URL
      Duration          float64 // 사용 시간(초)
      Session           int     // 세션 번호
      SessionDate       time.Time
      // ... 기타 필드
  }
  ```

#### Domain Models

**Pomodoro Domain**:
- `CategorizedData`: 앱/URL/제목의 분류 결과 저장
- `PomodoroUsageLog`: 사용자별 세션 로그

**Leaderboard Domain**:
- `LeaderboardEntry`: 리더보드 업데이트 단위
- `LeaderboardResult`: 순위 조회 결과

**CategoryPattern Domain**:
- `CategoryPattern`: 카테고리별 패턴 정의 (앱, 도메인 패턴)

**PatternClassifier Domain**:
- `PatternClassifier`: 핵심 분류 엔진
- `LLMClient`: OpenAI API 연동 클라이언트
- **자료구조**:
  - `Trie`: 앱 패턴 매칭용
  - `AhoCorasick`: URL 도메인 매칭용

### 2. Infrastructure Layer (`infrastructure/`)

#### MongoDB Adapters
- **CategorizedDataRepositoryAdapter**: 
  - 기본 CRUD + **배치 조회/저장** 최적화
  - `FindManyByAppUrlTitleBatch()`: N+1 문제 해결의 핵심
- **PomodoroUsageLogRepositoryAdapter**: 
  - 사용자 세션 로그 관리
  - `SaveBatch()`: 대량 로그 일괄 저장
- **CategoryPatternRepositoryAdapter**:
  - 카테고리 패턴 관리
  - 시작 시 패턴 로드 및 카테고리 맵핑

#### Redis Adapters  
- **LeaderboardCacheAdapter**: Redis ZSet 조작
  - `BatchIncreaseScore()`: 여러 사용자 점수를 한 번에 업데이트
  - 일별/카테고리별/전체 리더보드 지원
- **PatternClassifierAdapter**:
  - PatternClassifier 도메인을 Port 인터페이스로 래핑
  - AI 분류와 패턴 기반 분류 통합

### 3. Consumer (`infrastructure/redis/consumer/`)

#### PomodoroPatternConsumer
**핵심 최적화가 집중된 컴포넌트**

**주요 메서드**:
- `consume()`: Redis Stream에서 배치 단위로 메시지 읽기
- `processBatchMessages()`: 배치 단위로 메시지를 처리. 이 떄, 

**배치 처리 최적화**:
```go
// AS-IS: N+1 문제
for msg := range messages {
    db.FindByKey(msg.key)  // N번 호출
    db.Save(processedData) // N번 호출
}

// TO-BE: 배치 최적화
uniqueKeys := deduplicateKeys(messages)
existingData := db.FindManyByKeyBatch(uniqueKeys)  // 1번 호출
newData := processNewData(messages, existingData)
db.SaveBatch(newData)  // 1번 호출
```

## 🚀 Performance Optimizations

### 1. 배치 처리 최적화
- **Database 호출 감소**: N+1 문제 해결로 98% DB 호출 감소
- **메시지 배치 처리**: 50개씩 묶어서 처리 (개별 처리 대비 50배 성능 향상)
- **중복 키 제거**: 배치 내 중복 키를 사전에 제거하여 불필요한 DB 조회 방지

### 2. 동시성 제어
- **Worker Pool 패턴**: 10개 워커로 CPU 집약적 분류 작업 병렬 처리
- **Go Channel 활용**: 비동기 메시지 전달로 블로킹 최소화
- **Context 기반 취소**: Graceful shutdown 지원

### 3. 캐싱 전략
- **분류 결과 캐싱**: sync.Map을 활용한 스레드 안전 캐싱
- **카테고리 ID 맵핑**: 시작 시 카테고리-ID 맵핑 캐싱으로 조회 최적화

### 4. 데이터 구조 최적화
- **Trie 구조**: 앱 패턴 매칭에 Trie 자료구조 사용
- **Aho-Corasick 알고리즘**: URL 패턴 매칭에 효율적인 문자열 검색 알고리즘 적용

### 5. 리더보드 직접 업데이트
- **Stream 제거**: 중간 Stream 없이 Redis ZSet 직접 업데이트
- **집계 처리**: 같은 사용자+카테고리 조합을 메모리에서 집계 후 한 번에 업데이트 

## 🔧 Configuration

### Stream Configuration (`infrastructure/redis/config/`)
```go
PomodoroPatternMatch = StreamInfo{
    StreamKey: "pomodoro_pattern_match_stream",
    Group:     "pomodoro_pattern_match_group", 
    Consumer:  "pomodoro_pattern_match_consumer",
}
```

### Batch Sizes
- **Stream Read**: 50개 메시지/배치
- **Worker Pool**: 10개 워커 (CPU 집약적 분류 작업용)
- **Database Batch**: 제한 없음 (메모리 허용 범위)
- **Stream Block Time**: 2초 (메시지 대기 시간)

## 🏃‍♂️ Running the Pipeline

### Dependencies
- **MongoDB**: 영구 데이터 저장
- **Redis**: Stream + ZSet (리더보드)
- **Go 1.24+**

### Environment Variables
```bash
MONGO_URI=${your_mongodb_uri}
MONGO_DATABASE=${your_mongodb_database}
REDIS_ADDR=${your_redis_addr}
REDIS_PASSWORD=${your_redis_password}  # Optional
OPENAI_API_KEY=${your_api_key}
```

### Docker Deployment
```bash
# Build image
docker build -t pomocore-data-pipeline .

# Run container
docker run -d \
  --name pomocore-consumer \
  --env-file .env \
  pomocore-data-pipeline
```

### Startup
```bash
go run cmd/stream-consumer/main.go
```

### Main Components Initialization
```go
// Pattern Classifier 초기화
patternClassifier := core.NewPatternClassifier()
initializePatternClassifier(patternClassifier, db) // MongoDB에서 패턴 로드

// MongoDB Adapters
categorizedDataRepo := mongoAdapter.NewCategorizedDataRepositoryPort(db)
pomodoroUsageLogRepo := mongoAdapter.NewPomodoroUsageLogRepositoryPort(db)
categoryPatternRepo := mongoAdapter.NewCategoryPatternRepositoryPort(db)

// Redis Adapters
leaderboardCache := redisAdapter.NewLeaderboardCachePort(redisClient)
classifierAdapter := redisAdapter.NewPatternClassifierAdapter(patternClassifier)

// Services
categoryPatternUseCase := categoryPatternService.NewCategoryPatternService(categoryPatternRepo)

// Consumer는 모든 의존성을 받음
pomodoroConsumer := consumer.NewPomodoroPatternConsumer(
    redisClient,
    classifierAdapter,
    categorizedDataRepo,
    pomodoroUsageLogRepo,
    categoryPatternUseCase,
    leaderboardCache,  // 직접 주입으로 Stream 우회
)
```

## 📊 Monitoring & Observability

### Logging
- 배치 처리 결과: `"Successfully processed batch of N messages"`
- 리더보드 업데이트: `"Successfully updated leaderboard with N aggregated entries"`
- DB 저장 결과: 개별 컴포넌트별 상세 로깅

### Error Handling
- **MongoDB 실패**: 메시지 acknowledge하지 않음 → 재처리
- **Redis 실패**: 로깅 후 continue (핵심 기능 아님)
- **분류 실패**: 해당 메시지만 skip

## 🎯 Key Design Decisions

### 1. 왜 배치 처리인가?
- **처리량**: 개별 처리 대비 50배 향상
- **자원 효율**: DB 커넥션, 네트워크 비용 절약
- **일관성**: 트랜잭션 단위 축소

### 2. 왜 헥사고날 아키텍처인가?
- **테스트 용이성**: Port/Adapter 패턴으로 Mock 주입 간단
- **기술 독립성**: MongoDB → PostgreSQL 교체 시 Adapter만 변경
- **비즈니스 로직 보호**: Infrastructure 변경이 Domain에 영향 없음

### 3. 왜 Trie와 Aho-Corasick인가?
- **Trie**: 앱 이름의 prefix 매칭에 최적화 (O(m) 검색 시간)
- **Aho-Corasick**: 다중 패턴 문자열 검색에 최적화 (O(n+m+z) 복잡도)
- **메모리 효율**: 패턴이 많아져도 검색 속도 일정

### 4. 왜 Go 언어인가?
- **동시성**: Goroutine과 Channel로 효율적인 병렬 처리
- **성능**: 컴파일 언어로 Python 대비 10-20배 빠른 실행
- **메모리 효율**: GC가 있으면서도 메모리 사용량 최소화

## 🔄 Evolution History

1. **v1**: 개별 메시지 처리 (N+1 문제)
2. **v2**: 배치 DB 처리 추가 (98% DB 호출 감소)  
3. **v3**: 리더보드 집계 최적화 (90% 네트워크 감소)
4. **v4**: Stream 제거로 아키텍처 간소화
5. **v5**: 패턴 기반 분류 + LLM 하이브리드 (현재)
   - Trie/Aho-Corasick 도입
   - CategoryPattern 도메인 추가
   - Docker 컨테이너화

## 🚨 Known Limitations & Future Work

### Current Limitations
- **메모리**: 배치 크기가 클 경우 메모리 사용량 증가
- **지연**: 배치 처리로 인한 약간의 지연 (2초 block time)
- **복잡성**: 배치 로직이 개별 처리 대비 복잡

### Future Improvements
- [ ] 메트릭 수집 (Prometheus)
- [ ] 동적 배치 크기 조절
- [ ] Circuit Breaker 패턴 적용
- [ ] 분산 처리 (여러 Consumer 인스턴스)
- [ ] 패턴 학습 자동화 (ML 기반)
- [ ] Redis Cluster 지원
- [ ] Kubernetes 배포 매니페스트

---

## 📚 Code Reading Guide

### 시작점
1. `cmd/stream-consumer/main.go` - 전체 의존성 구조 파악
2. `infrastructure/redis/consumer/pomodoro_pattern_consumer.go` - 핵심 비즈니스 로직
3. `domains/` 디렉토리 - 도메인 모델과 인터페이스

### 핵심 파일들
- **배치 최적화**: `infrastructure/redis/consumer/pomodoro_pattern_consumer.go:processBatchMessages()`
- **리더보드 집계**: `infrastructure/redis/adapter/leaderboard_cache_adapter.go:BatchIncreaseScore()`
- **패턴 분류 엔진**: `domains/patternClassifier/domain/core/pattern_classifier.go`
- **Trie 구조**: `domains/patternClassifier/domain/structure/trie.go`
- **Aho-Corasick**: `domains/patternClassifier/domain/structure/aho_corasick.go`

이 문서를 통해 코드의 전체적인 구조와 최적화 포인트를 이해할 수 있으며, 향후 유지보수나 기능 확장 시 참고자료로 활용할 수 있습니다.