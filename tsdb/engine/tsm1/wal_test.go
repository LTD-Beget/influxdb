package tsm1_test

import (
	"os"
	"testing"
	"time"

	"github.com/influxdb/influxdb/tsdb/engine/tsm1"
)

func TestWALWriter_WritePoints_Single(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)
	w := tsm1.NewWALSegmentWriter(f)

	p1 := tsm1.NewValue(time.Unix(1, 0), 1.1)
	p2 := tsm1.NewValue(time.Unix(1, 0), int64(1))
	p3 := tsm1.NewValue(time.Unix(1, 0), true)
	p4 := tsm1.NewValue(time.Unix(1, 0), "string")

	values := map[string][]tsm1.Value{
		"cpu,host=A#!~#float":  []tsm1.Value{p1},
		"cpu,host=A#!~#int":    []tsm1.Value{p2},
		"cpu,host=A#!~#bool":   []tsm1.Value{p3},
		"cpu,host=A#!~#string": []tsm1.Value{p4},
	}

	entry := &tsm1.WriteWALEntry{
		Values: values,
	}

	if err := w.Write(entry); err != nil {
		fatal(t, "write points", err)
	}

	if _, err := f.Seek(0, os.SEEK_SET); err != nil {
		fatal(t, "seek", err)
	}

	r := tsm1.NewWALSegmentReader(f)

	if !r.Next() {
		t.Fatalf("expected next, got false")
	}

	we, err := r.Read()
	if err != nil {
		fatal(t, "read entry", err)
	}

	e, ok := we.(*tsm1.WriteWALEntry)
	if !ok {
		t.Fatalf("expected WriteWALEntry: got %#v", e)
	}

	for k, v := range e.Values {
		for i, vv := range v {
			if got, exp := vv.String(), values[k][i].String(); got != exp {
				t.Fatalf("points mismatch: got %v, exp %v", got, exp)
			}
		}
	}
}

func TestWALWriter_WritePoints_Multiple(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)
	w := tsm1.NewWALSegmentWriter(f)

	p1 := tsm1.NewValue(time.Unix(1, 0), int64(1))
	p2 := tsm1.NewValue(time.Unix(1, 0), int64(2))

	exp := []struct {
		key    string
		values []tsm1.Value
	}{
		{"cpu,host=A#!~#value", []tsm1.Value{p1}},
		{"cpu,host=B#!~#value", []tsm1.Value{p2}},
	}

	for _, v := range exp {
		entry := &tsm1.WriteWALEntry{
			Values: map[string][]tsm1.Value{v.key: v.values},
		}

		if err := w.Write(entry); err != nil {
			fatal(t, "write points", err)
		}
	}

	// Seek back to the beinning of the file for reading
	if _, err := f.Seek(0, os.SEEK_SET); err != nil {
		fatal(t, "seek", err)
	}

	r := tsm1.NewWALSegmentReader(f)

	for _, ep := range exp {
		if !r.Next() {
			t.Fatalf("expected next, got false")
		}

		we, err := r.Read()
		if err != nil {
			fatal(t, "read entry", err)
		}

		e, ok := we.(*tsm1.WriteWALEntry)
		if !ok {
			t.Fatalf("expected WriteWALEntry: got %#v", e)
		}

		for k, v := range e.Values {
			if got, exp := k, ep.key; got != exp {
				t.Fatalf("key mismatch. got %v, exp %v", got, exp)
			}

			if got, exp := len(v), len(ep.values); got != exp {
				t.Fatalf("values length mismatch: got %v, exp %v", got, exp)
			}

			for i, vv := range v {
				if got, exp := vv.String(), ep.values[i].String(); got != exp {
					t.Fatalf("points mismatch: got %v, exp %v", got, exp)
				}
			}
		}
	}
}

func TestWALWriter_WriteDelete_Single(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)
	w := tsm1.NewWALSegmentWriter(f)

	entry := &tsm1.DeleteWALEntry{
		Keys: []string{"cpu"},
	}

	if err := w.Write(entry); err != nil {
		fatal(t, "write points", err)
	}

	if _, err := f.Seek(0, os.SEEK_SET); err != nil {
		fatal(t, "seek", err)
	}

	r := tsm1.NewWALSegmentReader(f)

	if !r.Next() {
		t.Fatalf("expected next, got false")
	}

	we, err := r.Read()
	if err != nil {
		fatal(t, "read entry", err)
	}

	e, ok := we.(*tsm1.DeleteWALEntry)
	if !ok {
		t.Fatalf("expected WriteWALEntry: got %#v", e)
	}

	if got, exp := len(e.Keys), len(entry.Keys); got != exp {
		t.Fatalf("key length mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := e.Keys[0], entry.Keys[0]; got != exp {
		t.Fatalf("key mismatch: got %v, exp %v", got, exp)
	}
}

func TestWALWriter_WritePointsDelete_Multiple(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)
	w := tsm1.NewWALSegmentWriter(f)

	p1 := tsm1.NewValue(time.Unix(1, 0), true)
	values := map[string][]tsm1.Value{
		"cpu,host=A#!~#value": []tsm1.Value{p1},
	}

	writeEntry := &tsm1.WriteWALEntry{
		Values: values,
	}

	if err := w.Write(writeEntry); err != nil {
		fatal(t, "write points", err)
	}

	// Write the delete entry
	deleteEntry := &tsm1.DeleteWALEntry{
		Keys: []string{"cpu,host=A#!~value"},
	}

	if err := w.Write(deleteEntry); err != nil {
		fatal(t, "write points", err)
	}

	// Seek back to the beinning of the file for reading
	if _, err := f.Seek(0, os.SEEK_SET); err != nil {
		fatal(t, "seek", err)
	}

	r := tsm1.NewWALSegmentReader(f)

	// Read the write points first
	if !r.Next() {
		t.Fatalf("expected next, got false")
	}

	we, err := r.Read()
	if err != nil {
		fatal(t, "read entry", err)
	}

	e, ok := we.(*tsm1.WriteWALEntry)
	if !ok {
		t.Fatalf("expected WriteWALEntry: got %#v", e)
	}

	for k, v := range e.Values {
		if got, exp := len(v), len(values[k]); got != exp {
			t.Fatalf("values length mismatch: got %v, exp %v", got, exp)
		}

		for i, vv := range v {
			if got, exp := vv.String(), values[k][i].String(); got != exp {
				t.Fatalf("points mismatch: got %v, exp %v", got, exp)
			}
		}
	}

	// Read the delete second
	if !r.Next() {
		t.Fatalf("expected next, got false")
	}

	we, err = r.Read()
	if err != nil {
		fatal(t, "read entry", err)
	}

	de, ok := we.(*tsm1.DeleteWALEntry)
	if !ok {
		t.Fatalf("expected DeleteWALEntry: got %#v", e)
	}

	if got, exp := len(de.Keys), len(deleteEntry.Keys); got != exp {
		t.Fatalf("key length mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := de.Keys[0], deleteEntry.Keys[0]; got != exp {
		t.Fatalf("key mismatch: got %v, exp %v", got, exp)
	}
}

func TestWAL_ClosedSegments(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)

	w := tsm1.NewWAL(dir)
	if err := w.Open(); err != nil {
		t.Fatalf("error opening WAL: %v", err)
	}

	files, err := w.ClosedSegments()
	if err != nil {
		t.Fatalf("error getting closed segments: %v", err)
	}

	if got, exp := len(files), 0; got != exp {
		t.Fatalf("close segment length mismatch: got %v, exp %v", got, exp)
	}

	if err := w.WritePoints(map[string][]tsm1.Value{
		"cpu,host=A#!~#value": []tsm1.Value{
			tsm1.NewValue(time.Unix(1, 0), 1.1),
		},
	}); err != nil {
		t.Fatalf("error writing points: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("error closing wal: %v", err)
	}

	// Re-open the WAL
	w = tsm1.NewWAL(dir)
	defer w.Close()
	if err := w.Open(); err != nil {
		t.Fatalf("error opening WAL: %v", err)
	}

	files, err = w.ClosedSegments()
	if err != nil {
		t.Fatalf("error getting closed segments: %v", err)
	}
	if got, exp := len(files), 1; got != exp {
		t.Fatalf("close segment length mismatch: got %v, exp %v", got, exp)
	}
}

func TestWAL_Delete(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)

	w := tsm1.NewWAL(dir)
	if err := w.Open(); err != nil {
		t.Fatalf("error opening WAL: %v", err)
	}

	files, err := w.ClosedSegments()
	if err != nil {
		t.Fatalf("error getting closed segments: %v", err)
	}

	if got, exp := len(files), 0; got != exp {
		t.Fatalf("close segment length mismatch: got %v, exp %v", got, exp)
	}

	if err := w.Delete([]string{"cpu"}); err != nil {
		t.Fatalf("error writing points: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("error closing wal: %v", err)
	}

	// Re-open the WAL
	w = tsm1.NewWAL(dir)
	defer w.Close()
	if err := w.Open(); err != nil {
		t.Fatalf("error opening WAL: %v", err)
	}

	files, err = w.ClosedSegments()
	if err != nil {
		t.Fatalf("error getting closed segments: %v", err)
	}
	if got, exp := len(files), 1; got != exp {
		t.Fatalf("close segment length mismatch: got %v, exp %v", got, exp)
	}
}

func BenchmarkWALSegmentWriter(b *testing.B) {
	points := map[string][]tsm1.Value{}
	for i := 0; i < 5000; i++ {
		k := "cpu,host=A#!~#value"
		points[k] = append(points[k], tsm1.NewValue(time.Unix(int64(i), 0), 1.1))
	}

	dir := MustTempDir()
	defer os.RemoveAll(dir)

	f := MustTempFile(dir)
	w := tsm1.NewWALSegmentWriter(f)

	write := &tsm1.WriteWALEntry{
		Values: points,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := w.Write(write); err != nil {
			b.Fatalf("unexpected error writing entry: %v", err)
		}
	}
}

func BenchmarkWALSegmentReader(b *testing.B) {
	points := map[string][]tsm1.Value{}
	for i := 0; i < 5000; i++ {
		k := "cpu,host=A#!~#value"
		points[k] = append(points[k], tsm1.NewValue(time.Unix(int64(i), 0), 1.1))
	}

	dir := MustTempDir()
	defer os.RemoveAll(dir)

	f := MustTempFile(dir)
	w := tsm1.NewWALSegmentWriter(f)

	write := &tsm1.WriteWALEntry{
		Values: points,
	}

	for i := 0; i < 100; i++ {
		if err := w.Write(write); err != nil {
			b.Fatalf("unexpected error writing entry: %v", err)
		}
	}

	r := tsm1.NewWALSegmentReader(f)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		f.Seek(0, os.SEEK_SET)
		b.StartTimer()

		for r.Next() {
			_, err := r.Read()
			if err != nil {
				b.Fatalf("unexpected error reading entry: %v", err)
			}
		}
	}
}
