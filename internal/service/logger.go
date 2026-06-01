package service

// Logger представляет интерфейс логирования для бизнес-сервисов
type Logger interface {
	Log(msg string, level string)
}
