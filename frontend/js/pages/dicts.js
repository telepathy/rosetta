var selectedDictId = null;

async function pageDicts() {
  document.title = 'Rosetta - 字典维护';
  var html = sidebarHtml('/dicts') + pageHeader('字典维护', '管理标准字典、类型映射、参考数据');
  html += '<div class="page-toolbar"><button class="btn btn-primary" onclick="showDictCreateModal()">+ 新建字典</button></div>';
  html += '<div class="split-layout"><div class="split-left"><div class="card"><div class="card-title">字典列表</div><div id="dict-tree">加载中...</div></div></div>';
  html += '<div class="split-right"><div class="card" id="dict-detail"><div class="empty-state"><div class="empty-icon">📖</div><p>请从左侧选择一个字典</p></div></div></div></div>';
  html += '</div></div>';
  setTimeout(loadDictTree, 0);
  return html;
}

async function loadDictTree() {
  try {
    var data = await api.get('/dicts?page=1&page_size=50');
    var dicts = data.data.items;
    var html = '<ul class="tree">';
    dicts.forEach(function(d) {
      html += '<li class="tree-item ' + (d.id === selectedDictId ? 'active' : '') + '" onclick="selectDict(' + d.id + ')" style="display:flex;justify-content:space-between">';
      html += '<span>' + d.dict_name + '</span><span class="badge badge-gray" style="font-size:10px">' + d.dict_type + '</span></li>';
    });
    html += '</ul>';
    document.getElementById('dict-tree').innerHTML = html;
  } catch (e) {
    document.getElementById('dict-tree').innerHTML = '<div class="empty-state"><p>加载失败</p></div>';
  }
}

async function selectDict(id) {
  selectedDictId = id;
  loadDictTree();
  try {
    var dictRes = await api.get('/dicts/' + id);
    var itemsRes = await api.get('/dicts/' + id + '/items');
    var dict = dictRes.data;
    var items = itemsRes.data;

    var h = '<div class="card-title">' + dict.dict_name + ' <span class="badge badge-info">' + dict.dict_type + '</span>';
    h += '<span style="float:right">';
    h += btnSm('编辑', function() { showDictEditModal(id); }) + ' ';
    h += btnSm('删除', function() { deleteDict(id, dict.dict_name); }, 'btn-outline');
    h += '</span></div>';
    h += '<p style="color:var(--text-secondary);font-size:13px;margin-bottom:12px">编码: ' + dict.dict_code + (dict.remark ? ' | ' + dict.remark : '') + '</p>';
    h += '<div style="margin-bottom:8px"><button class="btn btn-sm btn-primary" onclick="showDictItemCreateModal(' + id + ')">+ 添加条目</button></div>';
    h += '<table><thead><tr><th>编码</th><th>名称</th><th>值</th><th>排序</th><th>操作</th></tr></thead><tbody>';

    var list = Array.isArray(items) ? items : (items.items || []);
    if (list.length === 0) {
      h += '<tr><td colspan="5" style="text-align:center;color:var(--text-secondary);padding:20px">暂无条目</td></tr>';
    }
    list.forEach(function(item) {
      h += '<tr><td>' + item.item_code + '</td><td>' + item.item_name + '</td><td>' + (item.item_value || '-') + '</td><td>' + item.ordinal + '</td><td>' +
        btnSm('编辑', function() { showDictItemEditModal(id, item); }) + ' ' +
        btnSm('删除', function() { deleteDictItem(id, item.id); }, 'btn-outline') + '</td></tr>';
    });
    h += '</tbody></table>';
    document.getElementById('dict-detail').innerHTML = h;
  } catch (e) {
    document.getElementById('dict-detail').innerHTML = '<div class="empty-state"><p>加载失败: ' + e.message + '</p></div>';
  }
}
window.selectDict = selectDict;

async function showDictCreateModal() {
  openModal('新建字典',
    '<div class="form-group"><label>字典名称</label><input id="md-name"></div>' +
    '<div class="form-group"><label>字典编码</label><input id="md-code"></div>' +
    '<div class="form-group"><label>类型</label><select id="md-type"><option value="STANDARD">标准字典</option><option value="TYPE_MAPPING">类型映射</option><option value="REFERENCE">参考数据</option></select></div>' +
    '<div class="form-group"><label>备注</label><input id="md-remark"></div>',
    '<button class="btn btn-outline" onclick="closeModal()">取消</button>' +
    '<button class="btn btn-primary" onclick="saveNewDict()">创建</button>'
  );
}
window.showDictCreateModal = showDictCreateModal;

async function saveNewDict() {
  try {
    await api.post('/dicts', {
      dict_name: document.getElementById('md-name').value,
      dict_code: document.getElementById('md-code').value,
      dict_type: document.getElementById('md-type').value,
      remark: document.getElementById('md-remark').value
    });
    closeModal();
    toast('创建成功', 'success');
    loadDictTree();
  } catch (e) { toast(e.message, 'error'); }
}
window.saveNewDict = saveNewDict;

async function showDictEditModal(dictId) {
  var dict;
  try { dict = (await api.get('/dicts/' + dictId)).data; } catch (e) { return; }
  openModal('编辑字典',
    '<div class="form-group"><label>字典名称</label><input id="md-name" value="' + dict.dict_name + '"></div>' +
    '<div class="form-group"><label>字典编码</label><input id="md-code" value="' + dict.dict_code + '" disabled></div>' +
    '<div class="form-group"><label>类型</label><select id="md-type"><option value="STANDARD" ' + (dict.dict_type === 'STANDARD' ? 'selected' : '') + '>标准字典</option><option value="TYPE_MAPPING" ' + (dict.dict_type === 'TYPE_MAPPING' ? 'selected' : '') + '>类型映射</option><option value="REFERENCE" ' + (dict.dict_type === 'REFERENCE' ? 'selected' : '') + '>参考数据</option></select></div>' +
    '<div class="form-group"><label>备注</label><input id="md-remark" value="' + (dict.remark || '') + '"></div>',
    '<button class="btn btn-outline" onclick="closeModal()">取消</button>' +
    '<button class="btn btn-primary" onclick="saveEditDict(' + dictId + ')">保存</button>'
  );
}
window.showDictEditModal = showDictEditModal;

async function saveEditDict(dictId) {
  try {
    await api.put('/dicts/' + dictId, {
      dict_name: document.getElementById('md-name').value,
      dict_type: document.getElementById('md-type').value,
      remark: document.getElementById('md-remark').value
    });
    closeModal();
    toast('保存成功', 'success');
    loadDictTree();
    selectDict(dictId);
  } catch (e) { toast(e.message, 'error'); }
}
window.saveEditDict = saveEditDict;

async function deleteDict(id, name) {
  if (!confirm('确定删除字典 "' + name + '" 吗？所有条目也会被删除。')) return;
  try {
    await api.del('/dicts/' + id);
    selectedDictId = null;
    toast('已删除', 'success');
    loadDictTree();
    document.getElementById('dict-detail').innerHTML = '<div class="empty-state"><div class="empty-icon">📖</div><p>请从左侧选择一个字典</p></div>';
  } catch (e) { toast(e.message, 'error'); }
}
window.deleteDict = deleteDict;

async function showDictItemCreateModal(dictId) {
  openModal('添加条目',
    '<div class="form-group"><label>条目编码</label><input id="di-code"></div>' +
    '<div class="form-group"><label>条目名称</label><input id="di-name"></div>' +
    '<div class="form-group"><label>条目值</label><input id="di-value"></div>' +
    '<div class="form-group"><label>排序</label><input id="di-ordinal" type="number" value="0"></div>',
    '<button class="btn btn-outline" onclick="closeModal()">取消</button>' +
    '<button class="btn btn-primary" onclick="saveNewDictItem(' + dictId + ')">添加</button>'
  );
}
window.showDictItemCreateModal = showDictItemCreateModal;

async function saveNewDictItem(dictId) {
  try {
    await api.post('/dicts/' + dictId + '/items', {
      item_code: document.getElementById('di-code').value,
      item_name: document.getElementById('di-name').value,
      item_value: document.getElementById('di-value').value,
      ordinal: parseInt(document.getElementById('di-ordinal').value) || 0
    });
    closeModal();
    toast('创建成功', 'success');
    selectDict(dictId);
  } catch (e) { toast(e.message, 'error'); }
}
window.saveNewDictItem = saveNewDictItem;

async function showDictItemEditModal(dictId, item) {
  openModal('编辑条目',
    '<div class="form-group"><label>条目编码</label><input id="di-code" value="' + item.item_code + '" disabled></div>' +
    '<div class="form-group"><label>条目名称</label><input id="di-name" value="' + item.item_name + '"></div>' +
    '<div class="form-group"><label>条目值</label><input id="di-value" value="' + (item.item_value || '') + '"></div>' +
    '<div class="form-group"><label>排序</label><input id="di-ordinal" type="number" value="' + item.ordinal + '"></div>',
    '<button class="btn btn-outline" onclick="closeModal()">取消</button>' +
    '<button class="btn btn-primary" onclick="saveEditDictItem(' + dictId + ',' + item.id + ')">保存</button>'
  );
}
window.showDictItemEditModal = showDictItemEditModal;

async function saveEditDictItem(dictId, itemId) {
  try {
    await api.put('/dicts/' + dictId + '/items/' + itemId, {
      item_code: document.getElementById('di-code').value,
      item_name: document.getElementById('di-name').value,
      item_value: document.getElementById('di-value').value,
      ordinal: parseInt(document.getElementById('di-ordinal').value) || 0
    });
    closeModal();
    toast('保存成功', 'success');
    selectDict(dictId);
  } catch (e) { toast(e.message, 'error'); }
}
window.saveEditDictItem = saveEditDictItem;

async function deleteDictItem(dictId, itemId) {
  if (!confirm('确定删除该条目吗？')) return;
  try {
    await api.del('/dicts/' + dictId + '/items/' + itemId);
    toast('已删除', 'success');
    selectDict(dictId);
  } catch (e) { toast(e.message, 'error'); }
}
window.deleteDictItem = deleteDictItem;
