package project

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotAcceptsClosedInventory(t *testing.T) {
	root := t.TempDir()
	files := map[string][]byte{
		"assets/site.css":   []byte("body{color:black}\n"),
		"assets/note.txt":   []byte("hello\n"),
		"assets/rows.csv":   []byte("a,b\n1,2\n"),
		"assets/data.json":  []byte(`{"ok":true}`),
		"assets/pixel.png":  pngBytes(),
		"assets/pixel.jpg":  jpegBytes(),
		"assets/pixel.jpeg": jpegBytes(),
		"assets/pixel.gif":  gifBytes(),
		"assets/pixel.webp": webpBytes(),
		"assets/pixel.avif": avifBytes(),
		"assets/pixel.ico":  icoBytes(),
		"assets/font.woff":  woffBytes(),
		"assets/font.woff2": woff2Bytes(),
	}
	manifest := map[string]string{}
	for name, data := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
			t.Fatal(err)
		}
		manifest[assetKeyName(name)] = name
	}
	assets, err := Snapshot(root, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != len(files) {
		t.Fatalf("snapshot count %d", len(assets))
	}
	for _, asset := range assets {
		if asset.Digest == "" || asset.URL != "/__can/project/"+asset.Digest+"/"+asset.SafeName || asset.Artifact != "assets/"+asset.Digest+"/"+asset.SafeName {
			t.Fatalf("unbound asset %+v", asset)
		}
		if asset.MediaType == "" || len(asset.Bytes) == 0 {
			t.Fatal(asset)
		}
	}
	moved := t.TempDir()
	for name, data := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(moved, name)), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(moved, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	again, err := Snapshot(moved, manifest)
	if err != nil {
		t.Fatal(err)
	}
	for i := range assets {
		if assets[i].URL != again[i].URL || assets[i].Digest != again[i].Digest {
			t.Fatal("relocation changed the asset manifest")
		}
	}
}

func TestSnapshotRejectsEscapeSignatureAndExecutableFormats(t *testing.T) {
	root := t.TempDir()
	write := func(name string, data []byte) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("assets/site.css", []byte("body{}\n"))
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.css"), []byte("body{}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.css"), filepath.Join(root, "assets", "linked.css")); err != nil {
		t.Fatal(err)
	}
	if _, err := Snapshot(root, map[string]string{"linked": "assets/linked.css"}); err == nil {
		t.Fatal("symlink escape admitted")
	}
	if _, err := Snapshot(root, map[string]string{"up": "../secret.css"}); err == nil {
		t.Fatal("parent path admitted")
	}
	rejected := map[string][]byte{
		"app.js":    []byte("alert(1)\n"),
		"app.html":  []byte("<html></html>\n"),
		"app.svg":   []byte("<svg></svg>\n"),
		"app.wasm":  []byte{0, 'a', 's', 'm', 1, 0, 0, 0},
		"app.map":   []byte(`{"version":3}`),
		"app.ts":    []byte("export {}\n"),
		"app.xml":   []byte("<root/>\n"),
		"bad.png":   []byte("not a png"),
		"bad.json":  []byte("{"),
		"bad.css":   []byte{0xff, 0xfe},
		"trunc.gif": []byte("GIF89a"),
		"zero.png":  pngBytes(),
	}
	rejected["zero.png"] = append([]byte(nil), pngBytes()...)
	binary.BigEndian.PutUint32(rejected["zero.png"][16:20], 0)
	for name, data := range rejected {
		write("assets/"+name, data)
		key := "file_" + assetKeyName(name)
		if _, err := Snapshot(root, map[string]string{key: "assets/" + name}); err == nil {
			t.Fatalf("accepted %s", name)
		}
	}
	write("assets/site.css", []byte("body{}\n"))
	if _, err := Snapshot(root, map[string]string{"site.css": "assets/site.css"}); err == nil {
		t.Fatal("dotted asset key admitted")
	}
}

func assetKeyName(name string) string {
	out := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		switch name[i] {
		case '/', '.':
			out = append(out, '_')
		default:
			out = append(out, name[i])
		}
	}
	return string(out)
}

func pngBytes() []byte {
	ihdr := []byte{0, 0, 0, 1, 0, 0, 0, 1, 8, 2, 0, 0, 0}
	out := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 13, 'I', 'H', 'D', 'R'}
	out = append(out, ihdr...)
	return append(out, 0, 0, 0, 0)
}

func jpegBytes() []byte {
	return []byte{0xff, 0xd8, 0xff, 0xc0, 0x00, 0x0b, 0x08, 0x00, 0x01, 0x00, 0x01, 0x01, 0x01, 0x11, 0x00, 0xff, 0xd9}
}

func gifBytes() []byte {
	return []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00")
}

func webpBytes() []byte {
	body := make([]byte, 10)
	chunk := []byte("VP8X")
	chunk = append(chunk, 10, 0, 0, 0)
	chunk = append(chunk, body...)
	riff := []byte("RIFF")
	size := make([]byte, 4)
	binary.LittleEndian.PutUint32(size, uint32(4+len(chunk)))
	out := append(append(riff, size...), []byte("WEBP")...)
	return append(out, chunk...)
}

func avifBytes() []byte {
	ftyp := make([]byte, 20)
	binary.BigEndian.PutUint32(ftyp[:4], 20)
	copy(ftyp[4:], "ftypavif")
	copy(ftyp[16:], "avif")
	ispe := make([]byte, 20)
	binary.BigEndian.PutUint32(ispe[:4], 20)
	copy(ispe[4:], "ispe")
	binary.BigEndian.PutUint32(ispe[12:16], 1)
	binary.BigEndian.PutUint32(ispe[16:20], 1)
	ipco := boxWrap("ipco", ispe)
	iprp := boxWrap("iprp", ipco)
	meta := make([]byte, 4+len(iprp))
	copy(meta[4:], iprp)
	return append(ftyp, boxWrap("meta", meta)...)
}

func boxWrap(kind string, payload []byte) []byte {
	out := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(out[:4], uint32(len(out)))
	copy(out[4:], kind)
	copy(out[8:], payload)
	return out
}

func icoBytes() []byte {
	image := pngBytes()
	entry := make([]byte, 16)
	entry[0], entry[1] = 1, 1
	binary.LittleEndian.PutUint16(entry[4:6], 1)
	binary.LittleEndian.PutUint32(entry[8:12], uint32(len(image)))
	binary.LittleEndian.PutUint32(entry[12:16], 22)
	head := []byte{0, 0, 1, 0, 1, 0}
	return append(append(head, entry...), image...)
}

func woffBytes() []byte {
	dir := make([]byte, 20)
	copy(dir, "OS/2")
	binary.BigEndian.PutUint32(dir[4:8], 64)
	binary.BigEndian.PutUint32(dir[8:12], 4)
	out := make([]byte, 68)
	copy(out, "wOFF")
	binary.BigEndian.PutUint32(out[4:8], 0x00010000)
	binary.BigEndian.PutUint32(out[8:12], 68)
	binary.BigEndian.PutUint16(out[12:14], 1)
	binary.BigEndian.PutUint32(out[16:20], 64)
	copy(out[44:], dir)
	return out
}

func woff2Bytes() []byte {
	out := make([]byte, 64)
	copy(out, "wOF2")
	binary.BigEndian.PutUint32(out[4:8], 0x00010000)
	binary.BigEndian.PutUint32(out[8:12], 64)
	binary.BigEndian.PutUint16(out[12:14], 1)
	binary.BigEndian.PutUint32(out[16:20], 32)
	binary.BigEndian.PutUint32(out[20:24], 1)
	out[48] = 0
	return out
}

func TestSnapshotCopiesBytes(t *testing.T) {
	root := t.TempDir()
	original := []byte("body{}\n")
	if err := os.WriteFile(filepath.Join(root, "site.css"), original, 0600); err != nil {
		t.Fatal(err)
	}
	assets, err := Snapshot(root, map[string]string{"site_css": "site.css"})
	if err != nil {
		t.Fatal(err)
	}
	assets[0].Bytes[0] = 'X'
	disk, err := os.ReadFile(filepath.Join(root, "site.css"))
	if err != nil || !bytes.Equal(disk, original) {
		t.Fatal("snapshot aliased source bytes")
	}
}
