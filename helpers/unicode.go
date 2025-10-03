// nolint:mnd
package helpers

import (
	"unsafe"

	"github.com/txix-open/isp-kit/json"
)

func UnescapeUnicodeJson(data []byte) []byte {
	var body any
	err := json.Unmarshal(data, &body)
	if err != nil {
		return data
	}

	newData, err := json.Marshal(body)
	if err != nil {
		return data
	}

	return newData
}

func UnescapeUnicode(data []byte) []byte {
	n := len(data)
	out := make([]byte, len(data)) // максимально возможный размер
	j := 0                         // индекс записи

	for i := 0; i < n; {
		if data[i] != '\\' {
			// копируем весь блок до следующего '\'
			start := i
			for i < n && data[i] != '\\' {
				i++
			}
			j += copy(out[j:], data[start:i])
			continue
		}

		if isUnicodeEscape(data, i) {
			r, consumed := decodeUnicodeEscape(data[i:])
			if r >= 0 {
				j = appendRuneAt(out, j, uint32(r))
				i += consumed
				continue
			}
		}

		// обычный байт (одиночный '\')
		out[j] = data[i]
		j++
		i++
	}

	return out[:j]
}

// Проверяет, начинается ли здесь \uXXXX
func isUnicodeEscape(data []byte, i int) bool {
	return i+5 < len(data) && data[i] == '\\' && data[i+1] == 'u'
}

// Декодирует \uXXXX, учитывая суррогатную пару.
// Возвращает руну и сколько байт было прочитано.
func decodeUnicodeEscape(data []byte) (rune, int) {
	v1, ok := parseHex4(data[2:6])
	if !ok {
		return -1, 1
	}

	// Проверка суррогатной пары
	if 0xD800 <= v1 && v1 <= 0xDBFF && isUnicodeEscape(data, 6) {
		v2, ok2 := parseHex4(data[8:12])
		if ok2 && 0xDC00 <= v2 && v2 <= 0xDFFF {
			r := 0x10000 + ((uint32(v1)-0xD800)<<10 | (uint32(v2) - 0xDC00))
			return rune(r), 12
		}
	}

	return rune(v1), 6
}

// Парсим 4 hex-символа через hexValue
func parseHex4(b []byte) (uint16, bool) {
	v := uint16(0)
	for _, c := range b {
		h, ok := hexValue(c)
		if !ok {
			return 0, false
		}
		v = v<<4 | h
	}
	return v, true
}

func hexValue(b byte) (uint16, bool) {
	switch {
	case '0' <= b && b <= '9':
		return uint16(b - '0'), true
	case 'a' <= b && b <= 'f':
		return uint16(b - 'a' + 10), true
	case 'A' <= b && b <= 'F':
		return uint16(b - 'A' + 10), true
	default:
		return 0, false
	}
}

// appendRune добавляет руну в UTF-8
func appendRuneAt(buf []byte, j int, r uint32) int {
	p := (*[4]byte)(unsafe.Pointer(&buf[j]))
	switch {
	case r < 0x80:
		p[0] = byte(r)
		return j + 1
	case r < 0x800:
		p[0] = 0xC0 | byte(r>>6)
		p[1] = 0x80 | byte(r&0x3F)
		return j + 2
	case r < 0x10000:
		p[0] = 0xE0 | byte(r>>12)
		p[1] = 0x80 | byte((r>>6)&0x3F)
		p[2] = 0x80 | byte(r&0x3F)
		return j + 3
	default:
		p[0] = 0xF0 | byte(r>>18)
		p[1] = 0x80 | byte((r>>12)&0x3F)
		p[2] = 0x80 | byte((r>>6)&0x3F)
		p[3] = 0x80 | byte(r&0x3F)
		return j + 4
	}
}
