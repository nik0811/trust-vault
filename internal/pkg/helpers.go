package pkg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

var validate = validator.New()

type ctxKey string

const (
	CtxTenantID     ctxKey = "tenant_id"
	CtxUserID       ctxKey = "user_id"
	CtxPermissions  ctxKey = "permissions"
	CtxIsSuperAdmin ctxKey = "is_super_admin"
	CtxClientIP     ctxKey = "client_ip"
)

type Claims struct {
	UserID       string   `json:"sub"`
	TenantID     string   `json:"tenant_id"`
	Permissions  []string `json:"permissions"`
	IsSuperAdmin bool     `json:"is_super_admin"`
	jwt.RegisteredClaims
}

var jwtSecret []byte

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Warn().Msg("JWT_SECRET not set - using insecure default (DO NOT USE IN PRODUCTION)")
		secret = "insecure-dev-only-change-me"
	}
	jwtSecret = []byte(secret)
}

func MustInitJWTSecret() {
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal().Msg("JWT_SECRET environment variable is required")
	}
}

func SetJWTSecret(secret string) {
	jwtSecret = []byte(secret)
}

func GenerateToken(userID, tenantID string, permissions []string, isSuperAdmin bool) (string, error) {
	claims := Claims{
		UserID:       userID,
		TenantID:     tenantID,
		Permissions:  permissions,
		IsSuperAdmin: isSuperAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func TenantFromCtx(ctx context.Context) string {
	if v := ctx.Value(CtxTenantID); v != nil {
		return v.(string)
	}
	return ""
}

func UserFromCtx(ctx context.Context) string {
	if v := ctx.Value(CtxUserID); v != nil {
		return v.(string)
	}
	return ""
}

func PermissionsFromCtx(ctx context.Context) []string {
	if v := ctx.Value(CtxPermissions); v != nil {
		return v.([]string)
	}
	return nil
}

func IsSuperAdminFromCtx(ctx context.Context) bool {
	if v := ctx.Value(CtxIsSuperAdmin); v != nil {
		return v.(bool)
	}
	return false
}

func ClientIPFromCtx(ctx context.Context) string {
	if v := ctx.Value(CtxClientIP); v != nil {
		return v.(string)
	}
	return ""
}

// GetClientIP extracts the real client IP from request headers
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (set by proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

func HasPermission(ctx context.Context, perm string) bool {
	if IsSuperAdminFromCtx(ctx) {
		return true
	}
	for _, p := range PermissionsFromCtx(ctx) {
		if p == perm || p == "*" {
			return true
		}
		if strings.HasSuffix(p, ":*") && strings.HasPrefix(perm, strings.TrimSuffix(p, "*")) {
			return true
		}
	}
	return false
}

func JSON(w http.ResponseWriter, data any, status ...int) {
	w.Header().Set("Content-Type", "application/json")
	code := http.StatusOK
	if len(status) > 0 {
		code = status[0]
	}
	w.WriteHeader(code)
	// Coerce nil slices/maps to empty so JSON encodes [] / {} instead of null
	if data != nil {
		v := reflect.ValueOf(data)
		if v.Kind() == reflect.Slice && v.IsNil() {
			data = reflect.MakeSlice(v.Type(), 0, 0).Interface()
		} else if v.Kind() == reflect.Map && v.IsNil() {
			data = reflect.MakeMap(v.Type()).Interface()
		}
	}
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Error().Err(err).Msg("Failed to encode JSON response")
	}
}

func Error(w http.ResponseWriter, err error, status ...int) {
	code := http.StatusInternalServerError
	if len(status) > 0 {
		code = status[0]
	}
	if errors.Is(err, ErrNotFound) {
		code = http.StatusNotFound
	} else if errors.Is(err, ErrUnauthorized) {
		code = http.StatusUnauthorized
	} else if errors.Is(err, ErrForbidden) {
		code = http.StatusForbidden
	} else if errors.Is(err, ErrBadRequest) {
		code = http.StatusBadRequest
	}

	log.Error().Err(err).Int("status", code).Msg("API error")
	JSON(w, map[string]string{"error": err.Error()}, code)
}

func Bind(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		log.Error().Err(err).Msg("JSON decode failed")
		return ErrBadRequest
	}
	if err := validate.Struct(v); err != nil {
		log.Error().Err(err).Msg("Validation failed")
		return ErrBadRequest
	}
	return nil
}

func ParseListOpts(r *http.Request) (limit, offset int) {
	limit = 50
	offset = 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}
	return
}

var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrBadRequest   = errors.New("bad request")
)

type Match struct {
	Value string
	Start int
	End   int
}

func FindAllMatches(text, pattern string) []Match {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringIndex(text, -1)
	var results []Match
	for _, m := range matches {
		results = append(results, Match{
			Value: text[m[0]:m[1]],
			Start: m[0],
			End:   m[1],
		})
	}
	return results
}

func Sprintf(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

// GenerateID generates a new UUID string
func GenerateID() string {
	return uuid.New().String()
}

// IsValidUUID checks if a string is a valid UUID
func IsValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// DerefStr safely dereferences a *string, returning "" if nil.
func DerefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
