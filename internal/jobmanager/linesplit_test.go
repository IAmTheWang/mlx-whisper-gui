package jobmanager

import (
	"bufio"
	"strings"
	"testing"
)

type splitResult struct {
	text string
	term byte
}

func runSplitter(t *testing.T, input string) []splitResult {
	t.Helper()
	s := &lineSplitter{}
	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Split(s.split)
	var results []splitResult
	for scanner.Scan() {
		results = append(results, splitResult{text: scanner.Text(), term: s.lastTerm})
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}
	return results
}

func TestLineSplitter_PlainNewline(t *testing.T) {
	got := runSplitter(t, "hello\nworld\n")
	want := []splitResult{{"hello", '\n'}, {"world", '\n'}}
	assertEqual(t, got, want)
}

func TestLineSplitter_BareCR_TqdmRedraw(t *testing.T) {
	got := runSplitter(t, "10%|#\r20%|##\r30%|###\n")
	want := []splitResult{
		{"10%|#", '\r'},
		{"20%|##", '\r'},
		{"30%|###", '\n'},
	}
	assertEqual(t, got, want)
}

func TestLineSplitter_CRLF_TreatedAsOrdinaryNewline(t *testing.T) {
	got := runSplitter(t, "line one\r\nline two\r\n")
	want := []splitResult{{"line one", '\n'}, {"line two", '\n'}}
	assertEqual(t, got, want)
}

func TestLineSplitter_UnterminatedFragmentAtEOF(t *testing.T) {
	got := runSplitter(t, "no newline at all")
	want := []splitResult{{"no newline at all", 0}}
	assertEqual(t, got, want)
}

func TestLineSplitter_Mixed(t *testing.T) {
	got := runSplitter(t, "start\r\nmid\rprogress\rdone\ntrailing")
	want := []splitResult{
		{"start", '\n'},
		{"mid", '\r'},
		{"progress", '\r'},
		{"done", '\n'},
		{"trailing", 0},
	}
	assertEqual(t, got, want)
}

func TestLineSplitter_CRSplitAcrossReads(t *testing.T) {
	// Regression check: a '\r' landing exactly at a Read() buffer boundary must
	// not be split from a '\n' that arrives in the next chunk -- otherwise a
	// \r\n pair straddling two pipe reads would wrongly emit an extra empty
	// log line. bufio.Scanner's default buffer is large enough that a plain
	// strings.Reader won't fragment this on its own, so drive the SplitFunc
	// directly to simulate the boundary.
	s := &lineSplitter{}
	adv, tok, err := s.split([]byte("abc\r"), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adv != 0 || tok != nil {
		t.Fatalf("expected split to request more data when trailing \\r might be part of \\r\\n, got advance=%d token=%q", adv, tok)
	}

	adv, tok, err = s.split([]byte("abc\r\n"), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adv != 5 || string(tok) != "abc" || s.lastTerm != '\n' {
		t.Fatalf("expected full \\r\\n pair consumed as one newline-terminated token, got advance=%d token=%q term=%q", adv, tok, s.lastTerm)
	}
}

func assertEqual(t *testing.T, got, want []splitResult) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d results, want %d\ngot:  %+v\nwant: %+v", len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("result[%d]: got %+v, want %+v\nfull got:  %+v\nfull want: %+v", i, got[i], want[i], got, want)
		}
	}
}
