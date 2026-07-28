package display

type Values interface {
	DisplayValue(name string) (string, error)
}
