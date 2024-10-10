package prefix

import "testing"

func TestTrimPrefix(t *testing.T) {
	type args struct {
		org    string
		prefix string
		repl   string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "normal replace",
			args: args{
				org:    "cJSON_Init",
				prefix: "cJSON_",
				repl:   "JSON_",
			},
			want:    "JSON_Init",
			wantErr: false,
		},
		{
			name: "normal trim",
			args: args{
				org:    "cJSON_Init",
				prefix: "cJSON_",
			},
			want:    "Init",
			wantErr: false,
		},
		{
			name: "regexp replace",
			args: args{
				org:    "cJSON_Init",
				prefix: "^cJSON?_",
				repl:   "JSON_",
			},
			want:    "JSON_Init",
			wantErr: false,
		},
		{
			name: "regexp trim",
			args: args{
				org:    "cJSON_Init",
				prefix: "^cJSON?_",
			},
			want:    "Init",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TrimPrefix(tt.args.org, tt.args.prefix, tt.args.repl)
			if (err != nil) != tt.wantErr {
				t.Errorf("TrimPrefix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("TrimPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrimPrefixes(t *testing.T) {
	type args struct {
		org      string
		prefixes []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "normal trim",
			args: args{
				org:      "cJSON_Init",
				prefixes: []string{"cJSON_"},
			},
			want: "Init",
		},
		{
			name: "regexp trim",
			args: args{
				org:      "cJSON_Init",
				prefixes: []string{"^cJSON?_"},
			},
			want: "Init",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TrimPrefixes(tt.args.org, tt.args.prefixes); got != tt.want {
				t.Errorf("TrimPrefixes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReplacePrefixes(t *testing.T) {
	type args struct {
		org      string
		prefixes []string
		repls    []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "normal replace",
			args: args{
				org:      "cJSON_Init",
				prefixes: []string{"cJSON_"},
				repls:    []string{"JSON_"},
			},
			want: "JSON_Init",
		},
		{
			name: "regexp replace",
			args: args{
				org:      "cJSON_Init",
				prefixes: []string{"^cJSON?_"},
				repls:    []string{"JSON_"},
			},
			want: "JSON_Init",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReplacePrefixes(tt.args.org, tt.args.prefixes, tt.args.repls); got != tt.want {
				t.Errorf("ReplacePrefixes() = %v, want %v", got, tt.want)
			}
		})
	}
}
