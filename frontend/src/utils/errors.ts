const translations: Array<[RegExp, string]> = [
  [/auth\.users must contain at least one user when auth is enabled/i, '开启认证前，请先新增至少一个认证用户。'],
  [/username is required/i, '请输入用户名。'],
  [/password is required/i, '请输入密码。'],
  [/user "(.+)" already exists/i, '该用户名已存在，请换一个用户名。'],
  [/user "(.+)" not found/i, '用户不存在。'],
  [/at least one inbound protocol must be enabled/i, '请至少开启一种入站协议：SOCKS5 或 HTTP CONNECT。'],
  [/port must be between/i, '端口必须在 1 到 65535 之间。'],
  [/host must be an IP address/i, '监听地址必须是 IP 地址，例如 0.0.0.0 或 127.0.0.1。'],
  [/max_connections/i, '最大并发连接数必须大于 0。'],
  [/same address/i, 'SOCKS5 和 HTTP CONNECT 不能使用完全相同的监听地址和端口。'],
  [/duplicate username/i, '认证用户名重复，请使用唯一用户名。'],
  [/bcrypt hash/i, '密码格式无效，请通过页面新增或重置密码。'],
  [/proxy server is already running/i, '代理服务已经在运行。'],
  [/listen socks5/i, 'SOCKS5 监听启动失败，请检查端口是否被占用。'],
  [/listen http/i, 'HTTP CONNECT 监听启动失败，请检查端口是否被占用。'],
  [/bind: Only one usage of each socket address/i, '端口已被其它程序占用，请更换端口或关闭占用程序。'],
  [/address already in use/i, '端口已被其它程序占用，请更换端口或关闭占用程序。']
]

export function friendlyError(err: unknown) {
  const raw = err instanceof Error ? err.message : String(err)
  for (const [pattern, text] of translations) {
    if (pattern.test(raw)) return text
  }
  return raw || '操作失败，请稍后重试。'
}
