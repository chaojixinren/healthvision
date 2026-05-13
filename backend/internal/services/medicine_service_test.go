package services

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateMedicineInputNormalizesValidValues(t *testing.T) {
	name, imageURL, description, notes, err := validateMedicineInput(
		"  阿莫西林  ",
		" https://example.com/medicine.png ",
		"  饭后服用  ",
		"  每日两次  ",
	)
	if err != nil {
		t.Fatalf("validateMedicineInput returned error: %v", err)
	}
	if name != "阿莫西林" || imageURL != "https://example.com/medicine.png" ||
		description != "饭后服用" || notes != "每日两次" {
		t.Fatalf("values were not normalized: %q %q %q %q", name, imageURL, description, notes)
	}
}

func TestValidateMedicineInputRejectsBlankName(t *testing.T) {
	_, _, _, _, err := validateMedicineInput("   ", "", "", "")
	if !errors.Is(err, ErrInvalidMedicineName) {
		t.Fatalf("expected ErrInvalidMedicineName, got %v", err)
	}
}

func TestValidateMedicineInputRejectsInvalidImageURL(t *testing.T) {
	_, _, _, _, err := validateMedicineInput("阿莫西林", "javascript:alert(1)", "", "")
	if !errors.Is(err, ErrInvalidMedicineImageURL) {
		t.Fatalf("expected ErrInvalidMedicineImageURL, got %v", err)
	}
}

func TestValidateMedicineInputRejectsLongText(t *testing.T) {
	_, _, _, _, err := validateMedicineInput("阿莫西林", "", strings.Repeat("a", maxMedicineDescriptionLength+1), "")
	if !errors.Is(err, ErrInvalidMedicineText) {
		t.Fatalf("expected ErrInvalidMedicineText, got %v", err)
	}
}
