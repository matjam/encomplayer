package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	const def = "/home/u/.config/encomplayer/config.json"
	tests := []struct {
		name      string
		args      []string
		want      options
		wantUsage bool
	}{
		{name: "no args", args: nil, want: options{config: def}},
		{name: "short version", args: []string{"-v"}, want: options{config: def, version: true}},
		{name: "long version", args: []string{"--version"}, want: options{config: def, version: true}},
		{name: "short help", args: []string{"-h"}, want: options{config: def, help: true}},
		{name: "long help", args: []string{"--help"}, want: options{config: def, help: true}},
		{name: "short config", args: []string{"-c", "/tmp/c.json"}, want: options{config: "/tmp/c.json"}},
		{name: "long config with =", args: []string{"--config=/tmp/c.json"}, want: options{config: "/tmp/c.json"}},
		{name: "attached short value", args: []string{"-aoff"}, want: options{config: def, art: "off"}},
		{name: "art and folder in any order", args: []string{"/music", "--art", "blocks"}, want: options{config: def, art: "blocks", musicDir: "/music"}},
		{name: "end of options", args: []string{"--", "-weird-folder"}, want: options{config: def, musicDir: "-weird-folder"}},
		{name: "legacy -version", args: []string{"-version"}, want: options{config: def, version: true}},
		{name: "legacy -config value", args: []string{"-config", "/tmp/c.json", "/music"}, want: options{config: "/tmp/c.json", musicDir: "/music"}},
		{name: "legacy -art=value", args: []string{"-art=off"}, want: options{config: def, art: "off"}},
		{name: "legacy form after -- is a folder", args: []string{"--", "-config"}, want: options{config: def, musicDir: "-config"}},
		{name: "unknown flag", args: []string{"-x"}, wantUsage: true},
		{name: "unknown single-dash long flag", args: []string{"-verbose"}, wantUsage: true},
		{name: "missing value", args: []string{"--config"}, wantUsage: true},
		{name: "two folders", args: []string{"/a", "/b"}, wantUsage: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseArgs(tc.args, def)
			if tc.wantUsage {
				if !isUsage(err) {
					t.Fatalf("parseArgs(%q) error = %v, want a usage error", tc.args, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseArgs(%q): %v", tc.args, err)
			}
			if got != tc.want {
				t.Errorf("parseArgs(%q) = %+v, want %+v", tc.args, got, tc.want)
			}
		})
	}
}

func TestUsageListsShortAndLongForms(t *testing.T) {
	var b bytes.Buffer
	printUsage(&b, "/cfg.json")
	for _, want := range []string{"-a, --art", "-c, --config", "-h, --help", "-v, --version", "[music-folder]"} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("usage missing %q:\n%s", want, b.String())
		}
	}
}
