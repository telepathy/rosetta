async function pageDiagram() {
  document.title = 'Rosetta - ER 图';
  var html = sidebarHtml('/diagram');
  html += '<div class="page-header"><h2>📊 数据库模型图</h2><div class="page-desc">表结构与关系可视化 ｜ 点击表卡片进入详情</div></div>';
  html += '<div class="page-toolbar">';
  html += '<select id="diagram-schema" onchange="loadDiagram()"><option value="">选择 Schema...</option></select>';
  html += '<div class="flex-spacer"></div>';
  html += '<button class="btn btn-sm btn-outline" onclick="zoomDiagram(0.2)">🔍+</button>';
  html += '<button class="btn btn-sm btn-outline" onclick="zoomDiagram(-0.2)">🔍-</button>';
  html += '<button class="btn btn-sm btn-outline" onclick="resetZoom()">重置</button>';
  html += '</div>';
  html += '<div id="diagram-container" style="overflow:auto;border:1px solid var(--border);border-radius:var(--radius);background:#fafbfc;min-height:500px;position:relative">';
  html += '<div class="empty-state"><div class="empty-icon">📊</div><p>请先选择一个 Schema 查看 ER 图</p></div>';
  html += '</div>';
  html += '</div></div>';

  setTimeout(loadSchemaOptions, 0);
  return html;
}

async function loadSchemaOptions() {
  try {
    var schemas = [];
    var instances = (await api.get('/instances?page=1&page_size=50')).data.items;
    for (var i = 0; i < instances.length; i++) {
      var s = (await api.get('/instances/' + instances[i].id + '/schemas')).data;
      s.forEach(function(sc) { schemas.push({ id: sc.id, name: instances[i].name + ' / ' + sc.schema_name, instanceId: instances[i].id }); });
    }
    var sel = document.getElementById('diagram-schema');
    schemas.forEach(function(s) {
      sel.innerHTML += '<option value="' + s.id + '">' + s.name + '</option>';
    });
  } catch(e) {}
}

async function loadDiagram() {
  var schemaId = document.getElementById('diagram-schema').value;
  if (!schemaId) return;

  var container = document.getElementById('diagram-container');
  container.innerHTML = '<div style="text-align:center;padding:60px;color:var(--text-secondary)">加载中...</div>';

  try {
    var er = (await api.get('/schemas/' + schemaId + '/er-diagram')).data;
    if (!er.data || !er.data.tables || er.data.tables.length === 0) {
      container.innerHTML = '<div class="empty-state"><div class="empty-icon">📊</div><p>该 Schema 下没有部署的模型</p></div>';
      return;
    }

    var tables = er.data.tables;
    var edges = er.data.edges || [];

    var modelDetails = {};
    for (var i = 0; i < tables.length; i++) {
      try {
        var d = (await api.get('/models/' + tables[i].id)).data;
        modelDetails[tables[i].id] = d.data;
      } catch(e) {}
    }

    renderDiagramSVG(container, tables, edges, modelDetails);
  } catch(e) {
    container.innerHTML = '<div class="empty-state"><p>加载失败: ' + e.message + '</p></div>';
  }
}

function renderDiagramSVG(container, tables, edges, modelDetails) {
  var cardW = 280, cardH = 40, headerH = 36, colH = 24, padX = 40, padY = 60;
  var cols = Math.max(1, Math.floor(Math.min(3, Math.ceil(Math.sqrt(tables.length)))));
  var positions = {};

  tables.forEach(function(t, i) {
    var col = i % cols, row = Math.floor(i / cols);
    positions[t.id] = { x: padX + col * (cardW + padX), y: padY + row * (cardH + padY + 60) };
  });

  var totalH = 0;
  tables.forEach(function(t) {
    var cols = (modelDetails[t.id] && modelDetails[t.id].columns) ? modelDetails[t.id].columns.length : 0;
    var h = headerH + cols * colH + 8;
    var p = positions[t.id];
    totalH = Math.max(totalH, p.y + h + padY);
  });
  var totalW = cols * (cardW + padX) + padX;

  var svgNS = 'http://www.w3.org/2000/svg';
  var svg = document.createElementNS(svgNS, 'svg');
  svg.setAttribute('width', totalW);
  svg.setAttribute('height', totalH);
  svg.setAttribute('id', 'diagram-svg');
  svg.style.minWidth = totalW + 'px';

  var defs = document.createElementNS(svgNS, 'defs');
  var marker = document.createElementNS(svgNS, 'marker');
  marker.setAttribute('id', 'arrowhead');
  marker.setAttribute('markerWidth', '8');
  marker.setAttribute('markerHeight', '6');
  marker.setAttribute('refX', '8');
  marker.setAttribute('refY', '3');
  marker.setAttribute('orient', 'auto');
  var arrowPath = document.createElementNS(svgNS, 'path');
  arrowPath.setAttribute('d', 'M0,0 L8,3 L0,6 Z');
  arrowPath.setAttribute('fill', '#94a3b8');
  marker.appendChild(arrowPath);
  defs.appendChild(marker);
  svg.appendChild(defs);

  var groups = {};
  tables.forEach(function(t) {
    var p = positions[t.id];
    var detail = modelDetails[t.id];
    var columns = detail ? (detail.columns || []) : [];
    var height = headerH + columns.length * colH + 8;
    var g = document.createElementNS(svgNS, 'g');
    g.setAttribute('transform', 'translate(' + p.x + ',' + p.y + ')');
    g.setAttribute('class', 'er-table-group');
    g.style.cursor = 'pointer';
    g.addEventListener('click', function() { router.navigate('/models/' + t.id); });

    var r = document.createElementNS(svgNS, 'rect');
    r.setAttribute('x', '0'); r.setAttribute('y', '0');
    r.setAttribute('width', cardW); r.setAttribute('height', height);
    r.setAttribute('rx', '6'); r.setAttribute('ry', '6');
    r.setAttribute('fill', '#fff'); r.setAttribute('stroke', '#cbd5e1'); r.setAttribute('stroke-width', '1.5');
    r.setAttribute('filter', 'drop-shadow(0 1px 2px rgba(0,0,0,0.06))');
    g.appendChild(r);

    var headerR = document.createElementNS(svgNS, 'rect');
    headerR.setAttribute('x', '1'); headerR.setAttribute('y', '1');
    headerR.setAttribute('width', cardW - 2); headerR.setAttribute('height', headerH - 2);
    headerR.setAttribute('rx', '5'); headerR.setAttribute('ry', '5');
    headerR.setAttribute('fill', '#4f46e5');
    g.appendChild(headerR);

    var headerR2 = document.createElementNS(svgNS, 'rect');
    headerR2.setAttribute('x', '1'); headerR2.setAttribute('y', headerH - 4);
    headerR2.setAttribute('width', cardW - 2); headerR2.setAttribute('height', '4');
    headerR2.setAttribute('fill', '#4f46e5');
    g.appendChild(headerR2);

    var title = document.createElementNS(svgNS, 'text');
    title.setAttribute('x', '12'); title.setAttribute('y', '24');
    title.setAttribute('fill', '#fff'); title.setAttribute('font-size', '13'); title.setAttribute('font-weight', '600');
    title.textContent = t.table_name;
    g.appendChild(title);

    if (t.column_count > 0) {
      var countBadge = document.createElementNS(svgNS, 'text');
      countBadge.setAttribute('x', cardW - 12); countBadge.setAttribute('y', '24');
      countBadge.setAttribute('fill', 'rgba(255,255,255,0.7)'); countBadge.setAttribute('font-size', '11'); countBadge.setAttribute('text-anchor', 'end');
      countBadge.textContent = t.column_count + ' cols';
      g.appendChild(countBadge);
    }

    columns.forEach(function(col, ci) {
      var cy = headerH + 4 + ci * colH + colH / 2 + 2;

      if (col.is_primary_key) {
        var pk = document.createElementNS(svgNS, 'text');
        pk.setAttribute('x', '10'); pk.setAttribute('y', cy);
        pk.setAttribute('fill', '#f59e0b'); pk.setAttribute('font-size', '10');
        pk.textContent = '🔑';
        g.appendChild(pk);
      }

      var fkCol = false;
      edges.forEach(function(e) {
        if (e.source === t.id && e.source_col === col.column_name) fkCol = true;
      });
      if (fkCol) {
        var fk = document.createElementNS(svgNS, 'text');
        fk.setAttribute('x', '28'); fk.setAttribute('y', cy);
        fk.setAttribute('fill', '#4f46e5'); fk.setAttribute('font-size', '9');
        fk.textContent = 'FK';
        g.appendChild(fk);
      }

      var colName = document.createElementNS(svgNS, 'text');
      var colX = fkCol ? 46 : 28;
      colName.setAttribute('x', colX); colName.setAttribute('y', cy);
      colName.setAttribute('fill', '#334155'); colName.setAttribute('font-size', '11'); colName.setAttribute('font-weight', '500');
      colName.textContent = col.column_name;
      g.appendChild(colName);

      var typeStr = col.logical_type + ((col.type_length) ? '(' + col.type_length + (col.type_scale ? ',' + col.type_scale : '') + ')' : '');
      var colType = document.createElementNS(svgNS, 'text');
      colType.setAttribute('x', cardW - 10); colType.setAttribute('y', cy);
      colType.setAttribute('fill', '#94a3b8'); colType.setAttribute('font-size', '10'); colType.setAttribute('text-anchor', 'end');
      colType.textContent = typeStr;
      g.appendChild(colType);
    });

    groups[t.id] = { g: g, cols: columns, pos: p, headerH: headerH, colH: colH };
    svg.appendChild(g);
  });

  edges.forEach(function(e) {
    if (!positions[e.source] || !positions[e.target]) return;
    var src = groups[e.source], tgt = groups[e.target];
    if (!src || !tgt) return;

    var srcColIdx = -1, tgtColIdx = -1;
    src.cols.forEach(function(c, i) { if (c.column_name === e.source_col) srcColIdx = i; });
    tgt.cols.forEach(function(c, i) { if (c.column_name === e.target_col) tgtColIdx = i; });

    var srcY = src.pos.y + src.headerH + 4 + (srcColIdx >= 0 ? srcColIdx : 0) * src.colH + src.colH / 2 + 2;
    var tgtY = tgt.pos.y + tgt.headerH + 4 + (tgtColIdx >= 0 ? tgtColIdx : 0) * tgt.colH + tgt.colH / 2 + 2;

    var srcX = src.pos.x + 280;
    var tgtX = tgt.pos.x;

    var path = document.createElementNS(svgNS, 'path');
    var mx = (srcX + tgtX) / 2;
    var d = 'M' + srcX + ',' + srcY + ' C' + mx + ',' + srcY + ' ' + mx + ',' + tgtY + ' ' + tgtX + ',' + tgtY;
    path.setAttribute('d', d);
    path.setAttribute('fill', 'none');
    path.setAttribute('stroke', '#94a3b8');
    path.setAttribute('stroke-width', '1.5');
    path.setAttribute('marker-end', 'url(#arrowhead)');
    svg.appendChild(path);

    var labelBg = document.createElementNS(svgNS, 'rect');
    var labelX = mx - 30, labelY = (srcY + tgtY) / 2 - 8;
    labelBg.setAttribute('x', labelX); labelBg.setAttribute('y', labelY);
    labelBg.setAttribute('width', '60'); labelBg.setAttribute('height', '16');
    labelBg.setAttribute('rx', '4'); labelBg.setAttribute('fill', '#f1f5f9'); labelBg.setAttribute('stroke', '#e2e8f0');
    svg.appendChild(labelBg);

    var label = document.createElementNS(svgNS, 'text');
    label.setAttribute('x', mx); label.setAttribute('y', (srcY + tgtY) / 2 + 3);
    label.setAttribute('fill', '#64748b'); label.setAttribute('font-size', '9'); label.setAttribute('text-anchor', 'middle');
    label.textContent = e.source_col + '→' + e.target_col;
    svg.appendChild(label);
  });

  container.innerHTML = '';
  container.appendChild(svg);
  window._diagramZoom = 1;
}

function zoomDiagram(delta) {
  window._diagramZoom = Math.max(0.3, Math.min(2, (window._diagramZoom || 1) + delta));
  var svg = document.getElementById('diagram-svg');
  if (svg) svg.style.transform = 'scale(' + window._diagramZoom + ')';
}

function resetZoom() {
  window._diagramZoom = 1;
  var svg = document.getElementById('diagram-svg');
  if (svg) svg.style.transform = 'scale(1)';
}

window.loadDiagram = loadDiagram;
window.zoomDiagram = zoomDiagram;
window.resetZoom = resetZoom;
