package handler

import (
	"fmt"
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
