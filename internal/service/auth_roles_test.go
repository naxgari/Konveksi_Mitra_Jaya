package service

import (
	"github.com/anggyputra/go-class/internal/model"
	"testing"
)

func TestPeranSesuaiRancangan(t *testing.T) {
	for _, tc := range []struct {
		role    model.PeranPengguna
		allowed bool
	}{
		{model.PeranAdmin, true}, {model.PeranPemilik, true}, {model.PeranSupervisor, false},
	} {
		if got := peranValid(tc.role); got != tc.allowed {
			t.Errorf("peran %q: got %v, want %v", tc.role, got, tc.allowed)
		}
	}
}
