package testpayload

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
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

	// Handle file:// placeholder
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
			// #nosec G304 -- reading file for test payload generation
			content, err := os.ReadFile(filePath)
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
