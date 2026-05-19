var usersPage = 1;

async function pageUsers() {
  document.title = 'Rosetta - 用户管理';
  var html = sidebarHtml('/users') + pageHeader('用户管理', '管理系统用户账号与角色');
  html += '<div class="page-toolbar"><button class="btn btn-primary" onclick="showUserCreateModal()">+ 新建用户</button><div class="flex-spacer"></div></div>';
  html += '<div id="users-table">加载中...</div>';
  html += '<div id="users-pagination"></div>';
  html += '</div></div>';
  setTimeout(loadUsers, 0);
  return html;
}

async function loadUsers() {
  try {
    var data = await api.get('/users?page=' + usersPage + '&page_size=15');
    var users = data.data.items;
    var html = renderTable(['ID', '用户名', '显示名', '邮箱', '状态'], users.map(function(u) {
      return [
        u.id, u.username, u.display_name, u.email || '-',
        u.status === 'ACTIVE' ? '<span class="badge badge-success">启用</span>' : '<span class="badge badge-danger">禁用</span>'
      ];
    }), function(row) {
      var id = row[0];
      var name = row[1];
      var isActive = row[4].indexOf('启用') >= 0;
      var newStatus = isActive ? 'DISABLED' : 'ACTIVE';
      var actions = '';
      actions += btnSm('编辑', function() { showUserEditModal(id); }) + ' ';
      actions += btnSm('角色', function() { showUserRoleModal(id, name); }) + ' ';
      actions += btnSm(isActive ? '禁用' : '启用', function() { toggleUserStatus(id, newStatus); }, 'btn-outline');
      return actions;
    });
    document.getElementById('users-table').innerHTML = html;
    document.getElementById('users-pagination').innerHTML = renderPagination(usersPage, data.data.total, 15, function(p) { usersPage = p; loadUsers(); });
  } catch (e) {
    document.getElementById('users-table').innerHTML = '<div class="empty-state"><p>加载失败: ' + e.message + '</p></div>';
  }
}

async function showUserCreateModal() {
  var roles = [];
  try { roles = (await api.get('/roles')).data; } catch (e) {}

  var rolesHtml = roles.map(function(r) {
    return '<label class="checkbox-group"><input type="checkbox" value="' + r.id + '"> ' + r.role_name + '</label>';
  }).join('');

  openModal('新建用户',
    '<div class="form-group"><label>用户名</label><input id="mu-username"></div>' +
    '<div class="form-group"><label>密码</label><input id="mu-password" type="password" placeholder="至少6位"></div>' +
    '<div class="form-group"><label>显示名</label><input id="mu-displayname"></div>' +
    '<div class="form-group"><label>邮箱</label><input id="mu-email"></div>' +
    '<div class="form-group"><label>角色</label><div id="mu-roles">' + rolesHtml + '</div></div>',
    '<button class="btn btn-outline" onclick="closeModal()">取消</button>' +
    '<button class="btn btn-primary" onclick="saveNewUser()">创建</button>'
  );
}
window.showUserCreateModal = showUserCreateModal;

async function saveNewUser() {
  var roleChecks = document.querySelectorAll('#mu-roles input:checked');
  var roleIds = Array.prototype.map.call(roleChecks, function(c) { return parseInt(c.value); });

  try {
    await api.post('/users', {
      username: document.getElementById('mu-username').value,
      password: document.getElementById('mu-password').value,
      display_name: document.getElementById('mu-displayname').value,
      email: document.getElementById('mu-email').value,
      role_ids: roleIds
    });
    closeModal();
    toast('创建成功', 'success');
    loadUsers();
  } catch (e) { toast(e.message, 'error'); }
}
window.saveNewUser = saveNewUser;

async function showUserEditModal(userId) {
  var user, roles;
  try { user = (await api.get('/users/' + userId)).data; roles = (await api.get('/roles')).data; } catch (e) { return; }

  var rolesHtml = roles.map(function(r) {
    var checked = user.roles && user.roles.some(function(ur) { return ur.id === r.id; }) ? 'checked' : '';
    return '<label class="checkbox-group"><input type="checkbox" value="' + r.id + '" ' + checked + '> ' + r.role_name + '</label>';
  }).join('');

  openModal('编辑用户 - ' + user.username,
    '<div class="form-group"><label>用户名</label><input id="mu-username" value="' + user.username + '" disabled></div>' +
    '<div class="form-group"><label>显示名</label><input id="mu-displayname" value="' + user.display_name + '"></div>' +
    '<div class="form-group"><label>邮箱</label><input id="mu-email" value="' + (user.email || '') + '"></div>' +
    '<div class="form-group"><label>角色</label><div id="mu-roles">' + rolesHtml + '</div></div>',
    '<button class="btn btn-outline" onclick="resetPassword(' + userId + ')">重置密码</button>' +
    '<button class="btn btn-outline" onclick="closeModal()">取消</button>' +
    '<button class="btn btn-primary" onclick="saveEditUser(' + userId + ')">保存</button>'
  );
}
window.showUserEditModal = showUserEditModal;

async function saveEditUser(userId) {
  var roleChecks = document.querySelectorAll('#mu-roles input:checked');
  var roleIds = Array.prototype.map.call(roleChecks, function(c) { return parseInt(c.value); });

  try {
    await api.put('/users/' + userId, {
      display_name: document.getElementById('mu-displayname').value,
      email: document.getElementById('mu-email').value
    });
    await api.put('/users/' + userId + '/roles', { role_ids: roleIds });
    closeModal();
    toast('保存成功', 'success');
    loadUsers();
  } catch (e) { toast(e.message, 'error'); }
}
window.saveEditUser = saveEditUser;

async function resetPassword(userId) {
  var pwd = prompt('请输入新密码（至少6位）:');
  if (!pwd || pwd.length < 6) return;
  try {
    await api.put('/users/' + userId + '/password', { password: pwd });
    toast('密码已重置', 'success');
  } catch (e) { toast(e.message, 'error'); }
}
window.resetPassword = resetPassword;

async function toggleUserStatus(userId, status) {
  try {
    await api.put('/users/' + userId + '/status', { status: status });
    toast(status === 'ACTIVE' ? '已启用' : '已禁用', 'success');
    loadUsers();
  } catch (e) { toast(e.message, 'error'); }
}
window.toggleUserStatus = toggleUserStatus;

async function showUserRoleModal(userId, username) {
  var roles, user;
  try { roles = (await api.get('/roles')).data; user = (await api.get('/users/' + userId)).data; } catch (e) { return; }

  var rolesHtml = roles.map(function(r) {
    var checked = user.roles && user.roles.some(function(ur) { return ur.id === r.id; }) ? 'checked' : '';
    return '<label class="checkbox-group"><input type="checkbox" value="' + r.id + '" ' + checked + '> ' + r.role_name + ' <span style="font-size:11px;color:var(--text-secondary)">' + r.description + '</span></label>';
  }).join('');

  openModal('分配角色 - ' + username,
    '<p style="margin-bottom:12px">为用户 <b>' + username + '</b> 分配角色:</p>' +
    '<div id="modal-roles">' + rolesHtml + '</div>',
    '<button class="btn btn-outline" onclick="closeModal()">取消</button>' +
    '<button class="btn btn-primary" onclick="saveUserRoles(' + userId + ')">保存</button>'
  );
}
window.showUserRoleModal = showUserRoleModal;

async function saveUserRoles(userId) {
  var checks = document.querySelectorAll('#modal-roles input:checked');
  var roleIds = Array.prototype.map.call(checks, function(c) { return parseInt(c.value); });
  try {
    await api.put('/users/' + userId + '/roles', { role_ids: roleIds });
    closeModal();
    toast('角色已更新', 'success');
  } catch (e) { toast(e.message, 'error'); }
}
window.saveUserRoles = saveUserRoles;
