package utils

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

// Hashable interface defines methods that models must implement for hash generation
type Hashable interface {
	GetHashableFields() map[string]interface{}
	SetContentHash(hash string)
	GetContentHash() string
}

// GenerateEntityHash creates a SHA-256 hash from an entity's hashable fields
func GenerateEntityHash(entity Hashable) (string, error) {
	fields := entity.GetHashableFields()
	return HashFields(fields), nil
}

// HashFields creates a deterministic hash from a map of fields
func HashFields(fields map[string]interface{}) string {
	// Create a sorted slice of keys for deterministic ordering
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Build ordered map for consistent JSON serialization
	orderedMap := make(map[string]interface{})
	for _, key := range keys {
		orderedMap[key] = normalizeValue(fields[key])
	}

	// Convert to JSON for hashing
	jsonBytes, err := json.Marshal(orderedMap)
	if err != nil {
		// Fallback to empty string if marshaling fails
		jsonBytes = []byte("{}")
	}

	// Generate SHA-256 hash
	hash := sha256.Sum256(jsonBytes)
	return fmt.Sprintf("%x", hash)
}

// normalizeValue ensures consistent representation of values for hashing
func normalizeValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	v := reflect.ValueOf(value)

	// Handle pointers
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		return normalizeValue(v.Elem().Interface())
	}

	// Handle different types to ensure consistency
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint()
	case reflect.Float32, reflect.Float64:
		return v.Float()
	case reflect.Bool:
		return v.Bool()
	default:
		// For other types, return as-is and let JSON marshaling handle them
		return value
	}
}

// PrepareHashableFields extracts hashable fields from a struct using reflection
// This is a helper function for models that want to auto-generate hashable fields
func PrepareHashableFields(entity interface{}, excludeFields ...string) map[string]interface{} {
	fields := make(map[string]interface{})
	excludeMap := make(map[string]bool)

	// Create exclusion map
	for _, field := range excludeFields {
		excludeMap[field] = true
	}

	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fields
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !fieldValue.CanInterface() {
			continue
		}

		// Skip excluded fields
		if excludeMap[field.Name] {
			continue
		}

		// Skip certain field types that shouldn't be in hash
		if shouldSkipField(field.Name, field.Type) {
			continue
		}

		fields[field.Name] = fieldValue.Interface()
	}

	return fields
}

// shouldSkipField determines if a field should be excluded from hash calculation
func shouldSkipField(fieldName string, fieldType reflect.Type) bool {
	// Skip timestamp fields
	if fieldName == "CreatedAt" || fieldName == "UpdatedAt" || fieldName == "DeletedAt" {
		return true
	}

	// Skip the hash field itself
	if fieldName == "ContentHash" {
		return true
	}

	// Skip relationship fields (slices of structs)
	if fieldType.Kind() == reflect.Slice {
		elemType := fieldType.Elem()
		if elemType.Kind() == reflect.Struct {
			return true
		}
	}

	// Skip pointer to struct fields (relationships)
	if fieldType.Kind() == reflect.Ptr && fieldType.Elem().Kind() == reflect.Struct {
		return true
	}

	return false
}

// CompareHashes compares two hash strings for equality
func CompareHashes(hash1, hash2 string) bool {
	return hash1 == hash2
}

// ValidateHash checks if a hash string is valid (64 character hex string)
func ValidateHash(hash string) bool {
	if len(hash) != 64 {
		return false
	}

	for _, char := range hash {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}

	return true
}

// BatchCategories categorizes records into insert/update/skip based on hash comparison
type BatchCategories struct {
	Insert []interface{} // New records (not in DB)
	Update []interface{} // Changed records (hash mismatch)
	Skip   []interface{} // Unchanged records (hash match)
}

// DiscogsHashable represents an entity with DiscogsID and hash capabilities
type DiscogsHashable interface {
	Hashable
	GetDiscogsID() int64
}

// CategorizeRecordsByHash compares incoming records with existing hashes
// and categorizes them into insert/update/skip operations
func CategorizeRecordsByHash(incomingRecords []DiscogsHashable, existingHashes map[int64]string) *BatchCategories {
	categories := &BatchCategories{
		Insert: make([]interface{}, 0),
		Update: make([]interface{}, 0),
		Skip:   make([]interface{}, 0),
	}

	for _, record := range incomingRecords {
		discogsID := record.GetDiscogsID()
		incomingHash := record.GetContentHash()

		// If hash is empty, generate it
		if incomingHash == "" {
			hash, err := GenerateEntityHash(record)
			if err == nil {
				record.SetContentHash(hash)
				incomingHash = hash
			}
		}

		existingHash, exists := existingHashes[discogsID]

		if !exists {
			// Record doesn't exist in DB - insert
			categories.Insert = append(categories.Insert, record)
		} else if existingHash != incomingHash {
			// Record exists but hash changed - update
			categories.Update = append(categories.Update, record)
		} else {
			// Record exists and hash matches - skip
			categories.Skip = append(categories.Skip, record)
		}
	}

	return categories
}

// NameHashable represents an entity with Name and hash capabilities (like Genre)
type NameHashable interface {
	Hashable
	GetName() string
}

// CategorizeRecordsByNameHash compares incoming records with existing hashes using names
// and categorizes them into insert/update/skip operations
func CategorizeRecordsByNameHash(incomingRecords []NameHashable, existingHashes map[string]string) *BatchCategories {
	categories := &BatchCategories{
		Insert: make([]interface{}, 0),
		Update: make([]interface{}, 0),
		Skip:   make([]interface{}, 0),
	}

	for _, record := range incomingRecords {
		name := record.GetName()
		incomingHash := record.GetContentHash()

		// If hash is empty, generate it
		if incomingHash == "" {
			hash, err := GenerateEntityHash(record)
			if err == nil {
				record.SetContentHash(hash)
				incomingHash = hash
			}
		}

		existingHash, exists := existingHashes[name]

		if !exists {
			// Record doesn't exist in DB - insert
			categories.Insert = append(categories.Insert, record)
		} else if existingHash != incomingHash {
			// Record exists but hash changed - update
			categories.Update = append(categories.Update, record)
		} else {
			// Record exists and hash matches - skip
			categories.Skip = append(categories.Skip, record)
		}
	}

	return categories
}