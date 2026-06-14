package text

import (
	"testing"

	"github.com/slack-go/slack"
)

func TestIsUnfurlingEnabled(t *testing.T) {
	tests := []struct {
		name string
		opt  string
		text string
		want bool
	}{
		{
			name: "no domains",
			opt:  "example.com,foo.io",
			text: "Hello world, no domains here.",
			want: true,
		},
		{
			name: "allowed URL",
			opt:  "example.com,foo.io",
			text: "Check this link: http://example.com/page",
			want: true,
		},
		{
			name: "disallowed URL",
			opt:  "example.com,foo.io",
			text: "Visit http://bad.com now",
			want: false,
		},
		{
			name: "allowed bare domain",
			opt:  "example.com,foo.io",
			text: "Visit example.com for info.",
			want: true,
		},
		{
			name: "disallowed bare domain",
			opt:  "example.com,foo.io",
			text: "Visit bad.com for info.",
			want: false,
		},
		{
			name: "multiple allowed mixed",
			opt:  "example.com,foo.io",
			text: "example.com and foo.io and https://example.com/test",
			want: true,
		},
		{
			name: "one disallowed among many",
			opt:  "example.com,foo.io",
			text: "example.com and bar.org",
			want: false,
		},
		{
			name: "subdomain not allowed",
			opt:  "example.com,foo.io",
			text: "Visit sub.example.com",
			want: false,
		},
		{
			name: "bare domain with port",
			opt:  "example.com",
			text: "Service at example.com:8080 is running",
			want: true,
		},
		{
			name: "invalid TLD skipped",
			opt:  "example.com",
			text: "Check foo.invalidtld and example.com",
			want: true, // foo.invalidtld is ignored, example.com is allowed
		},
		{
			name: "allowed subdomain check",
			opt:  "sub.example.com,bar.com",
			text: "Check sub.example.com forsubdomain",
			want: true,
		},
		{
			name: "enable for all - YOLO mode",
			opt:  "yes",
			text: "YOLO mode, any link works http://anydomain.com",
			want: true,
		},
		{
			name: "enable for all - YOLO mode",
			opt:  "1",
			text: "YOLO mode, any link works http://anydomain.com",
			want: true,
		},
		{
			name: "enable for all - YOLO mode",
			opt:  "true",
			text: "YOLO mode, any link works http://anydomain.com",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUnfurlingEnabled(tt.text, tt.opt, nil)
			if got != tt.want {
				t.Fatalf("opt=%q text=%q → got %v; want %v",
					tt.opt, tt.text, got, tt.want)
			}
		})
	}
}

func TestFilterSpecialCharsWithCommas(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Slack-style link in middle",
			input:    "aaabbcc <https://google.com|This is a link> aabbcc",
			expected: "aaabbcc https://google.com - This is a link, aabbcc",
		},
		{
			name:     "Slack-style link at end",
			input:    "aaabbcc <https://google.com|This is a link>",
			expected: "aaabbcc https://google.com - This is a link",
		},
		{
			name:     "Slack-style link at end with spaces",
			input:    "aaabbcc <https://google.com|This is a link>   ",
			expected: "aaabbcc https://google.com - This is a link",
		},
		{
			name:     "Two links, second at end",
			input:    "First <https://site1.com|Site One> then <https://site2.com|Site Two>",
			expected: "First https://site1.com - Site One, then https://site2.com - Site Two",
		},
		{
			name:     "Two links, text after second",
			input:    "First <https://site1.com|Site One> then <https://site2.com|Site Two> done",
			expected: "First https://site1.com - Site One, then https://site2.com - Site Two, done",
		},
		{
			name:     "Markdown link at end",
			input:    "Check this [Google](https://google.com)",
			expected: "Check this https://google.com - Google",
		},
		{
			name:     "Markdown link in middle",
			input:    "Check this [Google](https://google.com) out",
			expected: "Check this https://google.com - Google, out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterSpecialChars(tt.input)
			if result != tt.expected {
				t.Errorf("filterSpecialChars() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestResolveUserMentions(t *testing.T) {
	users := map[string]slack.User{
		"U04T78D9XFF": {ID: "U04T78D9XFF", Name: "ngayal", RealName: "Nikhil Gayal"},
		"UBS6G3ZEF":   {ID: "UBS6G3ZEF", Name: "jlucky", RealName: "Jake Lucky"},
		"W01NAMEONLY":  {ID: "W01NAMEONLY", Name: "alice"},
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single mention",
			input:    "<@U04T78D9XFF> I cant submit an MR",
			expected: "@Nikhil Gayal I cant submit an MR",
		},
		{
			name:     "mention with display name hint",
			input:    "<@UBS6G3ZEF|jlucky> filed a bug",
			expected: "@Jake Lucky filed a bug",
		},
		{
			name:     "unknown user ID kept as-is",
			input:    "<@U99UNKNOWN> said something",
			expected: "<@U99UNKNOWN> said something",
		},
		{
			name:     "multiple mentions",
			input:    "<@U04T78D9XFF> and <@UBS6G3ZEF> are on the call",
			expected: "@Nikhil Gayal and @Jake Lucky are on the call",
		},
		{
			name:     "no mentions",
			input:    "just a regular message with no mentions",
			expected: "just a regular message with no mentions",
		},
		{
			name:     "W-prefix user ID",
			input:    "<@W01NAMEONLY> joined",
			expected: "@alice joined",
		},
		{
			name:     "mention in middle of text",
			input:    "cc <@U04T78D9XFF> for review",
			expected: "cc @Nikhil Gayal for review",
		},
		{
			name:     "nil users map",
			input:    "<@U04T78D9XFF> test",
			expected: "<@U04T78D9XFF> test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var u map[string]slack.User
			if tt.name != "nil users map" {
				u = users
			}
			result := ResolveUserMentions(tt.input, u)
			if result != tt.expected {
				t.Errorf("ResolveUserMentions() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
