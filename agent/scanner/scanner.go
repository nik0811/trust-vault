package scanner

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Finding struct {
	FilePath   string  `json:"file_path"`
	LineNumber int     `json:"line_number"`
	PIIType    string  `json:"pii_type"`
	Value      string  `json:"value"`
	Masked     string  `json:"masked"`
	Confidence float64 `json:"confidence"`
	Severity   string  `json:"severity"`
}

type ScanResult struct {
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	FilesScanned int       `json:"files_scanned"`
	BytesScanned int64     `json:"bytes_scanned"`
	Findings     []Finding `json:"findings"`
	Errors       []string  `json:"errors,omitempty"`
}

type Scanner struct {
	excludePatterns []string
	maxFileSize     int64
	workers         int
	verbose         bool
	onProgress      func(file string, findings int)
}

type ScannerOption func(*Scanner)

func WithExclude(patterns []string) ScannerOption {
	return func(s *Scanner) {
		s.excludePatterns = patterns
	}
}

func WithMaxFileSize(size int64) ScannerOption {
	return func(s *Scanner) {
		s.maxFileSize = size
	}
}

func WithWorkers(n int) ScannerOption {
	return func(s *Scanner) {
		s.workers = n
	}
}

func WithVerbose(v bool) ScannerOption {
	return func(s *Scanner) {
		s.verbose = v
	}
}

func WithProgress(fn func(file string, findings int)) ScannerOption {
	return func(s *Scanner) {
		s.onProgress = fn
	}
}

func New(opts ...ScannerOption) *Scanner {
	s := &Scanner{
		maxFileSize: 50 * 1024 * 1024, // 50MB default
		workers:     4,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Scanner) Scan(paths []string) (*ScanResult, error) {
	result := &ScanResult{
		StartTime: time.Now(),
		Findings:  []Finding{},
	}

	files := make(chan string, 100)
	findings := make(chan Finding, 100)
	errors := make(chan string, 100)
	var filesScanned int64

	var wg sync.WaitGroup
	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range files {
				fileFindings, bytesRead, err := s.scanFile(file)
				atomic.AddInt64(&filesScanned, 1)
				if err != nil {
					errors <- fmt.Sprintf("%s: %v", file, err)
					continue
				}
				atomic.AddInt64(&result.BytesScanned, bytesRead)
				for _, f := range fileFindings {
					findings <- f
				}
				if s.onProgress != nil {
					s.onProgress(file, len(fileFindings))
				}
			}
		}()
	}

	go func() {
		for _, path := range paths {
			s.walkPath(path, files)
		}
		close(files)
	}()

	go func() {
		wg.Wait()
		close(findings)
		close(errors)
	}()

	var findingsMu sync.Mutex
	var errorsMu sync.Mutex
	var findingsWg sync.WaitGroup
	var errorsWg sync.WaitGroup

	findingsWg.Add(1)
	go func() {
		defer findingsWg.Done()
		for f := range findings {
			findingsMu.Lock()
			result.Findings = append(result.Findings, f)
			findingsMu.Unlock()
		}
	}()

	errorsWg.Add(1)
	go func() {
		defer errorsWg.Done()
		for e := range errors {
			errorsMu.Lock()
			result.Errors = append(result.Errors, e)
			errorsMu.Unlock()
		}
	}()

	findingsWg.Wait()
	errorsWg.Wait()

	result.FilesScanned = int(filesScanned)
	result.EndTime = time.Now()
	return result, nil
}

func (s *Scanner) walkPath(root string, files chan<- string) {
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if s.shouldExclude(path) {
				return filepath.SkipDir
			}
			return nil
		}
		if s.shouldExclude(path) {
			return nil
		}
		if info.Size() > s.maxFileSize {
			return nil
		}
		if !s.isTextFile(path) {
			return nil
		}
		files <- path
		return nil
	})
}

func (s *Scanner) shouldExclude(path string) bool {
	base := filepath.Base(path)
	for _, pattern := range s.excludePatterns {
		if matched, _ := filepath.Match(pattern, base); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
	}
	skipDirs := []string{".git", ".svn", "node_modules", "__pycache__", ".venv", "venv", ".idea", ".vscode"}
	for _, skip := range skipDirs {
		if base == skip {
			return true
		}
	}
	skipExts := []string{".exe", ".dll", ".so", ".dylib", ".bin", ".o", ".a", ".pyc", ".class",
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".ico", ".svg", ".webp",
		".mp3", ".mp4", ".avi", ".mov", ".mkv", ".wav", ".flac",
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".zip", ".tar", ".gz", ".bz2", ".7z", ".rar", ".jar", ".war"}
	ext := strings.ToLower(filepath.Ext(path))
	for _, skip := range skipExts {
		if ext == skip {
			return true
		}
	}
	return false
}

func (s *Scanner) isTextFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}
	buf = buf[:n]

	for _, b := range buf {
		if b == 0 {
			return false
		}
	}
	return true
}

func (s *Scanner) scanFile(path string) ([]Finding, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}

	var findings []Finding
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		for _, pattern := range PIIPatterns {
			matches := pattern.Regex.FindAllStringIndex(line, -1)
			for _, match := range matches {
				value := line[match[0]:match[1]]
				confidence := pattern.Confidence

				if pattern.Validator != nil && !pattern.Validator(value) {
					confidence *= 0.5
				}
				if confidence < 0.5 {
					continue
				}

				findings = append(findings, Finding{
					FilePath:   path,
					LineNumber: lineNum,
					PIIType:    pattern.Name,
					Value:      value,
					Masked:     maskValue(value, pattern.Name),
					Confidence: confidence,
					Severity:   pattern.Severity,
				})
			}
		}
	}

	return findings, info.Size(), scanner.Err()
}

func maskValue(value, piiType string) string {
	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}

	switch piiType {
	case "EMAIL":
		parts := strings.Split(value, "@")
		if len(parts) == 2 {
			local := parts[0]
			domain := parts[1]
			if len(local) > 2 {
				local = local[:1] + strings.Repeat("*", len(local)-2) + local[len(local)-1:]
			}
			domainParts := strings.Split(domain, ".")
			if len(domainParts) > 0 && len(domainParts[0]) > 2 {
				domainParts[0] = domainParts[0][:1] + strings.Repeat("*", len(domainParts[0])-1)
			}
			return local + "@" + strings.Join(domainParts, ".")
		}
	case "SSN":
		digits := extractDigits(value)
		if len(digits) == 9 {
			return "***-**-" + digits[5:]
		}
	case "CREDIT_CARD", "CREDIT_CARD_FORMATTED":
		digits := extractDigits(value)
		if len(digits) >= 4 {
			return "****-****-****-" + digits[len(digits)-4:]
		}
	case "PHONE":
		digits := extractDigits(value)
		if len(digits) >= 4 {
			return strings.Repeat("*", len(digits)-4) + digits[len(digits)-4:]
		}
	case "API_KEY", "AWS_SECRET_KEY", "JWT_TOKEN", "GITHUB_TOKEN", "SLACK_TOKEN", "STRIPE_KEY":
		if len(value) > 8 {
			return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
		}
	case "PRIVATE_KEY":
		return "-----BEGIN PRIVATE KEY-----[REDACTED]"
	case "DATABASE_URL":
		if idx := strings.Index(value, "://"); idx > 0 {
			return value[:idx+3] + "***:***@***"
		}
	}

	if len(value) > 6 {
		return value[:3] + strings.Repeat("*", len(value)-6) + value[len(value)-3:]
	}
	return value[:2] + strings.Repeat("*", len(value)-2)
}
