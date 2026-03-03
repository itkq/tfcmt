package controller

import (
	"regexp"
	"testing"

	"github.com/suzuki-shunsuke/tfcmt/v4/pkg/config"
)

func TestHasUnignoredWarnings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		warning        string
		ignoreWarnings []config.IgnoreWarning
		target         string
		want           bool
	}{
		{
			name:    "empty warning",
			warning: "",
			want:    false,
		},
		{
			name:    "whitespace only warning",
			warning: "  \n  ",
			want:    false,
		},
		{
			name:    "warning present, no ignore rules",
			warning: "Warning: Resource targeting is in effect",
			want:    true,
		},
		{
			name:    "warning matches ignore pattern",
			warning: "Warning: Resource targeting is in effect",
			ignoreWarnings: []config.IgnoreWarning{
				{
					Warning:       "Resource targeting",
					WarningRegexp: regexp.MustCompile("Resource targeting"),
				},
			},
			want: false,
		},
		{
			name:    "warning does not match ignore pattern",
			warning: "Warning: Something else",
			ignoreWarnings: []config.IgnoreWarning{
				{
					Warning:       "Resource targeting",
					WarningRegexp: regexp.MustCompile("Resource targeting"),
				},
			},
			want: true,
		},
		{
			name:    "target matches, warning matches",
			warning: "Warning: Resource targeting is in effect",
			target:  "foo",
			ignoreWarnings: []config.IgnoreWarning{
				{
					Target:        "foo",
					TargetRegexp:  regexp.MustCompile("foo"),
					Warning:       "Resource targeting",
					WarningRegexp: regexp.MustCompile("Resource targeting"),
				},
			},
			want: false,
		},
		{
			name:    "target does not match, warning matches",
			warning: "Warning: Resource targeting is in effect",
			target:  "bar",
			ignoreWarnings: []config.IgnoreWarning{
				{
					Target:        "foo",
					TargetRegexp:  regexp.MustCompile("foo"),
					Warning:       "Resource targeting",
					WarningRegexp: regexp.MustCompile("Resource targeting"),
				},
			},
			want: true,
		},
		{
			name:    "no target regexp, warning matches",
			warning: "Warning: Deprecated attribute",
			target:  "any-target",
			ignoreWarnings: []config.IgnoreWarning{
				{
					Warning:       "Deprecated attribute",
					WarningRegexp: regexp.MustCompile("Deprecated attribute"),
				},
			},
			want: false,
		},
		{
			name:    "target matches via regex",
			warning: "Warning: Something",
			target:  "my-service-prod",
			ignoreWarnings: []config.IgnoreWarning{
				{
					Target:        "my-service-.*",
					TargetRegexp:  regexp.MustCompile("my-service-.*"),
					Warning:       "Something",
					WarningRegexp: regexp.MustCompile("Something"),
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := hasUnignoredWarnings(tt.warning, tt.ignoreWarnings, tt.target)
			if got != tt.want {
				t.Errorf("hasUnignoredWarnings() = %v, want %v", got, tt.want)
			}
		})
	}
}
