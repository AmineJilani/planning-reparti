// CE FICHIER log.go SERT A SUIVRE OU VOIR A CE QUE LE PROGRAMME FAIT --> VISUEL
package utils

import (
	"fmt"
	"log"
	"os"
)

// Logger garde le nom de  site et un logger go
type Logger struct {
	site   string
	logger *log.Logger
}

func NewLogger(site string) *Logger {
	return &Logger{
		site:   site,
		logger: log.New(os.Stderr, "", 0),
	}
}

func (l *Logger) Info(where, what string) {
	//Méthode pour afficher l'action faite (ce qui se passe) --> information normale
	l.logger.Printf("+ [%s] %-12s : %s", l.site, where, what)
}

func (l *Logger) Warn(where, what string) {
	//Méthode pour afficher le problème possible
	l.logger.Printf("* [%s] %-12s : %s", l.site, where, what)
}

func (l *Logger) Error(where, what string) {
	//BUG
	l.logger.Printf("! [%s] %-12s : %s", l.site, where, what)
}

func (l *Logger) Debugf(where, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	l.logger.Printf("+ [%s] %-12s : %s", l.site, where, msg)
}
