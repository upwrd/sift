package logging

import (
	"fmt"
	log "gopkg.in/inconshreveable/log15.v2"
	//"gopkg.in/inconshreveable/log15.v2/ext"
	"gopkg.in/inconshreveable/log15.v2/stack"
)

// Log is the default log instance for all SIFT logs. Logs are disabled by
// default; use SetLevelStr to set log levels for all packages, or
// Log.SetHandler() to set a custom handler. (see:
// https://godoc.org/gopkg.in/inconshreveable/log15.v2)
var Log = log.New()

func init() {
	Log.SetHandler(log.DiscardHandler()) // Off by default
}

// SetLevel sets the log level of the default logger, using the default log handler
func setLevel(lvl log.Lvl) {
	Log.SetHandler(FuncAndLineHandler(log.StdoutHandler, lvl))
}

// SetLevelStr sets the log level of the default logger, using the default log
// handler. Possible values include "off", "debug", "info", "warn", "error" and
// "crit"
func SetLevelStr(lvlstr string) {
	switch lvlstr {
	case "off":
		Log.SetHandler(log.DiscardHandler()) // discard all output
	default:
		lvl, err := log.LvlFromString(lvlstr)
		if err == nil {
			setLevel(lvl)
			break
		}
		fmt.Printf("(!) error setting error level with string %s, will turn off logs", lvlstr)
		Log.SetHandler(log.DiscardHandler()) // discard all output
	}
}

// FuncAndLineHandler creates a log handler that adds the calling function's
// name and line number as context.
func FuncAndLineHandler(h log.Handler, lvl log.Lvl) log.Handler {
	return log.FuncHandler(func(r *log.Record) error {
		if r.Lvl <= lvl {
			call := stack.Call(r.CallPC[0])
			r.Ctx = append(r.Ctx, "fn", fmt.Sprintf("%+v", call))
			r.Ctx = append(r.Ctx, "ln", fmt.Sprint(call))
			return h.Log(r)
		}
		return nil
	})
}
