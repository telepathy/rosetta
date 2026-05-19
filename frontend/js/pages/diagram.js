var diagramState = { mode: 'view', schemaId: null, fkSource: null, cy: null };

async function pageDiagram() {
  document.title = 'Rosetta - ER 图';
  var html = sidebarHtml('/diagram');
  html += '<div class="page-header"><h2>📊 数据库模型图</h2><div class="page-desc">自动布局 ｜ 滚轮缩放 ｜ 拖拽平移 ｜ 点击表节点进入详情</div></div>';
  html += '<div class="page-toolbar">';
  html += '<select id="diagram-schema" onchange="loadDiagram()"><option value="">选择 Schema...</option></select>';
  html += '<div class="flex-spacer"></div>';
  html += '<button id="btn-mode-view" class="btn btn-sm btn-primary" onclick="setMode(\'view\')">👁 浏览</button>';
  html += '<button id="btn-mode-edit" class="btn btn-sm btn-outline" onclick="setMode(\'edit\')">✏️ 编辑</button>';
  html += '<button id="btn-mode-fk" class="btn btn-sm btn-outline" onclick="setMode(\'fk\')">🔗 建外键</button>';
  html += '<button id="btn-add-table" class="btn btn-sm btn-primary hidden" onclick="addTableOnDiagram()">+ 新表</button>';
  html += '<button class="btn btn-sm btn-outline" onclick="fitDiagram()">⊞ 适应</button>';
  html += '<button class="btn btn-sm btn-outline" onclick="if(diagramState.cy){diagramState.cy.zoom(1);diagramState.cy.center();}">1:1</button>';
  html += '</div>';
  html += '<div id="diagram-viewport" style="width:100%;height:calc(100vh - 200px);border:1px solid var(--border);border-radius:var(--radius);background:#fafbfc;position:relative"></div>';
  html += '</div></div>';
  document.getElementById('diagram-viewport').innerHTML = '<div class="empty-state"><div class="empty-icon">📊</div><p>请先选择一个 Schema 查看 ER 图</p></div>';
  setTimeout(loadSchemaOptions, 0);
  return html;
}

function setMode(mode) {
  diagramState.mode = mode; diagramState.fkSource = null;
  ['view','edit','fk'].forEach(function(m) {
    var btn = document.getElementById('btn-mode-' + m);
    if (btn) btn.className = 'btn btn-sm ' + (m === mode ? 'btn-primary' : 'btn-outline');
  });
  document.getElementById('btn-add-table').classList.toggle('hidden', mode === 'view');
  if (diagramState.schemaId) loadDiagram();
}

async function loadSchemaOptions() {
  try {
    var schemas = [];
    var instances = (await api.get('/instances?page=1&page_size=50')).data.items || [];
    for (var i = 0; i < instances.length; i++) {
      try {
        var s = (await api.get('/instances/' + instances[i].id + '/schemas')).data;
        if (Array.isArray(s)) s.forEach(function(sc) { schemas.push({ id: sc.id, name: instances[i].name + ' / ' + sc.schema_name }); });
      } catch(e) {}
    }
    var sel = document.getElementById('diagram-schema');
    sel.innerHTML = '<option value="">选择 Schema...</option>';
    schemas.forEach(function(s) { sel.innerHTML += '<option value="' + s.id + '">' + s.name + '</option>'; });
    if (schemas.length === 1) { sel.value = schemas[0].id; loadDiagram(); }
  } catch(e) {}
}

async function loadDiagram() {
  var schemaId = document.getElementById('diagram-schema').value;
  if (!schemaId) return;
  diagramState.schemaId = schemaId;
  if (diagramState.cy) { diagramState.cy.destroy(); diagramState.cy = null; }

  var vp = document.getElementById('diagram-viewport');
  vp.innerHTML = '<div style="text-align:center;padding:60px">加载中...</div>';

  try {
    var er = (await api.get('/schemas/' + schemaId + '/er-diagram')).data;
    if (!er || !er.tables || er.tables.length === 0) {
      vp.innerHTML = '<div class="empty-state"><p>该 Schema 下没有部署的模型</p></div>'; return;
    }
    var tables = er.tables, edges = er.edges || [], modelDetails = {};
    var batchSize = 8;
    for (var i = 0; i < tables.length; i += batchSize) {
      var batch = tables.slice(i, i + batchSize);
      var results = await Promise.all(batch.map(function(t) { return api.get('/models/' + t.id).catch(function() { return null; }); }));
      results.forEach(function(r, j) { if (r && r.data) modelDetails[batch[j].id] = r.data; });
    }
    vp.innerHTML = '';
    renderGraph(vp, tables, edges, modelDetails);
  } catch(e) {
    vp.innerHTML = '<div class="empty-state"><p>加载失败: ' + e.message + '</p></div>';
  }
}

function buildLabel(t, detail) {
  var cols = detail ? (detail.columns || []) : [];
  var lines = [t.table_name];
  lines.push('─'.repeat(Math.max(22, t.table_name.length + 6)));
  for (var i = 0; i < Math.min(cols.length, 15); i++) {
    var c = cols[i], pfx = c.is_primary_key ? '🔑' : '  ';
    var ts = c.logical_type + (c.type_length ? '(' + c.type_length + (c.type_scale ? ',' + c.type_scale : '') + ')' : '');
    var line = pfx + ' ' + c.column_name;
    while (line.length < 24) line += ' ';
    line += ' ' + ts;
    lines.push(line);
  }
  if (cols.length > 15) lines.push('  ... +' + (cols.length - 15) + ' more');
  return lines.join('\n');
}

function renderGraph(vp, tables, edges, modelDetails) {
  if (!window.cytoscape) { vp.innerHTML = '<div class="empty-state"><p>可视化库未加载，请刷新页面</p></div>'; return; }

  var elements = [];
  tables.forEach(function(t) {
    elements.push({ data: { id: 't' + t.id, label: buildLabel(t, modelDetails[t.id]), tableId: t.id, tableName: t.table_name }, classes: 'table-node' });
  });
  edges.forEach(function(e) {
    elements.push({ data: { id: 'e_' + e.source + '_' + e.target, source: 't' + e.source, target: 't' + e.target, label: e.source_col.substring(0, 10) + ' → ' + e.target_col.substring(0, 10) } });
  });

  var cy = window.cytoscape({
    container: vp, elements: elements,
    style: [
      { selector: '.table-node', style: {
        'shape': 'round-rectangle', 'width': 280, 'height': 'label',
        'background-color': '#ffffff', 'border-color': '#cbd5e1', 'border-width': 2,
        'padding': '12', 'text-valign': 'top', 'text-halign': 'left',
        'text-wrap': 'none', 'font-size': '11', 'font-family': 'monospace',
        'color': '#334155', 'text-max-width': '260', 'text-margin-y': 4
      }},
      { selector: 'edge', style: {
        'width': 1.5, 'line-color': '#94a3b8', 'target-arrow-color': '#94a3b8',
        'target-arrow-shape': 'triangle', 'curve-style': 'bezier',
        'text-rotation': 'autorotate', 'font-size': '9', 'color': '#64748b',
        'text-background-color': '#ffffff', 'text-background-opacity': 0.9,
        'text-background-padding': '2', 'text-background-shape': 'round-rectangle'
      }}
    ],
    layout: { name: 'dagre', rankDir: 'LR', spacingFactor: 1.3, nodeSep: 50, edgeSep: 25, rankSep: 100 },
    wheelSensitivity: 0.3, minZoom: 0.05, maxZoom: 3
  });

  cy.on('tap', 'node', function(ev) {
    var node = ev.target;
    if (diagramState.mode === 'fk') {
      if (!diagramState.fkSource) {
        diagramState.fkSource = { modelId: node.data('tableId'), nodeId: node.id(), name: node.data('tableName') };
        toast('已选择 ' + diagramState.fkSource.name + '，请点击目标表', 'info');
        node.style('border-color', '#f59e0b'); node.style('border-width', 3);
      } else if (node.data('tableId') !== diagramState.fkSource.modelId) {
        completeFK(node.data('tableId'));
      }
    } else if (diagramState.mode === 'view') {
      router.navigate('/models/' + node.data('tableId'));
    }
  });

  diagramState.cy = cy;
  setTimeout(function() { if (cy) cy.fit(undefined, 50); }, 500);
}

function completeFK(targetId) {
  var src = diagramState.fkSource; diagramState.fkSource = null;
  if (diagramState.cy) { diagramState.cy.nodes().style('border-color', '#cbd5e1'); diagramState.cy.nodes().style('border-width', 2); }
  var input = prompt('外键关系: 源列名,目标列名\n示例: user_id,id');
  if (!input) return;
  var parts = input.split(',').map(function(s) { return s.trim(); });
  if (parts.length < 2) { toast('请输入源列名和目标列名', 'error'); return; }
  api.post('/models/' + src.modelId + '/foreign-keys', { fk_name: 'fk_' + parts[0], column_name: parts[0], ref_model_id: targetId, ref_column_name: parts[1] }).then(function() { toast('外键已创建', 'success'); loadDiagram(); }).catch(function(e) { toast(e.message, 'error'); });
}

function fitDiagram() { if (diagramState.cy) diagramState.cy.fit(undefined, 50); }

function addColumnOnDiagram(modelId) {
  var input = prompt('添加字段\n格式: 字段名,类型\n类型: ' + ['BIGINT','INT','VARCHAR','DECIMAL','FLOAT','DOUBLE','DATE','DATETIME','TIMESTAMP','TEXT','BOOLEAN','JSON'].join(','));
  if (!input) return;
  var parts = input.split(',').map(function(s) { return s.trim(); });
  if (parts.length < 2) { toast('至少需要字段名和类型', 'error'); return; }
  api.get('/models/' + modelId).then(function(r) {
    var cols = r.data.columns || [];
    cols.push({ ordinal: cols.length + 1, column_name: parts[0], logical_type: parts[1].toUpperCase(), nullable: true, is_primary_key: false, default_value: '', comment: parts[2] || '', type_length: null, type_scale: null });
    api.put('/models/' + modelId + '/columns', { columns: cols }).then(function() { toast('字段已添加', 'success'); loadDiagram(); }).catch(function(e) { toast(e.message, 'error'); });
  });
}

function addTableOnDiagram() {
  var input = prompt('新建模型\n格式: 表名,备注');
  if (!input) return;
  var parts = input.split(',').map(function(s) { return s.trim(); });
  if (!parts[0]) { toast('请输入表名', 'error'); return; }
  api.post('/models', { table_name: parts[0], table_comment: parts[1] || '' }).then(function(r) {
    api.post('/models/' + r.data.id + '/deploy', { schema_id: parseInt(diagramState.schemaId), dialect: 'MYSQL' }).then(function() { toast('模型已创建并部署', 'success'); loadDiagram(); }).catch(function() { toast('模型已创建', 'success'); loadDiagram(); });
  }).catch(function(e) { toast(e.message, 'error'); });
}

window.loadDiagram = loadDiagram;
window.setMode = setMode;
window.fitDiagram = fitDiagram;
window.addColumnOnDiagram = addColumnOnDiagram;
window.addTableOnDiagram = addTableOnDiagram;
