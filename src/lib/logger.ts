import c from "ansi-colors";
import dayjs from "dayjs";

/**
 * Logger class options
 */
interface LoggerOptions {
  /**
   * Time format for logger (dayjs)
   */
  timeformat?: string;
}

/**
 * Each Logger function options
 */
interface LogOption {
  /**
   * Context of logger
   */
  context?: string;
}

const DEFAULT_TIME_FORMAT = "YYYY/MM/DD HH:mm:ss A";

/**
 * Custom Logger class
 * @param context Context of logger
 * @param options Options for logger
 */
export class Logger {
  private PID: number;
  private CONTEXT: string;
  constructor(context = "Next", private readonly options?: LoggerOptions) {
    this.CONTEXT = `[${context}]`;
    this.PID = process.pid;
  }

  private timestamp() {
    return dayjs().format(this.options?.timeformat || DEFAULT_TIME_FORMAT);
  }

  /**
   * Log out message
   * @param options Options for log
   * @param message Message to log
   */
  static log(message: string, options?: LogOption) {
    const title = c.green(`[LOG] ${process.pid} -`);
    const formattedMessage = c.green(message);
    const context = c.yellow(`[${options?.context}]` || "[Next]");
    console.log(
      [
        title,
        dayjs().format(DEFAULT_TIME_FORMAT),
        context,
        formattedMessage,
      ].join(" ")
    );
  }
  /**
   * Log out message
   * @param message Message to log
   * @param options Options for log
   */
  public log(message: string, options?: LogOption) {
    const title = c.green(`[LOG] ${this.PID} -`);
    const formattedMessage = c.green(message);
    const context = c.yellow(`[${options?.context}]` || this.CONTEXT);
    console.log([title, this.timestamp(), context, formattedMessage].join(" "));
  }

  /**
   * Log out error message
   * @param message Message to error
   * @param options Options for error
   */
  static error(message: string, options?: LogOption) {
    const title = c.red(`[ERROR] ${process.pid} -`);
    const formattedMessage = c.red(message);
    const context = c.yellow(`[${options?.context}]` || "[Next]");
    console.log(
      [
        title,
        dayjs().format(DEFAULT_TIME_FORMAT),
        context,
        formattedMessage,
      ].join(" ")
    );
  }
  /**
   * Log out error message
   * @param message Message to error
   * @param options Options for error
   */
  public error(message: string, options?: LogOption) {
    const title = c.red(`[ERROR] ${this.PID} -`);
    const formattedMessage = c.red(message);
    const context = c.yellow(`[${options?.context}]` || this.CONTEXT);
    console.log([title, this.timestamp(), context, formattedMessage].join(" "));
  }

  /**
   * Log out warn message
   * @param message Message to warn
   * @param options Options for warn
   */
  static warn(message: string, options?: LogOption) {
    const title = c.yellow(`[WARN] ${process.pid} -`);
    const formattedMessage = c.yellow(message);
    const context = c.yellow(`[${options?.context}]` || "[Next]");
    console.log(
      [
        title,
        dayjs().format(DEFAULT_TIME_FORMAT),
        context,
        formattedMessage,
      ].join(" ")
    );
  }
  /**
   * Log out warn message
   * @param message Message to warn
   * @param options Options for warn
   */
  public warn(message: string, options?: LogOption) {
    const title = c.yellow(`[WARN] ${this.PID} -`);
    const formattedMessage = c.yellow(message);
    const context = c.yellow(`[${options?.context}]` || this.CONTEXT);
    console.log([title, this.timestamp(), context, formattedMessage].join(" "));
  }

  /**
   * Log out debug message
   * @param message Message to debug
   * @param options Options for debug
   */
  static debug(message: string, options?: LogOption) {
    const title = c.blue(`[DEBUG] ${process.pid} -`);
    const formattedMessage = c.blue(message);
    const context = c.yellow(`[${options?.context}]` || "[Next]");
    console.log(
      [
        title,
        dayjs().format(DEFAULT_TIME_FORMAT),
        context,
        formattedMessage,
      ].join(" ")
    );
  }
  /**
   * Log out debug message
   * @param message Message to debug
   * @param options Options for debug
   */
  public debug(message: string, options?: LogOption) {
    const title = c.blue(`[DEBUG] ${this.PID} -`);
    const formattedMessage = c.blue(message);
    const context = c.yellow(`[${options?.context}]` || this.CONTEXT);
    console.log([title, this.timestamp(), context, formattedMessage].join(" "));
  }
}
