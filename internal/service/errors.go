package service

type ValidasiError struct {
	Pesan string
}

func (e *ValidasiError) Error() string { return e.Pesan }
