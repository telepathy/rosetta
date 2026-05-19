var modelsPage = 1;

async function pageModels() {
  document.title = 'Rosetta - 模型管理';
  var html = sidebarHtml('/models') + pageHeader('模型管理', '管理逻辑表模型，定义字段、索引、外键');
  html += '<div class="page-toolbar"><button class="btn btn-primary" onclick="showModelCreateModal()">+ 新建模型</button><div class="flex-spacer"></div><div class="search-box"><input id="model-search" placeholder="搜索表名..." onkeydown="if(event.key===\'Enter\')loadModels()"></div></div>';
  html += '<div id="models-table">加载中...</div>';
  html += '<div id="models-pagination"></div>';
  html += '</div></div>';
  setTimeout(loadModels, 0);
  return html;
}

async function loadModels() {
  try {
    var q = document.getElementById('model-search') ? document.getElementById('model-search').value : '';
    var data = await api.get('/models?page=' + modelsPage + '&page_size=15&keyword=' + encodeURIComponent(q));
    var items = data.data.items;
    var html = renderTable(['ID', '表名', '注释', '字段数', '状态'], items.map(function(m) {
      return [m.id, '<a href="#/models/' + m.id + '" style="color:var(--primary);text-decoration:none;font-weight:500">' + m.table_name + '</a>',
        m.table_comment || '-', m.column_count,
        m.table_status === 'DRAFT' ? '<span class="badge badge-warning">草稿</span>' : '<span class="badge badge-success">已发布</span>'];
    }), function(row) {
      var id = row[0];
      return btnSm('编辑', function() { router.navigate('/models/' + id); }) + ' ' +
        btnSm('DDL', function() { showModelDDL(id); }) + ' ' +
        btnSm('删除', function() { deleteModel(id, row[1]); }, 'btn-outline');
    });
    document.getElementById('models-table').innerHTML = html;
    document.getElementById('models-pagination').innerHTML = renderPagination(modelsPage, data.data.total, 15, function(p) { modelsPage = p; loadModels(); });
  } catch (e) {
    document.getElementById('models-table').innerHTML = '<div class="empty-state"><p>加载失败: ' + e.message + '</p></div>';
  }
}

async function showModelCreateModal() {
  openModal('新建模型',
    '<div class="form-group"><label>表名</label><input id="mm-tablename" placeholder="如 user_order (snake_case)"></div>' +
    '<div class="form-group"><label>注释</label><input id="mm-comment" placeholder="表注释"></div>',
    '<button class="btn btn-outline" onclick="closeModal()">取消</button>' +
    '<button class="btn btn-primary" onclick="saveNewModel()">创建</button>'
  );
}
window.showModelCreateModal = showModelCreateModal;

async function saveNewModel() {
  var name = document.getElementById('mm-tablename').value;
  if (!name) { toast('请输入表名', 'error'); return; }
  try {
    await api.post('/models', { table_name: name, table_comment: document.getElementById('mm-comment').value });
    closeModal();
    toast('创建成功', 'success');
    loadModels();
  } catch (e) { toast(e.message, 'error'); }
}
window.saveNewModel = saveNewModel;

async function deleteModel(id, name) {
  if (!confirm('确定删除模型 "' + name + '" 吗？所有字段、索引、外键也会被删除。')) return;
  try { await api.del('/models/' + id); toast('已删除', 'success'); loadModels(); }
  catch (e) { toast(e.message, 'error'); }
}
window.deleteModel = deleteModel;

async function showModelDDL(modelId) {
  try {
    var mysqlRes = await api.get('/models/' + modelId + '/ddl?dialect=MYSQL');
    var gaussRes = await api.get('/models/' + modelId + '/ddl?dialect=GAUSSDB');
    window._ddlData = { mysql: mysqlRes.data.ddl, gaussdb: gaussRes.data.ddl };
    openModal('DDL 预览',
      '<div class="tabs"><button class="tab active" onclick="switchDDLTab(\'mysql\')">MySQL</button><button class="tab" onclick="switchDDLTab(\'gaussdb\')">GaussDB M</button></div>' +
      '<div class="json-view" id="ddl-mysql">' + escapeHtml(window._ddlData.mysql) + '</div>' +
      '<div class="json-view hidden" id="ddl-gaussdb">' + escapeHtml(window._ddlData.gaussdb) + '</div>',
      '<button class="btn btn-outline" onclick="copyDDL()">复制</button>' +
      '<button class="btn btn-outline" onclick="closeModal()">关闭</button>', true
    );
  } catch (e) { toast(e.message, 'error'); }
}
window.showModelDDL = showModelDDL;

function switchDDLTab(dialect) {
  var tabs = document.querySelectorAll('#modal-content .tab');
  tabs.forEach(function(t) { t.classList.remove('active'); });
  if (dialect === 'mysql') { tabs[0].classList.add('active'); } else { tabs[1].classList.add('active'); }
  document.getElementById('ddl-mysql').classList.toggle('hidden', dialect !== 'mysql');
  document.getElementById('ddl-gaussdb').classList.toggle('hidden', dialect !== 'gaussdb');
}
window.switchDDLTab = switchDDLTab;

function copyDDL() {
  var ddl = document.querySelector('#ddl-mysql.hidden') ? window._ddlData.gaussdb : window._ddlData.mysql;
  if (ddl) { navigator.clipboard.writeText(ddl).then(function() { toast('已复制', 'success'); }); }
}
window.copyDDL = copyDDL;

async function pageModelDetail(params) {
  var modelId = parseInt(params.id);
  var detail;
  try { detail = (await api.get('/models/' + modelId)).data; } catch (e) {
    return '<h2>模型不存在</h2>';
  }

  document.title = 'Rosetta - ' + detail.table_name;
  var html = sidebarHtml('/models') + pageHeader('📐 ' + detail.table_name, (detail.table_comment || '无注释') + ' | ' + detail.table_status);
  html += '<div class="page-toolbar"><button class="btn btn-outline" onclick="router.navigate(\'/models\')">← 返回</button><div class="flex-spacer"></div><button class="btn btn-primary" onclick="showAddColumnModal(' + modelId + ')">+ 添加字段</button> <button class="btn btn-outline" onclick="showDeployModal(' + modelId + ')">🚀 部署</button></div>';
  html += '<div class="tabs"><button class="tab active" onclick="switchModelTab(event,\'columns\')">字段 (' + detail.columns.length + ')</button><button class="tab" onclick="switchModelTab(event,\'indexes\')">索引 (' + detail.indexes.length + ')</button><button class="tab" onclick="switchModelTab(event,\'fks\')">外键 (' + detail.foreign_keys.length + ')</button><button class="tab" onclick="switchModelTab(event,\'ddl\')">DDL 预览</button></div>';

  html += renderColumnsTab(detail.columns, modelId);
  html += renderIndexesTab(detail.indexes, modelId);
  html += renderFKsTab(detail.foreign_keys, modelId);

  try {
    var mysqlRes = await api.get('/models/' + modelId + '/ddl?dialect=MYSQL');
    var gaussRes = await api.get('/models/' + modelId + '/ddl?dialect=GAUSSDB');
    html += '<div id="tab-ddl" class="hidden"><div class="tabs"><button class="tab active" onclick="switchDDLTab2(event,\'mysql2\')">MySQL</button><button class="tab" onclick="switchDDLTab2(event,\'gaussdb2\')">GaussDB M</button></div>';
    html += '<div class="json-view" id="ddl2-mysql">' + escapeHtml(mysqlRes.data.ddl) + '</div>';
    html += '<div class="json-view hidden" id="ddl2-gaussdb">' + escapeHtml(gaussRes.data.ddl) + '</div></div>';
  } catch (e) {
    html += '<div id="tab-ddl" class="hidden"><div class="empty-state"><p>DDL 加载失败</p></div></div>';
  }

  html += '</div></div>';
  return html;
}

function switchModelTab(e, tab) {
  var tabs = document.querySelectorAll('.tabs .tab');
  tabs.forEach(function(t) { t.classList.remove('active'); });
  e.target.classList.add('active');
  ['columns', 'indexes', 'fks', 'ddl'].forEach(function(t) {
    var el = document.getElementById('tab-' + t);
    if (el) el.classList.toggle('hidden', t !== tab);
  });
}
window.switchModelTab = switchModelTab;

function switchDDLTab2(e, dialect) {
  var tabs = document.querySelectorAll('#tab-ddl .tab');
  tabs.forEach(function(t) { t.classList.remove('active'); });
  e.target.classList.add('active');
  document.getElementById('ddl2-mysql').classList.toggle('hidden', dialect !== 'mysql2');
  document.getElementById('ddl2-gaussdb').classList.toggle('hidden', dialect !== 'gaussdb2');
}
window.switchDDLTab2 = switchDDLTab2;

function renderColumnsTab(columns, modelId) {
  var h = '<div id="tab-columns"><table class="inline-edit-table"><thead><tr><th>#</th><th>字段名</th><th>类型</th><th>非空</th><th>主键</th><th>注释</th><th>操作</th></tr></thead><tbody>';
  columns.forEach(function(col) {
    var typeStr = col.logical_type + (col.type_length ? '(' + col.type_length + (col.type_scale ? ',' + col.type_scale : '') + ')' : '');
    h += '<tr><td>' + col.ordinal + '</td><td><strong>' + col.column_name + '</strong></td><td>' + typeStr + '</td>';
    h += '<td>' + (col.nullable ? '' : '✓') + '</td><td>' + (col.is_primary_key ? '🔑' : '') + '</td>';
    h += '<td>' + (col.comment || '') + '</td><td>' +
      btnSm('✏️', function() { showEditColumnModal(modelId, col); }) + ' ' +
      btnSm('🗑️', function() { deleteColumn(modelId, col); }, 'btn-outline') + '</td></tr>';
  });
  h += '</tbody></table></div>';
  return h;
}

function renderIndexesTab(indexes, modelId) {
  var h = '<div id="tab-indexes" class="hidden"><table><thead><tr><th>名称</th><th>类型</th><th>列</th><th>操作</th></tr></thead><tbody>';
  indexes.forEach(function(idx) {
    var cols = [];
    try { cols = JSON.parse(idx.columns); } catch(e) {}
    var colStr = cols.map(function(c) { return c.name + (c.order === 'DESC' ? ' DESC' : ''); }).join(', ');
    h += '<tr><td>' + idx.index_name + '</td><td><span class="badge badge-info">' + idx.index_type + '</span></td><td>' + colStr + '</td><td>' +
      btnSm('删除', function() { deleteIndex(modelId, idx.id); }, 'btn-outline') + '</td></tr>';
  });
  h += '</tbody></table><div style="margin-top:12px"><button class="btn btn-sm btn-primary" onclick="showAddIndexModal(' + modelId + ')">+ 添加索引</button></div></div>';
  return h;
}

function renderFKsTab(fks, modelId) {
  var h = '<div id="tab-fks" class="hidden"><table><thead><tr><th>外键名</th><th>列</th><th>引用表</th><th>引用列</th><th>操作</th></tr></thead><tbody>';
  fks.forEach(function(fk) {
    h += '<tr><td>' + fk.fk_name + '</td><td>' + fk.column_name + '</td><td>' + fk.ref_table_name + '</td><td>' + fk.ref_column_name + '</td><td>' +
      btnSm('删除', function() { deleteFK(modelId, fk.id); }, 'btn-outline') + '</td></tr>';
  });
  h += '</tbody></table><div style="margin-top:12px"><button class="btn btn-sm btn-primary" onclick="showAddFKModal(' + modelId + ')">+ 添加外键</button></div></div>';
  return h;
}

function showAddColumnModal(modelId) {
  var types = ['BIGINT','INT','VARCHAR','DECIMAL','FLOAT','DOUBLE','DATE','DATETIME','TIMESTAMP','TEXT','BOOLEAN','JSON'];
  openModal('添加字段',
    '<div class="form-row"><div class="form-group"><label>字段名</label><input id="mc-name" placeholder="snake_case"></div><div class="form-group"><label>序号</label><input id="mc-ordinal" type="number" value="1"></div></div>' +
    '<div class="form-row"><div class="form-group"><label>类型</label><select id="mc-type">' + types.map(function(t) { return '<option value="' + t + '">' + t + '</option>'; }).join('') + '</select></div>' +
    '<div class="form-group"><label>长度</label><input id="mc-length" type="number"></div>' +
    '<div class="form-group"><label>精度</label><input id="mc-scale" type="number"></div></div>' +
    '<div class="form-row"><div class="form-group"><label class="checkbox-group"><input type="checkbox" id="mc-nullable" checked> 可为空</label></div>' +
    '<div class="form-group"><label class="checkbox-group"><input type="checkbox" id="mc-pk"> 主键</label></div></div>' +
    '<div class="form-group"><label>默认值</label><input id="mc-default"></div>' +
    '<div class="form-group"><label>注释</label><input id="mc-comment"></div>',
    '<button class="btn btn-outline" onclick="closeModal()">取消</button>' +
    '<button class="btn btn-primary" onclick="saveAddColumn(' + modelId + ')">添加</button>'
  );
}
window.showAddColumnModal = showAddColumnModal;

async function saveAddColumn(modelId) {
  var col = {
    ordinal: parseInt(document.getElementById('mc-ordinal').value) || 1,
    column_name: document.getElementById('mc-name').value,
    logical_type: document.getElementById('mc-type').value,
    type_length: document.getElementById('mc-length').value ? parseInt(document.getElementById('mc-length').value) : null,
    type_scale: document.getElementById('mc-scale').value ? parseInt(document.getElementById('mc-scale').value) : null,
    nullable: document.getElementById('mc-nullable').checked,
    is_primary_key: document.getElementById('mc-pk').checked,
    default_value: document.getElementById('mc-default').value,
    comment: document.getElementById('mc-comment').value
  };
  if (!col.column_name) { toast('请输入字段名', 'error'); return; }

  try {
    var data = await api.get('/models/' + modelId);
    var existing = data.data.columns || [];
    existing.push(col);
    existing.forEach(function(c, i) { c.ordinal = i + 1; });
    await api.put('/models/' + modelId + '/columns', { columns: existing });
    closeModal();
    toast('字段已添加', 'success');
    router.navigate('/models/' + modelId);
  } catch (e) { toast(e.message, 'error'); }
}
window.saveAddColumn = saveAddColumn;

function showEditColumnModal(modelId, col) {
  var types = ['BIGINT','INT','VARCHAR','DECIMAL','FLOAT','DOUBLE','DATE','DATETIME','TIMESTAMP','TEXT','BOOLEAN','JSON'];
  var typeOpts = types.map(function(t) { return '<option value="' + t + '" ' + (col.logical_type === t ? 'selected' : '') + '>' + t + '</option>'; }).join('');
  openModal('编辑字段 - ' + col.column_name,
    '<div class="form-row"><div class="form-group"><label>字段名</label><input id="mc-name" value="' + col.column_name + '" disabled></div><div class="form-group"><label>序号</label><input id="mc-ordinal" type="number" value="' + col.ordinal + '"></div></div>' +
    '<div class="form-row"><div class="form-group"><label>类型</label><select id="mc-type">' + typeOpts + '</select></div>' +
    '<div class="form-group"><label>长度</label><input id="mc-length" type="number" value="' + (col.type_length || '') + '"></div>' +
    '<div class="form-group"><label>精度</label><input id="mc-scale" type="number" value="' + (col.type_scale || '') + '"></div></div>' +
    '<div class="form-row"><div class="form-group"><label class="checkbox-group"><input type="checkbox" id="mc-nullable" ' + (col.nullable ? 'checked' : '') + '> 可为空</label></div>' +
    '<div class="form-group"><label class="checkbox-group"><input type="checkbox" id="mc-pk" ' + (col.is_primary_key ? 'checked' : '') + '> 主键</label></div></div>' +
    '<div class="form-group"><label>默认值</label><input id="mc-default" value="' + (col.default_value || '') + '"></div>' +
    '<div class="form-group"><label>注释</label><input id="mc-comment" value="' + (col.comment || '') + '"></div>',
    '<button class="btn btn-outline" onclick="closeModal()">取消</button>' +
    '<button class="btn btn-primary" onclick="saveEditColumn(' + modelId + ',' + col.id + ')">保存</button>'
  );
}
window.showEditColumnModal = showEditColumnModal;

async function saveEditColumn(modelId, colId) {
  try {
    var data = await api.get('/models/' + modelId);
    var cols = data.data.columns || [];
    var col = cols.find(function(c) { return c.id === colId; });
    if (!col) return;
    col.ordinal = parseInt(document.getElementById('mc-ordinal').value) || col.ordinal;
    col.logical_type = document.getElementById('mc-type').value;
    col.type_length = document.getElementById('mc-length').value ? parseInt(document.getElementById('mc-length').value) : null;
    col.type_scale = document.getElementById('mc-scale').value ? parseInt(document.getElementById('mc-scale').value) : null;
    col.nullable = document.getElementById('mc-nullable').checked;
    col.is_primary_key = document.getElementById('mc-pk').checked;
    col.default_value = document.getElementById('mc-default').value;
    col.comment = document.getElementById('mc-comment').value;
    await api.put('/models/' + modelId + '/columns', { columns: cols });
    closeModal();
    toast('字段已更新', 'success');
    router.navigate('/models/' + modelId);
  } catch (e) { toast(e.message, 'error'); }
}
window.saveEditColumn = saveEditColumn;

async function deleteColumn(modelId, col) {
  if (!confirm('确定删除字段 "' + col.column_name + '" 吗？')) return;
  try {
    var data = await api.get('/models/' + modelId);
    var cols = (data.data.columns || []).filter(function(c) { return c.id !== col.id; });
    cols.forEach(function(c, i) { c.ordinal = i + 1; });
    await api.put('/models/' + modelId + '/columns', { columns: cols });
    toast('字段已删除', 'success');
    router.navigate('/models/' + modelId);
  } catch (e) { toast(e.message, 'error'); }
}
window.deleteColumn = deleteColumn;

function showAddIndexModal(modelId) {
  openModal('添加索引',
    '<div class="form-group"><label>索引名称</label><input id="mi-name"></div>' +
    '<div class="form-group"><label>类型</label><select id="mi-type"><option value="NORMAL">普通</option><option value="UNIQUE">唯一</option></select></div>' +
    '<div class="form-group"><label>索引列 (JSON)</label><input id="mi-cols" placeholder=\'[{"name":"col","order":"ASC"}]\'></div>',
    '<button class="btn btn-outline" onclick="closeModal()">取消</button>' +
    '<button class="btn btn-primary" onclick="saveAddIndex(' + modelId + ')">添加</button>'
  );
}
window.showAddIndexModal = showAddIndexModal;

async function saveAddIndex(modelId) {
  try {
    var cols = JSON.parse(document.getElementById('mi-cols').value);
    await api.post('/models/' + modelId + '/indexes', {
      index_name: document.getElementById('mi-name').value,
      index_type: document.getElementById('mi-type').value,
      columns: cols
    });
    closeModal();
    toast('索引已添加', 'success');
    router.navigate('/models/' + modelId);
  } catch (e) { toast(e.message, 'error'); }
}
window.saveAddIndex = saveAddIndex;

async function deleteIndex(modelId, indexId) {
  if (!confirm('确定删除该索引吗？')) return;
  try {
    await api.del('/models/' + modelId + '/indexes/' + indexId);
    toast('索引已删除', 'success');
    router.navigate('/models/' + modelId);
  } catch (e) { toast(e.message, 'error'); }
}
window.deleteIndex = deleteIndex;

function showAddFKModal(modelId) {
  openModal('添加外键',
    '<div class="form-group"><label>外键名称</label><input id="mfk-name"></div>' +
    '<div class="form-group"><label>外键列</label><input id="mfk-col" placeholder="如 user_id"></div>' +
    '<div class="form-row"><div class="form-group"><label>引用模型ID</label><input id="mfk-refid" type="number"></div><div class="form-group"><label>引用列</label><input id="mfk-refcol" placeholder="如 id"></div></div>',
    '<button class="btn btn-outline" onclick="closeModal()">取消</button>' +
    '<button class="btn btn-primary" onclick="saveAddFK(' + modelId + ')">添加</button>'
  );
}
window.showAddFKModal = showAddFKModal;

async function saveAddFK(modelId) {
  try {
    await api.post('/models/' + modelId + '/foreign-keys', {
      fk_name: document.getElementById('mfk-name').value,
      column_name: document.getElementById('mfk-col').value,
      ref_model_id: parseInt(document.getElementById('mfk-refid').value),
      ref_column_name: document.getElementById('mfk-refcol').value
    });
    closeModal();
    toast('外键已添加', 'success');
    router.navigate('/models/' + modelId);
  } catch (e) { toast(e.message, 'error'); }
}
window.saveAddFK = saveAddFK;

async function deleteFK(modelId, fkId) {
  if (!confirm('确定删除该外键吗？')) return;
  try {
    await api.del('/models/' + modelId + '/foreign-keys/' + fkId);
    toast('外键已删除', 'success');
    router.navigate('/models/' + modelId);
  } catch (e) { toast(e.message, 'error'); }
}

async function showDeployModal(modelId) {
  var schemas = [];
  try {
    var instances = (await api.get('/instances?page=1&page_size=50')).data.items || [];
    for (var i = 0; i < instances.length; i++) {
      var s = (await api.get('/instances/' + instances[i].id + '/schemas')).data;
      if (Array.isArray(s)) {
        s.forEach(function(sc) { schemas.push({ id: sc.id, name: instances[i].name + ' / ' + sc.schema_name, instanceId: instances[i].id }); });
      }
    }
  } catch(e) {}

  if (schemas.length === 0) {
    toast('没有可用的 Schema，请先在实例管理中创建', 'error');
    return;
  }

  var opts = schemas.map(function(s) { return '<option value="' + s.id + '" data-inst="' + s.instanceId + '">' + s.name + '</option>'; }).join('');

  openModal('部署模型',
    '<div class="form-group"><label>目标 Schema</label><select id="deploy-schema">' + opts + '</select></div>' +
    '<div class="form-group"><label>方言</label><select id="deploy-dialect"><option value="MYSQL">MySQL</option><option value="GAUSSDB">GaussDB M</option></select></div>',
    '<button class="btn btn-outline" onclick="closeModal()">取消</button>' +
    '<button class="btn btn-primary" onclick="doDeploy(' + modelId + ')">部署</button>'
  );
}
window.showDeployModal = showDeployModal;

async function doDeploy(modelId) {
  var schemaId = parseInt(document.getElementById('deploy-schema').value);
  var dialect = document.getElementById('deploy-dialect').value;
  try {
    await api.post('/models/' + modelId + '/deploy', { schema_id: schemaId, dialect: dialect });
    closeModal();
    toast('部署成功', 'success');
    router.navigate('/models/' + modelId);
  } catch(e) { toast(e.message, 'error'); }
}
window.doDeploy = doDeploy;
window.deleteFK = deleteFK;

window.loadModels = loadModels;
