package project

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Asset is one manifest file after confinement, signature, and media-type checks.
// The URL and artifact path are content-addressed; callers cannot choose either.
type Asset struct {
	Name, Project, Relative, SafeName, Digest, MediaType, URL, Artifact string
	Bytes                                                               []byte
}

var assetKey = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)
var assetBase = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
var assetDevice = regexp.MustCompile(`(?i)^(con|prn|aux|nul|com[1-9]|lpt[1-9])(?:\.|$)`)

var assetMedia = map[string]string{
	".css":   "text/css; charset=utf-8",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".webp":  "image/webp",
	".avif":  "image/avif",
	".ico":   "image/x-icon",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".txt":   "text/plain; charset=utf-8",
	".json":  "application/json",
	".csv":   "text/csv; charset=utf-8",
}

var rejectedAssetExt = map[string]bool{
	".js": true, ".mjs": true, ".cjs": true, ".jsx": true, ".ts": true, ".tsx": true,
	".html": true, ".htm": true, ".svg": true, ".xml": true, ".wasm": true, ".map": true,
}

// Snapshot reads every manifest asset inside its project root. Extensions select
// the fixed media type; signatures are checked from the bytes, never from a
// caller-supplied type. Executable and unknown formats fail the build.
func Snapshot(root string, assets map[string]string) ([]Asset, error) {
	out := make([]Asset, 0, len(assets))
	for _, name := range sortedKeys(assets) {
		if !assetKey.MatchString(name) {
			return nil, fmt.Errorf("asset %q: unsafe name", name)
		}
		relative := assets[name]
		base := path.Base(relative)
		ext := strings.ToLower(path.Ext(base))
		if rejectedAssetExt[ext] {
			return nil, fmt.Errorf("asset %q: rejected %s content", name, ext)
		}
		media, ok := assetMedia[ext]
		if !ok || !assetBase.MatchString(base) || strings.HasSuffix(base, ".") || assetDevice.MatchString(base) {
			return nil, fmt.Errorf("asset %q: unsupported file name %q", name, base)
		}
		file, err := ConfinedPath(root, relative, false)
		if err != nil {
			return nil, fmt.Errorf("asset %q: %w", name, err)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("asset %q: %w", name, err)
		}
		if err := validateAssetBytes(ext, data); err != nil {
			return nil, fmt.Errorf("asset %q: %w", name, err)
		}
		sum := sha256.Sum256(data)
		digest := hex.EncodeToString(sum[:])
		artifact := "assets/" + digest + "/" + base
		copied := append([]byte(nil), data...)
		out = append(out, Asset{
			Name: name, Relative: relative, SafeName: base, Digest: digest, MediaType: media,
			URL: "/__can/project/" + digest + "/" + base, Artifact: artifact, Bytes: copied,
		})
	}
	return out, nil
}

func validateAssetBytes(ext string, data []byte) error {
	switch ext {
	case ".css", ".txt", ".csv":
		return validateText(data)
	case ".json":
		return validateJSONAsset(data)
	case ".png":
		return validatePNG(data)
	case ".jpg", ".jpeg":
		return validateJPEG(data)
	case ".gif":
		return validateGIF(data)
	case ".webp":
		return validateWEBP(data)
	case ".avif":
		return validateAVIF(data)
	case ".ico":
		return validateICO(data)
	case ".woff":
		return validateWOFF(data)
	case ".woff2":
		return validateWOFF2(data)
	default:
		return fmt.Errorf("unsupported file type")
	}
}

func validateText(data []byte) error {
	if !utf8.Valid(data) || bytes.Contains(data, []byte{0}) {
		return fmt.Errorf("text asset is not UTF-8")
	}
	return nil
}

func validateJSONAsset(data []byte) error {
	if err := validateText(data); err != nil {
		return fmt.Errorf("json signature mismatch")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("json signature mismatch")
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("json signature mismatch")
	}
	return nil
}

func validatePNG(data []byte) error {
	sig := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	if len(data) < 8+12+13 || !bytes.Equal(data[:8], sig) {
		return fmt.Errorf("png signature mismatch")
	}
	length := binary.BigEndian.Uint32(data[8:12])
	if length != 13 || string(data[12:16]) != "IHDR" || int(length)+12 > len(data)-8 {
		return fmt.Errorf("png signature mismatch")
	}
	ihdr := data[16:29]
	width := binary.BigEndian.Uint32(ihdr[0:4])
	height := binary.BigEndian.Uint32(ihdr[4:8])
	depth, color := ihdr[8], ihdr[9]
	if width == 0 || height == 0 || ihdr[10] != 0 || ihdr[11] != 0 || (ihdr[12] != 0 && ihdr[12] != 1) {
		return fmt.Errorf("png signature mismatch")
	}
	legal := map[byte]string{0: "\x01\x02\x04\x08\x10", 2: "\x08\x10", 3: "\x01\x02\x04\x08", 4: "\x08\x10", 6: "\x08\x10"}
	if !bytes.Contains([]byte(legal[color]), []byte{depth}) {
		return fmt.Errorf("png signature mismatch")
	}
	return nil
}

func validateJPEG(data []byte) error {
	if len(data) < 4 || data[0] != 0xff || data[1] != 0xd8 {
		return fmt.Errorf("jpeg signature mismatch")
	}
	i := 2
	found := false
	for step := 0; step < 64 && i < len(data); step++ {
		if data[i] != 0xff {
			return fmt.Errorf("jpeg signature mismatch")
		}
		for i < len(data) && data[i] == 0xff {
			i++
		}
		if i >= len(data) {
			return fmt.Errorf("jpeg signature mismatch")
		}
		marker := data[i]
		i++
		if marker == 0xd9 {
			break
		}
		if marker == 0x01 || (marker >= 0xd0 && marker <= 0xd7) {
			continue
		}
		if i+2 > len(data) {
			return fmt.Errorf("jpeg signature mismatch")
		}
		seg := int(data[i])<<8 | int(data[i+1])
		if seg < 2 || i+seg > len(data) {
			return fmt.Errorf("jpeg signature mismatch")
		}
		if marker == 0xda {
			if !found {
				return fmt.Errorf("jpeg signature mismatch")
			}
			return nil
		}
		if isSOF(marker) {
			if seg < 8 {
				return fmt.Errorf("jpeg signature mismatch")
			}
			precision := data[i+2]
			height := int(data[i+3])<<8 | int(data[i+4])
			width := int(data[i+5])<<8 | int(data[i+6])
			if (precision != 8 && precision != 12) || width == 0 || height == 0 {
				return fmt.Errorf("jpeg signature mismatch")
			}
			found = true
		}
		i += seg
	}
	if !found {
		return fmt.Errorf("jpeg signature mismatch")
	}
	return nil
}

func isSOF(marker byte) bool {
	switch marker {
	case 0xc0, 0xc1, 0xc2, 0xc3, 0xc5, 0xc6, 0xc7, 0xc9, 0xca, 0xcb, 0xcd, 0xce, 0xcf:
		return true
	default:
		return false
	}
}

func validateGIF(data []byte) error {
	if len(data) < 13 || (string(data[:6]) != "GIF87a" && string(data[:6]) != "GIF89a") {
		return fmt.Errorf("gif signature mismatch")
	}
	width := binary.LittleEndian.Uint16(data[6:8])
	height := binary.LittleEndian.Uint16(data[8:10])
	if width == 0 || height == 0 {
		return fmt.Errorf("gif signature mismatch")
	}
	return nil
}

func validateWEBP(data []byte) error {
	if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return fmt.Errorf("webp signature mismatch")
	}
	size := binary.LittleEndian.Uint32(data[4:8])
	if uint64(size)+8 != uint64(len(data)) || len(data) < 30 {
		return fmt.Errorf("webp signature mismatch")
	}
	kind := string(data[12:16])
	chunk := binary.LittleEndian.Uint32(data[16:20])
	if uint64(20)+uint64(chunk) > uint64(len(data)) {
		return fmt.Errorf("webp signature mismatch")
	}
	body := data[20 : 20+chunk]
	switch kind {
	case "VP8 ":
		if len(body) < 10 || body[0]&1 != 0 || body[3] != 0x9d || body[4] != 0x01 || body[5] != 0x2a {
			return fmt.Errorf("webp signature mismatch")
		}
		width := int(body[6]) | int(body[7]&0x3f)<<8
		height := int(body[8]) | int(body[9]&0x3f)<<8
		if width == 0 || height == 0 {
			return fmt.Errorf("webp signature mismatch")
		}
	case "VP8L":
		if len(body) < 5 || body[0] != 0x2f {
			return fmt.Errorf("webp signature mismatch")
		}
		// Width and height are stored minus one, so a complete header is at least 1 by 1.
	case "VP8X":
		if len(body) < 10 {
			return fmt.Errorf("webp signature mismatch")
		}
	default:
		return fmt.Errorf("webp signature mismatch")
	}
	return nil
}

func validateAVIF(data []byte) error {
	brand, rest, err := box(data)
	if err != nil || string(brand.kind) != "ftyp" || len(brand.data) < 8 {
		return fmt.Errorf("avif signature mismatch")
	}
	major := string(brand.data[:4])
	if major != "avif" && major != "avis" {
		return fmt.Errorf("avif signature mismatch")
	}
	found := false
	var walk func([]byte, int) error
	walk = func(payload []byte, depth int) error {
		if depth > 8 {
			return fmt.Errorf("avif signature mismatch")
		}
		for len(payload) > 0 {
			next, remain, err := box(payload)
			if err != nil {
				return err
			}
			if string(next.kind) == "ispe" && len(next.data) >= 12 {
				width := binary.BigEndian.Uint32(next.data[4:8])
				height := binary.BigEndian.Uint32(next.data[8:12])
				if next.data[0] == 0 && width > 0 && height > 0 {
					found = true
				}
			}
			if kind := string(next.kind); kind == "meta" || kind == "iprp" || kind == "ipco" {
				inner := next.data
				if kind == "meta" {
					if len(inner) < 4 {
						return fmt.Errorf("avif signature mismatch")
					}
					inner = inner[4:]
				}
				if err := walk(inner, depth+1); err != nil {
					return err
				}
			}
			payload = remain
		}
		return nil
	}
	whole := append([]byte(nil), brand.raw...)
	whole = append(whole, rest...)
	if err := walk(whole, 0); err != nil || !found {
		return fmt.Errorf("avif signature mismatch")
	}
	return nil
}

type bmffBox struct {
	kind, data, raw []byte
}

func box(data []byte) (bmffBox, []byte, error) {
	if len(data) < 8 {
		return bmffBox{}, nil, fmt.Errorf("avif signature mismatch")
	}
	size := binary.BigEndian.Uint32(data[:4])
	header := 8
	total := uint64(size)
	if size == 1 {
		if len(data) < 16 {
			return bmffBox{}, nil, fmt.Errorf("avif signature mismatch")
		}
		total = binary.BigEndian.Uint64(data[8:16])
		header = 16
	} else if size == 0 {
		total = uint64(len(data))
	}
	if total < uint64(header) || total > uint64(len(data)) {
		return bmffBox{}, nil, fmt.Errorf("avif signature mismatch")
	}
	n := int(total)
	return bmffBox{kind: data[4:8], data: data[header:n], raw: data[:n]}, data[n:], nil
}

func validateICO(data []byte) error {
	if len(data) < 6 || binary.LittleEndian.Uint16(data[0:2]) != 0 {
		return fmt.Errorf("ico signature mismatch")
	}
	kind := binary.LittleEndian.Uint16(data[2:4])
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if (kind != 1 && kind != 2) || count < 1 || 6+count*16 > len(data) {
		return fmt.Errorf("ico signature mismatch")
	}
	for i := 0; i < count; i++ {
		entry := data[6+i*16 : 22+i*16]
		planes := binary.LittleEndian.Uint16(entry[4:6])
		bytesLen := binary.LittleEndian.Uint32(entry[8:12])
		offset := binary.LittleEndian.Uint32(entry[12:16])
		if (planes != 0 && planes != 1) || bytesLen == 0 || uint64(offset)+uint64(bytesLen) > uint64(len(data)) {
			return fmt.Errorf("ico signature mismatch")
		}
		image := data[offset : offset+bytesLen]
		if len(image) >= 8 && bytes.Equal(image[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
			if err := validatePNG(image); err != nil {
				return fmt.Errorf("ico signature mismatch")
			}
			continue
		}
		if err := validateDIB(image); err != nil {
			return fmt.Errorf("ico signature mismatch")
		}
	}
	return nil
}

func validateDIB(data []byte) error {
	if len(data) < 16 {
		return fmt.Errorf("dib signature mismatch")
	}
	size := binary.LittleEndian.Uint32(data[:4])
	if size != 40 && size != 108 && size != 124 || int(size) > len(data) {
		return fmt.Errorf("dib signature mismatch")
	}
	width := int32(binary.LittleEndian.Uint32(data[4:8]))
	height := int32(binary.LittleEndian.Uint32(data[8:12]))
	if width == 0 || height == 0 {
		return fmt.Errorf("dib signature mismatch")
	}
	return nil
}

func validateWOFF(data []byte) error {
	if len(data) < 44 || string(data[:4]) != "wOFF" {
		return fmt.Errorf("woff signature mismatch")
	}
	flavor := binary.BigEndian.Uint32(data[4:8])
	length := binary.BigEndian.Uint32(data[8:12])
	tables := int(binary.BigEndian.Uint16(data[12:14]))
	reserved := binary.BigEndian.Uint16(data[14:16])
	total := binary.BigEndian.Uint32(data[16:20])
	if !fontFlavor(flavor) || uint64(length) != uint64(len(data)) || tables < 1 || tables > 256 || reserved != 0 || total == 0 {
		return fmt.Errorf("woff signature mismatch")
	}
	if 44+tables*20 > len(data) {
		return fmt.Errorf("woff signature mismatch")
	}
	for i := 0; i < tables; i++ {
		entry := data[44+i*20 : 64+i*20]
		offset := binary.BigEndian.Uint32(entry[4:8])
		comp := binary.BigEndian.Uint32(entry[8:12])
		if comp == 0 || uint64(offset)+uint64(comp) > uint64(length) || offset < uint32(44+tables*20) {
			return fmt.Errorf("woff signature mismatch")
		}
	}
	metaOff := binary.BigEndian.Uint32(data[24:28])
	metaLen := binary.BigEndian.Uint32(data[28:32])
	privOff := binary.BigEndian.Uint32(data[36:40])
	privLen := binary.BigEndian.Uint32(data[40:44])
	if !optionalRange(metaOff, metaLen, length) || !optionalRange(privOff, privLen, length) {
		return fmt.Errorf("woff signature mismatch")
	}
	return nil
}

func validateWOFF2(data []byte) error {
	if len(data) < 48 || string(data[:4]) != "wOF2" {
		return fmt.Errorf("woff2 signature mismatch")
	}
	flavor := binary.BigEndian.Uint32(data[4:8])
	length := binary.BigEndian.Uint32(data[8:12])
	tables := int(binary.BigEndian.Uint16(data[12:14]))
	reserved := binary.BigEndian.Uint16(data[14:16])
	total := binary.BigEndian.Uint32(data[16:20])
	compressed := binary.BigEndian.Uint32(data[20:24])
	if !fontFlavor(flavor) || uint64(length) != uint64(len(data)) || tables < 1 || tables > 256 || reserved != 0 || total == 0 || compressed == 0 {
		return fmt.Errorf("woff2 signature mismatch")
	}
	i := 48
	for n := 0; n < tables; n++ {
		if i >= len(data) {
			return fmt.Errorf("woff2 signature mismatch")
		}
		flags := data[i]
		i++
		if flags&0x3f == 0x3f {
			if i+4 > len(data) {
				return fmt.Errorf("woff2 signature mismatch")
			}
			i += 4
		}
	}
	if uint64(i)+uint64(compressed) > uint64(length) {
		return fmt.Errorf("woff2 signature mismatch")
	}
	return nil
}

func fontFlavor(flavor uint32) bool {
	return flavor == 0x00010000 || flavor == 0x4f54544f || flavor == 0x74727565
}

func optionalRange(offset, length, file uint32) bool {
	if offset == 0 && length == 0 {
		return true
	}
	return offset > 0 && length > 0 && uint64(offset)+uint64(length) <= uint64(file)
}
