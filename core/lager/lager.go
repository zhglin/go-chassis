package lager

import (
	"github.com/go-chassis/seclog"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chassis/openlog"
	"github.com/go-chassis/seclog/third_party/forked/cloudfoundry/lager"
)

// constant values for log rotate parameters
const (
	LogRotateDate  = 1
	LogRotateSize  = 10
	LogBackupCount = 7
)

// log level
const (
	LevelDebug = "DEBUG"
	LevelInfo  = "INFO"
)

// output type
const (
	Stdout = "stdout"
	File   = "file"
)

// logFilePath log file path
var logFilePath string

//Options is the struct for lager information(lager.yaml)
type Options struct {
	Writers       string `yaml:"logWriters"` // 写入目的地
	LoggerLevel   string `yaml:"logLevel"`   // log级别
	LoggerFile    string `yaml:"logFile"`    // log文件
	LogFormatText bool   `yaml:"logFormatText"`

	LogRotateDisable  bool `yaml:"logRotateDisable"` // 是否开启日志轮转
	LogRotateCompress bool `yaml:"logRotateCompress"`
	LogRotateAge      int  `yaml:"logRotateAge"`
	LogRotateSize     int  `yaml:"logRotateSize"`
	LogBackupCount    int  `yaml:"logBackupCount"`

	AccessLogFile string `yaml:"accessLogFile"`
}

// Init Build constructs a *Lager.logger with the configured parameters.
func Init(option *Options) {
	var err error
	logger, err := NewLog(option)
	// log文件打开失败
	if err != nil {
		panic(err)
	}
	openlog.SetLogger(logger)
	openlog.Debug("logger init success")
}

// NewLog returns a logger
// 创建log
func NewLog(option *Options) (lager.Logger, error) {
	checkPassLagerDefinition(option)

	localPath := ""
	if !filepath.IsAbs(option.LoggerFile) { // 不是绝对路径
		localPath = os.Getenv("CHASSIS_HOME")
	}

	// 创建log文件
	err := createLogFile(localPath, option.LoggerFile)
	if err != nil {
		return nil, err
	}

	logFilePath = filepath.Join(localPath, option.LoggerFile)

	// 日志写入的目的地
	writers := strings.Split(strings.TrimSpace(option.Writers), ",")

	option.LoggerFile = logFilePath
	seclog.Init(seclog.Config{
		LoggerLevel:   option.LoggerLevel,
		LogFormatText: option.LogFormatText,
		Writers:       writers,
		LoggerFile:    logFilePath,
		RotateDisable: option.LogRotateDisable,
		MaxSize:       option.LogRotateSize,
		MaxAge:        option.LogRotateAge,
		MaxBackups:    option.LogBackupCount,
		Compress:      option.LogRotateCompress,
	})
	logger := seclog.NewLogger("ut")
	return logger, nil
}

// checkPassLagerDefinition check pass lager definition
// 校验log配置
func checkPassLagerDefinition(option *Options) {
	if option.LoggerLevel == "" {
		option.LoggerLevel = "DEBUG"
	}

	if option.LoggerFile == "" {
		option.LoggerFile = "log/chassis.log"
	}

	if option.LogRotateAge <= 0 || option.LogRotateAge > 10 {
		option.LogRotateAge = LogRotateDate
	}

	if option.LogRotateSize <= 0 || option.LogRotateSize > 50 {
		option.LogRotateSize = LogRotateSize
	}

	if option.LogBackupCount < 0 || option.LogBackupCount > 100 {
		option.LogBackupCount = LogBackupCount
	}
	if option.Writers == "" {
		option.Writers = "file,stdout"
	}
}

// createLogFile create log file
// 创建log文件
func createLogFile(localPath, out string) error {
	_, err := os.Stat(strings.Replace(filepath.Dir(filepath.Join(localPath, out)), "\\", "/", -1))
	// 文件不存在 创建目录
	if err != nil && os.IsNotExist(err) {
		err := os.MkdirAll(strings.Replace(filepath.Dir(filepath.Join(localPath, out)), "\\", "/", -1), os.ModePerm)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	// 尝试打开一次
	f, err := os.OpenFile(strings.Replace(filepath.Join(localPath, out), "\\", "/", -1), os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	return f.Close()
}
