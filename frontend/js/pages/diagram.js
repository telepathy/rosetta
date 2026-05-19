var diagramState = { mode: 'view', schemaId: null, dragInfo: null, fkSource: null, viewX: 0, viewY: 0, scale: 1 };

async function pageDiagram() {
  document.title = 'Rosetta - ER 图';
  var html = sidebarHtml('/diagram');
  html += '<div class="page-header"><h2>📊 数据库模型图</h2><div class="page-desc">所见即所得编辑 ｜ 拖拽移动表 ｜ 滚轮缩放 ｜ 右键拖拽平移 ｜ 连线建立外键</div></div>';
  html += '<div class="page-toolbar">';
  html += '<select id="diagram-schema" onchange="loadDiagram()"><option value="">选择 Schema...</option></select>';
  html += '<div class="flex-spacer"></div>';
  html += '<button id="btn-mode-view" class="btn btn-sm btn-primary" onclick="setMode(\'view\')">👁 浏览</button>';
  html += '<button id="btn-mode-edit" class="btn btn-sm btn-outline" onclick="setMode(\'edit\')">✏️ 编辑</button>';
  html += '<button id="btn-mode-fk" class="btn btn-sm btn-outline" onclick="setMode(\'fk\')">🔗 建外键</button>';
  html += '<button id="btn-add-table" class="btn btn-sm btn-primary hidden" onclick="addTableOnDiagram()">+ 新表</button>';
  html += '<button class="btn btn-sm btn-outline" onclick="fitDiagram()">⊞ 适应</button>';
  html += '<button class="btn btn-sm btn-outline" onclick="resetDiagram()">1:1</button>';
  html += '</div>';
  html += '<div id="diagram-viewport" style="overflow:hidden;border:1px solid var(--border);border-radius:var(--radius);background:#fafbfc;min-height:500px;position:relative;cursor:grab">';
  html += '<div class="empty-state"><div class="empty-icon">📊</div><p>请先选择一个 Schema 查看 ER 图</p></div>';
  html += '</div>';
  html += '</div></div>';
  setTimeout(loadSchemaOptions, 0);
  return html;
}

function setMode(mode) {
  diagramState.mode = mode;
  diagramState.fkSource = null;
  ['view','edit','fk'].forEach(function(m) {
    var btn = document.getElementById('btn-mode-' + m);
    if (btn) btn.className = 'btn btn-sm ' + (m === mode ? 'btn-primary' : 'btn-outline');
  });
  var addBtn = document.getElementById('btn-add-table');
  if (addBtn) addBtn.classList.toggle('hidden', mode === 'view');
  if (diagramState.schemaId) loadDiagram();
}

async function loadSchemaOptions() {
  try {
    var schemas = [];
    var instances = (await api.get('/instances?page=1&page_size=50')).data.items || [];
    for (var i = 0; i < instances.length; i++) {
      try {
        var s = (await api.get('/instances/' + instances[i].id + '/schemas')).data;
        if (Array.isArray(s)) {
          s.forEach(function(sc) { schemas.push({ id: sc.id, name: instances[i].name + ' / ' + sc.schema_name, instanceId: instances[i].id }); });
        }
      } catch(e) {}
    }
    var sel = document.getElementById('diagram-schema');
    sel.innerHTML = '<option value="">选择 Schema...</option>';
    schemas.forEach(function(s) { sel.innerHTML += '<option value="' + s.id + '">' + s.name + '</option>'; });
    if (schemas.length === 0) {
      document.getElementById('diagram-viewport').innerHTML = '<div class="empty-state"><div class="empty-icon">📊</div><p>没有可用的 Schema</p><p style="font-size:12px;color:var(--text-secondary);margin-top:8px">请先在「实例管理」中创建实例和 Schema，然后在「模型管理」中将模型部署到 Schema</p></div>';
    } else if (schemas.length === 1) {
      sel.value = schemas[0].id;
      loadDiagram();
    }
  } catch(e) {
    document.getElementById('diagram-viewport').innerHTML = '<div class="empty-state"><p>加载失败: ' + e.message + '</p></div>';
  }
}

async function loadDiagram() {
  var schemaId = document.getElementById('diagram-schema').value;
  if (!schemaId) return;
  diagramState.schemaId = schemaId;
  diagramState.viewX = 0; diagramState.viewY = 0; diagramState.scale = 1;

  var container = document.getElementById('diagram-viewport');
  container.innerHTML = '<div style="text-align:center;padding:60px;color:var(--text-secondary)">加载中...</div>';

  try {
    var er = (await api.get('/schemas/' + schemaId + '/er-diagram')).data;
    if (!er || !er.tables || er.tables.length === 0) {
      container.innerHTML = '<div class="empty-state"><div class="empty-icon">📊</div><p>该 Schema 下没有部署的模型，请先部署模型到此 Schema</p></div>';
      return;
    }
    var tables = er.tables;
    var edges = er.edges || [];
    var modelDetails = {};
    for (var i = 0; i < tables.length; i++) {
      try {
        var d = (await api.get('/models/' + tables[i].id)).data;
        modelDetails[tables[i].id] = d;
      } catch(e) {}
    }
    renderDiagram(container, tables, edges, modelDetails);
  } catch(e) {
    container.innerHTML = '<div class="empty-state"><p>加载失败: ' + e.message + '</p></div>';
  }
}

function renderDiagram(container, tables, edges, modelDetails) {
  var cardW = 300, colH = 26, padX = 60, padY = 20;
  var cols = Math.max(1, Math.floor(Math.min(5, Math.ceil(Math.sqrt(tables.length * 1.5)))));
  var positions = {};
  tables.forEach(function(t, i) {
    var col = i % cols, row = Math.floor(i / cols);
    positions[t.id] = { x: padX + col * (cardW + padX), y: padY + row * 200 };
  });
  var headerH = diagramState.mode === 'view' ? 32 : 42;

  var maxX = 0, maxY = 0;
  tables.forEach(function(t) {
    var detail = modelDetails[t.id];
    var colCount = detail ? (detail.columns ? detail.columns.length : 0) : 0;
    var h = headerH + colCount * colH + (diagramState.mode !== 'view' ? 30 : 12);
    var p = positions[t.id];
    maxX = Math.max(maxX, p.x + cardW + padX);
    maxY = Math.max(maxY, p.y + h + padY + 20);
  });

  var svgNS = 'http://www.w3.org/2000/svg';
  var svg = document.createElementNS(svgNS, 'svg');
  svg.setAttribute('id', 'diagram-svg');
  svg.setAttribute('viewBox', '0 0 ' + maxX + ' ' + maxY);
  svg.setAttribute('width', maxX);
  svg.setAttribute('height', maxY);
  svg.style.width = maxX + 'px';
  svg.style.height = maxY + 'px';

  var defs = document.createElementNS(svgNS, 'defs');

  var marker = document.createElementNS(svgNS, 'marker');
  marker.setAttribute('id', 'arrowhead'); marker.setAttribute('markerWidth', '6'); marker.setAttribute('markerHeight', '5');
  marker.setAttribute('refX', '6'); marker.setAttribute('refY', '2.5'); marker.setAttribute('orient', 'auto');
  var ap = document.createElementNS(svgNS, 'path'); ap.setAttribute('d', 'M0,0 L6,2.5 L0,5 Z'); ap.setAttribute('fill', '#cbd5e1');
  marker.appendChild(ap); defs.appendChild(marker);

  var shadow = document.createElementNS(svgNS, 'filter');
  shadow.setAttribute('id', 'card-shadow'); shadow.setAttribute('x', '-5%'); shadow.setAttribute('y', '-5%');
  shadow.setAttribute('width', '110%'); shadow.setAttribute('height', '110%');
  var feDrop = document.createElementNS(svgNS, 'feDropShadow');
  feDrop.setAttribute('dx', '0'); feDrop.setAttribute('dy', '1'); feDrop.setAttribute('stdDeviation', '2');
  feDrop.setAttribute('flood-color', '#000'); feDrop.setAttribute('flood-opacity', '0.1');
  shadow.appendChild(feDrop); defs.appendChild(shadow);
  svg.appendChild(defs);

  var bgGroup = document.createElementNS(svgNS, 'g');
  bgGroup.setAttribute('id', 'bg-layer');
  tables.forEach(function(t) {
    var detail = modelDetails[t.id];
    var columns = detail ? (detail.columns || []) : [];
    var height = headerH + columns.length * colH + (diagramState.mode !== 'view' ? 30 : 12);
    bgGroup.appendChild(tableCardBg(svgNS, t, columns.length, positions[t.id], cardW, height, headerH, colH));
  });
  svg.appendChild(bgGroup);

  var edgeGroup = document.createElementNS(svgNS, 'g');
  edgeGroup.setAttribute('id', 'edge-layer');
  edgeGroup.setAttribute('style', 'pointer-events:none');
  edges.forEach(function(e) {
    if (!positions[e.source] || !positions[e.target]) return;
    var sp = positions[e.source], tp = positions[e.target];
    var sd = modelDetails[e.source], td = modelDetails[e.target];
    var sc = sd ? (sd.columns || []) : []; var tc = td ? (td.columns || []) : [];
    var si = sc.findIndex(function(c) { return c.column_name === e.source_col; });
    var ti = tc.findIndex(function(c) { return c.column_name === e.target_col; });
    var srcY = sp.y + headerH + 4 + (si >= 0 ? si : 0) * colH + colH / 2 + 2;
    var tgtY = tp.y + headerH + 4 + (ti >= 0 ? ti : 0) * colH + colH / 2 + 2;

    var path = document.createElementNS(svgNS, 'path');
    var sx = sp.x + cardW, tx = tp.x;
    var mx = (sx + tx) / 2;
    path.setAttribute('d', 'M' + sx + ',' + srcY + ' C' + mx + ',' + srcY + ' ' + mx + ',' + tgtY + ' ' + tx + ',' + tgtY);
    path.setAttribute('fill', 'none'); path.setAttribute('stroke', '#94a3b8'); path.setAttribute('stroke-width', '2.5');
    path.setAttribute('stroke-opacity', '0.5'); path.setAttribute('marker-end', 'url(#arrowhead)');
    edgeGroup.appendChild(path);

    var lb = document.createElementNS(svgNS, 'rect');
    lb.setAttribute('x', mx - 30); lb.setAttribute('y', (srcY + tgtY) / 2 - 9);
    lb.setAttribute('width', '60'); lb.setAttribute('height', '18');
    lb.setAttribute('rx', '4'); lb.setAttribute('fill', '#fff'); lb.setAttribute('stroke', '#e2e8f0');
    lb.setAttribute('stroke-width', '1');
    edgeGroup.appendChild(lb);

    var lt = document.createElementNS(svgNS, 'text');
    lt.setAttribute('x', mx); lt.setAttribute('y', (srcY + tgtY) / 2 + 4);
    lt.setAttribute('fill', '#64748b'); lt.setAttribute('font-size', '9'); lt.setAttribute('text-anchor', 'middle');
    lt.textContent = (e.source_col.length > 8 ? e.source_col.slice(0,7) + '…' : e.source_col) + '→' + (e.target_col.length > 8 ? e.target_col.slice(0,7) + '…' : e.target_col);
    edgeGroup.appendChild(lt);
  });
  svg.appendChild(edgeGroup);

  var cardGroup = document.createElementNS(svgNS, 'g');
  cardGroup.setAttribute('id', 'card-layer');
  tables.forEach(function(t) {
    var detail = modelDetails[t.id];
    var columns = detail ? (detail.columns || []) : [];
    var height = headerH + columns.length * colH + (diagramState.mode !== 'view' ? 30 : 12);
    cardGroup.appendChild(tableCardContent(svgNS, t, columns, positions[t.id], cardW, height, headerH, colH, edges));
  });
  svg.appendChild(cardGroup);

  container.innerHTML = '';
  container.appendChild(svg);

  setupViewport(container);

  if (diagramState.mode !== 'view') setupDrag(container);
  diagramState.viewX = 0; diagramState.viewY = 0; diagramState.scale = 1;
  fitDiagram();
}

function tableCardBg(ns, t, colCount, pos, cardW, height, headerH) {
  var g = document.createElementNS(ns, 'g');
  g.setAttribute('transform', 'translate(' + pos.x + ',' + pos.y + ')');

  var bg = document.createElementNS(ns, 'rect');
  bg.setAttribute('x', '0'); bg.setAttribute('y', '0'); bg.setAttribute('width', cardW); bg.setAttribute('height', height);
  bg.setAttribute('rx', '6'); bg.setAttribute('fill', '#fff'); bg.setAttribute('stroke', '#cbd5e1'); bg.setAttribute('stroke-width', '2');
  bg.setAttribute('filter', 'url(#card-shadow)');
  g.appendChild(bg);

  var hbg = document.createElementNS(ns, 'rect');
  hbg.setAttribute('x', '2'); hbg.setAttribute('y', '2'); hbg.setAttribute('width', cardW - 4); hbg.setAttribute('height', headerH - 2);
  hbg.setAttribute('rx', '4'); hbg.setAttribute('fill', '#4f46e5');
  g.appendChild(hbg);

  return g;
}

function tableCardContent(ns, t, columns, pos, cardW, height, headerH, colH, edges) {
  var g = document.createElementNS(ns, 'g');
  g.setAttribute('transform', 'translate(' + pos.x + ',' + pos.y + ')');
  g.setAttribute('data-table-id', t.id);
  g.classList.add('er-table-group');

  var title = svgText(ns, 12, 22, '#fff', 13, '600', t.table_name);
  g.appendChild(title);
  g.appendChild(svgText(ns, cardW - 20, 22, 'rgba(255,255,255,0.6)', 10, 'normal', columns.length + ' cols', 'end'));

  if (diagramState.mode !== 'view') {
    var editBtn = svgText(ns, cardW - 70, 22, 'rgba(255,255,255,0.85)', 11, 'normal', '✏️');
    editBtn.style.cursor = 'pointer';
    editBtn.addEventListener('click', function(ev) { ev.stopPropagation(); editTableName(t.id); });
    g.appendChild(editBtn);
  }

  columns.forEach(function(col, ci) {
    var cy = headerH + 4 + ci * colH + colH / 2 + 2;
    if (col.is_primary_key) g.appendChild(svgText(ns, 8, cy, '#f59e0b', 10, 'normal', '🔑'));

    var isFK = false;
    edges.forEach(function(e) { if (e.source === t.id && e.source_col === col.column_name) isFK = true; });
    var colX = 26;
    if (isFK) { g.appendChild(svgText(ns, 24, cy, '#4f46e5', 9, '600', 'FK')); colX = 44; }

    g.appendChild(svgText(ns, colX, cy, '#334155', 11, '500', col.column_name));
    var typeStr = col.logical_type + (col.type_length ? '(' + col.type_length + (col.type_scale ? ',' + col.type_scale : '') + ')' : '');
    g.appendChild(svgText(ns, cardW - 10, cy, '#94a3b8', 10, 'normal', typeStr, 'end'));

    if (diagramState.mode === 'edit') {
      var editIcon = svgText(ns, cardW - 80, cy, '#94a3b8', 10, 'normal', '✎');
      editIcon.style.cursor = 'pointer';
      (function(mid, colId) { editIcon.addEventListener('click', function(ev) { ev.stopPropagation(); editColumnOnDiagram(mid, colId); }); })(t.id, col.id);
      g.appendChild(editIcon);
    }
    if (diagramState.mode === 'fk') {
      var fkDot = document.createElementNS(ns, 'circle');
      fkDot.setAttribute('cx', cardW - 80); fkDot.setAttribute('cy', cy); fkDot.setAttribute('r', '6');
      fkDot.setAttribute('fill', '#e2e8f0'); fkDot.setAttribute('stroke', '#94a3b8');
      fkDot.style.cursor = 'pointer';
      (function(mid, colName) {
        fkDot.addEventListener('click', function(ev) { ev.stopPropagation(); startFK(mid, colName); });
        fkDot.addEventListener('mouseenter', function() { fkDot.setAttribute('fill', '#4f46e5'); });
        fkDot.addEventListener('mouseleave', function() { fkDot.setAttribute('fill', diagramState.fkSource && diagramState.fkSource.modelId === mid && diagramState.fkSource.colName === colName ? '#f59e0b' : '#e2e8f0'); });
      })(t.id, col.column_name);
      g.appendChild(fkDot);
    }
  });

  if (diagramState.mode === 'edit') {
    var addY = headerH + columns.length * colH + 16;
    var addBg = document.createElementNS(ns, 'rect');
    addBg.setAttribute('x', '4'); addBg.setAttribute('y', addY - 4); addBg.setAttribute('width', cardW - 8); addBg.setAttribute('height', '24');
    addBg.setAttribute('rx', '4'); addBg.setAttribute('fill', '#f8fafc'); addBg.style.cursor = 'pointer';
    g.appendChild(addBg);
    var addTxt = svgText(ns, cardW / 2, addY + 8, '#94a3b8', 11, 'normal', '+ 添加字段', 'middle');
    addTxt.style.cursor = 'pointer';
    (function(mid) {
      addTxt.addEventListener('click', function(ev) { ev.stopPropagation(); addColumnOnDiagram(mid); });
      addBg.addEventListener('click', function(ev) { ev.stopPropagation(); addColumnOnDiagram(mid); });
    })(t.id);
    g.appendChild(addTxt);
  }

  g.addEventListener('click', function() { if (diagramState.mode === 'view') router.navigate('/models/' + t.id); });
  return g;
}

function setupViewport(container) {
  var svg = document.getElementById('diagram-svg');
  if (!svg) return;
  container.style.cursor = 'grab';
  var isPanning = false, sx, sy, vx, vy;

  container.addEventListener('wheel', function(e) {
    e.preventDefault();
    var delta = e.deltaY > 0 ? -0.1 : 0.1;
    var newScale = Math.max(0.15, Math.min(3, diagramState.scale + delta));
    var rect = container.getBoundingClientRect();
    var mx = e.clientX - rect.left, my = e.clientY - rect.top;
    diagramState.viewX -= mx * (newScale - diagramState.scale);
    diagramState.viewY -= my * (newScale - diagramState.scale);
    diagramState.scale = newScale;
    applyView();
  }, { passive: false });

  container.addEventListener('mousedown', function(e) {
    if (diagramState.mode !== 'view' && e.target.closest('.er-table-group')) return;
    if (e.button === 0 || e.button === 2) {
      isPanning = true; container.style.cursor = 'grabbing';
      sx = e.clientX; sy = e.clientY;
      vx = diagramState.viewX; vy = diagramState.viewY;
      e.preventDefault();
    }
  });

  window.addEventListener('mousemove', function(e) {
    if (!isPanning) return;
    diagramState.viewX = vx + (e.clientX - sx) / diagramState.scale;
    diagramState.viewY = vy + (e.clientY - sy) / diagramState.scale;
    applyView();
  });

  window.addEventListener('mouseup', function() { if (isPanning) { isPanning = false; container.style.cursor = diagramState.mode === 'view' ? 'grab' : 'default'; } });

  container.addEventListener('contextmenu', function(e) { if (!e.target.closest('.er-table-group')) e.preventDefault(); });
}

function applyView() {
  var svg = document.getElementById('diagram-svg');
  if (!svg) return;
  var vp = document.getElementById('diagram-viewport');
  var pw = vp ? vp.clientWidth : 1200, ph = vp ? vp.clientHeight : 800;
  svg.style.transformOrigin = '0 0';
  svg.style.transform = 'translate(' + diagramState.viewX * diagramState.scale + 'px,' + diagramState.viewY * diagramState.scale + 'px) scale(' + diagramState.scale + ')';
  svg.style.minWidth = 'auto';
  svg.style.width = (pw / diagramState.scale) + 'px';
}

function fitDiagram() {
  var svg = document.getElementById('diagram-svg');
  var vp = document.getElementById('diagram-viewport');
  if (!svg || !vp) return;
  var vb = svg.getAttribute('viewBox').split(' ').map(Number);
  var scaleX = vp.clientWidth / vb[2], scaleY = vp.clientHeight / vb[3];
  diagramState.scale = Math.min(scaleX, scaleY, 1) * 0.9;
  diagramState.viewX = (vp.clientWidth / diagramState.scale - vb[2]) / 2;
  diagramState.viewY = (vp.clientHeight / diagramState.scale - vb[3]) / 2;
  applyView();
}

function resetDiagram() {
  diagramState.scale = 1; diagramState.viewX = 0; diagramState.viewY = 0;
  applyView();
}

var colTypes = ['BIGINT','INT','VARCHAR','DECIMAL','FLOAT','DOUBLE','DATE','DATETIME','TIMESTAMP','TEXT','BOOLEAN','JSON'];

function editColumnOnDiagram(modelId, colId) {
  var input = prompt('编辑字段 (格式: 字段名,类型,备注)\n类型选: ' + colTypes.join(','));
  if (!input) return;
  var parts = input.split(',').map(function(s) { return s.trim(); });
  if (parts.length < 1 || !parts[0]) return;
  api.get('/models/' + modelId).then(function(r) {
    var cols = r.data.columns || [];
    var col = cols.find(function(c) { return c.id === colId; });
    if (!col) { toast('字段未找到', 'error'); return; }
    col.column_name = parts[0];
    if (parts[1]) col.logical_type = parts[1].toUpperCase();
    if (parts[2]) col.comment = parts[2];
    api.put('/models/' + modelId + '/columns', { columns: cols }).then(function() {
      toast('字段已更新', 'success'); loadDiagram();
    }).catch(function(e) { toast(e.message, 'error'); });
  });
}

function addColumnOnDiagram(modelId) {
  var input = prompt('添加字段\n格式: 字段名,类型,备注\n类型选: ' + colTypes.join(','));
  if (!input) return;
  var parts = input.split(',').map(function(s) { return s.trim(); });
  if (parts.length < 2 || !parts[0] || !parts[1]) { toast('至少需要字段名和类型', 'error'); return; }
  api.get('/models/' + modelId).then(function(r) {
    var cols = r.data.columns || [];
    cols.push({ ordinal: cols.length + 1, column_name: parts[0], logical_type: parts[1].toUpperCase(), nullable: true, is_primary_key: false, default_value: '', comment: parts[2] || '', type_length: null, type_scale: null });
    api.put('/models/' + modelId + '/columns', { columns: cols }).then(function() { toast('字段已添加', 'success'); loadDiagram(); }).catch(function(e) { toast(e.message, 'error'); });
  });
}

function editTableName(modelId) {
  var input = prompt('编辑表名和备注\n格式: 表名,备注');
  if (!input) return;
  var parts = input.split(',').map(function(s) { return s.trim(); });
  api.get('/models/' + modelId).then(function(r) {
    api.put('/models/' + modelId, { table_name: parts[0] || r.data.table_name, table_comment: parts[1] || '', table_status: r.data.table_status }).then(function() { toast('已更新', 'success'); loadDiagram(); }).catch(function(e) { toast(e.message, 'error'); });
  });
}

function startFK(modelId, colName) {
  if (!diagramState.fkSource) { diagramState.fkSource = { modelId: modelId, colName: colName }; toast('已选择 ' + colName + ' 作为外键源，请点击目标列', 'info'); }
  else completeFK(modelId, colName);
}

function completeFK(modelId, colName) {
  var src = diagramState.fkSource; diagramState.fkSource = null;
  if (src.modelId === modelId && src.colName === colName) { toast('不能引用自身', 'error'); return; }
  api.post('/models/' + src.modelId + '/foreign-keys', { fk_name: 'fk_' + src.colName, column_name: src.colName, ref_model_id: modelId, ref_column_name: colName }).then(function() { toast('外键已创建', 'success'); loadDiagram(); }).catch(function(e) { toast(e.message, 'error'); });
}

function addTableOnDiagram() {
  var input = prompt('新建模型\n格式: 表名,备注');
  if (!input) return;
  var parts = input.split(',').map(function(s) { return s.trim(); });
  if (!parts[0]) { toast('请输入表名', 'error'); return; }
  api.post('/models', { table_name: parts[0], table_comment: parts[1] || '' }).then(function(r) {
    var modelId = r.data.id;
    api.post('/models/' + modelId + '/deploy', { schema_id: parseInt(diagramState.schemaId), dialect: 'MYSQL' }).then(function() { toast('模型已创建并部署', 'success'); loadDiagram(); }).catch(function() { toast('模型已创建，请手动部署到 Schema', 'success'); loadDiagram(); });
  }).catch(function(e) { toast(e.message, 'error'); });
}

function setupDrag(container) {
  container.addEventListener('mousedown', function(e) {
    if (diagramState.mode === 'view') return;
    var target = e.target.closest('.er-table-group');
    if (target && !e.target.closest('text') && !e.target.closest('circle')) {
      var tableId = target.getAttribute('data-table-id');
      var m = target.getAttribute('transform').match(/translate\(([\d.]+),\s*([\d.]+)\)/);
      if (!m) return;
      diagramState.dragInfo = { elem: target, sx: e.clientX, sy: e.clientY, ox: parseFloat(m[1]), oy: parseFloat(m[2]) };
      e.preventDefault(); e.stopPropagation();
    }
  });
  container.addEventListener('mousemove', function(e) {
    if (!diagramState.dragInfo) return;
    var dx = (e.clientX - diagramState.dragInfo.sx) / diagramState.scale;
    var dy = (e.clientY - diagramState.dragInfo.sy) / diagramState.scale;
    diagramState.dragInfo.elem.setAttribute('transform', 'translate(' + (diagramState.dragInfo.ox + dx) + ',' + (diagramState.dragInfo.oy + dy) + ')');
  });
  container.addEventListener('mouseup', function() { diagramState.dragInfo = null; });
}

function svgText(ns, x, y, fill, size, weight, content, anchor) {
  var t = document.createElementNS(ns, 'text');
  t.setAttribute('x', x); t.setAttribute('y', y); t.setAttribute('fill', fill);
  t.setAttribute('font-size', size); t.setAttribute('font-weight', weight || 'normal');
  if (anchor) t.setAttribute('text-anchor', anchor);
  t.textContent = content; return t;
}

window.loadDiagram = loadDiagram;
window.setMode = setMode;
window.fitDiagram = fitDiagram;
window.resetDiagram = resetDiagram;
window.editColumnOnDiagram = editColumnOnDiagram;
window.addColumnOnDiagram = addColumnOnDiagram;
window.editTableName = editTableName;
window.startFK = startFK;
window.completeFK = completeFK;
window.addTableOnDiagram = addTableOnDiagram;
