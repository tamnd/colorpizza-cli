package color

import "testing"

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "color" {
		t.Errorf("Scheme = %q, want color", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "color" {
		t.Errorf("Binary = %q, want color", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	_, _, err := Domain{}.Classify("")
	if err == nil {
		t.Error("Classify empty string should return error")
	}

	typ, id, err := Domain{}.Classify("ff0000")
	if err != nil {
		t.Errorf("Classify: unexpected error: %v", err)
	}
	if typ != "color" {
		t.Errorf("Classify type = %q, want color", typ)
	}
	if id != "ff0000" {
		t.Errorf("Classify id = %q, want ff0000", id)
	}

	// strip # prefix
	typ2, id2, err2 := Domain{}.Classify("#ff0000")
	if err2 != nil {
		t.Errorf("Classify #ff0000: unexpected error: %v", err2)
	}
	if typ2 != "color" || id2 != "ff0000" {
		t.Errorf("Classify #ff0000 = (%q, %q), want (color, ff0000)", typ2, id2)
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("color", "ff0000")
	if err != nil {
		t.Fatalf("Locate: unexpected error: %v", err)
	}
	if got == "" {
		t.Error("Locate returned empty URL")
	}

	_, err = Domain{}.Locate("unknown", "x")
	if err == nil {
		t.Error("Locate with unknown type should return error")
	}
}

func TestParseHex(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"ff0000", []string{"ff0000"}},
		{"#ff0000", []string{"ff0000"}},
		{"ff0000,00ff00", []string{"ff0000", "00ff00"}},
		{"#ff0000, #00ff00", []string{"ff0000", "00ff00"}},
		{"", nil},
	}
	for _, tc := range cases {
		got := parseHex(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("parseHex(%q) = %v, want %v", tc.in, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("parseHex(%q)[%d] = %q, want %q", tc.in, i, got[i], tc.want[i])
			}
		}
	}
}
