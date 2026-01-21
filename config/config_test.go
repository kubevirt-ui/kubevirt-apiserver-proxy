package config

import (
	"flag"
	"fmt"
	"math"
	"reflect"
	"testing"
)

func TestGetTLSCipherSuites(t *testing.T) {
	for _, tc := range []struct {
		name    string
		flagVal string
		want    []uint16
	}{
		{name: "valid input", flagVal: "1,2,3,4", want: []uint16{1, 2, 3, 4}},
		{name: "no flag", flagVal: "", want: nil},
		{name: "max val", flagVal: fmt.Sprintf("%d", math.MaxUint16), want: []uint16{math.MaxUint16}},
	} {
		t.Run(tc.flagVal, func(t *testing.T) {
			err := setFlags("0", tc.flagVal)
			if err != nil {
				t.Fatal(err)
			}

			cfg, err := GetConfig()
			if err != nil {
				t.Fatal(err)
			}

			if got, want := cfg.GetTLSCipherSuites(), tc.want; !reflect.DeepEqual(got, want) {
				t.Errorf("GetTLSCipherSuites() = %v, want %v", got, want)
			}
		})
	}
}

func TestWrongGetTLSCipherSuites_tooLarge(t *testing.T) {
	err := setFlags("0", fmt.Sprintf("%d", math.MaxUint16+1))
	if err != nil {
		t.Fatal(err)
	}

	_, err = GetConfig()
	if err == nil {
		t.Fatal("error should have errored")
	}

	t.Logf("got expected error: %v", err)
}

func TestWrongGetTLSCipherSuites_negative(t *testing.T) {
	err := setFlags("0", "-42")
	if err != nil {
		t.Fatal(err)
	}

	_, err = GetConfig()
	if err == nil {
		t.Fatal("error should have errored")
	}

	t.Logf("got expected error: %v", err)
}

func TestWrongGetTLSCipherSuites_notNum(t *testing.T) {
	err := setFlags("0", "not a number")
	if err != nil {
		t.Fatal(err)
	}

	_, err = GetConfig()
	if err == nil {
		t.Fatal("error should have errored")
	}

	t.Logf("got expected error: %v", err)
}

func TestWrongGetTLSCipherSuites_float(t *testing.T) {
	err := setFlags("0", "1.42")
	if err != nil {
		t.Fatal(err)
	}

	_, err = GetConfig()
	if err == nil {
		t.Fatal("error should have errored")
	}

	t.Logf("got expected error: %v", err)
}

func TestGetMinTLSVersion(t *testing.T) {
	for _, tc := range []struct {
		name    string
		flagVal string
		want    uint16
	}{
		{name: "valid input", flagVal: "42", want: 42},
		{name: "no flag", flagVal: "0", want: 0},
		{name: "max val", flagVal: fmt.Sprintf("%d", math.MaxUint16), want: math.MaxUint16},
	} {
		t.Run(tc.flagVal, func(t *testing.T) {
			err := setFlags(tc.flagVal, "")
			if err != nil {
				t.Fatal(err)
			}

			cfg, err := GetConfig()
			if err != nil {
				t.Fatal(err)
			}

			if got, want := cfg.GetMinTLSVersion(), tc.want; !reflect.DeepEqual(got, want) {
				t.Errorf("GetMinTLSVersion() = %d, want %d", got, want)
			}
		})
	}
}

func TestWrongGetMinTLSVersion_tooLarge(t *testing.T) {
	err := setFlags(fmt.Sprintf("%d", math.MaxUint16+1), "")
	if err != nil {
		t.Fatal(err)
	}

	_, err = GetConfig()
	if err == nil {
		t.Fatal("error should have errored")
	}

	t.Logf("got expected error: %v", err)
}

func setFlags(minVer, ciphers string) error {
	err := flag.Set("tls-min-version", minVer)
	if err != nil {
		return err
	}
	err = flag.Set("tls-cipher-suites", ciphers)
	if err != nil {
		return err
	}

	return nil
}
