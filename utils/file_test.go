package utils

import "testing"

func TestSanitize(t *testing.T) {
	input := `My:Invalid/Name*?<>|`
	expected := "My_Invalid_Name_____"

	output := Sanitize(input)
	if output != expected {
		t.Errorf("expected '%s', got '%s'", expected, output)
	}
}
