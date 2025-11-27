package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func useTempEnvFile(t *testing.T, content string, create bool) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, ".env")

	if create {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create temp env file: %v", err)
		}
	}

	prev := EnvFile
	EnvFile = path
	t.Cleanup(func() { EnvFile = prev })
}

func TestParsePlans(t *testing.T) {
	t.Run("normalizes and filters blanks", func(t *testing.T) {
		got := ParsePlans("FREE, PRO , ,PLUS,,")
		want := []string{"FREE", "PRO", "PLUS"}
		if !slices.Equal(got, want) {
			t.Fatalf("ParsePlans() = %v, want %v", got, want)
		}
	})

	t.Run("empty string returns empty slice", func(t *testing.T) {
		if got := ParsePlans(""); len(got) != 0 {
			t.Fatalf("ParsePlans() empty expected empty slice, got %v", got)
		}
	})
}

func TestMaskAPIKey(t *testing.T) {
	if got := MaskAPIKey("abcdefghijk"); got != "abcdefgh****" {
		t.Fatalf("MaskAPIKey long = %s", got)
	}
	if got := MaskAPIKey("short"); got != "****" {
		t.Fatalf("MaskAPIKey short = %s", got)
	}
}

func TestGetTimezone(t *testing.T) {
	useTempEnvFile(t, "", false)

	t.Setenv("TZ", "")
	t.Setenv("TIMEZONE", "")

	t.Run("command line overrides", func(t *testing.T) {
		if got := GetTimezone("Asia/Tokyo"); got != "Asia/Tokyo" {
			t.Fatalf("GetTimezone command value = %s", got)
		}
	})

	t.Run("environment TZ", func(t *testing.T) {
		t.Setenv("TZ", "Asia/Seoul")
		defer t.Setenv("TZ", "")
		if got := GetTimezone(""); got != "Asia/Seoul" {
			t.Fatalf("expected Asia/Seoul, got %s", got)
		}
	})

	t.Run("environment TIMEZONE", func(t *testing.T) {
		t.Setenv("TZ", "")
		t.Setenv("TIMEZONE", "Asia/Kuala_Lumpur")
		defer t.Setenv("TIMEZONE", "")
		if got := GetTimezone(""); got != "Asia/Kuala_Lumpur" {
			t.Fatalf("expected Asia/Kuala_Lumpur, got %s", got)
		}
	})

	t.Run("env file fallback", func(t *testing.T) {
		t.Setenv("TZ", "")
		t.Setenv("TIMEZONE", "")
		useTempEnvFile(t, "TZ=Asia/Manila\n", true)
		if got := GetTimezone(""); got != "Asia/Manila" {
			t.Fatalf("expected Asia/Manila, got %s", got)
		}
	})

	t.Run("default fallback", func(t *testing.T) {
		t.Setenv("TZ", "")
		t.Setenv("TIMEZONE", "")
		useTempEnvFile(t, "", false)
		if got := GetTimezone(""); got != DefaultTimezone {
			t.Fatalf("expected default timezone, got %s", got)
		}
	})
}

func TestGetCreditThresholds(t *testing.T) {
	useTempEnvFile(t, "", false)
	t.Setenv("CREDIT_THRESHOLD_MAX", "")
	t.Setenv("CREDIT_THRESHOLD_MIN", "")

	t.Run("command line max and min", func(t *testing.T) {
		max, min, useMax := GetCreditThresholds(70, 20)
		if max != 70 || min != 20 || !useMax {
			t.Fatalf("expected (70,20,true), got (%v,%v,%v)", max, min, useMax)
		}
	})

	t.Run("environment max", func(t *testing.T) {
		t.Setenv("CREDIT_THRESHOLD_MAX", "75")
		defer t.Setenv("CREDIT_THRESHOLD_MAX", "")
		max, min, useMax := GetCreditThresholds(0, 0)
		if max != 75 || min != 0 || !useMax {
			t.Fatalf("expected (75,0,true), got (%v,%v,%v)", max, min, useMax)
		}
	})

	t.Run("environment min no max", func(t *testing.T) {
		t.Setenv("CREDIT_THRESHOLD_MAX", "")
		t.Setenv("CREDIT_THRESHOLD_MIN", "55")
		defer t.Setenv("CREDIT_THRESHOLD_MIN", "")
		max, min, useMax := GetCreditThresholds(0, 0)
		if max != 0 || min != 55 || useMax {
			t.Fatalf("expected (0,55,false), got (%v,%v,%v)", max, min, useMax)
		}
	})

	t.Run("default fallback", func(t *testing.T) {
		t.Setenv("CREDIT_THRESHOLD_MIN", "")
		max, min, useMax := GetCreditThresholds(0, 0)
		if max != DefaultCreditThresholdMax || min != 0 || !useMax {
			t.Fatalf("expected default max, got (%v,%v,%v)", max, min, useMax)
		}
	})
}

func TestGetEnableFirstReset(t *testing.T) {
	useTempEnvFile(t, "", false)
	t.Setenv("ENABLE_FIRST_RESET", "")

	t.Run("command line true", func(t *testing.T) {
		if !GetEnableFirstReset(true) {
			t.Fatalf("expected true when cmd flag true")
		}
	})

	t.Run("environment", func(t *testing.T) {
		t.Setenv("ENABLE_FIRST_RESET", "1")
		defer t.Setenv("ENABLE_FIRST_RESET", "")
		if !GetEnableFirstReset(false) {
			t.Fatalf("expected true from env")
		}
	})

	t.Run("env file", func(t *testing.T) {
		t.Setenv("ENABLE_FIRST_RESET", "")
		useTempEnvFile(t, "ENABLE_FIRST_RESET=true\n", true)
		if !GetEnableFirstReset(false) {
			t.Fatalf("expected true from env file")
		}
	})

	t.Run("default false", func(t *testing.T) {
		t.Setenv("ENABLE_FIRST_RESET", "")
		useTempEnvFile(t, "", false)
		if GetEnableFirstReset(false) {
			t.Fatalf("expected default false")
		}
	})
}

func TestGetAllAPIKeys(t *testing.T) {
	useTempEnvFile(t, "", false)
	t.Setenv("API_KEYS", "")
	t.Setenv("API_KEY", "")

	t.Run("command line precedence", func(t *testing.T) {
		got := GetAllAPIKeys("keyA", "keyB,keyC")
		want := []string{"keyB", "keyC", "keyA"}
		if !slices.Equal(got, want) {
			t.Fatalf("cmd precedence got %v want %v", got, want)
		}
	})

	t.Run("environment fallback", func(t *testing.T) {
		t.Setenv("API_KEYS", "env1, env2")
		defer t.Setenv("API_KEYS", "")
		got := GetAllAPIKeys("", "")
		want := []string{"env1", "env2"}
		if !slices.Equal(got, want) {
			t.Fatalf("env fallback got %v want %v", got, want)
		}
	})

	t.Run("env file fallback", func(t *testing.T) {
		t.Setenv("API_KEYS", "")
		t.Setenv("API_KEY", "")
		useTempEnvFile(t, strings.Join([]string{"API_KEYS=file1,file2"}, "\n"), true)
		got := GetAllAPIKeys("", "")
		want := []string{"file1", "file2"}
		if !slices.Equal(got, want) {
			t.Fatalf("env file fallback got %v want %v", got, want)
		}
	})

	t.Run("single key env fallback", func(t *testing.T) {
		t.Setenv("API_KEYS", "")
		useTempEnvFile(t, "API_KEY=single\n", true)
		got := GetAllAPIKeys("", "")
		want := []string{"single"}
		if !slices.Equal(got, want) {
			t.Fatalf("single env fallback got %v want %v", got, want)
		}
	})
}
