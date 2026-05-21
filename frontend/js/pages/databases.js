async function pageDatabases() {
  document.title = 'Rosetta - 逻辑库管理';
  var html = sidebarHtml('/databases') + pageHeader('逻辑库管理', '管理逻辑数据库、Schema 及物理实例映射');
  html += '<div class="page-toolbar"><button class="btn btn-primary" onclick="showDBCreateModal()">+ 新建逻辑库</button></div>';
  html += '<div id="dbs-container">加载中...</div></div></div>';
  setTimeout(loadDatabases, 0);
  return html;
}

async function loadDatabases() {
  var container = document.getElementById('dbs-container');
  try {
    var dbs = (await api.get('/databases')).data || [];
    if (dbs.length === 0) {
      container.innerHTML = '<div class="empty-state"><p>暂无逻辑库，请先创建</p></div>';
      return;
    }
    var html = '';
    for (var i = 0; i < dbs.length; i++) {
      var db = dbs[i];
      var schemas = (await api.get('/databases/' + db.id + '/schemas')).data || [];
      var instances = (await api.get('/databases/' + db.id + '/instances')).data || [];

      html += '<div class="card" style="margin-bottom:16px">';
      html += '<div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:12px">';
      html += '<div><h3 style="margin:0;font-size:16px">' + escapeHtml(db.name) + '</h3>';
      if (db.description) html += '<div style="font-size:12px;color:var(--text-secondary);margin-top:2px">' + escapeHtml(db.description) + '</div>';
      html += '</div>';
      html += '<div>' + btnSm('+ Schema', function() { showSchemaCreateModal(db.id, db.name); }) + ' ' +
        btnSm('编辑', function() { showDBEditModal(db); }) + ' ' +
        btnSm('删除', function() { deleteDB(db.id, db.name); }, 'btn-outline') + '</div>';
      html += '</div>';

      html += '<div style="margin-bottom:8px;font-size:13px;color:var(--text-secondary)">Schema 列表:</div>';
      html += '<table><thead><tr><th>名称</th><th>描述</th><th>操作</th></tr></thead><tbody>';
      if (schemas.length === 0) {
        html += '<tr><td colspan="3" style="text-align:center;color:var(--text-secondary);padding:20px">暂无 Schema</td></tr>';
      }
      schemas.forEach(function(s) {
        html += '<tr><td><strong>' + escapeHtml(s.name) + '</strong></td><td>' + escapeHtml(s.description || '-') + '</td><td>' +
          btnSm('删除', function() { deleteSchema(db.id, s.id, s.name); }, 'btn-outline') + '</td></tr>';
      });
      html += '</tbody></table>';

      html += '<div style="margin-top:12px;font-size:13px;color:var(--text-secondary)">已映射实例:</div>';
      html += '<div style="display:flex;flex-wrap:wrap;gap:6px;margin-top:6px">';
      if (instances.length === 0) {
        html += '<span style="font-size:12px;color:var(--text-secondary)">暂无映射</span>';
      }
      instances.forEach(function(inst) {
        html += '<span style="display:inline-flex;align-items:center;gap:4px;padding:2px 10px;background:#f1f5f9;border-radius:12px;font-size:12px">';
        html += escapeHtml(inst.name) + ' (' + inst.type + ')';
        html += ' <span style="cursor:pointer;color:var(--danger)" onclick="unmapInstance(' + db.id + ',' + inst.id + ')">✕</span>';
        html += '</span>';
      });
      html += '</div>';
      html += '<div style="margin-top:8px">' + btnSm('+ 映射实例', function() { showMapInstanceModal(db.id, db.name); }, 'btn-outline') + '</div>';

      html += '</div>';
    }
    container.innerHTML = html;
  } catch (e) {
    container.innerHTML = '<div class="empty-state"><p>加载失败: ' + e.message + '</p></div>';
  }
}

function showDBCreateModal() {
  openModal('新建逻辑库',
    '<div class="form-group"><label>名称</label><input id="db-name" placeholder="如 ERP数据库"></div>' +
    '<div class="form-group"><label>描述</label><input id="db-desc" placeholder="描述"></div>',
    '<button class="btn btn-outline" onclick="closeModal()">取消</button>' +
    '<button class="btn btn-primary" onclick="saveNewDB()">创建</button>'
  );
}

async function saveNewDB() {
  var name = document.getElementById('db-name').value;
  if (!name) { toast('请输入名称', 'error'); return; }
  try {
    await api.post('/databases', { name: name, description: document.getElementById('db-desc').value });
    closeModal();
    toast('创建成功', 'success');
    loadDatabases();
  } catch (e) { toast(e.message, 'error'); }
}

function showDBEditModal(db) {
  openModal('编辑逻辑库',
    '<div class="form-group"><label>名称</label><input id="db-name" value="' + escapeHtml(db.name) + '"></div>' +
    '<div class="form-group"><label>描述</label><input id="db-desc" value="' + escapeHtml(db.description || '') + '"></div>',
    '<button class="btn btn-outline" onclick="closeModal()">取消</button>' +
    '<button class="btn btn-primary" onclick="saveEditDB(' + db.id + ')">保存</button>'
  );
}

async function saveEditDB(id) {
  try {
    await api.put('/databases/' + id, { name: document.getElementById('db-name').value, description: document.getElementById('db-desc').value });
    closeModal();
    toast('保存成功', 'success');
    loadDatabases();
  } catch (e) { toast(e.message, 'error'); }
}

async function deleteDB(id, name) {
  if (!confirm('确定删除逻辑库 "' + name + '" 吗？（关联的 Schema 和模型也会被删除）')) return;
  try {
    await api.del('/databases/' + id);
    toast('已删除', 'success');
    loadDatabases();
  } catch (e) { toast(e.message, 'error'); }
}

function showSchemaCreateModal(dbId, dbName) {
  openModal('新建 Schema - ' + escapeHtml(dbName),
    '<div class="form-group"><label>名称</label><input id="s-name" placeholder="如 ods"></div>' +
    '<div class="form-group"><label>描述</label><input id="s-desc" placeholder="描述"></div>',
    '<button class="btn btn-outline" onclick="closeModal()">取消</button>' +
    '<button class="btn btn-primary" onclick="saveNewSchema(' + dbId + ')">创建</button>'
  );
}

async function saveNewSchema(dbId) {
  var name = document.getElementById('s-name').value;
  if (!name) { toast('请输入名称', 'error'); return; }
  try {
    await api.post('/databases/' + dbId + '/schemas', { name: name, description: document.getElementById('s-desc').value });
    closeModal();
    toast('Schema 创建成功', 'success');
    loadDatabases();
  } catch (e) { toast(e.message, 'error'); }
}

async function deleteSchema(dbId, schemaId, name) {
  if (!confirm('确定删除 Schema "' + name + '" 吗？')) return;
  try {
    await api.del('/databases/' + dbId + '/schemas/' + schemaId);
    toast('已删除', 'success');
    loadDatabases();
  } catch (e) { toast(e.message, 'error'); }
}

async function showMapInstanceModal(dbId, dbName) {
  try {
    var data = (await api.get('/instances?page=1&page_size=100')).data;
    var instances = data.items || [];
    if (instances.length === 0) {
      toast('没有可用实例，请先在实例管理中创建', 'error');
      return;
    }
    var opts = instances.map(function(i) { return '<option value="' + i.id + '">' + escapeHtml(i.name) + ' (' + i.type + ')</option>'; }).join('');
    openModal('映射实例 - ' + escapeHtml(dbName),
      '<div class="form-group"><label>选择实例</label><select id="map-instance">' + opts + '</select></div>',
      '<button class="btn btn-outline" onclick="closeModal()">取消</button>' +
      '<button class="btn btn-primary" onclick="doMapInstance(' + dbId + ')">映射</button>'
    );
  } catch(e) { toast(e.message, 'error'); }
}

async function doMapInstance(dbId) {
  var instId = parseInt(document.getElementById('map-instance').value);
  try {
    await api.post('/databases/' + dbId + '/instances', { instance_id: instId });
    closeModal();
    toast('映射成功', 'success');
    loadDatabases();
  } catch(e) { toast(e.message, 'error'); }
}

async function unmapInstance(dbId, instId) {
  if (!confirm('确定解除该实例映射吗？')) return;
  try {
    await api.del('/databases/' + dbId + '/instances/' + instId);
    toast('已解除映射', 'success');
    loadDatabases();
  } catch(e) { toast(e.message, 'error'); }
}

window.loadDatabases = loadDatabases;
window.showDBCreateModal = showDBCreateModal;
window.saveNewDB = saveNewDB;
window.showDBEditModal = showDBEditModal;
window.saveEditDB = saveEditDB;
window.deleteDB = deleteDB;
window.showSchemaCreateModal = showSchemaCreateModal;
window.saveNewSchema = saveNewSchema;
window.deleteSchema = deleteSchema;
window.showMapInstanceModal = showMapInstanceModal;
window.doMapInstance = doMapInstance;
window.unmapInstance = unmapInstance;
