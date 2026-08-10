package app

import (
	"fmt"
	"math"
	"regexp"
)

// hexColorRe — строгий формат status_color: ровно #RRGGBB.
// Прописные и строчные буквы допустимы. Любое другое значение
// на входе API отклоняется; при рендере служит защитным слоем
// от старых/грязных данных в БД.
var hexColorRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// validStatusColor проверяет формат #RRGGBB. Пустая строка не является
// цветом и обрабатывается вызывающим кодом как «удалить окраску».
func validStatusColor(value string) bool {
	return hexColorRe.MatchString(value)
}

// statusColorIfValid возвращает цвет (#RRGGBB), если значение допустимо,
// иначе "". Используется как защитный слой при рендере: старые/грязные
// данные в БД не должны попадать в CSS.
func statusColorIfValid(value string) string {
	if validStatusColor(value) {
		return value
	}
	return ""
}

// statusColorRGB возвращает компоненты RGB (0–255) валидного #RRGGBB.
func statusColorRGB(value string) (r, g, b int, err error) {
	if !validStatusColor(value) {
		return 0, 0, 0, fmt.Errorf("invalid status color %q", value)
	}
	_, err = fmt.Sscanf(value[1:], "%02x%02x%02x", &r, &g, &b)
	return r, g, b, err
}

// statusTextColor выбирает цвет текста для фона #RRGGBB: чёрный или белый,
// в зависимости от того, какой даёт больший WCAG-контраст. Фон не меняется
// светлой/тёмной темой — выбирается только цвет текста, поэтому семантика
// status_color остаётся единой в обеих темах.
//
//	nil/невалидный цвет → "" (окраска не применяется, обычный вид).
func statusTextColor(value string) string {
	r, g, b, err := statusColorRGB(value)
	if err != nil {
		return ""
	}

	lum := relativeLuminance(r, g, b)

	contrastWhite := (1.0 + 0.05) / (lum + 0.05)
	contrastBlack := (lum + 0.05) / (0.0 + 0.05)

	if contrastWhite >= contrastBlack {
		return "#ffffff"
	}
	return "#000000"
}

// relativeLuminance считает WCAG-относительную яркость цвета (sRGB → linear).
func relativeLuminance(r, g, b int) float64 {
	return 0.2126*linearize(r) + 0.7152*linearize(g) + 0.0722*linearize(b)
}

func linearize(channel int) float64 {
	c := float64(channel) / 255.0
	if c <= 0.03928 {
		return c / 12.92
	}
	return math.Pow(c, 2.4)
}
