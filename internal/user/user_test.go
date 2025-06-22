package user

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/antonminaichev/gophermart-loyalty/internal/types/user"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

type stubUserRepo struct {
	users       map[string]*user.User
	errOnCreate error
	errOnFind   error
}

func newStubUserRepo() *stubUserRepo {
	return &stubUserRepo{users: make(map[string]*user.User)}
}

func (r *stubUserRepo) Create(ctx context.Context, u *user.User) error {
	if r.errOnCreate != nil {
		return r.errOnCreate
	}
	if _, exists := r.users[u.Login]; exists {
		return ErrUserExists
	}
	u.ID = int64(len(r.users) + 1)
	r.users[u.Login] = u
	return nil
}

func (r *stubUserRepo) FindByLogin(ctx context.Context, login string) (*user.User, error) {
	if r.errOnFind != nil {
		return nil, r.errOnFind
	}
	u, ok := r.users[login]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

func TestServiceRegister(t *testing.T) {
	repo := newStubUserRepo()
	svc := NewService(repo, []byte("secret"), time.Hour)

	t.Run("successful registration", func(t *testing.T) {
		u, err := svc.Register(context.Background(), "login1", "password123")
		if err != nil {
			t.Fatal(err)
		}
		if u.Login != "login1" {
			t.Errorf("expected login 'login1', got '%s'", u.Login)
		}
		if u.ID == 0 {
			t.Errorf("expected assigned ID, got 0")
		}
		if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("password123")) != nil {
			t.Error("password hash does not match original password")
		}
	})

	t.Run("password too short", func(t *testing.T) {
		_, err := svc.Register(context.Background(), "login2", "short")
		if !errors.Is(err, ErrPasswordTooShort) {
			t.Errorf("expected ErrPasswordTooShort, got %v", err)
		}
	})

	t.Run("user already exists", func(t *testing.T) {
		_, err := svc.Register(context.Background(), "login1", "anotherpass")
		if !errors.Is(err, ErrUserExists) {
			t.Errorf("expected ErrUserExists, got %v", err)
		}
	})

	t.Run("empty login", func(t *testing.T) {
		u, err := svc.Register(context.Background(), "", "password123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.Login != "" {
			t.Errorf("expected empty login, got %s", u.Login)
		}
	})

	t.Run("repo create returns error", func(t *testing.T) {
		repo := newStubUserRepo()
		repo.errOnCreate = errors.New("db error")
		svc := NewService(repo, []byte("secret"), time.Hour)

		_, err := svc.Register(context.Background(), "login3", "password123")
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})
}

func TestServiceAuthenticate(t *testing.T) {
	repo := newStubUserRepo()
	svc := NewService(repo, []byte("secret"), time.Hour)

	password := "password123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	repo.users["login1"] = &user.User{ID: 1, Login: "login1", PasswordHash: string(hash)}

	t.Run("successful authentication", func(t *testing.T) {
		token, err := svc.Authenticate(context.Background(), "login1", password)
		if err != nil {
			t.Fatal(err)
		}
		if token == "" {
			t.Error("expected non-empty token")
		}
	})

	t.Run("invalid login", func(t *testing.T) {
		_, err := svc.Authenticate(context.Background(), "no-user", "password")
		if !errors.Is(err, ErrInvalidCreds) {
			t.Errorf("expected ErrInvalidCreds, got %v", err)
		}
	})

	t.Run("invalid password", func(t *testing.T) {
		_, err := svc.Authenticate(context.Background(), "login1", "wrongpass")
		if !errors.Is(err, ErrInvalidCreds) {
			t.Errorf("expected ErrInvalidCreds, got %v", err)
		}
	})

	t.Run("repo find returns error", func(t *testing.T) {
		repo := newStubUserRepo()
		repo.errOnFind = errors.New("db find error")
		svc := NewService(repo, []byte("secret"), time.Hour)

		_, err := svc.Authenticate(context.Background(), "login1", "password123")
		if !errors.Is(err, ErrInvalidCreds) {
			t.Errorf("expected ErrInvalidCreds, got %v", err)
		}
	})

	t.Run("authenticate returns valid JWT", func(t *testing.T) {
		token, err := svc.Authenticate(context.Background(), "login1", password)
		if err != nil {
			t.Fatal(err)
		}
		if token == "" {
			t.Fatal("token is empty")
		}

		parsed, _, err := new(jwt.Parser).ParseUnverified(token, &jwt.RegisteredClaims{})
		if err != nil {
			t.Fatalf("failed to parse token: %v", err)
		}
		claims, ok := parsed.Claims.(*jwt.RegisteredClaims)
		if !ok {
			t.Fatal("token claims have wrong type")
		}
		if claims.Subject != "login1" {
			t.Errorf("expected subject 'login1', got %q", claims.Subject)
		}
	})
}

func TestServiceRegisterWithCancelledContext(t *testing.T) {
	repo := newStubUserRepo()
	svc := NewService(repo, []byte("secret"), time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.Register(ctx, "login2", "password123")
	if err != nil {
		t.Logf("got error with cancelled context: %v", err)
	}
}
func setupUserHandler() (*Handler, *stubUserRepo) {
	repo := newStubUserRepo()
	svc := NewService(repo, []byte("secret"), time.Hour)
	return NewHandler(svc), repo
}

func TestUserHandlerRegister(t *testing.T) {
	handler, _ := setupUserHandler()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"Valid registration", `{"login":"testuser","password":"password123"}`, http.StatusOK},
		{"Invalid JSON", `{"login":"testuser",password:"badjson"}`, http.StatusBadRequest},
		{"Password too short", `{"login":"testuser","password":"short"}`, http.StatusBadRequest},
		{"User already exists", `{"login":"testuser","password":"password123"}`, http.StatusConflict},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(tt.body))
		rec := httptest.NewRecorder()

		handler.Register(rec, req)
		res := rec.Result()
		defer res.Body.Close()

		if res.StatusCode != tt.wantStatus {
			t.Errorf("%s: got status %d, want %d", tt.name, res.StatusCode, tt.wantStatus)
		}
	}
}

func TestUserHandlerLogin(t *testing.T) {
	handler, repo := setupUserHandler()

	pass := "password123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	repo.users["testuser"] = &user.User{
		ID:           1,
		Login:        "testuser",
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"Valid login", `{"login":"testuser","password":"password123"}`, http.StatusOK},
		{"Invalid password", `{"login":"testuser","password":"wrongpass"}`, http.StatusUnauthorized},
		{"Invalid JSON", `{"login":"testuser",password:"badjson"}`, http.StatusBadRequest},
		{"User not found", `{"login":"nouser","password":"pass"}`, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(tt.body))
		rec := httptest.NewRecorder()

		handler.Login(rec, req)
		res := rec.Result()

		defer res.Body.Close()

		if res.StatusCode != tt.wantStatus {
			t.Errorf("%s: got status %d, want %d", tt.name, res.StatusCode, tt.wantStatus)
		}
	}
}
