package protocol_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"example.com/kvd/internal/protocol"
)

func TestRequestRoundTrip(t *testing.T) {
	cases := []protocol.Request{
		{Op: protocol.OpGet, Key: "alpha"},
		{Op: protocol.OpDel, Key: "à_supprimer"},
		{Op: protocol.OpSet, Key: "k", Value: []byte("une valeur binaire \x00\x01")},
		{Op: protocol.OpSet, Key: "vide", Value: []byte{}},
		{Op: protocol.OpPing},
	}
	for _, want := range cases {
		t.Run(want.Key+"/"+string(rune('0'+byte(want.Op))), func(t *testing.T) {
			var buf bytes.Buffer
			if err := protocol.WriteRequest(&buf, want); err != nil {
				t.Fatalf("WriteRequest : %v", err)
			}
			got, err := protocol.ReadRequest(&buf)
			if err != nil {
				t.Fatalf("ReadRequest : %v", err)
			}
			if got.Op != want.Op || got.Key != want.Key || !bytes.Equal(got.Value, want.Value) {
				t.Errorf("aller-retour : got %+v, voulu %+v", got, want)
			}
		})
	}
}

func TestResponseRoundTrip(t *testing.T) {
	cases := []protocol.Response{
		{Status: protocol.StatusOK, Value: []byte("résultat")},
		{Status: protocol.StatusNotFound},
		{Status: protocol.StatusError, Err: "clé interdite"},
	}
	for _, want := range cases {
		var buf bytes.Buffer
		if err := protocol.WriteResponse(&buf, want); err != nil {
			t.Fatalf("WriteResponse : %v", err)
		}
		got, err := protocol.ReadResponse(&buf)
		if err != nil {
			t.Fatalf("ReadResponse : %v", err)
		}
		if got.Status != want.Status || got.Err != want.Err || !bytes.Equal(got.Value, want.Value) {
			t.Errorf("aller-retour : got %+v, voulu %+v", got, want)
		}
	}
}

func TestReadFrameTruncated(t *testing.T) {
	// En-tête annonçant 10 octets, mais seulement 3 fournis : doit échouer proprement.
	var buf bytes.Buffer
	buf.Write([]byte{0, 0, 0, 10})
	buf.Write([]byte{1, 2, 3})
	if _, err := protocol.ReadFrame(&buf); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("frame tronquée : err = %v, voulu io.ErrUnexpectedEOF", err)
	}
}

func TestReadFrameTooLarge(t *testing.T) {
	var buf bytes.Buffer
	buf.Write([]byte{0xff, 0xff, 0xff, 0xff}) // ~4 Gio annoncés
	if _, err := protocol.ReadFrame(&buf); err == nil {
		t.Error("frame surdimensionnée : une erreur était attendue")
	}
}

func TestUnmarshalMalformed(t *testing.T) {
	// OpSet sans clé ni valeur : payload tronqué.
	var req protocol.Request
	if err := req.UnmarshalBinary([]byte{byte(protocol.OpSet)}); err == nil {
		t.Error("payload malformé : une erreur était attendue")
	}
}
