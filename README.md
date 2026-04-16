# Project Superlink

A system consisting of a Go backend, an Android client, and Docker orchestration.

## Structure

- `backend/`: Go server implementation.
- `android/`: Kotlin Android application.
- `docker/`: Docker configuration files for bootstrapping the system.
- `docs/`: Architecture and documentation.

## Getting Started

### Backend
```bash
cd backend
go run main.go
```

### Docker
```bash
cd docker
docker-compose up --build
```

### Android
Requires Android Studio or Gradle to build.
