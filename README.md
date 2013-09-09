# go-nicelog

go-nicelog is a standard-library-compatible logging package for Go.  It adds some nice features, such as coloring the logging prefix, levels, and filtering, all of which is designed to be as simple as possible.

Documentation is kind of sparse right now, but in short:

    import (
        "os"
        "github.com/andrew-d/go-nicelog"
    )

    func main() {
        log := nicelog.New(os.Stderr, "", nicelog.LdefaultFlags)

        // The following two lines won't get displayed by default, since
        // the default filter level is INFO.  If you'd like to see them,
        // use:
        //    log.SetLevelFilter(nicelog.TRACE)
        // Level filtering to level "X" hides all log messages with levels
        // lower than "X".
        log.Trace("this is a trace")
        log.Debug("debugging")

        if log.WouldLog(nicelog.TRACE) {
            log.Trace("Something expensive that will only run if we're logging it")
        }

        // This will log as INFO by default.  Use:
        //    log.SetDefaultLevel(nicelog.WARN)
        // to change the level that the standard-library-compatible functions use.
        log.Print("Default log message")

        log.Info("information")
        log.Warnf("foo != %d", 2)
        log.Error("Something went wrong!")
        log.Fatal("goodbye, cruel world")
    }
