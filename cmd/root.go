// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"os"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "grctool",
	Short: "Security Program Manager for GRC systems integration",
	Long: `A CLI application for managing security program compliance through GRC systems integration.

This tool helps automate the collection, assembly, and submission of evidence for security compliance
frameworks like SOC 2, ISO 27001, and others by integrating with GRC systems like Tugboat Logic and
extracting relevant configuration snippets from your infrastructure code.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default searches $PWD then $HOME for .grctool.yaml)")
	rootCmd.PersistentFlags().Bool("verbose", false, "verbose output")
	rootCmd.PersistentFlags().String("log-level", "warn", "console log level (trace, debug, info, warn, error)")
	rootCmd.PersistentFlags().String("log-file", "", "log file location (default: OS-appropriate path)")
	rootCmd.PersistentFlags().String("log-file-level", "info", "file log level (trace, debug, info, warn, error)")
	rootCmd.PersistentFlags().Bool("no-log-file", false, "disable file logging")

	// Bind flags to viper
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	_ = viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
	_ = viper.BindPFlag("log-file", rootCmd.PersistentFlags().Lookup("log-file"))
	_ = viper.BindPFlag("log-file-level", rootCmd.PersistentFlags().Lookup("log-file-level"))
	_ = viper.BindPFlag("no-log-file", rootCmd.PersistentFlags().Lookup("no-log-file"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in current directory first, then home directory.
		// This allows project-specific configs to override global configs.
		viper.AddConfigPath(".") // Current working directory ($PWD)

		// Find home directory and add as fallback
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(home) // Home directory ($HOME)

		viper.SetConfigType("yaml")
		viper.SetConfigName(".grctool")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if viper.GetBool("verbose") {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}

	// Initialize logging system
	initLogging()
}

// initLogging initializes the centralized logging system
func initLogging() {
	// Check if this is a tool subcommand to adjust logging behavior
	isToolCommand := len(os.Args) > 1 && os.Args[1] == "tool"

	// Check if file logging is disabled before anything else
	noLogFile := viper.GetBool("no-log-file")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		// Even if config fails, we should still set up file logging if requested
		var loggers []logger.Logger

		// Create default console logger
		defaultConfig := logger.DefaultConfig()
		if viper.IsSet("log-level") {
			defaultConfig.Level = logger.ParseLogLevel(viper.GetString("log-level"))
		}

		// For tool commands, use stderr and warn level by default
		if isToolCommand {
			defaultConfig.Output = "stderr"
			if !viper.IsSet("log-level") {
				defaultConfig.Level = logger.WarnLevel
			}
		}

		consoleLogger, consoleErr := logger.NewZerologLogger(defaultConfig)
		if consoleErr != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize default logger: %v\n", consoleErr)
			return
		}
		loggers = append(loggers, consoleLogger)

		// Create file logger if not disabled
		logFilePath := viper.GetString("log-file")
		if logFilePath == "" {
			// No explicit path specified, use OS-appropriate default
			logFilePath = logger.DefaultLogFilePath()
		}
		if !noLogFile {
			fileConfig := &logger.Config{
				Level:         logger.TraceLevel, // Default to trace for comprehensive logging
				Format:        "text",
				Output:        "file",
				FilePath:      logFilePath,
				SanitizeURLs:  true,
				RedactFields:  defaultConfig.RedactFields,
				ShowCaller:    true,
				BufferSize:    defaultConfig.BufferSize,
				FlushInterval: defaultConfig.FlushInterval,
			}

			// Override file log level if specified
			if viper.IsSet("log-file-level") {
				fileConfig.Level = logger.ParseLogLevel(viper.GetString("log-file-level"))
			}

			fileLogger, fileErr := logger.NewZerologLogger(fileConfig)
			if fileErr != nil {
				consoleLogger.Warn("failed to initialize file logger",
					logger.Error(fileErr),
					logger.String("file", fileConfig.FilePath))
			} else {
				loggers = append(loggers, fileLogger)
			}
		}

		multiLogger := logger.NewMultiLogger(loggers...)
		logger.InitGlobalWithLogger(multiLogger)

		// Only warn about config issues if a config file was actually found
		// No config file is expected for new users (they should run 'grctool init' first)
		if viper.ConfigFileUsed() != "" {
			multiLogger.Warn("configuration failed to load, using default logging", logger.Error(err))
		}
		return
	}

	// Use default logging config if no loggers are configured
	if len(cfg.Logging.Loggers) == 0 {
		cfg.Logging = *config.DefaultLoggingConfig()
	}

	// Override console logger level from command line flag if provided
	if viper.IsSet("log-level") && cfg.Logging.Loggers["console"].Enabled {
		consoleCfg := cfg.Logging.Loggers["console"]
		consoleCfg.Level = viper.GetString("log-level")
		cfg.Logging.Loggers["console"] = consoleCfg
	}

	// Override file logger level from command line flag if provided
	if viper.IsSet("log-file-level") && cfg.Logging.Loggers["file"].Enabled {
		fileCfg := cfg.Logging.Loggers["file"]
		fileCfg.Level = viper.GetString("log-file-level")
		cfg.Logging.Loggers["file"] = fileCfg
	}

	// Override file logger path from command line flag if provided
	if viper.IsSet("log-file") && cfg.Logging.Loggers["file"].Enabled {
		fileCfg := cfg.Logging.Loggers["file"]
		fileCfg.FilePath = viper.GetString("log-file")
		cfg.Logging.Loggers["file"] = fileCfg
	}

	// Disable file logger if --no-log-file flag is set
	if noLogFile && cfg.Logging.Loggers["file"].Enabled {
		fileCfg := cfg.Logging.Loggers["file"]
		fileCfg.Enabled = false
		cfg.Logging.Loggers["file"] = fileCfg
	}

	var loggers []logger.Logger
	var loggerInfo []string

	// Create each enabled logger
	for name, loggerCfg := range cfg.Logging.Loggers {
		if !loggerCfg.Enabled {
			continue
		}

		// For tool commands, override console logger to use stderr and warn level
		if isToolCommand && name == "console" {
			loggerCfg.Output = "stderr"
			if !viper.IsSet("log-level") {
				loggerCfg.Level = "warn"
			}
		}

		loggerConfig := loggerCfg.ToLoggerConfig()
		l, err := logger.NewZerologLogger(loggerConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize %s logger: %v\n", name, err)
			continue
		}

		loggers = append(loggers, l)
		loggerInfo = append(loggerInfo, fmt.Sprintf("%s=%s@%s", name, loggerCfg.Level, loggerCfg.Output))
	}

	// Fallback to default logger if no loggers were created
	if len(loggers) == 0 {
		fmt.Fprintf(os.Stderr, "No loggers initialized, falling back to default\n")
		if err := logger.InitGlobal(logger.DefaultConfig()); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize default logger: %v\n", err)
		}
		return
	}

	// Initialize global logger with multi-logger
	multiLogger := logger.NewMultiLogger(loggers...)
	logger.InitGlobalWithLogger(multiLogger)

	multiLogger.Info("logging system initialized",
		logger.String("loggers", fmt.Sprintf("%v", loggerInfo)),
	)
}
