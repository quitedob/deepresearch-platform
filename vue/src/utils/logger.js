/**
 * 条件日志工具 — 生产环境静默，开发环境输出
 * 替代散落在代码中的 console.log/warn/error
 */
const isDev = import.meta.env.DEV

const logger = {
  log: (...args) => { if (isDev) console.log(...args) },
  warn: (...args) => { if (isDev) console.warn(...args) },
  debug: (...args) => { if (isDev) console.debug(...args) },
  // error 始终输出，生产环境也需要错误信息
  error: (...args) => console.error(...args),
}

export default logger
