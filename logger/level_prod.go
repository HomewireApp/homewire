//go:build production

package logger

func init() {
	defaultLevel = LevelError
}
