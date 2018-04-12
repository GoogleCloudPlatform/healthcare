package com.google.cloud.healthcare;

import java.util.Properties;
import org.apache.log4j.PropertyConfigurator;

/** LogUtil contains utilities for logging. */
public class LogUtil {
  // Print all logs emitted by log4j (as low as DEBUG level) to stdout. This
  // is useful for seeing dcm4che errors in verbose mode.
  public static void Log4jToStdout() {
    Properties log4jProperties = new Properties();
    log4jProperties.setProperty("log4j.rootLogger", "DEBUG, console");
    log4jProperties.setProperty("log4j.appender.console", "org.apache.log4j.ConsoleAppender");
    log4jProperties.setProperty("log4j.appender.console.layout", "org.apache.log4j.PatternLayout");
    log4jProperties.setProperty(
        "log4j.appender.console.layout.ConversionPattern", "%-5p %c %x - %m%n");
    PropertyConfigurator.configure(log4jProperties);
  }

  private LogUtil() {}
}
