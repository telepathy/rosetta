function toast(msg, type) {
  type = type || 'info';
  const c = document.getElementById('toast-container');
  const el = document.createElement('div');
  el.className = 'toast toast-' + type;
  el.textContent = msg;
  c.appendChild(el);
  setTimeout(function() { el.style.opacity = '0'; setTimeout(function() { el.remove(); }, 300); }, 3000);
}

function openModal(title, bodyHTML, footerHTML, wide) {
  var overlay = document.getElementById('modal-overlay');
  var content = document.getElementById('modal-content');
  if (wide) content.classList.add('wide'); else content.classList.remove('wide');
  content.innerHTML =
    '<div class="modal-header"><h3>' + title + '</h3><button class="modal-close" onclick="closeModal()">✕</button></div>' +
    '<div class="modal-body">' + bodyHTML + '</div>' +
    (footerHTML ? '<div class="modal-footer">' + footerHTML + '</div>' : '');
  overlay.classList.remove('hidden');
}

function closeModal(e) {
  if (e && e.target !== document.getElementById('modal-overlay')) return;
  document.getElementById('modal-overlay').classList.add('hidden');
}

function btnSm(label, fn, cls) {
  var id = 'btn_' + Math.random().toString(36).slice(2);
  window[id] = fn;
  return '<button class="btn btn-sm ' + (cls || 'btn-outline') + '" onclick="window.' + id + '()">' + label + '</button>';
}

function btnSmId(label, id, fn, cls) {
  window[id] = fn;
  return '<button class="btn btn-sm ' + (cls || 'btn-outline') + '" onclick="window.' + id + '()">' + label + '</button>';
}

function renderTable(headers, rows, actionsFn) {
  var html = '<table><thead><tr>';
  headers.forEach(function(h) { html += '<th>' + h + '</th>'; });
  if (actionsFn) html += '<th style="width:140px">操作</th>';
  html += '</tr></thead><tbody>';
  if (rows.length === 0) {
    var n = headers.length + (actionsFn ? 1 : 0);
    html += '<tr><td colspan="' + n + '" style="text-align:center;color:var(--text-secondary);padding:40px">暂无数据</td></tr>';
  }
  rows.forEach(function(row) {
    html += '<tr>';
    row.forEach(function(cell) { html += '<td>' + cell + '</td>'; });
    if (actionsFn) html += '<td>' + actionsFn(row) + '</td>';
    html += '</tr>';
  });
  html += '</tbody></table>';
  return html;
}

function renderPagination(page, total, pageSize, onPageFn) {
  var totalPages = Math.ceil(total / pageSize) || 1;
  if (totalPages <= 1) return '';
  window._pagination_cb = onPageFn;
  var html = '<div class="pagination">';
  html += '<button ' + (page <= 1 ? 'disabled' : '') + ' onclick="window._pagination_cb(' + (page - 1) + ')">上一页</button>';
  for (var i = 1; i <= totalPages; i++) {
    html += '<button class="' + (i === page ? 'active' : '') + '" onclick="window._pagination_cb(' + i + ')">' + i + '</button>';
  }
  html += '<button ' + (page >= totalPages ? 'disabled' : '') + ' onclick="window._pagination_cb(' + (page + 1) + ')">下一页</button>';
  html += '<span class="page-info">共 ' + total + ' 条</span></div>';
  return html;
}

function sidebarHtml(currentPath) {
  var items = [
    ['/', '📊', '仪表盘'],
    ['/users', '👥', '用户管理'],
    ['/instances', '🗄️', '实 例管理'],
    ['/dicts', '📖', '字典维护'],
    ['/models', '📐', '模型管理'],
  ];
  var html = '<div class="app-layout"><div class="sidebar">';
  html += '<div class="sidebar-header"><h2>Rosetta</h2><div class="version">数据治理平台</div></div>';
  html += '<nav class="sidebar-nav">';
  items.forEach(function(item) {
    html += '<a href="#' + item[0] + '" class="' + (currentPath === item[0] ? 'active' : '') + '"><span class="nav-icon">' + item[1] + '</span><span>' + item[2] + '</span></a>';
  });
  html += '</nav><div class="sidebar-footer">';
  if (api.user) {
    html += '<div class="user-info"><div class="user-name">' + api.user.display_name + '</div>';
    html += '<div class="user-role">' + (api.user.roles || []).join(', ') + '</div></div>';
  }
  html += '<button class="btn-logout" onclick="doLogout()">退出登录</button>';
  html += '</div></div><div class="main-content" id="main-content">';
  return html;
}

function pageHeader(title, desc) {
  return '<div class="page-header"><h2>' + title + '</h2>' + (desc ? '<div class="page-desc">' + desc + '</div>' : '') + '</div>';
}

function escapeHtml(str) {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

async function doLogout() {
  try { await api.post('/auth/logout'); } catch (e) {}
  api.clearToken();
  router.navigate('/login');
}

window.closeModal = closeModal;
window.doLogout = doLogout;
window.toast = toast;
