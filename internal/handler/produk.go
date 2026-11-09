package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/anggyputra/go-class/internal/model"
	"github.com/anggyputra/go-class/internal/service"
)

type ProdukForm struct {
	Judul  string
	Action string
	Produk model.Produk
}

type ProdukHandler struct {
	Svc  *service.ProdukService
	View *Renderer
}

func (h *ProdukHandler) Daftar(w http.ResponseWriter, r *http.Request) {
	daftar, err := h.Svc.Daftar(r.Context())
	if err != nil {
		gagal(w, err)
		return
	}
	h.View.Tampil(w, "produk/list.html", daftar)
}

func (h *ProdukHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	p, err := h.Svc.Detail(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}
	h.View.Tampil(w, "produk/detail.html", p)
}

func (h *ProdukHandler) FormTambah(w http.ResponseWriter, r *http.Request) {
	h.View.Tampil(w, "produk/form.html", ProdukForm{
		Judul:  "Form Tambah Data Produk",
		Action: "/produk",
	})
}

func (h *ProdukHandler) Simpan(w http.ResponseWriter, r *http.Request) {
	if _, err := h.Svc.Buat(r.Context(), produkDariForm(r)); err != nil {
		gagal(w, err)
		return
	}
	http.Redirect(w, r, "/produk", http.StatusSeeOther)
}

func (h *ProdukHandler) FormEdit(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	p, err := h.Svc.Detail(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}
	h.View.Tampil(w, "produk/form.html", ProdukForm{
		Judul:  "Form Update Produk",
		Action: fmt.Sprintf("/produk/%d/edit", id),
		Produk: p,
	})
}

func (h *ProdukHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	p := produkDariForm(r)
	p.ID = id
	if err := h.Svc.Ubah(r.Context(), p); err != nil {
		gagal(w, err)
		return
	}
	http.Redirect(w, r, "/produk", http.StatusSeeOther)
}

func (h *ProdukHandler) Hapus(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if err := h.Svc.Hapus(r.Context(), id); err != nil {
		gagal(w, err)
		return
	}
	http.Redirect(w, r, "/produk", http.StatusSeeOther)
}

func produkDariForm(r *http.Request) model.Produk {
	price, _ := strconv.Atoi(r.PostFormValue("price"))
	stok, _ := strconv.Atoi(r.PostFormValue("stok"))
	return model.Produk{
		Name:  r.PostFormValue("name"),
		Price: price,
		Stok:  stok,
		Ket:   r.PostFormValue("keterangan"),
	}
}

func idDariPath(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "ID tidak valid", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

// gagal menerjemahkan error dari service menjadi status HTTP.
func gagal(w http.ResponseWriter, err error) {
	var ve *service.ValidasiError
	switch {
	case errors.As(err, &ve):
		http.Error(w, ve.Pesan, http.StatusBadRequest)
	case errors.Is(err, model.ErrNotFound):
		http.Error(w, "Data tidak ditemukan", http.StatusNotFound)
	default:
		log.Printf("error: %v", err)
		http.Error(w, "Terjadi kesalahan pada server", http.StatusInternalServerError)
	}
}
