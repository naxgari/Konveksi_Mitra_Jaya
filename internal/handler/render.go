package handler

import (
	"bytes"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/anggyputra/go-class/internal/model"
)

// rupiah memformat angka int jadi "Rp39.000.000"
func rupiah(n int) string {
	negatif := n < 0
	if negatif {
		n = -n
	}
	s := strconv.Itoa(n)
	hasil := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			hasil += "."
		}
		hasil += string(c)
	}
	if negatif {
		return "-Rp" + hasil
	}
	return "Rp" + hasil
}

// rupiahFloat memformat angka desimal float64 (dibulatkan ke rupiah terdekat) jadi "Rp150.000"
func rupiahFloat(n float64) string {
	return rupiah(int(math.Round(n)))
}

var namaBulanIndo = [...]string{
	"", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun",
	"Jul", "Agu", "Sep", "Okt", "Nov", "Des",
}

// tglIndo memformat time.Time menjadi "02 Nov 2026"
func tglIndo(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	m := int(t.Month())
	bln := ""
	if m >= 1 && m <= 12 {
		bln = namaBulanIndo[m]
	}
	return t.Format("02 ") + bln + t.Format(" 2006")
}

func addOne(n int) int {
	return n + 1
}

var funcs = template.FuncMap{
	"rupiah":       rupiah,
	"rupiah_float": rupiahFloat,
	"tgl_indo":     tglIndo,
	"add_one":      addOne,
	"currentUser":  func() *model.Pengguna { return nil },
}

type penggunaCarrier interface {
	CurrentPengguna() *model.Pengguna
}

type Renderer struct {
	tpl map[string]*template.Template
}

func NewRenderer() *Renderer {
	halamanWajib := []string{
		"home.html",
		"modul.html",
		"profil.html",
		"auth/login.html",
		"auth/register.html",
		"produk/list.html",
		"produk/form.html",
		"produk/detail.html",
	}

	// Daftar template modul Anggota 2 s.d. 5 (otomatis di-parse begitu file HTML-nya dibuat)
	halamanModulTim := []string{
		"pekerja/list.html",
		"pekerja/form.html",
		"pekerja/detail.html",
		"absensi/jadwal_list.html",
		"absensi/jadwal_form.html",
		"absensi/absensi_form.html",
		"absensi/pemeriksaan.html",
		"gaji/periode_list.html",
		"gaji/periode_form.html",
		"gaji/periode_detail.html",
		"gaji/gaji_detail.html",
		"kasbon/kasbon_list.html",
		"kasbon/kasbon_form.html",
		"kasbon/koreksi_list.html",
		"kasbon/koreksi_form.html",
		"kasbon/slip.html",
		"kasbon/rekap.html",
	}

	rd := &Renderer{tpl: make(map[string]*template.Template)}
	for _, file := range halamanWajib {
		rd.tpl[file] = parseDenganLayout(file)
	}
	for _, file := range halamanModulTim {
		if _, err := os.Stat("templates/" + file); err == nil {
			rd.tpl[file] = parseDenganLayout(file)
		}
	}
	return rd
}

func parseDenganLayout(file string) *template.Template {
	return template.Must(template.New("layout").Funcs(funcs).ParseFiles(
		"templates/layout.html",
		"templates/header.html",
		"templates/sidebar.html",
		"templates/footer.html",
		"templates/"+file,
	))
}

// Tampil me-render template ke buffer dulu supaya error template tidak menghasilkan halaman setengah jadi.
// Jika response writer berasal dari middleware RequireLogin, fungsi template `currentUser` otomatis berisi data pengguna aktif.
func (rd *Renderer) Tampil(w http.ResponseWriter, file string, data any) {
	baseTpl, ok := rd.tpl[file]
	if !ok {
		log.Printf("template %s belum terdaftar", file)
		http.Error(w, "Template halaman tidak ditemukan", http.StatusInternalServerError)
		return
	}

	var penggunaAktif *model.Pengguna
	if pc, ok := w.(penggunaCarrier); ok {
		penggunaAktif = pc.CurrentPengguna()
	}

	tmpl, err := baseTpl.Clone()
	if err != nil {
		log.Printf("clone template %s: %v", file, err)
		http.Error(w, "Terjadi kesalahan pada server", http.StatusInternalServerError)
		return
	}
	tmpl.Funcs(template.FuncMap{
		"currentUser": func() *model.Pengguna { return penggunaAktif },
	})

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "layout", data); err != nil {
		log.Printf("render %s: %v", file, err)
		http.Error(w, "Terjadi kesalahan pada server", http.StatusInternalServerError)
		return
	}
	_, _ = buf.WriteTo(w)
}
