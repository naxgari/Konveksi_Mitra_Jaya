package handler

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"strconv"
)

// rupiah memformat angka jadi "Rp39.000.000"
func rupiah(n int) string {
	s := strconv.Itoa(n)
	hasil := ""
	for i, c := range s {
		// sisipkan titik tiap 3 digit dari kanan, kecuali setelah tanda minus
		if i > 0 && (len(s)-i)%3 == 0 && s[i-1] != '-' {
			hasil += "."
		}
		hasil += string(c)
	}
	return "Rp" + hasil
}

func addOne(n int) int {
	return n + 1
}

var funcs = template.FuncMap{"rupiah": rupiah, "add_one": addOne}

type Renderer struct {
	tpl map[string]*template.Template
}

func NewRenderer() *Renderer {
	halaman := []string{"home.html", "profil.html", "produk/list.html", "produk/form.html", "produk/detail.html"}

	rd := &Renderer{tpl: make(map[string]*template.Template)}
	for _, file := range halaman {
		rd.tpl[file] = template.Must(template.New("layout").Funcs(funcs).ParseFiles(
			"templates/layout.html",
			"templates/header.html",
			"templates/sidebar.html",
			"templates/footer.html",
			"templates/"+file,
		))
	}
	return rd
}

// Tampil me-render ke buffer dulu supaya error template tidak menghasilkan halaman setengah jadi.
func (rd *Renderer) Tampil(w http.ResponseWriter, file string, data any) {
	var buf bytes.Buffer
	if err := rd.tpl[file].ExecuteTemplate(&buf, "layout", data); err != nil {
		log.Printf("render %s: %v", file, err)
		http.Error(w, "Terjadi kesalahan pada server", http.StatusInternalServerError)
		return
	}
	buf.WriteTo(w)
}
