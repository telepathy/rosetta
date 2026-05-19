var instancesPage = 1;

async function pageInstances() {
  document.title = 'Rosetta - 实例管理';
  var html = sidebarHtml('/instances') + pageHeader('实例管理', '管理数据源连接实例与 Schema');
  html += '<div class="page-toolbar"><button class="btn btn-primary" onclick="showInstanceCreateModal()">+ 注册实例</button></div>';
  html += '<div id="instances-table">加载中...</div>';
  html += '<div id="instances-pagination"></div>';
  html += '</div></div>';
  setTimeout(loadInstances, 0);
  return html;
}

async function loadInstances() {
  try {
    var data = await api.get('/instances?page=' + instancesPage + '&page_size=15');
    var items = data.data.items;
    var html = renderTable(['ID', '名称', '类型', '主机', '端口', '状态'], items.map(function(i) {
      return [i.id, i.name, '<span class="badge badge-info">' + i.type + '</span>', i.host, i.port,
        i.status === 'ACTIVE' ? '<span class="badge badge-success">正常</span>' : '<span class="badge badge-danger">停用</span>'];
    }), function(row) {
      var id = row[0];
      var name = row[1];
      return btnSm('Schema', function() { showSchemas(id, name); }) + ' ' +
        btnSm('编辑', function() { showInstanceEditModal(id); }) + ' ' +
        btnSm('删除', function() { deleteInstance(id, name); }, 'btn-outline');
    });
    document.getElementById('instances-table').innerHTML = html;
    document.getElementById('instances-pagination').innerHTML = renderPagination(instancesPage, data.data.total, 15, function(p) { instancesPage = p; loadInstances(); });
  } catch (e) {
    document.getElementById('instances-table').innerHTML = '<div class="empty-state"><p>加载失败: ' + e.message + '</p></div>';
  }
}

async function showInstanceCreateModal() {
  openModal('注册实例',
    '<div class="form-row"><div class="form-group"><label>名称</label><input id="mi-name"></div><div class="form-group"><label>类型</label><select id="mi-type"><option value="MYSQL">MySQL</option><option value="GAUSSDB_M">GaussDB M</option></select></div></div>' +
    '<div class="form-row"><div class="form-group"><label>主机</label><input id="mi-host"></div><div class="form-group"><label>端口</label><input id="mi-port" type="number" value="3306"></div></div>' +
    '<div class="form-row"><div class="form-group"><label>用户名</label><input id="mi-user"></div><div class="form-group"><label>密码</label><input id="mi-password" type="password"></div></div>' +
    '<div class="form-group"><label>数据库</label><input id="mi-database"></div>',
    '<button class="btn btn-outline" onclick="closeModal()">取消</button>' +
    '<button class="btn btn-primary" onclick="saveNewInstance()">保存</button>'
  );
}
window.showInstanceCreateModal = showInstanceCreateModal;

async function saveNewInstance() {
  try {
    await api.post('/instances', {
      name: document.getElementById('mi-name').value,
      type: document.getElementById('mi-type').value,
      host: document.getElementById('mi-host').value,
      port: parseInt(document.getElementById('mi-port').value) || 3306,
      user: document.getElementById('mi-user').value,
      password: document.getElementById('mi-password').value,
      database: document.getElementById('mi-database').value
    });
    closeModal();
    toast('创建成功', 'success');
    loadInstances();
  } catch (e) { toast(e.message, 'error'); }
}
window.saveNewInstance = saveNewInstance;

async function showInstanceEditModal(instId) {
  var inst;
  try { inst = (await api.get('/instances/' + instId)).data; } catch (e) { return; }

  openModal('编辑实例',
    '<div class="form-row"><div class="form-group"><label>名称</label><input id="mi-name" value="' + inst.name + '"></div><div class="form-group"><label>类型</label><select id="mi-type"><option value="MYSQL" ' + (inst.type === 'MYSQL' ? 'selected' : '') + '>MySQL</option><option value="GAUSSDB_M" ' + (inst.type === 'GAUSSDB_M' ? 'selected' : '') + '>GaussDB M</option></select></div></div>' +
    '<div class="form-row"><div class="form-group"><label>主机</label><input id="mi-host" value="' + inst.host + '"></div><div class="form-group"><label>端口</label><input id="mi-port" type="number" value="' + inst.port + '"></div></div>' +
    '<div class="form-row"><div class="form-group"><label>用户名</label><input id="mi-user" placeholder="留空则不修改"></div><div class="form-group"><label>密码</label><input id="mi-password" type="password" placeholder="留空则不修改"></div></div>' +
    '<div class="form-group"><label>数据库</label><input id="mi-database" placeholder="留空则不修改"></div>',
    '<button class="btn btn-outline" onclick="closeModal()">取消</button>' +
    '<button class="btn btn-primary" onclick="saveEditInstance(' + instId + ')">保存</button>'
  );
}
window.showInstanceEditModal = showInstanceEditModal;

async function saveEditInstance(instId) {
  try {
    await api.put('/instances/' + instId, {
      name: document.getElementById('mi-name').value,
      type: document.getElementById('mi-type').value,
      host: document.getElementById('mi-host').value,
      port: parseInt(document.getElementById('mi-port').value) || 3306,
      user: document.getElementById('mi-user').value,
      password: document.getElementById('mi-password').value,
      database: document.getElementById('mi-database').value
    });
    closeModal();
    toast('保存成功', 'success');
    loadInstances();
  } catch (e) { toast(e.message, 'error'); }
}
window.saveEditInstance = saveEditInstance;

async function deleteInstance(id, name) {
  if (!confirm('确定删除实例 "' + name + '" 吗？关联的 Schema 也会被删除。')) return;
  try {
    await api.del('/instances/' + id);
    toast('已删除', 'success');
    loadInstances();
  } catch (e) { toast(e.message, 'error'); }
}
window.deleteInstance = deleteInstance;

async function showSchemas(instId, instName) {
  var schemas = [];
  try { schemas = (await api.get('/instances/' + instId + '/schemas')).data; } catch (e) {}

  var body = '<p style="margin-bottom:12px">实例 <b>' + instName + '</b> 的 Schema 列表:</p>';
  body += '<table><thead><tr><th>Schema</th><th>层级</th></tr></thead><tbody>';
  if (schemas.length === 0) {
    body += '<tr><td colspan="2" style="text-align:center;padding:20px;color:var(--text-secondary)">暂无 Schema</td></tr>';
  }
  schemas.forEach(function(s) {
    body += '<tr><td>' + s.schema_name + '</td><td><span class="badge badge-info">' + s.layer + '</span></td></tr>';
  });
  body += '</tbody></table>';
  body += '<div style="margin-top:16px;padding-top:12px;border-top:1px solid var(--border)"><b>新建 Schema</b></div>';
  body += '<div class="form-row"><div class="form-group"><label>Schema名</label><input id="ms-name" placeholder="如 ods"></div><div class="form-group"><label>层级</label><select id="ms-layer"><option>ODS</option><option>DWD</option><option>DWS</option><option>ADS</option></select></div></div>';

  openModal('Schema 管理 - ' + instName, body,
    '<button class="btn btn-outline" onclick="closeModal()">关闭</button>' +
    '<button class="btn btn-primary" onclick="createSchema(' + instId + ')">+ 创建 Schema</button>', true
  );
}
window.showSchemas = showSchemas;

async function createSchema(instId) {
  var name = document.getElementById('ms-name').value;
  var layer = document.getElementById('ms-layer').value;
  if (!name) { toast('请输入Schema名称', 'error'); return; }
  try {
    await api.post('/instances/' + instId + '/schemas', { schema_name: name, layer: layer });
    closeModal();
    toast('Schema 创建成功', 'success');
    loadInstances();
  } catch (e) { toast(e.message, 'error'); }
}
window.createSchema = createSchema;
