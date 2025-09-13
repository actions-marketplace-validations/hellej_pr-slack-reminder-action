package inputhelpers_test

import (
	"testing"

	"github.com/hellej/pr-slack-reminder-action/internal/config/inputhelpers"
)

func TestReadInput(t *testing.T) {
	t.Setenv("INPUT_TEST", "test_value")
	value := inputhelpers.GetInput("test")
	if value != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", value)
	}

	nonSetInputValue := inputhelpers.GetInput("notSet")
	if nonSetInputValue != "" {
		t.Errorf("Expected '', got '%s'", value)
	}
}

func TestReadInputRequired(t *testing.T) {
	_, err := inputhelpers.GetInputRequired("test")
	if err == nil {
		t.Errorf("Expected error for missing required input, got nil")
	}
}

func TestReadInputIntOk(t *testing.T) {
	t.Setenv("INPUT_TEST", "1")
	value, _ := inputhelpers.GetInputInt("test")
	expected := 1
	if value != expected {
		t.Errorf("Expected '%d', got '%v'", expected, value)
	}
}

func TestReadInputIntNotSet(t *testing.T) {
	value, err := inputhelpers.GetInputInt("notSet")
	if err != nil {
		t.Errorf("Expected no error for missing int input, got %v", err)
	}
	expected := 0
	if value != expected {
		t.Errorf("Expected '%d', got '%v'", expected, value)
	}
}

func TestReadInputIntInvalid(t *testing.T) {
	t.Setenv("INPUT_TEST", "a")
	_, err := inputhelpers.GetInputInt("test")
	if err == nil {
		t.Errorf("Expected error for invalid int input, got nil")
	}
	expectedError := "error parsing input test as integer: strconv.Atoi: parsing \"a\": invalid syntax"
	if err.Error() != expectedError {
		t.Errorf("Expected error %v, got '%v'", expectedError, err)
	}
}

func TestReadStringMapping(t *testing.T) {
	t.Setenv("INPUT_TEST", "a:b;c:d")
	mapping, _ := inputhelpers.GetInputMapping("test")
	expected := map[string]string{"a": "b", "c": "d"}

	for key, expected := range expected {
		value, exists := (mapping)[key]
		if !exists {
			t.Errorf("Expected key '%s' to exist in mapping", key)
		}
		if value != expected {
			t.Errorf("Expected '%v', got '%v'", expected, value)
		}
	}
}
func TestReadInputMappingInvalid1(t *testing.T) {
	t.Setenv("INPUT_TEST", "a:b;c")
	_, err := inputhelpers.GetInputMapping("test")
	if err == nil {
		t.Errorf("Expected error for invalid mapping input, got nil")
	}
}

func TestReadInputMappingInvalid2(t *testing.T) {
	t.Setenv("INPUT_TEST", " ;a:b;c: ")
	_, err := inputhelpers.GetInputMapping("test")
	if err == nil {
		t.Errorf("Expected error for invalid mapping input, got nil")
	}
}
