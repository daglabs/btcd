// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2017 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/daglabs/btcd/logs"
	"github.com/jrick/logrotate/rotator"
)

// logWriter implements an io.Writer that outputs to both standard output and
// the write-end pipe of an initialized log rotator.
type logWriter struct{}

func (logWriter) Write(p []byte) (n int, err error) {
	if initiated {
		os.Stdout.Write(p)
		LogRotator.Write(p)
	}
	return len(p), nil
}

// errLogWriter implements an io.Writer that outputs to both standard output and
// the write-end pipe of an initialized log rotator.
type errLogWriter struct{}

func (errLogWriter) Write(p []byte) (n int, err error) {
	if initiated {
		os.Stdout.Write(p)
		ErrLogRotator.Write(p)
	}
	return len(p), nil
}

// Loggers per subsystem.  A single backend logger is created and all subsytem
// loggers created from it will write to the backend.  When adding new
// subsystems, add the subsystem logger variable here and to the
// subsystemLoggers map.
//
// Loggers can not be used before the log rotator has been initialized with a
// log file.  This must be performed early during application startup by calling
// InitLogRotators.
var (
	// backendLog is the logging backend used to create all subsystem loggers.
	// The backend must not be used before the log rotator has been initialized,
	// or data races and/or nil pointer dereferences will occur.
	backendLog = logs.NewBackend([]*logs.BackendWriter{
		logs.NewAllLevelsBackendWriter(logWriter{}),
		logs.NewErrorBackendWriter(errLogWriter{}),
	})

	// LogRotator is one of the logging outputs.  It should be closed on
	// application shutdown.
	LogRotator *rotator.Rotator
	ErrLogRotator *rotator.Rotator

	adxrLog = backendLog.Logger("ADXR")
	amgrLog = backendLog.Logger("AMGR")
	cmgrLog = backendLog.Logger("CMGR")
	bcdbLog = backendLog.Logger("BCDB")
	btcdLog = backendLog.Logger("BTCD")
	chanLog = backendLog.Logger("CHAN")
	cnfgLog = backendLog.Logger("CNFG")
	discLog = backendLog.Logger("DISC")
	indxLog = backendLog.Logger("INDX")
	minrLog = backendLog.Logger("MINR")
	peerLog = backendLog.Logger("PEER")
	rpcsLog = backendLog.Logger("RPCS")
	scrpLog = backendLog.Logger("SCRP")
	srvrLog = backendLog.Logger("SRVR")
	syncLog = backendLog.Logger("SYNC")
	txmpLog = backendLog.Logger("TXMP")
	utilLog = backendLog.Logger("UTIL")

	initiated = false
)

// SubsystemTags is an enum of all sub system tags
var SubsystemTags = struct {
	ADXR,
	AMGR,
	CMGR,
	BCDB,
	BTCD,
	CHAN,
	CNFG,
	DISC,
	INDX,
	MINR,
	PEER,
	RPCS,
	SCRP,
	SRVR,
	SYNC,
	TXMP,
	UTIL string
}{
	ADXR: "ADXR",
	AMGR: "AMGR",
	CMGR: "CMGR",
	BCDB: "BCDB",
	BTCD: "BTCD",
	CHAN: "CHAN",
	CNFG: "CNFG",
	DISC: "DISC",
	INDX: "INDX",
	MINR: "MINR",
	PEER: "PEER",
	RPCS: "RPCS",
	SCRP: "SCRP",
	SRVR: "SRVR",
	SYNC: "SYNC",
	TXMP: "TXMP",
	UTIL: "UTIL",
}

// subsystemLoggers maps each subsystem identifier to its associated logger.
var subsystemLoggers = map[string]logs.Logger{
	SubsystemTags.ADXR: adxrLog,
	SubsystemTags.AMGR: amgrLog,
	SubsystemTags.CMGR: cmgrLog,
	SubsystemTags.BCDB: bcdbLog,
	SubsystemTags.BTCD: btcdLog,
	SubsystemTags.CHAN: chanLog,
	SubsystemTags.CNFG: cnfgLog,
	SubsystemTags.DISC: discLog,
	SubsystemTags.INDX: indxLog,
	SubsystemTags.MINR: minrLog,
	SubsystemTags.PEER: peerLog,
	SubsystemTags.RPCS: rpcsLog,
	SubsystemTags.SCRP: scrpLog,
	SubsystemTags.SRVR: srvrLog,
	SubsystemTags.SYNC: syncLog,
	SubsystemTags.TXMP: txmpLog,
	SubsystemTags.UTIL: utilLog,
}

// InitLogRotators initializes the logging rotaters to
// write logs to logFile, errLogFile, and create roll
// files in the same directory.  It must be called
// before the package-global log rotater variables
// are used.
func InitLogRotators(logFile, errLogFile string) {
	initiated = true
	LogRotator = initLogRotator(logFile)
	ErrLogRotator = initLogRotator(errLogFile)
}

func initLogRotator(logFile string) *rotator.Rotator{
	logDir, _ := filepath.Split(logFile)
	err := os.MkdirAll(logDir, 0700)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create log directory: %s\n", err)
		os.Exit(1)
	}
	r, err := rotator.New(logFile, 10*1024, false, 3)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create file rotator: %s\n", err)
		os.Exit(1)
	}
	return r
}

// SetLogLevel sets the logging level for provided subsystem.  Invalid
// subsystems are ignored.  Uninitialized subsystems are dynamically created as
// needed.
func SetLogLevel(subsystemID string, logLevel string) {
	// Ignore invalid subsystems.
	logger, ok := subsystemLoggers[subsystemID]
	if !ok {
		return
	}

	// Defaults to info if the log level is invalid.
	level, _ := logs.LevelFromString(logLevel)
	logger.SetLevel(level)
}

// SetLogLevels sets the log level for all subsystem loggers to the passed
// level.  It also dynamically creates the subsystem loggers as needed, so it
// can be used to initialize the logging system.
func SetLogLevels(logLevel string) {
	// Configure all sub-systems with the new logging level.  Dynamically
	// create loggers as needed.
	for subsystemID := range subsystemLoggers {
		SetLogLevel(subsystemID, logLevel)
	}
}

// DirectionString is a helper function that returns a string that represents
// the direction of a connection (inbound or outbound).
func DirectionString(inbound bool) string {
	if inbound {
		return "inbound"
	}
	return "outbound"
}

// PickNoun returns the singular or plural form of a noun depending
// on the count n.
func PickNoun(n uint64, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

// SupportedSubsystems returns a sorted slice of the supported subsystems for
// logging purposes.
func SupportedSubsystems() []string {
	// Convert the subsystemLoggers map keys to a slice.
	subsystems := make([]string, 0, len(subsystemLoggers))
	for subsysID := range subsystemLoggers {
		subsystems = append(subsystems, subsysID)
	}

	// Sort the subsystems for stable display.
	sort.Strings(subsystems)
	return subsystems
}

// Get returns a logger of a specific sub system
func Get(tag string) (logger logs.Logger, ok bool) {
	logger, ok = subsystemLoggers[tag]
	return
}

// ParseAndSetDebugLevels attempts to parse the specified debug level and set
// the levels accordingly.  An appropriate error is returned if anything is
// invalid.
func ParseAndSetDebugLevels(debugLevel string) error {
	// When the specified string doesn't have any delimters, treat it as
	// the log level for all subsystems.
	if !strings.Contains(debugLevel, ",") && !strings.Contains(debugLevel, "=") {
		// Validate debug log level.
		if !validLogLevel(debugLevel) {
			str := "The specified debug level [%s] is invalid"
			return fmt.Errorf(str, debugLevel)
		}

		// Change the logging level for all subsystems.
		SetLogLevels(debugLevel)

		return nil
	}

	// Split the specified string into subsystem/level pairs while detecting
	// issues and update the log levels accordingly.
	for _, logLevelPair := range strings.Split(debugLevel, ",") {
		if !strings.Contains(logLevelPair, "=") {
			str := "The specified debug level contains an invalid " +
				"subsystem/level pair [%s]"
			return fmt.Errorf(str, logLevelPair)
		}

		// Extract the specified subsystem and log level.
		fields := strings.Split(logLevelPair, "=")
		subsysID, logLevel := fields[0], fields[1]

		// Validate subsystem.
		if _, exists := Get(subsysID); !exists {
			str := "The specified subsystem [%s] is invalid -- " +
				"supported subsytems %s"
			return fmt.Errorf(str, subsysID, strings.Join(SupportedSubsystems(), ", "))
		}

		// Validate log level.
		if !validLogLevel(logLevel) {
			str := "The specified debug level [%s] is invalid"
			return fmt.Errorf(str, logLevel)
		}

		SetLogLevel(subsysID, logLevel)
	}

	return nil
}

// validLogLevel returns whether or not logLevel is a valid debug log level.
func validLogLevel(logLevel string) bool {
	switch logLevel {
	case "trace":
		fallthrough
	case "debug":
		fallthrough
	case "info":
		fallthrough
	case "warn":
		fallthrough
	case "error":
		fallthrough
	case "critical":
		return true
	}
	return false
}
