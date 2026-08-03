package search

import (
	"database/sql/driver"
	"sync"

	sqlite "modernc.org/sqlite"
)

// RegisterFunctions registers SQL functions required by the search package.
//
// Функции регистрируются на драйвере SQLite до открытия соединений:
// новые коннекты получают их автоматически. Вызывается из
// database.OpenPath. Не использует init(): жизненный цикл очевиден.
//
// Будущие функции пакета (search_unaccent, search_soundex, search_rank)
// регистрируются здесь же — API вызывающей стороны не меняется.
func RegisterFunctions() {
	registerOnce.Do(func() {
		sqlite.MustRegisterDeterministicScalarFunction("search_normalize", 1,
			func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
				if args[0] == nil {
					return nil, nil
				}
				s, ok := args[0].(string)
				if !ok {
					return args[0], nil
				}
				return normalizeWord(s), nil
			})
	})
}

// registerOnce делает регистрацию идемпотентной: OpenPath вызывается
// многократно (в том числе в тестах), а регистрация допустима один раз.
var registerOnce sync.Once
