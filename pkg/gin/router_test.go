package gin

import (
	"testing"

	"github.com/DougTea/go-common/pkg/web"
)

func TestHttpMethod_CamelString(t *testing.T) {
	tests := []struct {
		name string
		h    HttpMethod
		want string
	}{
		{
			name: "GET",
			h:    HttpMethod(web.MethodGet),
			want: "Get",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.h.CamelString(); got != tt.want {
				t.Errorf("HttpMethod.CamelString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHttpMethod_LowerString(t *testing.T) {
	tests := []struct {
		name string
		h    HttpMethod
		want string
	}{
		{
			name: "GET",
			h:    HttpMethod(web.MethodGet),
			want: "get",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.h.LowerString(); got != tt.want {
				t.Errorf("HttpMethod.LowerString() = %v, want %v", got, tt.want)
			}
		})
	}
}
