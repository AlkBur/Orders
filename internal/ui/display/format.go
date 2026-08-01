package display

import "strconv"

// IntegerNumber — целочисленные типы, поддерживаемые FormatNumber.
// Ограничение намеренно уже, чем cmp.Ordered: группировка разрядов
// не имеет смысла для чисел с плавающей точкой и строк.
type IntegerNumber interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// FormatNumber возвращает число с разделителями тысяч (пробел).
//
//	FormatNumber(12547)  // "12 547"
//	FormatNumber(-1234)  // "-1 234"
//	FormatNumber(0)      // "0"
func FormatNumber[T IntegerNumber](v T) string {
	n := int64(v)

	digits := strconv.FormatInt(n, 10)
	neg := false
	if n < 0 {
		neg = true
		digits = digits[1:]
	}

	var b []byte
	for i := 0; i < len(digits); i++ {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b = append(b, ' ')
		}
		b = append(b, digits[i])
	}

	s := string(b)
	if neg {
		s = "-" + s
	}
	return s
}
