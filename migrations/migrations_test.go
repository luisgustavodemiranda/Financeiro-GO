package migrations

import "testing"

func TestPendingHistory(t *testing.T) {
	known := []migration{{1, "primeira"}, {2, "segunda"}}
	first := record{1, checksumSQL("primeira")}
	second := record{2, checksumSQL("segunda")}
	for _, tc := range []struct {
		name    string
		applied []record
		count   int
		invalid bool
	}{
		{"novo", nil, 2, false},
		{"incremental", []record{first}, 1, false},
		{"atualizado", []record{first, second}, 0, false},
		{"checksum alterado", []record{{1, "alterado"}}, 0, true},
		{"lacuna", []record{second}, 0, true},
		{"desconhecida", []record{first, second, {3, "terceira"}}, 0, true},
		{"duplicada", []record{first, first}, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := pending(known, tc.applied)
			if (err != nil) != tc.invalid {
				t.Fatalf("erro inesperado: %v", err)
			}
			if !tc.invalid && len(result) != tc.count {
				t.Fatal("lote pendente incorreto")
			}
			if tc.name == "incremental" && result[0].version != 2 {
				t.Fatal("versão aplicada seria repetida")
			}
		})
	}
}

func TestCatalogAndExistingChecksum(t *testing.T) {
	known, err := catalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(known) != 3 || known[2].version != 3 {
		t.Fatal("catálogo inesperado")
	}
	// Hash conferido nos bancos da fase 3: migration aplicada não pode ser editada.
	if checksumSQL(known[0].sql) != "aeaa37f7ed292cff6d042a53fb5c6167fce2795c5f7511ecf5ec4960b867971e" {
		t.Fatal("migration 0001 aplicada foi modificada")
	}
	if checksumSQL("a\r\nb\r\n") != checksumSQL("a\nb\n") {
		t.Fatal("hash depende da plataforma")
	}
	if checksumSQL(known[1].sql) != "497fe8ed741c58a31014a466f5c0c4cebe93c42c15b7ddc6b01b237e9221acee" {
		t.Fatal("migration 0002 aplicada foi modificada")
	}
}
