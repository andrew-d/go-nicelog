package nicelog

import (
    "fmt"
    "io"
    "os"
    "runtime"
    "strings"
    "sync"
    "time"

    stdlog "log"
)

const (
    TRACE = iota
    DEBUG
    INFO
    WARN
    ERROR
    FATAL
)

const (
    // Constants directly from standard library's log package
    Ldate         = stdlog.Ldate
    Ltime         = stdlog.Ltime
    Lmicroseconds = stdlog.Lmicroseconds
    Llongfile     = stdlog.Llongfile
    Lshortfile    = stdlog.Lshortfile
    LstdFlags     = stdlog.LstdFlags

    // Extra flags
    Lcolor = Lshortfile << 1
    Llevel = Lshortfile << 2

    // We use color and level by default
    LdefaultFlags = LstdFlags | Lcolor | Llevel
)

type Logger struct {
    mu     sync.Mutex                 // for atomic writes, protecting fields
    flag   int                        // flags
    out    io.Writer                  // output destination
    fmt    func(*LogMessage, *[]byte) // formatting function
    buf    []byte                     // accumulated text to write
    prefix string                     // this text will be prepended to all log lines

    defaultLevel int // log level if none given
    levelFilter  int // don't emit log levels below this one
}

type LogMessage struct {
    Time  time.Time
    File  string
    Line  int
    Level int

    // Settings from the root logger
    Prefix string
    Flag   int
}

var root = New(os.Stderr, "", LdefaultFlags)

var colorMap = map[int]string{
    TRACE: "\x1b[34m", // Blue
    DEBUG: "\x1b[34m", // Blue
    INFO:  "\x1b[32m", // Green
    WARN:  "\x1b[33m", // Yellow
    ERROR: "\x1b[31m", // Red
    FATAL: "\x1b[31m", // Red
}

var levelMap = map[int]string{
    TRACE: "[T]",
    DEBUG: "[D]",
    INFO:  "[I]",
    WARN:  "[W]",
    ERROR: "[E]",
    FATAL: "[F]",
}

func New(out io.Writer, prefix string, flag int) *Logger {
    return &Logger{out: out, prefix: prefix, flag: flag,
        fmt: defaultFormat, defaultLevel: INFO, levelFilter: INFO}
}

func defaultFormat(msg *LogMessage, buf *[]byte) {
    if msg.Flag&Lcolor != 0 {
        color, ok := colorMap[msg.Level]
        if ok {
            *buf = append(*buf, color...)
        }
    }

    *buf = append(*buf, msg.Prefix...)

    if msg.Flag&Llevel != 0 {
        ltxt, ok := levelMap[msg.Level]
        if ok {
            *buf = append(*buf, ltxt...)
            *buf = append(*buf, ' ')
        }
    }

    if msg.Flag&(Ldate|Ltime|Lmicroseconds) != 0 {
        var s string

        if msg.Flag&Ldate != 0 {
            year, month, day := msg.Time.Date()

            s = fmt.Sprintf("%04d/%02d/%02d ", year, month, day)
            *buf = append(*buf, s...)
        }
        if msg.Flag&(Ltime|Lmicroseconds) != 0 {
            hour, min, sec := msg.Time.Clock()
            s := fmt.Sprintf("%02d:%02d:%02d", hour, min, sec)
            *buf = append(*buf, s...)

            if msg.Flag&Lmicroseconds != 0 {
                ns := msg.Time.Nanosecond() / 1e3
                *buf = append(*buf, fmt.Sprintf(".%06d", ns)...)
            }

            *buf = append(*buf, ' ')
        }
    }

    if msg.Flag&(Lshortfile|Llongfile) != 0 {
        file := msg.File

        if msg.Flag&Lshortfile != 0 {
            slashIdx := strings.LastIndex(file, "/")
            if slashIdx != -1 {
                file = file[slashIdx+1:]
            }
        }

        *buf = append(*buf, file...)
        *buf = append(*buf, ':')
        *buf = append(*buf, fmt.Sprintf("%d", msg.Line)...)
        *buf = append(*buf, ": "...)
    }

    if msg.Flag&Lcolor != 0 {
        // Reset colors
        *buf = append(*buf, "\x1b[0m"...)
    }
}

func (l *Logger) SetFormatter(f func(*LogMessage, *[]byte)) {
    l.fmt = f
}

func (l *Logger) Output(calldepth int, level int, s string) error {
    now := time.Now()
    var file string
    var line int

    l.mu.Lock()
    defer l.mu.Unlock()

    if level < l.levelFilter {
        return nil
    }

    // Much of this is taken from the standard library
    if l.flag&(Lshortfile|Llongfile) != 0 {
        l.mu.Unlock()
        var ok bool

        _, file, line, ok = runtime.Caller(calldepth)
        if !ok {
            file = "???"
            line = 0
        }
        l.mu.Lock()
    }

    l.buf = l.buf[:0]
    msg := LogMessage{now, file, line, level,
        l.prefix, l.flag}

    l.fmt(&msg, &l.buf)
    l.buf = append(l.buf, s...)
    if len(s) > 0 && s[len(s)-1] != '\n' {
        l.buf = append(l.buf, '\n')
    }

    _, err := l.out.Write(l.buf)
    return err
}

func (l *Logger) Flags() int {
    l.mu.Lock()
    defer l.mu.Unlock()
    return l.flag
}

func (l *Logger) SetFlags(flag int) {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.flag = flag
}

func (l *Logger) Prefix() string {
    l.mu.Lock()
    defer l.mu.Unlock()
    return l.prefix
}

func (l *Logger) SetPrefix(prefix string) {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.prefix = prefix
}

func (l *Logger) DefaultLevel() int {
    l.mu.Lock()
    defer l.mu.Unlock()
    return l.defaultLevel
}

func (l *Logger) SetDefaultLevel(defaultLevel int) {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.defaultLevel = defaultLevel
}

func (l *Logger) LevelFilter() int {
    l.mu.Lock()
    defer l.mu.Unlock()
    return l.levelFilter
}

func (l *Logger) SetLevelFilter(lvl int) {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.levelFilter = lvl
}

func (l *Logger) WouldLog(lvl int) bool {
    l.mu.Lock()
    defer l.mu.Unlock()

    return lvl >= l.levelFilter
}

// ------------------------------------------------------------

func (l *Logger) Print(v ...interface{}) {
    l.Output(2, l.defaultLevel, fmt.Sprint(v...))
}

func (l *Logger) Printf(format string, v ...interface{}) {
    l.Output(2, l.defaultLevel, fmt.Sprintf(format, v...))
}

func (l *Logger) Println(v ...interface{}) {
    l.Output(2, l.defaultLevel, fmt.Sprintln(v...))
}

// ------------------------------------------------------------

func (l *Logger) Trace(v ...interface{}) {
    l.Output(2, TRACE, fmt.Sprint(v...))
}

func (l *Logger) Tracef(format string, v ...interface{}) {
    l.Output(2, TRACE, fmt.Sprintf(format, v...))
}

func (l *Logger) Traceln(v ...interface{}) {
    l.Output(2, TRACE, fmt.Sprintln(v...))
}

// ------------------------------------------------------------

func (l *Logger) Debug(v ...interface{}) {
    l.Output(2, DEBUG, fmt.Sprint(v...))
}

func (l *Logger) Debugf(format string, v ...interface{}) {
    l.Output(2, DEBUG, fmt.Sprintf(format, v...))
}

func (l *Logger) Debugln(v ...interface{}) {
    l.Output(2, DEBUG, fmt.Sprintln(v...))
}

// ------------------------------------------------------------

func (l *Logger) Info(v ...interface{}) {
    l.Output(2, INFO, fmt.Sprint(v...))
}

func (l *Logger) Infof(format string, v ...interface{}) {
    l.Output(2, INFO, fmt.Sprintf(format, v...))
}

func (l *Logger) Infoln(v ...interface{}) {
    l.Output(2, INFO, fmt.Sprintln(v...))
}

// ------------------------------------------------------------

func (l *Logger) Warn(v ...interface{}) {
    l.Output(2, WARN, fmt.Sprint(v...))
}

func (l *Logger) Warnf(format string, v ...interface{}) {
    l.Output(2, WARN, fmt.Sprintf(format, v...))
}

func (l *Logger) Warnln(v ...interface{}) {
    l.Output(2, WARN, fmt.Sprintln(v...))
}

// ------------------------------------------------------------

func (l *Logger) Error(v ...interface{}) {
    l.Output(2, ERROR, fmt.Sprint(v...))
}

func (l *Logger) Errorf(format string, v ...interface{}) {
    l.Output(2, ERROR, fmt.Sprintf(format, v...))
}

func (l *Logger) Errorln(v ...interface{}) {
    l.Output(2, ERROR, fmt.Sprintln(v...))
}

// ------------------------------------------------------------

func (l *Logger) Fatal(v ...interface{}) {
    l.Output(2, FATAL, fmt.Sprint(v...))
    os.Exit(1)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
    l.Output(2, FATAL, fmt.Sprintf(format, v...))
    os.Exit(1)
}

func (l *Logger) Fatalln(v ...interface{}) {
    l.Output(2, FATAL, fmt.Sprintln(v...))
    os.Exit(1)
}

// ------------------------------------------------------------

func (l *Logger) Panic(v ...interface{}) {
    s := fmt.Sprint(v...)
    l.Output(2, FATAL, s)
    panic(s)
}

func (l *Logger) Panicf(format string, v ...interface{}) {
    s := fmt.Sprintf(format, v...)
    l.Output(2, FATAL, s)
    panic(s)
}

func (l *Logger) Panicln(v ...interface{}) {
    s := fmt.Sprintln(v...)
    l.Output(2, FATAL, s)
    panic(s)
}

// ============================================================

var (
    LevelFilter    = root.LevelFilter
    SetLevelFilter = root.SetLevelFilter
    Flags          = root.Flags
    SetFlags       = root.SetFlags
    Prefix         = root.Prefix
    SetPrefix      = root.SetPrefix
    WouldLog       = root.WouldLog
)

func Print(v ...interface{}) {
    root.Output(2, root.defaultLevel, fmt.Sprint(v...))
}

func Printf(format string, v ...interface{}) {
    root.Output(2, root.defaultLevel, fmt.Sprintf(format, v...))
}

func Println(v ...interface{}) {
    root.Output(2, root.defaultLevel, fmt.Sprintln(v...))
}

// ------------------------------------------------------------

func Trace(v ...interface{}) {
    root.Output(2, TRACE, fmt.Sprint(v...))
}

func Tracef(format string, v ...interface{}) {
    root.Output(2, TRACE, fmt.Sprintf(format, v...))
}

func Traceln(v ...interface{}) {
    root.Output(2, TRACE, fmt.Sprintln(v...))
}

// ------------------------------------------------------------

func Debug(v ...interface{}) {
    root.Output(2, DEBUG, fmt.Sprint(v...))
}

func Debugf(format string, v ...interface{}) {
    root.Output(2, DEBUG, fmt.Sprintf(format, v...))
}

func Debugln(v ...interface{}) {
    root.Output(2, DEBUG, fmt.Sprintln(v...))
}

// ------------------------------------------------------------

func Info(v ...interface{}) {
    root.Output(2, INFO, fmt.Sprint(v...))
}

func Infof(format string, v ...interface{}) {
    root.Output(2, INFO, fmt.Sprintf(format, v...))
}

func Infoln(v ...interface{}) {
    root.Output(2, INFO, fmt.Sprintln(v...))
}

// ------------------------------------------------------------

func Warn(v ...interface{}) {
    root.Output(2, WARN, fmt.Sprint(v...))
}

func Warnf(format string, v ...interface{}) {
    root.Output(2, WARN, fmt.Sprintf(format, v...))
}

func Warnln(v ...interface{}) {
    root.Output(2, WARN, fmt.Sprintln(v...))
}

// ------------------------------------------------------------

func Error(v ...interface{}) {
    root.Output(2, ERROR, fmt.Sprint(v...))
}

func Errorf(format string, v ...interface{}) {
    root.Output(2, ERROR, fmt.Sprintf(format, v...))
}

func Errorln(v ...interface{}) {
    root.Output(2, ERROR, fmt.Sprintln(v...))
}

// ------------------------------------------------------------

func Fatal(v ...interface{}) {
    root.Output(2, FATAL, fmt.Sprint(v...))
    os.Exit(1)
}

func Fatalf(format string, v ...interface{}) {
    root.Output(2, FATAL, fmt.Sprintf(format, v...))
    os.Exit(1)
}

func Fatalln(v ...interface{}) {
    root.Output(2, FATAL, fmt.Sprintln(v...))
    os.Exit(1)
}

// ------------------------------------------------------------

func Panic(v ...interface{}) {
    s := fmt.Sprint(v...)
    root.Output(2, FATAL, s)
    panic(s)
}

func Panicf(format string, v ...interface{}) {
    s := fmt.Sprintf(format, v...)
    root.Output(2, FATAL, s)
    panic(s)
}

func Panicln(v ...interface{}) {
    s := fmt.Sprintln(v...)
    root.Output(2, FATAL, s)
    panic(s)
}
