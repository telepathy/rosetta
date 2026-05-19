function pageLogin() {
  document.title = 'Rosetta - 登录';
  return (
    '<div class="login-page">' +
    '<div class="login-card">' +
    '<h1>Rosetta</h1>' +
    '<div class="subtitle">企业数据治理平台</div>' +
    '<div class="form-group"><label>用户名</label><input id="login-username" placeholder="请输入用户名"></div>' +
    '<div class="form-group"><label>密码</label><input id="login-password" type="password" placeholder="请输入密码" onkeydown="if(event.key===\'Enter\')doLogin()"></div>' +
    '<button class="btn btn-primary" onclick="doLogin()">登 录</button>' +
    '<div id="login-error" class="error-msg hidden"></div>' +
    '</div></div>'
  );
}

async function doLogin() {
  var u = document.getElementById('login-username').value;
  var p = document.getElementById('login-password').value;
  var err = document.getElementById('login-error');
  if (!u || !p) { err.textContent = '请输入用户名和密码'; err.classList.remove('hidden'); return; }
  try {
    await api.login(u, p);
    router.navigate('/');
  } catch (e) {
    err.textContent = e.message;
    err.classList.remove('hidden');
  }
}
window.doLogin = doLogin;

async function pageDashboard() {
  document.title = 'Rosetta - 仪表盘';
  var html = sidebarHtml('/') + pageHeader('仪表盘', '系统概览');
  html += '<div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:16px;">';

  var counts = { users: '-', instances: '-', dicts: '-', models: '-' };
  try {
    var results = await Promise.all([
      api.get('/users?page_size=1'), api.get('/instances?page_size=1'),
      api.get('/dicts?page_size=1'), api.get('/models?page_size=1')
    ]);
    counts.users = results[0].data.total;
    counts.instances = results[1].data.total;
    counts.dicts = results[2].data.total;
    counts.models = results[3].data.total;
  } catch (e) {}

  var cards = [
    ['👥 用户', '/users', counts.users],
    ['🗄️ 实例', '/instances', counts.instances],
    ['📖 字典', '/dicts', counts.dicts],
    ['📐 模型', '/models', counts.models],
  ];
  cards.forEach(function(c) {
    html += '<a href="#' + c[1] + '" style="text-decoration:none"><div class="card" style="text-align:center;padding:32px 20px;cursor:pointer"><div style="font-size:36px;margin-bottom:8px">' + c[2] + '</div><div style="color:var(--text-secondary)">' + c[0] + '</div></div></a>';
  });
  html += '</div></div></div>';
  return html;
}
