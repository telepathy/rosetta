var diagramState = { mode: 'view', schemaId: null, fkSource: null };

async function pageDiagram() {
  document.title = 'Rosetta - ER 图';
  var html = sidebarHtml('/diagram');
  html += '<div class="page-header"><h2>📊 数据库模型图</h2><div class="page-desc">自动分层布局 ｜ 滚轮缩放 ｜ 拖拽平移 ｜ 点击表进入详情</div></div>';
  html += '<div class="page-toolbar">';
  html += '<select id="diagram-schema" onchange="loadDiagram()"><option value="">选择 Schema...</option></select>';
  html += '<div class="flex-spacer"></div>';
  html += '<button id="btn-mode-view" class="btn btn-sm btn-primary" onclick="setMode(\'view\')">👁 浏览</button>';
  html += '<button id="btn-mode-edit" class="btn btn-sm btn-outline" onclick="setMode(\'edit\')">✏️ 编辑</button>';
  html += '<button id="btn-mode-fk" class="btn btn-sm btn-outline" onclick="setMode(\'fk\')">🔗 建外键</button>';
  html += '<button id="btn-add-table" class="btn btn-sm btn-primary hidden" onclick="addTableOnDiagram()">+ 新表</button>';
  html += '<button class="btn btn-sm btn-outline" onclick="fitDiagram()">⊞ 适应</button>';
  html += '<button class="btn btn-sm btn-outline" onclick="diagramState.scale = 1; applyZoom();">1:1</button>';
  html += '</div>';
  html += '<div id="diagram-viewport" style="width:100%;height:calc(100vh - 200px);overflow:auto;border:1px solid var(--border);border-radius:var(--radius);background:#fafbfc;position:relative">';
  html += '<div class="empty-state"><div class="empty-icon">📊</div><p>请先选择一个 Schema 查看 ER 图</p></div>';
  html += '</div>';
  html += '<div style="position:fixed;bottom:20px;right:20px;z-index:100;background:var(--card-bg);border:1px solid var(--border);border-radius:8px;padding:4px;box-shadow:var(--shadow-lg)" id="zoom-controls">';
  html += '<button class="btn btn-sm btn-outline" onclick="diagramState.scale = Math.max(0.2, (diagramState.scale||1) - 0.15); applyZoom();" style="font-size:16px">−</button>';
  html += '<span id="zoom-level" style="margin:0 6px;font-size:12px">100%</span>';
  html += '<button class="btn btn-sm btn-outline" onclick="diagramState.scale = Math.min(3, (diagramState.scale||1) + 0.15); applyZoom();" style="font-size:16px">+</button>';
  html += '</div>';
  html += '</div></div>';
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
    if (schemas.length === 0) {
      document.getElementById('diagram-viewport').innerHTML = '<div class="empty-state"><p>没有可用的 Schema</p></div>';
    } else if (schemas.length === 1) { sel.value = schemas[0].id; loadDiagram(); }
  } catch(e) {}
}

async function loadDiagram() {
  var schemaId = document.getElementById('diagram-schema').value;
  if (!schemaId) return;
  diagramState.schemaId = schemaId;
  diagramState.scale = 1;

  var vp = document.getElementById('diagram-viewport');
  vp.innerHTML = '<div style="text-align:center;padding:60px">加载中...</div>';

  try {
    var er = (await api.get('/schemas/' + schemaId + '/er-diagram')).data;
    if (!er || !er.tables || er.tables.length === 0) {
      vp.innerHTML = '<div class="empty-state"><p>该 Schema 下没有部署的模型</p></div>';
      return;
    }
    var tables = er.tables, edges = er.edges || [], modelDetails = {};
    var batchSize = 8;
    for (var i = 0; i < tables.length; i += batchSize) {
      var batch = tables.slice(i, i + batchSize);
      var results = await Promise.all(batch.map(function(t) { return api.get('/models/' + t.id).catch(function() { return null; }); }));
      results.forEach(function(r, j) { if (r && r.data) modelDetails[batch[j].id] = r.data; });
    }
    vp.innerHTML = '';
    renderDiagram(vp, tables, edges, modelDetails);
  } catch(e) {
    vp.innerHTML = '<div class="empty-state"><p>加载失败: ' + e.message + '</p></div>';
  }
}

function computeLayout(tables, edges) {
  var adjIn = {}, adjOut = {};
  tables.forEach(function(t) { adjIn[t.id] = []; adjOut[t.id] = []; });
  edges.forEach(function(e) { adjOut[e.source].push(e.target); adjIn[e.target].push(e.source); });

  var layers = {}, visited = {};
  function assignLayer(id, depth) {
    if (visited[id] && layers[id] >= depth) return;
    visited[id] = true; layers[id] = Math.max(layers[id] || 0, depth);
    (adjOut[id] || []).forEach(function(t) { assignLayer(t, depth + 1); });
  }

  var roots = [];
  tables.forEach(function(t) { if ((adjIn[t.id] || []).length === 0) roots.push(t.id); });
  if (roots.length === 0) roots = [tables[0].id];
  roots.forEach(function(r) { assignLayer(r, 0); });
  tables.forEach(function(t) { if (!visited[t.id]) { assignLayer(t.id, 0); } });

  var maxLayer = 0, layerCounts = {};
  Object.keys(layers).forEach(function(id) { maxLayer = Math.max(maxLayer, layers[id]); layerCounts[layers[id]] = (layerCounts[layers[id]] || 0) + 1; });

  var colOffsets = {}, colPositions = {};
  var cardW = 290, cardSpacing = 30, colSpacing = 80;

  Object.keys(layerCounts).sort(function(a,b) { return a - b; }).forEach(function(layer) {
    colOffsets[layer] = 0;
    Object.keys(layerCounts).filter(function(l) { return parseInt(l) < parseInt(layer); }).forEach(function(l) { colOffsets[layer] += (layerCounts[l] || 0) * (cardW + cardSpacing) + colSpacing; });
  });

  var layerRows = {};
  tables.forEach(function(t) { var l = layers[t.id] || 0; if (!layerRows[l]) layerRows[l] = []; layerRows[l].push(t.id); });

  var positions = {}, maxH = 0;
  Object.keys(layerRows).sort(function(a,b) { return a - b; }).forEach(function(layer) {
    var ids = layerRows[layer], topY = 40;
    ids.forEach(function(id, i) { positions[id] = { x: colOffsets[layer], y: topY + i * 220 }; maxH = Math.max(maxH, positions[id].y + 210); });
  });

  return { positions: positions, totalW: colOffsets[maxLayer] + cardW + 40, totalH: maxH + 40 };
}

function renderDiagram(vp, tables, edges, modelDetails) {
  var layout = computeLayout(tables, edges);
  var positions = layout.positions;

  var canvas = document.createElement('div');
  canvas.id = 'diagram-canvas';
  canvas.style.position = 'relative';
  canvas.style.width = layout.totalW + 'px';
  canvas.style.height = layout.totalH + 'px';
  canvas.style.transformOrigin = '0 0';
  canvas.style.transform = 'scale(' + (diagramState.scale || 1) + ')';

  var svgNS = 'http://www.w3.org/2000/svg';
  var svg = document.createElementNS(svgNS, 'svg');
  svg.setAttribute('width', layout.totalW); svg.setAttribute('height', layout.totalH);
  svg.style.position = 'absolute'; svg.style.top = '0'; svg.style.left = '0';
  svg.style.pointerEvents = 'none'; svg.style.zIndex = '1';

  var defs = document.createElementNS(svgNS, 'defs');
  var marker = document.createElementNS(svgNS, 'marker');
  marker.setAttribute('id', 'arrow'); marker.setAttribute('markerWidth', '6'); marker.setAttribute('markerHeight', '5');
  marker.setAttribute('refX', '5'); marker.setAttribute('refY', '2.5'); marker.setAttribute('orient', 'auto');
  var mp = document.createElementNS(svgNS, 'path'); mp.setAttribute('d', 'M0,0 L6,2.5 L0,5 Z'); mp.setAttribute('fill', '#94a3b8');
  marker.appendChild(mp); defs.appendChild(marker); svg.appendChild(defs);

  edges.forEach(function(e) {
    if (!positions[e.source] || !positions[e.target]) return;
    var sx = positions[e.source].x + 290, sy = positions[e.source].y + 30;
    var tx = positions[e.target].x, ty = positions[e.target].y + 30;
    var mx = (sx + tx) / 2;
    var path = document.createElementNS(svgNS, 'path');
    path.setAttribute('d', 'M' + sx + ',' + sy + ' C' + mx + ',' + sy + ' ' + mx + ',' + ty + ' ' + tx + ',' + ty);
    path.setAttribute('fill', 'none'); path.setAttribute('stroke', '#94a3b8'); path.setAttribute('stroke-width', '1.5');
    path.setAttribute('marker-end', 'url(#arrow)');
    svg.appendChild(path);
  });

  canvas.appendChild(svg);

  tables.forEach(function(t) {
    var detail = modelDetails[t.id];
    var columns = detail ? (detail.columns || []) : [];
    var pos = positions[t.id];
    if (!pos) return;

    var card = document.createElement('div');
    card.className = 'er-card';
    card.setAttribute('data-table-id', t.id);
    card.style.position = 'absolute'; card.style.left = pos.x + 'px'; card.style.top = pos.y + 'px';
    card.style.width = '288px'; card.style.background = '#fff'; card.style.borderRadius = '8px';
    card.style.boxShadow = '0 2px 8px rgba(0,0,0,0.1)'; card.style.border = '2px solid #e2e8f0';
    card.style.zIndex = '10'; card.style.cursor = diagramState.mode === 'view' ? 'pointer' : 'default';
    card.style.fontSize = '12px'; card.style.lineHeight = '1.4';

    var header = document.createElement('div');
    header.style.background = '#4f46e5'; header.style.color = '#fff'; header.style.padding = '8px 12px';
    header.style.borderRadius = '6px 6px 0 0'; header.style.fontWeight = '600'; header.style.fontSize = '13px';
    header.style.display = 'flex'; header.style.justifyContent = 'space-between'; header.style.alignItems = 'center';
    header.innerHTML = '<span>' + t.table_name + '</span><span style="opacity:0.7;font-size:10px">' + columns.length + ' cols</span>';
    card.appendChild(header);

    var body = document.createElement('div');
    body.style.padding = '4px 0';
    var maxCols = 18;
    columns.slice(0, maxCols).forEach(function(col) {
      var row = document.createElement('div');
      row.style.padding = '3px 10px'; row.style.display = 'flex'; row.style.alignItems = 'center';
      row.style.borderBottom = '1px solid #f1f5f9';

      var icons = '';
      if (col.is_primary_key) icons += '<span style="color:#f59e0b;margin-right:4px">🔑</span>';
      var isFK = edges.some(function(e) { return e.source === t.id && e.source_col === col.column_name; });
      if (isFK) icons += '<span style="color:#4f46e5;font-size:9px;font-weight:600;margin-right:4px">FK</span>';

      var typeStr = col.logical_type + (col.type_length ? '(' + col.type_length + (col.type_scale ? ',' + col.type_scale : '') + ')' : '');
      row.innerHTML = '<span style="width:20px">' + icons + '</span>' +
        '<span style="flex:1;font-weight:500;color:#334155;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">' + col.column_name + '</span>' +
        '<span style="color:#94a3b8;font-size:10px;margin-left:8px;white-space:nowrap">' + typeStr + '</span>';
      body.appendChild(row);
    });
    if (columns.length > maxCols) {
      var more = document.createElement('div');
      more.style.padding = '4px 10px'; more.style.color = '#94a3b8'; more.style.fontSize = '11px'; more.style.textAlign = 'center';
      more.textContent = '... +' + (columns.length - maxCols) + ' more columns';
      body.appendChild(more);
    }
    card.appendChild(body);

    if (diagramState.mode === 'edit') {
      var footer = document.createElement('div');
      footer.style.padding = '4px 10px'; footer.style.borderTop = '1px solid #f1f5f9'; footer.style.textAlign = 'center';
      footer.innerHTML = '<button style="background:none;border:none;color:#4f46e5;cursor:pointer;font-size:11px" onclick="event.stopPropagation();addColumnOnDiagram(' + t.id + ')">+ 添加字段</button>';
      card.appendChild(footer);
    }

    card.addEventListener('click', function() {
      if (diagramState.mode === 'fk') handleFKClick(t.id);
      else if (diagramState.mode === 'view') router.navigate('/models/' + t.id);
    });
    canvas.appendChild(card);
  });

  vp.innerHTML = '';
  vp.appendChild(canvas);
  diagramState._canvas = canvas;
  applyZoom();
}

function handleFKClick(tableId) {
  if (!diagramState.fkSource) {
    diagramState.fkSource = { modelId: tableId };
    highlightFKSource(tableId);
  } else {
    var input = prompt('外键关系: 源列名,目标列名\n示例: user_id,id');
    if (!input) { diagramState.fkSource = null; clearHighlights(); return; }
    var parts = input.split(',').map(function(s) { return s.trim(); });
    if (parts.length < 2) { toast('请输入源列名和目标列名', 'error'); diagramState.fkSource = null; clearHighlights(); return; }
    api.post('/models/' + diagramState.fkSource.modelId + '/foreign-keys', {
      fk_name: 'fk_' + parts[0], column_name: parts[0], ref_model_id: tableId, ref_column_name: parts[1]
    }).then(function() { toast('外键已创建', 'success'); diagramState.fkSource = null; clearHighlights(); loadDiagram(); })
    .catch(function(e) { toast(e.message, 'error'); diagramState.fkSource = null; clearHighlights(); });
  }
}

function highlightFKSource(tableId) {
  clearHighlights();
  var card = document.querySelector('.er-card[data-table-id="' + tableId + '"]');
  if (card) card.style.border = '3px solid #f59e0b';
  toast('已选择源表，请点击目标表', 'info');
}

function clearHighlights() {
  document.querySelectorAll('.er-card').forEach(function(c) { c.style.border = '2px solid #e2e8f0'; });
}

function applyZoom() {
  var canvas = diagramState._canvas || document.getElementById('diagram-canvas');
  if (canvas) canvas.style.transform = 'scale(' + (diagramState.scale || 1) + ')';
  var zl = document.getElementById('zoom-level');
  if (zl) zl.textContent = Math.round((diagramState.scale || 1) * 100) + '%';
}

function fitDiagram() {
  diagramState.scale = 0.55; applyZoom();
}

var colTypes = ['BIGINT','INT','VARCHAR','DECIMAL','FLOAT','DOUBLE','DATE','DATETIME','TIMESTAMP','TEXT','BOOLEAN','JSON'];

function addColumnOnDiagram(modelId) {
  var input = prompt('添加字段\n格式: 字段名,类型,备注\n类型: ' + colTypes.join(','));
  if (!input) return;
  var parts = input.split(',').map(function(s) { return s.trim(); });
  if (parts.length < 2 || !parts[0] || !parts[1]) { toast('至少需要字段名和类型', 'error'); return; }
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
window.applyZoom = applyZoom;
