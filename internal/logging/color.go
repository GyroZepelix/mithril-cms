package logging

import "fmt"

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Yellow = "\033[33m"
	White  = "\033[97m"
)

func colorRed(s string) string {
	return Red + s + Reset
}

func colorYellow(s string) string {
	return Yellow + s + Reset
}

func colorWhite(s string) string {
	return White + s + Reset
}

func colorRedRange(v ...interface{}) string {
	return Red + fmt.Sprint(v...) + Reset
}

func colorYellowRange(v ...interface{}) string {
	return Yellow + fmt.Sprint(v...) + Reset
}

func colorWhiteRange(v ...interface{}) string {
	return White + fmt.Sprint(v...) + Reset
}
