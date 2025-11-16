package testpayload

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/go-faker/faker/v4"
)

// Payload represents the predictable payload structure
// faker annotates fields for automatic generation
// https://github.com/go-faker/faker#supported-tags
type Payload struct {
	ID     string  `faker:"uuid_hyphenated" json:"id"`
	Name   string  `faker:"name" json:"name"`
	Value  float64 `faker:"lat" json:"value"` // use lat as random float
	Active bool    `json:"active"`
	Time   int64   `faker:"unix_time" json:"time"`
}

// generates an instance of Payload with realistic random values
func generatePredictablePayload() Payload {
	var p Payload
	if err := faker.FakeData(&p); err != nil {
		// If faker fails, return a minimal valid payload
		p = Payload{
			ID:     "00000000-0000-0000-0000-000000000000",
			Name:   "default",
			Value:  0.0,
			Active: false,
			Time:   0,
		}
	}
	return p
}

// GenerateRandomJSON creates a JSON with predictable structure and random values
func GenerateRandomJSON() ([]byte, error) {
	return json.Marshal(generatePredictablePayload())
}

// GenerateRandomCBOR creates a CBOR with predictable structure and random values
func GenerateRandomCBOR() ([]byte, error) {
	return cbor.Marshal(generatePredictablePayload())
}

// GenerateSentence generates a random sentence for tests
func GenerateSentence() string {
	return faker.Sentence()
}

func GenerateSentimentPhrase() string {
	starts := []string{"I love", "I hate", "I think", "I feel", "I wish", "I see"}
	adjectives := []string{"great", "terrible", "amazing", "awful", "funny", "boring"}
	objects := []string{"this product", "the service", "the movie", "the food", "the weather", "the app"}
	return starts[rand.Intn(len(starts))] + " " + adjectives[rand.Intn(len(adjectives))] + " " + objects[rand.Intn(len(objects))] // #nosec G404 -- test data generator
}

func GenerateRandomDateTime() string {
	// Generate a random Unix timestamp between 1 and 10 years ago
	timestamp := rand.Int63n(10*365*24*3600) + (time.Now().Unix() - 10*365*24*3600) // #nosec G404 -- test data generator
	return time.Unix(timestamp, 0).Format(time.RFC3339Nano)
}

func GenerateNowDateTime() string {
	// Generate the current timestamp in RFC3339
	return time.Now().Format(time.RFC3339Nano)
}

var counter int = 0
var counterMutex = sync.Mutex{}

func GenerateCounter() int {
	counterMutex.Lock()
	defer counterMutex.Unlock()
	counter++
	return counter
}

func Interpolate(str string) ([]byte, error) {
	return InterpolateWithDelimiters(str, "{{", "}}")
}

// InterpolateWithDelimiters performs template variable interpolation with custom delimiters
// Supports placeholders: json, cbor, sentiment, sentence, datetime, nowtime, counter, file:/path
func InterpolateWithDelimiters(str string, openDelim string, closeDelim string) ([]byte, error) {
	placeholders := map[string]TestPayloadType{
		"json":      TestPayloadJSON,
		"cbor":      TestPayloadCBOR,
		"sentiment": TestPayloadSentiment,
		"sentence":  TestPayloadSentence,
		"datetime":  TestPayloadDateTime,
		"nowtime":   TestPayloadNowTime,
		"counter":   TestPayloadCounter,
	}

	result := str
	// Handle `var:` placeholders first (variable substitution)
	varPrefix := openDelim + "var:"
	if strings.Contains(result, varPrefix) {
		for key := range templateVars {
			ph := openDelim + "var:" + key + closeDelim
			if strings.Contains(result, ph) {
				result = strings.ReplaceAll(result, ph, templateVars[key])
			}
		}
		// Replace any var: placeholders not found in map with empty string
		for {
			startIdx := strings.Index(result, varPrefix)
			if startIdx == -1 {
				break
			}
			endIdx := strings.Index(result[startIdx:], closeDelim)
			if endIdx == -1 {
				break
			}
			endIdx += startIdx
			placeholder := result[startIdx : endIdx+len(closeDelim)]
			result = strings.Replace(result, placeholder, "", 1)
		}
	}
	// Process `raw:` and `str:` wrappers, these wrap inner placeholders or file: expressions
	wrappers := []string{"raw:", "str:"}
	for _, w := range wrappers {
		prefix := openDelim + w
		if strings.Contains(result, prefix) {
			for {
				startIdx := strings.Index(result, prefix)
				if startIdx == -1 {
					break
				}
				endIdx := strings.Index(result[startIdx:], closeDelim)
				if endIdx == -1 {
					return nil, fmt.Errorf("unclosed placeholder at position %d", startIdx)
				}
				endIdx += startIdx
				inner := result[startIdx+len(prefix) : endIdx]
				var val []byte
				var err error
				if strings.HasPrefix(inner, "file:") {
					// file read
					fp := inner[len("file:"):]
					if fp == "" {
						return nil, fmt.Errorf("empty file path in placeholder at position %d", startIdx)
					}
					if !AllowFileReads {
						return nil, fmt.Errorf("file reads are disabled: to enable allow file reads set testpayload.SetAllowFileReads(true)")
					}
					if FileRoot != "" {
						absRoot, _ := filepath.Abs(FileRoot)
						absPath, err2 := filepath.Abs(fp)
						if err2 != nil {
							return nil, fmt.Errorf("invalid file path: %s", fp)
						}
						if !strings.HasPrefix(absPath, absRoot) {
							return nil, fmt.Errorf("file %s outside allowed root %s", fp, FileRoot)
						}
					}
					// Check cache
					if c, ok := GetFileFromCache(fp); ok {
						val = c
					} else {
						val, err = os.ReadFile(fp)
						if err == nil {
							PutFileIntoCache(fp, val)
						}
					}
					if err != nil {
						return nil, fmt.Errorf("failed to read file %s: %w", fp, err)
					}
				} else if strings.HasPrefix(inner, "var:") {
					key := inner[len("var:"):]
					val = []byte(templateVars[key])
				} else if t, ok := placeholders[inner]; ok {
					val, err = t.Generate()
					if err != nil {
						return nil, err
					}
				} else {
					// Unknown inner expression, treat as raw text
					val = []byte(inner)
				}
				// For str: wrapper, JSON-escape the value (including quotes)
				if w == "str:" {
					esc, err := json.Marshal(string(val))
					if err != nil {
						return nil, fmt.Errorf("failed to escape value: %w", err)
					}
					val = esc
				}
				placeholder := result[startIdx : endIdx+len(closeDelim)]
				result = strings.Replace(result, placeholder, string(val), 1)
			}
		}
	}

	for key, typ := range placeholders {
		ph := openDelim + key + closeDelim

		if str == ph {
			// If the entire string is just the placeholder, return the generated value directly
			return typ.Generate()
		}

		if strings.Contains(result, ph) {
			val, err := typ.Generate()
			if err != nil {
				return nil, err
			}
			result = strings.ReplaceAll(result, ph, string(val))
		}
	}

	// Handle file:// placeholder (non-wrapped form)
	filePrefix := openDelim + "file:"
	fileSuffix := closeDelim
	if strings.Contains(result, filePrefix) {
		for {
			startIdx := strings.Index(result, filePrefix)
			if startIdx == -1 {
				break
			}
			endIdx := strings.Index(result[startIdx:], fileSuffix)
			if endIdx == -1 {
				return nil, fmt.Errorf("unclosed file placeholder at position %d", startIdx)
			}
			endIdx += startIdx

			// Extract file path
			filePath := result[startIdx+len(filePrefix) : endIdx]
			if filePath == "" {
				return nil, fmt.Errorf("empty file path in placeholder at position %d", startIdx)
			}

			// Read file content
			// File reads may be disabled by default for security in CI.
			if !AllowFileReads {
				return nil, fmt.Errorf("file reads are disabled: to enable allow file reads set testpayload.SetAllowFileReads(true)")
			}
			if FileRoot != "" {
				absRoot, _ := filepath.Abs(FileRoot)
				absPath, err2 := filepath.Abs(filePath)
				if err2 != nil {
					return nil, fmt.Errorf("invalid file path: %s", filePath)
				}
				if !strings.HasPrefix(absPath, absRoot) {
					return nil, fmt.Errorf("file %s outside allowed root %s", filePath, FileRoot)
				}
			}
			// #nosec G304 -- reading file for test payload generation
			// Fetch from cache or read and put into cache
			var content []byte
			var err error
			if c, ok := GetFileFromCache(filePath); ok {
				content = c
			} else {
				content, err = os.ReadFile(filePath)
				if err == nil {
					PutFileIntoCache(filePath, content)
				}
			}
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
			}

			// Replace placeholder with file content
			placeholder := result[startIdx : endIdx+len(fileSuffix)]
			result = strings.Replace(result, placeholder, string(content), 1)
		}
	}

	return []byte(result), nil
}

// AllowFileReads controls whether {{file:...}} placeholders are permitted.
// Disabled by default for safety; set via testpayload.SetAllowFileReads(true) or CLI flag.
var AllowFileReads bool = false

// SetAllowFileReads toggles file reading support for the test payload generator.
func SetAllowFileReads(v bool) {
	AllowFileReads = v
}

// SeedRandom seeds the global pseudo-random generator used by testpayload helpers.
// Useful to make generation deterministic for tests and reproducible scenarios.
func SeedRandom(seed int64) {
	rand.Seed(seed)
}

// Template variables for substitution using {{var:name}} placeholders
var templateVars = map[string]string{}

// SetTemplateVars replaces the full variables map used by InterpolateWithDelimiters.
func SetTemplateVars(vars map[string]string) {
	templateVars = map[string]string{}
	for k, v := range vars {
		templateVars[k] = v
	}
}

// AddTemplateVar adds a single template variable.
func AddTemplateVar(name, val string) {
	if templateVars == nil {
		templateVars = map[string]string{}
	}
	templateVars[name] = val
}

// ClearTemplateVars clears all configured template variables.
func ClearTemplateVars() {
	templateVars = map[string]string{}
}

// FileRoot is the optional root path for allowed file reads; empty means no root restriction.
var FileRoot string = ""

// SetFileRoot sets a root path that file placeholders must be under to be allowed.
func SetFileRoot(root string) {
	FileRoot = root
}

// File cache
var fileCacheEnabled bool = false
var fileCache = map[string][]byte{}
var fileCacheMutex = sync.RWMutex{}

// SetFileCacheEnabled toggles file content caching (process-lifetime cache).
func SetFileCacheEnabled(v bool) {
	fileCacheMutex.Lock()
	defer fileCacheMutex.Unlock()
	fileCacheEnabled = v
	if v && fileCache == nil {
		fileCache = map[string][]byte{}
	}
	if !v {
		fileCache = map[string][]byte{}
	}
}

// ClearFileCache clears the in-memory file cache.
func ClearFileCache() {
	fileCacheMutex.Lock()
	defer fileCacheMutex.Unlock()
	fileCache = map[string][]byte{}
}

// GetFileFromCache returns file content if present, else nil/false
func GetFileFromCache(path string) ([]byte, bool) {
	fileCacheMutex.RLock()
	defer fileCacheMutex.RUnlock()
	if !fileCacheEnabled {
		return nil, false
	}
	v, ok := fileCache[path]
	return v, ok
}

// PutFileIntoCache stores content in the cache if enabled
func PutFileIntoCache(path string, content []byte) {
	if !fileCacheEnabled {
		return
	}
	fileCacheMutex.Lock()
	fileCache[path] = content
	fileCacheMutex.Unlock()
}

type TestPayloadType string

const (
	TestPayloadJSON      TestPayloadType = "json"
	TestPayloadCBOR      TestPayloadType = "cbor"
	TestPayloadSentiment TestPayloadType = "sentiment"
	TestPayloadSentence  TestPayloadType = "sentence"
	TestPayloadDateTime  TestPayloadType = "datetime" // to generate a timestamp
	TestPayloadNowTime   TestPayloadType = "nowtime"  // to generate the current timestamp
	TestPayloadCounter   TestPayloadType = "counter"  // to generate an incrementing counter (not implemented yet
)

func (t TestPayloadType) IsValid() bool {
	switch t {
	case TestPayloadJSON, TestPayloadCBOR, TestPayloadSentiment, TestPayloadSentence, TestPayloadDateTime, TestPayloadNowTime:
		return true
	}
	return false
}

func (t TestPayloadType) GetContentType() string {
	switch t {
	case TestPayloadJSON:
		return "application/json"
	case TestPayloadCBOR:
		return "application/cbor"
	case TestPayloadSentiment, TestPayloadSentence, TestPayloadDateTime, TestPayloadNowTime:
		return "text/plain"
	}
	return "application/octet-stream"
}

func (t TestPayloadType) Generate() ([]byte, error) {
	switch t {
	case TestPayloadJSON:
		return GenerateRandomJSON()
	case TestPayloadCBOR:
		return GenerateRandomCBOR()
	case TestPayloadSentiment:
		return []byte(GenerateSentimentPhrase()), nil
	case TestPayloadSentence:
		return []byte(GenerateSentence()), nil
	case TestPayloadDateTime:
		return []byte(GenerateRandomDateTime()), nil
	case TestPayloadNowTime:
		return []byte(GenerateNowDateTime()), nil
	case TestPayloadCounter:
		return []byte(fmt.Sprintf("%d", GenerateCounter())), nil
	}
	return nil, fmt.Errorf("unsupported test payload type: %s", t)
}
