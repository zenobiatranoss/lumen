(() => {
  'use strict';

  const $ = (sel, root = document) => root.querySelector(sel);
  const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));
  const esc = (s) => String(s).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]);
  const clamp = (v, lo, hi) => Math.min(hi, Math.max(lo, v));
  const qs = (obj) => new URLSearchParams(obj).toString();

  const store = {
    get(key, fallback) {
      try {
        const raw = localStorage.getItem('lumen.' + key);
        return raw === null ? fallback : JSON.parse(raw);
      } catch {
        return fallback;
      }
    },
    set(key, value) {
      try {
        localStorage.setItem('lumen.' + key, JSON.stringify(value));
      } catch {}
    },
  };

  const launchQuery = new URLSearchParams(location.search);
  const embedded = launchQuery.get('pane') === 'b';
  const paneId = embedded ? 'b' : 'a';
  const DRAG_TYPE = 'application/x-lumen-paths';

  const ICONS = {
    folder: '<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>',
    newfolder: '<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><path d="M12 10v5M9.5 12.5h5"/>',
    newfile: '<path d="M6 3h8l4 4v14H6z"/><path d="M14 3v4h4M12 10v6M9 13h6"/>',
    image: '<rect x="3" y="4" width="18" height="16" rx="2"/><circle cx="9" cy="10" r="1.6"/><path d="m21 16-5-5-8 8"/>',
    video: '<rect x="3" y="5" width="18" height="14" rx="2"/><path d="m10 9 5 3-5 3z"/>',
    audio: '<path d="M9 18V6l10-2v12"/><circle cx="6.5" cy="18" r="2.5"/><circle cx="16.5" cy="16" r="2.5"/>',
    doc: '<path d="M6 3h8l4 4v14H6z"/><path d="M14 3v4h4M9 12h6M9 16h6"/>',
    code: '<path d="m8 8-4 4 4 4M16 8l4 4-4 4M13.5 5l-3 14"/>',
    archive: '<rect x="4" y="3" width="16" height="18" rx="2"/><path d="M12 3v8M10 6h2M12 8h2"/><rect x="10" y="12" width="4" height="5" rx="1"/>',
    other: '<path d="M6 3h8l4 4v14H6z"/><path d="M14 3v4h4"/>',
    home: '<path d="m3 11 9-7 9 7"/><path d="M5 10v10h14V10"/>',
    desktop: '<rect x="3" y="4" width="18" height="12" rx="2"/><path d="M8 20h8M12 16v4"/>',
    downloads: '<path d="M12 4v11m0 0-4-4m4 4 4-4M5 20h14"/>',
    root: '<rect x="3" y="4" width="18" height="6" rx="1.5"/><rect x="3" y="14" width="18" height="6" rx="1.5"/><path d="M7 7h.01M7 17h.01"/>',
    trash: '<path d="M4 7h16M9 7V4h6v3M6 7l1 13h10l1-13M10 11v6M14 11v6"/>',
    star: '<path d="m12 3 2.7 5.6 6.1.9-4.4 4.3 1 6.1L12 17l-5.4 2.9 1-6.1L3.2 9.5l6.1-.9z"/>',
    search: '<circle cx="11" cy="11" r="6"/><path d="m20 20-4-4"/>',
    back: '<path d="m15 5-7 7 7 7"/>',
    forward: '<path d="m9 5 7 7-7 7"/>',
    up: '<path d="m5 15 7-7 7 7"/>',
    refresh: '<path d="M20 5v5h-5"/><path d="M19 10a7 7 0 1 0 1 4"/>',
    gallery: '<rect x="3" y="4" width="10" height="7" rx="1"/><rect x="15" y="4" width="6" height="7" rx="1"/><rect x="3" y="13" width="6" height="7" rx="1"/><rect x="11" y="13" width="10" height="7" rx="1"/>',
    grid: '<rect x="4" y="4" width="7" height="7" rx="1"/><rect x="13" y="4" width="7" height="7" rx="1"/><rect x="4" y="13" width="7" height="7" rx="1"/><rect x="13" y="13" width="7" height="7" rx="1"/>',
    list: '<path d="M8 6h12M8 12h12M8 18h12M4 6h.01M4 12h.01M4 18h.01"/>',
    gear: '<circle cx="12" cy="12" r="3"/><path d="M12 3v2.5M12 18.5V21M3 12h2.5M18.5 12H21M5.6 5.6l1.8 1.8M16.6 16.6l1.8 1.8M5.6 18.4l1.8-1.8M16.6 7.4l1.8-1.8"/>',
    close: '<path d="M6 6l12 12M18 6 6 18"/>',
    info: '<circle cx="12" cy="12" r="9"/><path d="M12 11v5M12 8h.01"/>',
    play: '<path d="M8 5v14l11-7z" fill="currentColor"/>',
    pause: '<path d="M8 5v14M16 5v14"/>',
    eye: '<path d="M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12z"/><circle cx="12" cy="12" r="3"/>',
    asc: '<path d="M12 19V5m0 0-5 5m5-5 5 5"/>',
    desc: '<path d="M12 5v14m0 0-5-5m5 5 5-5"/>',
    expand: '<path d="M4 9V4h5M20 9V4h-5M4 15v5h5M20 15v5h-5"/>',
    external: '<path d="M14 4h6v6M20 4l-9 9M18 14v5H5V6h5"/>',
    check: '<path d="m5 12 5 5 9-10"/>',
    plus: '<path d="M12 5v14M5 12h14"/>',
    filter: '<path d="M4 5h16l-6 7v6l-4 2v-8z"/>',
    split: '<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M12 4v16"/>',
    minus: '<path d="M5 12h14"/>',
    size: '<rect x="4" y="4" width="16" height="16" rx="2"/><path d="m4 14 5-5 4 4 3-3 4 4"/>',
  };
  ICONS.desktop = ICONS.desktop;
  ICONS.documents = ICONS.doc;
  ICONS.pictures = ICONS.image;
  ICONS.videos = ICONS.video;
  ICONS.music = ICONS.audio;
  ICONS.drive = ICONS.root;
  ICONS.download = ICONS.downloads;
  ICONS.text = ICONS.doc;
  ICONS.chevron = ICONS.forward;

  const icon = (name, size = 18) =>
    `<svg class="ic" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">${ICONS[name] || ICONS.other}</svg>`;

  async function request(url, options) {
    const res = await fetch(url, options);
    if (!res.ok) {
      let message = res.statusText || 'request failed';
      try {
        message = (await res.json()).error || message;
      } catch {}
      const err = new Error(message);
      err.status = res.status;
      throw err;
    }
    return res;
  }
  const getJSON = async (url) => (await request(url)).json();
  const sendJSON = async (url, body, method = 'POST') =>
    (await request(url, { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })).json();
  const fileURL = (p) => '/api/file?' + qs({ path: p });
  const thumbURL = (p, w) => '/api/thumb?' + qs({ path: p, w });

  function fmtSize(n) {
    if (!n) return '0 B';
    if (n < 1024) return n + ' B';
    const units = ['KB', 'MB', 'GB', 'TB'];
    let i = -1;
    do {
      n /= 1024;
      i++;
    } while (n >= 1024 && i < units.length - 1);
    return (n >= 100 ? n.toFixed(0) : n >= 10 ? n.toFixed(1) : n.toFixed(2)) + ' ' + units[i];
  }
  const dateFmt = new Intl.DateTimeFormat(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
  const timeFmt = new Intl.DateTimeFormat(undefined, { hour: '2-digit', minute: '2-digit' });
  const fmtDate = (ms) => (ms ? dateFmt.format(ms) + ', ' + timeFmt.format(ms) : '');
  function fmtDuration(s) {
    s = Math.round(s);
    const h = Math.floor(s / 3600);
    const m = Math.floor((s % 3600) / 60);
    const sec = String(s % 60).padStart(2, '0');
    return h ? `${h}:${String(m).padStart(2, '0')}:${sec}` : `${m}:${sec}`;
  }
  const typeLabel = (e) => (e.kind === 'folder' ? 'Folder' : e.ext ? e.ext.toUpperCase() : e.kind[0].toUpperCase() + e.kind.slice(1));
  const metaOf = (e) => {
    if (e.kind === 'folder') return '';
    if (e.w && e.h) return `${e.w} × ${e.h} · ${fmtSize(e.size)}`;
    return fmtSize(e.size);
  };

  const FILTERS = {
    all: null,
    media: ['image', 'video'],
    image: ['image'],
    video: ['video'],
    audio: ['audio'],
    docs: ['doc', 'text', 'code'],
    code: ['code'],
    text: ['text'],
    archives: ['archive'],
    folders: [],
  };
  const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' });
  const sorters = {
    name: (a, b) => collator.compare(a.name, b.name),
    size: (a, b) => a.size - b.size,
    date: (a, b) => a.mtime - b.mtime,
    type: (a, b) => collator.compare(a.ext || '', b.ext || ''),
    dims: (a, b) => (a.w * a.h || 0) - (b.w * b.h || 0),
  };

  const state = {
    mode: 'dir',
    path: '',
    parent: '',
    entries: [],
    shown: [],
    els: [],
    selection: new Set(),
    anchor: -1,
    cursor: -1,
    view: store.get('view', 'gallery'),
    size: store.get('size', 300),
    sortKey: store.get('sortKey', 'name'),
    sortDir: store.get('sortDir', 'asc'),
    filter: store.get('filter', 'all'),
    hidden: store.get('hidden', false),
    singleClick: store.get('single', false),
    query: '',
    scope: 'folder',
    inFiles: false,
    searchInfo: null,
    writable: true,
    free: 0,
    total: 0,
    meta: null,
    home: '',
    places: null,
    clipboard: null,
    drag: null,
    pendingSelect: null,
    pendingView: null,
    navId: 0,
  };

  const content = $('#content');
  const searchInput = $('#search');
  const scrolls = new Map();
  let typeBuffer = '';
  let typeTimer = 0;
  let renderFrame = 0;
  const tabs = { list: [], active: 0 };
  const other = { path: null };
  let split = false;
  let chan = null;

  function toast(message, kind = '') {
    const node = document.createElement('div');
    node.className = 'toast ' + kind;
    node.textContent = message;
    $('#toasts').append(node);
    setTimeout(() => node.remove(), kind === 'error' ? 6000 : 3200);
  }

  function pathToHash(p) {
    if (p === 'trash:') return '#trash';
    return '#' + p.split('/').map(encodeURIComponent).join('/');
  }

  function hashPath() {
    try {
      return decodeURIComponent(location.hash.slice(1));
    } catch {
      return '';
    }
  }

  function navigate(p) {
    const hash = pathToHash(p);
    if (location.hash === hash) refresh();
    else location.hash = hash;
  }

  function route() {
    closeMenu();
    scrolls.set(state.mode === 'trash' ? 'trash' : state.path, content.scrollTop);
    searchInput.value = '';
    state.query = '';
    const raw = hashPath();
    if (raw === 'trash') return loadTrash();
    return loadDir(raw || store.get('last', '') || state.home);
  }

  async function loadDir(path, keepScroll) {
    const id = ++state.navId;
    try {
      const data = await getJSON('/api/list?' + qs({ path, hidden: state.hidden ? 1 : 0 }));
      if (id !== state.navId) return;
      state.mode = 'dir';
      state.path = data.path;
      state.parent = data.parent;
      state.entries = data.entries;
      state.writable = data.writable;
      state.free = data.free;
      state.total = data.total;
      state.searchInfo = null;
      if (!embedded) store.set('last', data.path);
      afterLoad(keepScroll);
    } catch (err) {
      if (id !== state.navId) return;
      toast(err.message, 'error');
      if (!state.path && path !== state.home) loadDir(state.home);
    }
  }

  async function loadTrash(keepScroll) {
    const id = ++state.navId;
    try {
      const data = await getJSON('/api/trash');
      if (id !== state.navId) return;
      state.mode = 'trash';
      state.path = '';
      state.parent = '';
      state.entries = data.entries;
      state.writable = !state.meta.readOnly;
      state.searchInfo = null;
      afterLoad(keepScroll);
    } catch (err) {
      toast(err.message, 'error');
    }
  }

  async function runSearch(keepScroll) {
    const text = state.query.trim();
    if (!text) {
      state.query = '';
      if (state.mode === 'search') await refresh();
      return;
    }
    const id = ++state.navId;
    $('#st-mid').textContent = state.scope === 'system' ? 'Scanning the whole system…' : 'Searching…';
    try {
      const data = await getJSON(
        '/api/search?' + qs({ q: text, path: state.path || state.home, scope: state.scope, content: state.inFiles ? 1 : 0, hidden: state.hidden ? 1 : 0 })
      );
      if (id !== state.navId) return;
      state.mode = 'search';
      state.entries = data.results;
      state.searchInfo = data;
      afterLoad(keepScroll);
    } catch (err) {
      if (id !== state.navId) return;
      toast(err.message, 'error');
    }
  }

  function refresh() {
    const top = content.scrollTop;
    const keep = true;
    let job;
    if (state.mode === 'search' && state.query.trim()) job = runSearch(keep);
    else if (state.mode === 'trash') job = loadTrash(keep);
    else job = loadDir(state.path, keep);
    return job.then(() => {
      content.scrollTop = top;
    });
  }

  function afterLoad(keepScroll) {
    const alive = new Set(state.entries.map((e) => e.path));
    if (state.pendingSelect) {
      state.selection = new Set([state.pendingSelect]);
    } else {
      state.selection = new Set([...state.selection].filter((p) => alive.has(p)));
    }
    render();
    if (!keepScroll) content.scrollTop = scrolls.get(state.mode === 'trash' ? 'trash' : state.path) || 0;
    if (state.pendingSelect) {
      const i = state.shown.findIndex((e) => e.path === state.pendingSelect);
      if (i >= 0) {
        state.anchor = state.cursor = i;
        state.els[i]?.scrollIntoView({ block: 'center' });
      }
      state.pendingSelect = null;
    }
    renderCrumbs();
    markActivePlace();
    if (state.mode === 'dir') content.dataset.drop = state.path;
    else delete content.dataset.drop;
    syncTab();
    announce();
    updateToolbar();
    updateStatus();
    document.title = (state.mode === 'trash' ? 'Trash' : state.path.split('/').filter(Boolean).pop() || '/') + ' — Lumen';
    if (state.pendingView) {
      const target = state.entries.find((e) => e.path === state.pendingView);
      state.pendingView = null;
      if (target) openEntry(target);
    }
  }

  function visibleItems() {
    let items = state.entries;
    const allowed = FILTERS[state.filter];
    if (allowed) {
      const keepFolders = state.mode !== 'search' && state.filter !== 'folders';
      items = items.filter((e) => (e.kind === 'folder' ? state.filter === 'folders' || keepFolders : allowed.includes(e.kind)));
    }
    if (state.filter === 'folders') items = items.filter((e) => e.kind === 'folder');
    const cmp = sorters[state.sortKey] || sorters.name;
    const dir = state.sortDir === 'asc' ? 1 : -1;
    return items.slice().sort((a, b) => {
      const fa = a.kind === 'folder';
      const fb = b.kind === 'folder';
      if (fa !== fb) return fa ? -1 : 1;
      return dir * cmp(a, b) || collator.compare(a.name, b.name);
    });
  }

  const hasThumb = (e) => e.kind === 'image' || e.kind === 'video' || e.ext === 'pdf';
  function thumbSrc(e, width) {
    if (e.ext === 'svg' || e.ext === 'ico') return fileURL(e.path);
    return thumbURL(e.path, Math.ceil(width * (window.devicePixelRatio || 1)));
  }
  function aspectOf(e) {
    if (e.kind === 'image' && e.w && e.h) return clamp(e.w / e.h, 0.45, 3.6);
    if (e.kind === 'image') return 1.4;
    if (e.kind === 'video') return 16 / 9;
    if (e.ext === 'pdf') return 0.78;
    return 1.15;
  }

  const io = new IntersectionObserver(
    (records) => {
      for (const rec of records) {
        if (!rec.isIntersecting) continue;
        io.unobserve(rec.target);
        hydrate(rec.target);
      }
    },
    { root: content, rootMargin: '900px 0px' }
  );

  function hydrate(node) {
    if (node.tagName === 'IMG') {
      node.src = node.dataset.src;
      return;
    }
    if (node.classList.contains('folder')) loadFolderPreview(node);
  }

  async function loadFolderPreview(card) {
    const prev = $('.fprev', card);
    try {
      const data = await getJSON('/api/preview?' + qs({ path: card.dataset.path }));
      const found = data.images.slice(0, 3);
      const count = $('.mt', card);
      if (count && data.count) count.textContent = data.count + (data.count === 1 ? ' item' : ' items');
      if (!found.length || !card.isConnected) return;
      prev.innerHTML = found.map((p) => `<img alt="" src="${esc(thumbURL(p, Math.ceil(260 * (window.devicePixelRatio || 1))))}">`).join('');
      prev.classList.add('has');
    } catch {}
  }

  function justify(files, width, target, gap) {
    const rows = [];
    let row = [];
    let sum = 0;
    for (const f of files) {
      const a = aspectOf(f.e);
      row.push({ ...f, a });
      sum += a;
      if (sum * target + gap * (row.length - 1) >= width) {
        rows.push({ items: row, h: (width - gap * (row.length - 1)) / sum });
        row = [];
        sum = 0;
      }
    }
    if (row.length) rows.push({ items: row, h: Math.min(target, (width - gap * (row.length - 1)) / sum) });
    return rows;
  }

  function placeholder(e, size = 34) {
    const themed = window.LumenExtras && LumenExtras.placeholder(e, size);
    if (themed) return themed;
    return `<div class="ph">${icon(e.kind === 'other' || e.kind === 'archive' || e.kind === 'doc' ? e.kind : e.kind, size)}${size > 30 && e.ext ? `<span>${esc(e.ext.toUpperCase())}</span>` : ''}</div>`;
  }

  function thumbHTML(e, width, iconSize) {
    return placeholder(e, iconSize) + (hasThumb(e) ? `<img class="lazy" alt="" draggable="false" decoding="async" data-src="${esc(thumbSrc(e, width))}">` : '');
  }

  const checkHTML = `<button class="chk" tabindex="-1" type="button">${icon('check', 13)}</button>`;

  function make(html) {
    const t = document.createElement('template');
    t.innerHTML = html.trim();
    return t.content.firstChild;
  }

  function decorate(node, e, i) {
    node.dataset.i = i;
    node.dataset.path = e.path;
    if (state.mode === 'search') node.title = e.path;
    if (state.mode !== 'trash' && !state.meta.readOnly) node.draggable = true;
    if (e.kind === 'folder' && state.mode !== 'trash') node.dataset.drop = e.path;
    if (state.selection.has(e.path)) node.classList.add('sel');
    state.els[i] = node;
    return node;
  }

  function folderCard(e, i) {
    const node = make(`<div class="item folder"><div class="fprev"></div><div class="fbody">${icon('folder', 20)}<span class="nm">${esc(e.name)}</span><span class="mt"></span></div>${checkHTML}</div>`);
    io.observe(node);
    return decorate(node, e, i);
  }

  function galleryTile(e, i, w, h) {
    const badge = e.kind === 'video' ? `<span class="badge-play">${icon('play', 12)}</span>` : '';
    const node = make(
      `<div class="item k-${e.kind}" style="width:${Math.floor(w)}px;height:${Math.floor(h)}px"><div class="thumb">${thumbHTML(e, w, 40)}</div>${badge}${checkHTML}<div class="cap"><span class="nm">${esc(e.name)}</span><span class="mt">${esc(metaOf(e))}</span></div></div>`
    );
    const img = $('img.lazy', node);
    if (img) io.observe(img);
    return decorate(node, e, i);
  }

  function gridTile(e, i, w) {
    const badge = e.kind === 'video' ? `<span class="badge-play">${icon('play', 12)}</span>` : '';
    const node = make(
      `<div class="item k-${e.kind}"><div class="tile"><div class="thumb">${thumbHTML(e, w, 40)}</div>${badge}${checkHTML}</div><div class="cap2"><span class="nm">${esc(e.name)}</span><span class="mt">${esc(metaOf(e))}</span></div></div>`
    );
    const img = $('img.lazy', node);
    if (img) io.observe(img);
    return decorate(node, e, i);
  }

  function listRow(e, i) {
    let sub = '';
    if (state.mode === 'search') sub = e.path.replace(/\/[^/]*$/, '') || '/';
    if (state.mode === 'trash' && e.original) sub = e.original;
    const dims = e.w && e.h ? `${e.w} × ${e.h}` : '';
    const node = make(
      `<div class="item lrow k-${e.kind}"><div class="c-name"><span class="lthumb">${thumbHTML(e, 34, 18)}</span><span class="lt"><span class="nm">${esc(e.name)}</span>${sub ? `<span class="sub">${esc(sub)}</span>` : ''}</span></div><div class="c-size">${e.kind === 'folder' ? '' : fmtSize(e.size)}</div><div class="c-date">${fmtDate(e.mtime)}</div><div class="c-type">${esc(typeLabel(e))}</div><div class="c-dim">${dims}</div></div>`
    );
    const img = $('img.lazy', node);
    if (img) io.observe(img);
    return decorate(node, e, i);
  }

  function emptyState() {
    const wrap = document.createElement('div');
    wrap.className = 'empty';
    let text = 'This folder is empty';
    let ic = 'folder';
    if (state.mode === 'search') {
      text = 'Nothing matched your search';
      ic = 'search';
    } else if (state.mode === 'trash') {
      text = 'The trash is empty';
      ic = 'trash';
    } else if (state.entries.length) {
      text = 'Nothing to show with the current filter';
    }
    wrap.innerHTML = `${icon(ic, 44)}<p>${text}</p>`;
    return wrap;
  }

  function render() {
    io.disconnect();
    const items = visibleItems();
    state.shown = items;
    state.els = new Array(items.length);
    state.cursor = Math.min(state.cursor, items.length - 1);
    content.replaceChildren();
    if (!items.length) {
      content.append(emptyState());
      return;
    }
    const frag = document.createDocumentFragment();
    const indexed = items.map((e, i) => ({ e, i }));
    if (state.view === 'list') {
      const list = document.createElement('div');
      list.className = 'list';
      const arrow = state.sortDir === 'asc' ? ' ↑' : ' ↓';
      const head = (key, label) => `<span data-sort="${key}" class="${state.sortKey === key ? 'on' : ''}">${label}${state.sortKey === key ? arrow : ''}</span>`;
      list.innerHTML = `<div class="lhead">${head('name', 'Name')}${head('size', 'Size')}${head('date', 'Modified')}${head('type', 'Type')}${head('dims', 'Dimensions')}</div>`;
      for (const { e, i } of indexed) list.append(listRow(e, i));
      frag.append(list);
    } else {
      const folders = indexed.filter((x) => x.e.kind === 'folder');
      const files = indexed.filter((x) => x.e.kind !== 'folder');
      if (folders.length) {
        const box = document.createElement('div');
        box.className = 'folders';
        for (const { e, i } of folders) box.append(folderCard(e, i));
        frag.append(box);
      }
      if (files.length && state.view === 'gallery') {
        const rowsBox = document.createElement('div');
        rowsBox.className = 'rows';
        const width = Math.max(300, content.clientWidth - 24);
        for (const row of justify(files, width, state.size, 6)) {
          const r = document.createElement('div');
          r.className = 'row';
          for (const it of row.items) r.append(galleryTile(it.e, it.i, it.a * row.h, row.h));
          rowsBox.append(r);
        }
        frag.append(rowsBox);
      } else if (files.length) {
        const grid = document.createElement('div');
        grid.className = 'grid';
        grid.style.gridTemplateColumns = `repeat(auto-fill, minmax(${Math.round(state.size * 0.62)}px, 1fr))`;
        for (const { e, i } of files) grid.append(gridTile(e, i, state.size * 0.7));
        frag.append(grid);
      }
    }
    content.append(frag);
  }

  function scheduleRender() {
    cancelAnimationFrame(renderFrame);
    renderFrame = requestAnimationFrame(() => {
      const top = content.scrollTop;
      render();
      content.scrollTop = top;
      updateStatus();
    });
  }

  function renderCrumbs() {
    const box = $('#crumbs');
    if (state.mode === 'trash') {
      box.innerHTML = `<span class="crumb cur">${icon('trash', 16)} Trash</span>`;
      return;
    }
    const parts = state.path.split('/').filter(Boolean);
    let acc = '';
    let html = `<a class="crumb" href="${pathToHash('/')}" data-drop="/">${icon('root', 16)}</a>`;
    parts.forEach((part, i) => {
      acc += '/' + part;
      html += `<span class="sep">/</span><a class="crumb${i === parts.length - 1 && state.mode !== 'search' ? ' cur' : ''}" href="${pathToHash(acc)}" data-drop="${esc(acc)}">${esc(part)}</a>`;
    });
    if (state.mode === 'search') html += `<span class="sep">/</span><span class="crumb tag">${icon('search', 14)} ${esc(state.query)}</span>`;
    box.innerHTML = html;
    box.scrollLeft = box.scrollWidth;
  }

  function editPath() {
    const box = $('#crumbs');
    const input = document.createElement('input');
    input.value = state.mode === 'trash' ? '' : state.path;
    box.replaceChildren(input);
    input.focus();
    input.select();
    const done = (go) => {
      if (go && input.value.trim()) navigate(input.value.trim());
      else renderCrumbs();
    };
    input.addEventListener('keydown', (ev) => {
      if (ev.key === 'Enter') done(true);
      else if (ev.key === 'Escape') done(false);
      ev.stopPropagation();
    });
    input.addEventListener('blur', () => done(false));
  }

  async function loadPlaces() {
    state.places = await getJSON('/api/places');
    renderPlaces();
  }

  function renderPlaces() {
    const d = state.places;
    if (!d) return;
    const link = (p, cls = '') =>
      `<a class="pl ${cls}" href="${pathToHash(p.path)}" data-path="${esc(p.path)}" data-drop="${esc(p.path)}">${icon(p.icon || 'folder')}<span>${esc(p.name)}</span></a>`;
    let html = `<div class="group">Places</div>` + d.places.map((p) => link(p)).join('');
    if (d.favorites.length) html += `<div class="group">Favorites</div>` + d.favorites.map((p) => link({ ...p, icon: 'star' }, 'fav')).join('');
    if (d.drives.length) {
      html += `<div class="group">Drives</div>`;
      html += d.drives
        .map((dr) => {
          const used = dr.total ? Math.round((1 - dr.free / dr.total) * 100) : 0;
          return `<a class="pl drive" href="${pathToHash(dr.path)}" data-path="${esc(dr.path)}" data-drop="${esc(dr.path)}" title="${fmtSize(dr.free)} free of ${fmtSize(dr.total)}">${icon('drive')}<span>${esc(dr.name)}</span><div class="bar"><i style="width:${used}%"></i></div></a>`;
        })
        .join('');
    }
    html += `<div class="group">Other</div><a class="pl" href="#trash" data-path="trash:">${icon('trash')}<span>Trash</span>${d.trashCount ? `<span class="badge">${d.trashCount}</span>` : ''}</a>`;
    $('#places').innerHTML = html;
    markActivePlace();
  }

  function markActivePlace() {
    $$('#places .pl').forEach((a) => {
      const p = a.dataset.path;
      a.classList.toggle('active', state.mode === 'trash' ? p === 'trash:' : p === state.path);
    });
  }

  function updateToolbar() {
    $$('#views button').forEach((b) => b.classList.toggle('on', b.dataset.view === state.view));
    $('#sortkey').value = state.sortKey;
    $('#filter').value = state.filter;
    $('#sortdir').innerHTML = icon(state.sortDir === 'asc' ? 'asc' : 'desc');
    $('#btn-hidden').classList.toggle('on', state.hidden);
    $('#size').value = state.size;
    $('#size').disabled = state.view === 'list';
    $('#scope').textContent = { folder: 'This folder', everywhere: 'Everywhere', system: 'Whole system' }[state.scope];
    $('#scope').classList.toggle('wide', state.scope !== 'folder');
    $('#btn-up').disabled = !state.parent || state.mode === 'trash';
    $('#btn-newfolder').disabled = state.mode !== 'dir' || !state.writable;
    $('#btn-newfile').disabled = state.mode !== 'dir' || !state.writable;
    $('#btn-split').classList.toggle('on', split || embedded);
    const sendable = (split || embedded) && !!other.path && state.mode !== 'trash';
    $('#btn-copyto').hidden = !sendable;
    $('#btn-moveto').hidden = !sendable;
    const arrow = paneId === 'a' ? '→' : '←';
    $('#btn-copyto').textContent = 'Copy ' + arrow;
    $('#btn-moveto').textContent = 'Move ' + arrow;
    $('#btn-home').hidden = !embedded;
  }

  function selectedEntries() {
    return state.shown.filter((e) => state.selection.has(e.path));
  }

  function updateStatus() {
    const total = state.shown.length;
    const folders = state.shown.filter((e) => e.kind === 'folder').length;
    const sel = selectedEntries();
    let left = `${total} ${total === 1 ? 'item' : 'items'}`;
    if (folders && folders !== total) left += ` · ${folders} ${folders === 1 ? 'folder' : 'folders'}`;
    if (sel.length) {
      const bytes = sel.reduce((n, e) => n + (e.kind === 'folder' ? 0 : e.size), 0);
      left += ` · ${sel.length} selected${bytes ? ' (' + fmtSize(bytes) + ')' : ''}`;
    }
    $('#st-left').textContent = left;
    let mid = '';
    const info = state.searchInfo;
    if (state.mode === 'search' && info) {
      mid = `${info.total}${info.truncated ? '+' : ''} results in ${info.tookMs} ms${info.live ? ' · live scan' : ''}`;
      if (info.indexing) mid += ' · index still building';
    }
    $('#st-mid').textContent = mid;
    const ix = state.meta?.index;
    let right = '';
    if (state.mode === 'dir' && state.total) right = `${fmtSize(state.free)} free`;
    if (ix) right += (right ? ' · ' : '') + (ix.building ? 'indexing…' : `${ix.count.toLocaleString()} indexed`);
    if (state.meta?.readOnly) right = 'read-only · ' + right;
    $('#st-right').textContent = right;
  }

  async function pollIndex() {
    for (let i = 0; i < 200; i++) {
      try {
        state.meta = await getJSON('/api/meta');
      } catch {
        return;
      }
      updateStatus();
      if (!state.meta.index.building) return;
      await new Promise((r) => setTimeout(r, 2000));
    }
  }

  function paintSelection() {
    state.els.forEach((el, i) => el && el.classList.toggle('sel', state.selection.has(state.shown[i].path)));
    updateStatus();
  }
  function selectOnly(i) {
    state.selection = new Set([state.shown[i].path]);
    state.anchor = state.cursor = i;
    paintSelection();
  }
  function toggleSel(i) {
    const p = state.shown[i].path;
    if (state.selection.has(p)) state.selection.delete(p);
    else state.selection.add(p);
    state.anchor = state.cursor = i;
    paintSelection();
  }
  function rangeSel(i) {
    const from = state.anchor < 0 ? 0 : state.anchor;
    const [lo, hi] = from < i ? [from, i] : [i, from];
    state.selection = new Set(state.shown.slice(lo, hi + 1).map((e) => e.path));
    state.cursor = i;
    paintSelection();
  }
  function clearSelection() {
    if (!state.selection.size) return;
    state.selection.clear();
    paintSelection();
  }
  function selectAll() {
    state.selection = new Set(state.shown.map((e) => e.path));
    paintSelection();
  }

  const canView = (e) => ['image', 'video', 'audio', 'text', 'code'].includes(e.kind) || e.ext === 'pdf' || e.sniffed === true;

  function openEntry(e) {
    if (window.LumenExtras && LumenExtras.openEntry(e)) return;
    if (e.kind === 'folder') navigate(e.path);
    else if (canView(e)) openViewer(e);
    else if (e.kind === 'other') openUnknown(e);
    else openExternal(e.path);
  }

  async function openUnknown(e) {
    let sniff;
    try {
      sniff = await getJSON('/api/sniff?' + qs({ path: e.path }));
    } catch {
      return openExternal(e.path);
    }
    if (sniff.text) {
      e.sniffed = true;
      return openViewer(e);
    }
    if (sniff.exec) return runProgram(e);
    openExternal(e.path);
  }

  async function runProgram(e, chmod = false) {
    if (!chmod) {
      const ok = await ask({ title: 'Run program', message: `Run ${e.name}? Only run programs you trust.`, confirm: 'Run' });
      if (ok === null) return;
    }
    try {
      await sendJSON('/api/run', { path: e.path, chmod });
      toast('Started ' + e.name);
    } catch (err) {
      if (err.status !== 409) return toast(err.message, 'error');
      const ok = await ask({ title: 'Make executable?', message: `${e.name} is not marked as executable. Make it executable and run it?`, confirm: 'Make executable and run' });
      if (ok !== null) runProgram(e, true);
    }
  }

  async function openExternal(path, mode = 'open') {
    try {
      await sendJSON('/api/open', { path, mode });
    } catch (err) {
      toast(err.message, 'error');
    }
  }

  function revealInApp(e) {
    state.pendingSelect = e.path;
    navigate(e.path.replace(/\/[^/]*$/, '') || '/');
  }

  const dlg = $('#dialog');

  function ask({ title, message = '', value = null, confirm = 'OK', danger = false }) {
    return new Promise((resolve) => {
      dlg.innerHTML = `<form method="dialog" class="dlg"><h3>${esc(title)}</h3>${message ? `<p>${esc(message)}</p>` : ''}${value !== null ? `<input type="text" name="v" value="${esc(value)}" autocomplete="off" spellcheck="false">` : ''}<div class="dlg-actions"><button type="button" class="btn" data-close>Cancel</button><button class="btn ${danger ? 'danger' : 'primary'}" value="ok">${esc(confirm)}</button></div></form>`;
      dlg.returnValue = '';
      dlg.addEventListener(
        'close',
        () => {
          if (dlg.returnValue !== 'ok') return resolve(null);
          const input = $('input', dlg);
          resolve(input ? input.value : true);
        },
        { once: true }
      );
      dlg.showModal();
      const input = $('input', dlg);
      if (input) {
        input.focus();
        const dot = input.value.lastIndexOf('.');
        input.setSelectionRange(0, dot > 0 ? dot : input.value.length);
      }
    });
  }

  dlg.addEventListener('click', (ev) => {
    if (ev.target.closest('[data-close]')) dlg.close('cancel');
    else if (ev.target === dlg) dlg.close('cancel');
  });

  async function runOp(body) {
    try {
      const res = await sendJSON('/api/op', body);
      const failed = res.results.filter((r) => r.error);
      if (failed.length) toast(failed.length === 1 ? failed[0].error : `${failed.length} items failed: ${failed[0].error}`, 'error');
      return res;
    } catch (err) {
      toast(err.message, 'error');
      return null;
    }
  }

  async function afterOp() {
    await refresh();
    loadPlaces().catch(() => {});
    if (chan) chan.postMessage({ type: 'refresh' });
  }

  async function renameEntry(e) {
    const name = await ask({ title: 'Rename', value: e.name, confirm: 'Rename' });
    if (name === null || name === e.name) return;
    const res = await runOp({ action: 'rename', path: e.path, name });
    if (res && res.results[0] && !res.results[0].error) state.pendingSelect = res.results[0].to;
    if (res) afterOp();
  }

  async function newFolder() {
    if (state.mode !== 'dir') return;
    const name = await ask({ title: 'New folder', value: 'New folder', confirm: 'Create' });
    if (name === null) return;
    const res = await runOp({ action: 'mkdir', path: state.path, name });
    if (res && res.results[0] && !res.results[0].error) state.pendingSelect = res.results[0].to;
    if (res) afterOp();
  }

  const ARCHIVE_RE = /\.(zip|jar|apk|7z|rar|iso|tar|tar\.gz|tgz|tar\.bz2|tbz2?|tar\.xz|txz|gz|bz2|xz)$/i;
  const isArchive = (e) => e.kind !== 'folder' && ARCHIVE_RE.test(e.name);
  const pathLeaf = (p) => p.slice(p.lastIndexOf('/') + 1);

  function formDialog({ title, body, confirm = 'OK', wide = false }) {
    return new Promise((resolve) => {
      dlg.innerHTML = `<form method="dialog" class="dlg${wide ? ' wide' : ''}"><h3>${esc(title)}</h3>${body}<div class="dlg-actions"><button type="button" class="btn" data-close>Cancel</button><button class="btn primary" value="ok">${esc(confirm)}</button></div></form>`;
      dlg.returnValue = '';
      dlg.addEventListener(
        'close',
        () => {
          if (dlg.returnValue !== 'ok') return resolve(null);
          resolve(Object.fromEntries(new FormData($('form', dlg))));
        },
        { once: true }
      );
      dlg.showModal();
      const first = $('input[type=text], input[type=password]', dlg);
      if (first) {
        first.focus();
        first.select();
      }
    });
  }

  function copyText(text) {
    const fallback = () => {
      const area = document.createElement('textarea');
      area.value = text;
      document.body.append(area);
      area.select();
      document.execCommand('copy');
      area.remove();
      toast('Copied to the clipboard');
    };
    if (!navigator.clipboard) return fallback();
    navigator.clipboard.writeText(text).then(() => toast('Copied to the clipboard'), fallback);
  }

  async function openTerminal(path) {
    try {
      await sendJSON('/api/terminal', { path });
    } catch (err) {
      toast(err.message, 'error');
    }
  }

  async function newFile() {
    if (state.mode !== 'dir') return;
    const name = await ask({ title: 'New file', value: 'untitled.txt', confirm: 'Create' });
    if (name === null) return;
    const res = await runOp({ action: 'newfile', path: state.path, name });
    if (res && res.results[0] && !res.results[0].error) state.pendingSelect = res.results[0].to;
    if (res) afterOp();
  }

  async function compressSelected() {
    const sel = selectedEntries();
    if (!sel.length) return;
    let base = state.path.split('/').filter(Boolean).pop() || 'archive';
    if (sel.length === 1) base = sel[0].kind === 'folder' ? sel[0].name : sel[0].name.replace(/\.[^.]*$/, '') || sel[0].name;
    const data = await formDialog({
      title: 'Compress',
      confirm: 'Compress',
      body: `<label>Archive name<input type="text" name="name" value="${esc(base)}" autocomplete="off" spellcheck="false"></label><label>Format<select name="format"><option value="zip">ZIP (.zip), works everywhere</option><option value="tgz">TAR.GZ (.tar.gz)</option><option value="txz">TAR.XZ (.tar.xz), smaller</option><option value="7z">7-Zip (.7z), smallest</option></select></label>`,
    });
    if (!data || !data.name.trim()) return;
    toast('Compressing…');
    const res = await runOp({ action: 'compress', paths: sel.map((e) => e.path), name: data.name.trim(), format: data.format });
    if (res && res.results[0] && !res.results[0].error) {
      state.pendingSelect = res.results[0].to;
      toast('Created ' + pathLeaf(res.results[0].to));
    }
    if (res) afterOp();
  }

  async function extractSelected(sel) {
    toast('Extracting…');
    const res = await runOp({ action: 'extract', paths: sel.map((e) => e.path) });
    const done = res ? res.results.filter((r) => !r.error) : [];
    if (done.length) {
      state.pendingSelect = done[0].to;
      toast(done.length === 1 ? 'Extracted to ' + pathLeaf(done[0].to) : `Extracted ${done.length} archives`);
    }
    if (res) afterOp();
  }

  async function duplicateSelected(sel) {
    const res = await runOp({ action: 'duplicate', paths: sel.map((e) => e.path) });
    if (res && res.results[0] && !res.results[0].error) state.pendingSelect = res.results[0].to;
    if (res) afterOp();
  }

  async function createLink(e) {
    const name = await ask({ title: 'Create link', value: e.name + ' link', confirm: 'Create' });
    if (name === null) return;
    const res = await runOp({ action: 'symlink', path: e.path, name });
    if (res && res.results[0] && !res.results[0].error) state.pendingSelect = res.results[0].to;
    if (res) afterOp();
  }

  async function bulkRename(sel) {
    const data = await formDialog({
      title: `Rename ${sel.length} items`,
      confirm: 'Rename',
      body: `<label>New name<input type="text" name="name" value="{name}-{n}" autocomplete="off" spellcheck="false"></label><p class="sp-hint">Use <code>{name}</code> for the current name, <code>{n}</code> for a counter and <code>{ext}</code> for the extension. The extension is kept unless you use <code>{ext}</code>.</p><label>Or replace text in the names<input type="text" name="find" placeholder="Find" autocomplete="off" spellcheck="false"></label><label><input type="text" name="replace" placeholder="Replace with" autocomplete="off" spellcheck="false"></label>`,
    });
    if (!data) return;
    const res = await runOp({ action: 'bulkrename', paths: sel.map((e) => e.path), name: data.name, find: data.find, replace: data.replace });
    if (res) afterOp();
  }

  async function permissionsDialog(e) {
    let info;
    try {
      info = await getJSON('/api/info?' + qs({ path: e.path }));
    } catch (err) {
      return toast(err.message, 'error');
    }
    const bits = info.mode.slice(-9);
    const rows = ['Owner', 'Group', 'Others'];
    const grid = rows
      .map((r, i) => `<tr><td>${r}</td>${[0, 1, 2].map((j) => `<td><input type="checkbox" name="b${i * 3 + j}" value="1" ${bits[i * 3 + j] !== '-' ? 'checked' : ''}></td>`).join('')}</tr>`)
      .join('');
    const data = await formDialog({
      title: 'Permissions',
      confirm: 'Apply',
      body: `<p>${esc(e.name)}</p><table class="perm"><tr><th></th><th>Read</th><th>Write</th><th>Execute</th></tr>${grid}</table>`,
    });
    if (!data) return;
    let mode = 0;
    for (let i = 0; i < 9; i++) if (data['b' + i]) mode |= 1 << (8 - i);
    const res = await runOp({ action: 'chmod', path: e.path, mode: mode.toString(8).padStart(3, '0') });
    if (res) afterOp();
  }

  async function showChecksums(e) {
    dlg.innerHTML = `<form method="dialog" class="dlg wide"><h3>Checksums</h3><dl class="props"><dt>File</dt><dd>${esc(e.name)}</dd><dt>MD5</dt><dd id="ck-md5">calculating…</dd><dt>SHA-1</dt><dd id="ck-sha1"></dd><dt>SHA-256</dt><dd id="ck-sha256"></dd></dl><label>Compare with<input type="text" id="ck-cmp" placeholder="Paste a hash to check it" autocomplete="off" spellcheck="false"></label><div id="ck-res" class="sp-hint"></div><div class="dlg-actions"><button type="button" class="btn" data-close>Close</button></div></form>`;
    dlg.showModal();
    try {
      const d = await getJSON('/api/checksum?' + qs({ path: e.path }));
      if (!$('#ck-md5')) return;
      $('#ck-md5').textContent = d.md5;
      $('#ck-sha1').textContent = d.sha1;
      $('#ck-sha256').textContent = d.sha256;
      $('#ck-cmp').addEventListener('input', (ev) => {
        const v = ev.target.value.trim().toLowerCase();
        const res = $('#ck-res');
        if (!v) return (res.textContent = '');
        const hit = [d.md5, d.sha1, d.sha256].includes(v);
        res.textContent = hit ? '✓ The hash matches' : '✗ No match';
      });
    } catch (err) {
      if ($('#ck-md5')) $('#ck-md5').textContent = err.message;
    }
  }

  async function openWithDialog(e) {
    let info;
    try {
      info = await getJSON('/api/openwith?' + qs({ path: e.path }));
    } catch (err) {
      return toast(err.message, 'error');
    }
    const row = (a, checked) => `<label class="radio"><input type="radio" name="app" value="${esc(a.id)}" ${checked ? 'checked' : ''}> ${window.LumenExtras ? LumenExtras.appIcon(a.icon, 20) : ''}<span>${esc(a.name)}${a.id === info.default ? ' (default)' : ''}</span></label>`;
    const rec = info.recommended.map((a, i) => row(a, i === 0)).join('');
    const rest = info.others.map((a) => row(a, false)).join('');
    const data = await formDialog({
      title: 'Open with',
      confirm: 'Open',
      body: `<div class="owlist">${rec || '<p>No recommended application was found.</p>'}${rest ? `<details${rec ? '' : ' open'}><summary>Other applications</summary>${rest}</details>` : ''}</div><label class="check"><input type="checkbox" name="remember" value="1"> Always use this application for ${esc(info.mime)}</label>`,
    });
    if (!data || !data.app) return;
    try {
      await sendJSON('/api/openwith', { path: e.path, app: data.app, makeDefault: !!data.remember });
    } catch (err) {
      toast(err.message, 'error');
    }
  }

  async function trashSelected() {
    const paths = selectedEntries().map((e) => e.path);
    if (!paths.length) return;
    if (state.mode === 'trash') return deleteSelected();
    const res = await runOp({ action: 'trash', paths });
    if (res) {
      if (res.count) toast(res.count === 1 ? 'Moved to trash' : `${res.count} items moved to trash`);
      state.selection.clear();
      afterOp();
    }
  }

  async function deleteSelected() {
    const sel = selectedEntries();
    if (!sel.length) return;
    const what = sel.length === 1 ? `“${sel[0].name}”` : `${sel.length} items`;
    const ok = await ask({ title: 'Delete permanently', message: `${what} will be deleted for good. This cannot be undone.`, confirm: 'Delete', danger: true });
    if (!ok) return;
    const res = await runOp({ action: 'delete', paths: sel.map((e) => e.path) });
    if (res) {
      state.selection.clear();
      afterOp();
    }
  }

  async function restoreSelected() {
    const paths = selectedEntries().map((e) => e.path);
    if (!paths.length) return;
    const res = await runOp({ action: 'restore', paths });
    if (res) {
      toast(`${res.count} restored`);
      state.selection.clear();
      afterOp();
    }
  }

  async function emptyTrash() {
    const ok = await ask({ title: 'Empty trash', message: 'Everything in the trash will be deleted for good.', confirm: 'Empty trash', danger: true });
    if (!ok) return;
    const res = await runOp({ action: 'emptytrash' });
    if (res) {
      toast('Trash emptied');
      afterOp();
    }
  }

  function setClip(mode) {
    const paths = selectedEntries().map((e) => e.path);
    if (!paths.length) return;
    state.clipboard = { mode, paths };
    store.set('clip', state.clipboard);
    toast(`${paths.length} ${paths.length === 1 ? 'item' : 'items'} ${mode === 'copy' ? 'copied' : 'ready to move'}`);
  }

  async function paste(target = state.path) {
    const clip = store.get('clip', null) || state.clipboard;
    if (!clip || !target || state.mode === 'trash') return;
    const { mode, paths } = clip;
    const res = await runOp({ action: mode === 'copy' ? 'copy' : 'move', paths, target });
    if (res) {
      if (mode === 'move') {
        state.clipboard = null;
        store.set('clip', null);
      }
      afterOp();
    }
  }

  async function toggleFavorite(path) {
    const cfg = await getJSON('/api/config');
    const at = cfg.favorites.findIndex((f) => f.path === path);
    if (at >= 0) cfg.favorites.splice(at, 1);
    else cfg.favorites.push({ name: path.split('/').filter(Boolean).pop() || '/', path });
    await sendJSON('/api/config', cfg, 'PUT');
    await loadPlaces();
  }

  const isFavorite = (path) => !!state.places?.favorites.some((f) => f.path === path);

  function infoRows(info) {
    const rows = [];
    const add = (k, v) => v !== undefined && v !== '' && v !== null && rows.push([k, v]);
    add('Name', info.name);
    add('Type', info.kind === 'folder' ? 'Folder' : info.mime);
    if (info.kind !== 'folder') add('Size', `${fmtSize(info.size)} (${info.size.toLocaleString()} bytes)`);
    if (info.w && info.h) add('Dimensions', `${info.w} × ${info.h}${info.kind === 'image' ? ` · ${((info.w * info.h) / 1e6).toFixed(1)} MP` : ''}`);
    if (info.items !== undefined && info.kind === 'folder') add('Contains', `${info.items} items`);
    add('Location', info.dir);
    if (info.target) add('Link to', info.target);
    add('Modified', fmtDate(info.mtime));
    add('Changed', fmtDate(info.changed));
    add('Accessed', fmtDate(info.accessed));
    add('Permissions', info.mode);
    add('Owner', info.owner ? `${info.owner}${info.group ? ' : ' + info.group : ''}` : '');
    const x = info.exif;
    if (x) {
      add('Camera', [x.make, x.model].filter(Boolean).join(' '));
      add('Lens', x.lens);
      add('Taken', x.taken);
      add('Exposure', [x.exposure, x.fnumber, x.iso ? 'ISO ' + x.iso : '', x.focal].filter(Boolean).join(' · '));
    }
    const p = info.probe;
    if (p) {
      if (p.duration) add('Duration', fmtDuration(p.duration));
      add('Video', [p.videoCodec, p.fps ? p.fps.toFixed(2).replace(/\.?0+$/, '') + ' fps' : ''].filter(Boolean).join(' · '));
      add('Audio', p.audioCodec);
      if (p.bitrate) add('Bitrate', Math.round(p.bitrate / 1000) + ' kb/s');
      add('Title', p.title);
      add('Artist', p.artist);
      add('Album', p.album);
    }
    return rows;
  }
  const rowsHTML = (rows) => rows.map(([k, v]) => `<dt>${esc(k)}</dt><dd>${esc(v)}</dd>`).join('');

  async function showProperties(path) {
    try {
      const info = await getJSON('/api/info?' + qs({ path }));
      dlg.innerHTML = `<form method="dialog" class="dlg wide"><h3>Properties</h3><dl class="props">${rowsHTML(infoRows(info))}<dt id="sz-k" hidden>Total size</dt><dd id="sz-v" hidden></dd></dl><div class="dlg-actions"><button type="button" class="btn" data-close>Close</button></div></form>`;
      dlg.showModal();
      if (info.kind === 'folder') {
        $('#sz-k').hidden = false;
        $('#sz-v').hidden = false;
        $('#sz-v').textContent = 'calculating…';
        getJSON('/api/dirsize?' + qs({ path }))
          .then((d) => {
            const el = $('#sz-v');
            if (el) el.textContent = `${fmtSize(d.size)} · ${d.files.toLocaleString()} files, ${d.folders.toLocaleString()} folders${d.complete ? '' : ' (partial)'}`;
          })
          .catch(() => {});
      }
    } catch (err) {
      toast(err.message, 'error');
    }
  }

  async function openSettings() {
    const [cfg, cache, meta] = await Promise.all([getJSON('/api/config'), getJSON('/api/cache'), getJSON('/api/meta')]);
    const tool = (ok, name) => `${ok ? '✓' : '✗'} ${name}`;
    const missing = [];
    if (!meta.ffmpeg) missing.push('ffmpeg (video thumbnails and playback)');
    if (!meta.pdf) missing.push('poppler-utils (PDF thumbnails)');
    if (!meta.convert) missing.push('imagemagick (HEIC, TIFF and other formats)');
    dlg.innerHTML = `<form method="dialog" class="dlg wide"><h3>Settings</h3>
      <label>Folders to index for fast search (one per line)<textarea name="roots" rows="3" spellcheck="false">${esc(cfg.indexRoots.join('\n'))}</textarea></label>
      <label>Folder names to skip while indexing (comma separated)<input type="text" name="skip" value="${esc(cfg.skipDirs.join(', '))}" spellcheck="false"></label>
      <label>Theme<select name="theme">${window.LumenExtras ? LumenExtras.themeOptions() : ''}</select></label>
      <label class="check"><input type="checkbox" name="single" ${state.singleClick ? 'checked' : ''}> Open items with a single click</label>
      <div class="row-info"><span>Search index: ${meta.index.count.toLocaleString()} items</span><button type="button" class="btn small" id="s-reindex">Rebuild</button></div>
      <div class="row-info"><span>Thumbnail cache: ${cache.files.toLocaleString()} files, ${fmtSize(cache.bytes)}</span><button type="button" class="btn small" id="s-cache">Clear</button></div>
      <div class="row-info"><span>${tool(meta.ffmpeg, 'ffmpeg')} · ${tool(meta.ffprobe, 'ffprobe')} · ${tool(meta.pdf, 'pdftoppm')} · ${tool(meta.convert, 'imagemagick')}</span></div>
      ${missing.length ? `<div class="note">Optional tools not found: ${missing.join(', ')}.<br>Install with: sudo apt install ffmpeg poppler-utils imagemagick</div>` : ''}
      <details class="keys"><summary>Keyboard shortcuts</summary><table>
        <tr><td>Arrows</td><td>Move selection</td></tr><tr><td>Enter / Space</td><td>Open in viewer</td></tr>
        <tr><td>/ or Ctrl+F</td><td>Search</td></tr><tr><td>Ctrl+L</td><td>Edit path</td></tr>
        <tr><td>Alt+Left / Right / Up</td><td>Back / forward / parent</td></tr><tr><td>F2</td><td>Rename</td></tr>
        <tr><td>Delete</td><td>Move to trash</td></tr><tr><td>Shift+Delete</td><td>Delete permanently</td></tr>
        <tr><td>Ctrl+C / X / V</td><td>Copy / cut / paste</td></tr><tr><td>Ctrl+H</td><td>Show hidden files</td></tr>
        <tr><td>Ctrl + wheel</td><td>Resize thumbnails</td></tr><tr><td>Viewer: I, F, S</td><td>Info, fullscreen, slideshow</td></tr>
        <tr><td>F3</td><td>Split view (two panes)</td></tr><tr><td>F5 / F6 (split)</td><td>Copy / move to the other pane</td></tr>
        <tr><td>Alt+T / Alt+W</td><td>New tab / close tab</td></tr><tr><td>Alt+1…9, Alt+[ ]</td><td>Switch tabs</td></tr>
        <tr><td>Viewer: + − 0 1</td><td>Zoom, fit, 100%</td></tr>
      </table></details>
      <div class="dlg-actions"><button type="button" class="btn" data-close>Cancel</button><button class="btn primary" value="ok">Save</button></div></form>`;
    dlg.returnValue = '';
    const theme = window.LumenExtras ? LumenExtras.theme() : 'rose';
    const themePick = $('select[name=theme]', dlg);
    if (themePick && window.LumenExtras) themePick.addEventListener('change', () => LumenExtras.applyTheme(themePick.value));
    $('#s-reindex').onclick = async () => {
      await sendJSON('/api/reindex', {});
      toast('Rebuilding the search index');
      pollIndex();
    };
    $('#s-cache').onclick = async () => {
      await request('/api/cache', { method: 'DELETE' });
      toast('Thumbnail cache cleared');
      scheduleRender();
    };
    dlg.addEventListener(
      'close',
      async () => {
        if (dlg.returnValue !== 'ok') {
          if (window.LumenExtras) LumenExtras.applyTheme(theme);
          return;
        }
        if (window.LumenExtras && themePick) LumenExtras.saveTheme(themePick.value);
        const form = $('form', dlg);
        const roots = form.roots.value.split('\n').map((s) => s.trim()).filter(Boolean);
        const skip = form.skip.value.split(',').map((s) => s.trim()).filter(Boolean);
        state.singleClick = form.single.checked;
        store.set('single', state.singleClick);
        try {
          await sendJSON('/api/config', { ...cfg, indexRoots: roots, skipDirs: skip }, 'PUT');
          pollIndex();
        } catch (err) {
          toast(err.message, 'error');
        }
      },
      { once: true }
    );
    dlg.showModal();
  }

  const menu = $('#menu');

  function showMenu(x, y, items) {
    menu.innerHTML = items
      .map((it, i) => (it === '-' ? '<hr>' : `<div class="mi${it.disabled ? ' dis' : ''}${it.danger ? ' danger' : ''}" data-i="${i}"><span>${esc(it.label)}</span>${it.key ? `<kbd>${esc(it.key)}</kbd>` : ''}</div>`))
      .join('');
    menu.hidden = false;
    menu._items = items;
    const w = menu.offsetWidth;
    const h = menu.offsetHeight;
    menu.style.left = clamp(x, 6, innerWidth - w - 6) + 'px';
    menu.style.top = clamp(y, 6, innerHeight - h - 6) + 'px';
  }
  function closeMenu() {
    menu.hidden = true;
  }
  menu.addEventListener('click', (ev) => {
    const row = ev.target.closest('.mi');
    if (!row) return;
    const item = menu._items[+row.dataset.i];
    closeMenu();
    item.run();
  });
  document.addEventListener('mousedown', (ev) => {
    if (!menu.hidden && !menu.contains(ev.target)) closeMenu();
  });
  window.addEventListener('blur', closeMenu);
  content.addEventListener('scroll', closeMenu, { passive: true });

  function itemMenu() {
    const sel = selectedEntries();
    const one = sel.length === 1 ? sel[0] : null;
    const ro = state.meta.readOnly;
    const m = [];
    if (one) m.push({ label: 'Open', key: 'Enter', run: () => openEntry(one) });
    if (one && one.kind === 'folder') m.push({ label: 'Open in new tab', run: () => newTab(one.path) });
    if (one && one.kind !== 'folder') m.push({ label: 'Open with default app', run: () => openExternal(one.path) });
    if (one && one.kind !== 'folder') m.push({ label: 'Open with…', run: () => openWithDialog(one) });
    if (one && (one.kind === 'text' || one.kind === 'code')) m.push({ label: 'Edit', disabled: ro, run: () => { state.editNext = one.path; openViewer(one); } });
    if (one && one.kind === 'other') m.push({ label: 'Open as text', run: () => { one.sniffed = true; openViewer(one); } }, { label: 'Run…', run: () => runProgram(one) });
    if (one && one.kind === 'folder') m.push({ label: 'Open terminal here', run: () => openTerminal(one.path) });
    if (one && state.mode === 'search') m.push({ label: 'Open containing folder', run: () => revealInApp(one) });
    if (state.mode === 'trash') {
      m.push('-', { label: 'Restore', run: restoreSelected }, { label: 'Delete permanently', danger: true, run: deleteSelected });
      return m;
    }
    m.push('-');
    if (one) m.push({ label: 'Rename', key: 'F2', disabled: ro, run: () => renameEntry(one) });
    m.push({ label: 'Copy', key: 'Ctrl+C', run: () => setClip('copy') });
    m.push({ label: 'Cut', key: 'Ctrl+X', disabled: ro, run: () => setClip('move') });
    if ((split || embedded) && other.path) {
      m.push({ label: 'Copy to other pane', key: 'F5', disabled: ro, run: () => sendToOther('copy') });
      m.push({ label: 'Move to other pane', key: 'F6', disabled: ro, run: () => sendToOther('move') });
    }
    if (one && one.kind === 'folder') {
      m.push({ label: 'Paste into folder', disabled: ro || !(state.clipboard || store.get('clip', null)), run: () => paste(one.path) });
      m.push({ label: isFavorite(one.path) ? 'Remove from favorites' : 'Add to favorites', run: () => toggleFavorite(one.path) });
    }
    if (one && one.kind !== 'folder') m.push({ label: 'Download copy', run: () => downloadFile(one) });
    if (state.mode !== 'trash') {
      m.push('-');
      m.push({ label: sel.length > 1 ? `Compress ${sel.length} items…` : 'Compress…', disabled: ro, run: compressSelected });
      if (sel.length && sel.every(isArchive)) m.push({ label: 'Extract here', disabled: ro, run: () => extractSelected(sel) });
      m.push({ label: 'Duplicate', disabled: ro, run: () => duplicateSelected(sel) });
      if (sel.length > 1) m.push({ label: 'Bulk rename…', disabled: ro, run: () => bulkRename(sel) });
      if (one) m.push({ label: 'Create link', disabled: ro, run: () => createLink(one) });
      if (one) m.push({ label: 'Permissions…', disabled: ro, run: () => permissionsDialog(one) });
      if (one && one.kind !== 'folder') m.push({ label: 'Checksums…', run: () => showChecksums(one) });
      m.push({ label: sel.length > 1 ? 'Copy paths' : 'Copy path', run: () => copyText(sel.map((x) => x.path).join('\n')) });
      if (window.LumenExtras) {
        const extra = LumenExtras.itemMenu(sel, ro);
        if (extra.length) m.push('-', ...extra);
      }
    }
    m.push('-', { label: 'Move to trash', key: 'Del', danger: true, disabled: ro, run: trashSelected });
    if (one) m.push({ label: 'Properties', run: () => showProperties(one.path) });
    return m;
  }

  function blankMenu() {
    const ro = state.meta.readOnly;
    if (state.mode === 'trash') return [{ label: 'Empty trash', danger: true, run: emptyTrash }, { label: 'Refresh', key: 'F5', run: refresh }];
    const m = [];
    if (state.mode === 'dir') {
      m.push({ label: 'New folder', key: 'Ctrl+Shift+N', disabled: ro || !state.writable, run: newFolder });
      m.push({ label: 'New file', disabled: ro || !state.writable, run: newFile });
      m.push({ label: 'Paste', key: 'Ctrl+V', disabled: ro || !(state.clipboard || store.get('clip', null)), run: () => paste() });
      m.push('-', { label: isFavorite(state.path) ? 'Remove folder from favorites' : 'Add folder to favorites', run: () => toggleFavorite(state.path) });
      m.push({ label: 'Open terminal here', run: () => openTerminal(state.path) });
    }
    m.push({ label: 'Select all', key: 'Ctrl+A', run: selectAll }, { label: 'Refresh', key: 'Ctrl+R', run: refresh }, { label: 'New tab', key: 'Alt+T', run: () => newTab() });
    if (state.mode === 'dir') m.push('-', { label: 'Properties', run: () => showProperties(state.path) });
    return m;
  }

  function downloadFile(e) {
    const a = document.createElement('a');
    a.href = '/api/file?' + qs({ path: e.path, dl: 1 });
    a.download = e.name;
    document.body.append(a);
    a.click();
    a.remove();
  }

  content.addEventListener('click', (ev) => {
    const item = ev.target.closest('.item');
    if (!item) {
      const head = ev.target.closest('[data-sort]');
      if (head) return setSort(head.dataset.sort);
      if (!ev.target.closest('.lhead')) clearSelection();
      return;
    }
    const i = +item.dataset.i;
    if (ev.target.closest('.chk')) return toggleSel(i);
    if (ev.shiftKey) return rangeSel(i);
    if (ev.ctrlKey || ev.metaKey) return toggleSel(i);
    selectOnly(i);
    if (state.singleClick) openEntry(state.shown[i]);
  });
  content.addEventListener('dblclick', (ev) => {
    const item = ev.target.closest('.item');
    if (!item || ev.target.closest('.chk') || state.singleClick) return;
    openEntry(state.shown[+item.dataset.i]);
  });
  content.addEventListener('contextmenu', (ev) => {
    ev.preventDefault();
    const item = ev.target.closest('.item');
    if (item) {
      const i = +item.dataset.i;
      if (!state.selection.has(state.shown[i].path)) selectOnly(i);
      showMenu(ev.clientX, ev.clientY, itemMenu());
    } else {
      clearSelection();
      showMenu(ev.clientX, ev.clientY, blankMenu());
    }
  });
  content.addEventListener(
    'load',
    (ev) => {
      if (ev.target.tagName === 'IMG') ev.target.classList.add('ready');
    },
    true
  );
  content.addEventListener(
    'error',
    (ev) => {
      if (ev.target.tagName === 'IMG') ev.target.classList.add('bad');
    },
    true
  );
  content.addEventListener(
    'wheel',
    (ev) => {
      if (!ev.ctrlKey || state.view === 'list') return;
      ev.preventDefault();
      setSize(state.size + (ev.deltaY < 0 ? 30 : -30));
    },
    { passive: false }
  );

  $('#places').addEventListener('contextmenu', (ev) => {
    const fav = ev.target.closest('.pl.fav');
    if (!fav) return;
    ev.preventDefault();
    showMenu(ev.clientX, ev.clientY, [{ label: 'Remove from favorites', run: () => toggleFavorite(fav.dataset.path) }]);
  });

  document.addEventListener('dragstart', (ev) => {
    const item = ev.target.closest?.('.item');
    if (!item || !content.contains(item)) return;
    const i = +item.dataset.i;
    if (!state.selection.has(state.shown[i].path)) selectOnly(i);
    state.drag = selectedEntries().map((e) => e.path);
    ev.dataTransfer.setData('text/plain', state.drag.join('\n'));
    const picked = selectedEntries();
    ev.dataTransfer.setData(
      'text/uri-list',
      picked.map((e) => 'file://' + e.path.split('/').map(encodeURIComponent).join('/')).join('\r\n')
    );
    if (picked.length === 1 && picked[0].kind !== 'folder') {
      const one = picked[0];
      ev.dataTransfer.setData('DownloadURL', `application/octet-stream:${one.name.replace(/:/g, '_')}:${location.origin}/api/file?${qs({ path: one.path, dl: 1 })}`);
    }
    ev.dataTransfer.setData(DRAG_TYPE, JSON.stringify(state.drag));
    ev.dataTransfer.effectAllowed = 'copyMove';
  });
  const clearDrops = () => $$('.drop').forEach((n) => n.classList.remove('drop'));
  const foreignDrag = (ev) => !state.drag && !!ev.dataTransfer && Array.from(ev.dataTransfer.types).includes(DRAG_TYPE);
  document.addEventListener('dragover', (ev) => {
    if (!state.drag && !foreignDrag(ev)) return;
    const target = ev.target.closest('[data-drop]');
    clearDrops();
    if (!target || (state.drag && (state.drag.includes(target.dataset.drop) || target === content))) return;
    ev.preventDefault();
    ev.dataTransfer.dropEffect = ev.ctrlKey ? 'copy' : 'move';
    target.classList.add('drop');
  });
  document.addEventListener('drop', async (ev) => {
    if (!state.drag && !foreignDrag(ev)) return;
    const target = ev.target.closest('[data-drop]');
    let paths = state.drag;
    if (!paths) {
      try {
        paths = JSON.parse(ev.dataTransfer.getData(DRAG_TYPE) || '[]');
      } catch {
        paths = [];
      }
    }
    const internal = !!state.drag;
    state.drag = null;
    clearDrops();
    if (!target || !paths.length || paths.includes(target.dataset.drop) || (internal && target === content)) return;
    ev.preventDefault();
    const res = await runOp({ action: ev.ctrlKey ? 'copy' : 'move', paths, target: target.dataset.drop });
    if (res) {
      state.selection.clear();
      afterOp();
    }
  });
  document.addEventListener('dragend', () => {
    state.drag = null;
    clearDrops();
  });

  function setSort(key) {
    if (state.sortKey === key) state.sortDir = state.sortDir === 'asc' ? 'desc' : 'asc';
    else {
      state.sortKey = key;
      state.sortDir = key === 'date' || key === 'size' ? 'desc' : 'asc';
    }
    store.set('sortKey', state.sortKey);
    store.set('sortDir', state.sortDir);
    render();
    updateToolbar();
    updateStatus();
  }
  function setSize(v) {
    state.size = clamp(v, 120, 640);
    $('#size').value = state.size;
    store.set('size', state.size);
    scheduleRender();
  }
  function setView(v) {
    state.view = v;
    store.set('view', v);
    render();
    updateToolbar();
  }

  $('#views').addEventListener('click', (ev) => {
    const b = ev.target.closest('button');
    if (b) setView(b.dataset.view);
  });
  $('#size').addEventListener('input', (ev) => setSize(+ev.target.value));
  $('#sortkey').addEventListener('change', (ev) => {
    state.sortKey = ev.target.value;
    store.set('sortKey', state.sortKey);
    render();
    updateStatus();
  });
  $('#sortdir').addEventListener('click', () => {
    state.sortDir = state.sortDir === 'asc' ? 'desc' : 'asc';
    store.set('sortDir', state.sortDir);
    render();
    updateToolbar();
  });
  $('#filter').addEventListener('change', (ev) => {
    state.filter = ev.target.value;
    store.set('filter', state.filter);
    render();
    updateStatus();
  });
  $('#btn-hidden').addEventListener('click', toggleHidden);
  function toggleHidden() {
    state.hidden = !state.hidden;
    store.set('hidden', state.hidden);
    refresh();
    updateToolbar();
  }
  $('#btn-newfolder').addEventListener('click', newFolder);
  $('#btn-newfile').addEventListener('click', newFile);
  $('#btn-back').addEventListener('click', () => history.back());
  $('#btn-forward').addEventListener('click', () => history.forward());
  $('#btn-up').addEventListener('click', goUp);
  $('#btn-refresh').addEventListener('click', refresh);
  $('#btn-settings').addEventListener('click', openSettings);
  $('#btn-newtab').addEventListener('click', () => newTab());
  $('#btn-split').addEventListener('click', toggleSplit);
  $('#btn-copyto').addEventListener('click', () => sendToOther('copy'));
  $('#btn-moveto').addEventListener('click', () => sendToOther('move'));
  $('#btn-home').addEventListener('click', () => navigate(state.home));
  function goUp() {
    if (state.parent && state.mode !== 'trash') navigate(state.parent);
  }
  $('#crumbs').addEventListener('dblclick', (ev) => {
    if (ev.target.id === 'crumbs') editPath();
  });
  $('#scope').addEventListener('click', () => {
    const order = ['folder', 'everywhere', 'system'];
    state.scope = order[(order.indexOf(state.scope) + 1) % order.length];
    updateToolbar();
    if (state.query.trim()) runSearch();
    if (!$('#spanel').hidden) renderSearchPanel();
  });

  let searchTimer = 0;
  searchInput.addEventListener('input', () => {
    state.query = searchInput.value;
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => {
      if (state.query.trim()) runSearch();
      else if (state.mode === 'search') refresh();
    }, 220);
  });
  searchInput.addEventListener('keydown', (ev) => {
    if (ev.key === 'Escape') {
      searchInput.value = '';
      state.query = '';
      searchInput.blur();
      if (state.mode === 'search') refresh();
    } else if (ev.key === 'ArrowDown' || ev.key === 'Enter') {
      if (state.shown.length) {
        searchInput.blur();
        selectOnly(0);
        content.focus({ preventScroll: true });
      }
    }
    ev.stopPropagation();
  });

  function neighbor(from, key) {
    const els = state.els;
    if (from < 0) return 0;
    if (key === 'ArrowLeft') return Math.max(0, from - 1);
    if (key === 'ArrowRight') return Math.min(els.length - 1, from + 1);
    const r = els[from].getBoundingClientRect();
    const cx = r.left + r.width / 2;
    const down = key === 'ArrowDown';
    let best = -1;
    let bestScore = Infinity;
    for (let i = 0; i < els.length; i++) {
      const b = els[i].getBoundingClientRect();
      if (down ? b.top <= r.top + 4 : b.top >= r.top - 4) continue;
      const score = Math.abs(b.top - r.top) * 4 + Math.abs(b.left + b.width / 2 - cx);
      if (score < bestScore) {
        bestScore = score;
        best = i;
      }
    }
    return best < 0 ? from : best;
  }

  function moveTo(i, extend) {
    if (i < 0 || i >= state.shown.length) return;
    if (extend) rangeSel(i);
    else selectOnly(i);
    state.els[i].scrollIntoView({ block: 'nearest' });
  }

  function typeAhead(ch) {
    clearTimeout(typeTimer);
    typeBuffer += ch.toLowerCase();
    typeTimer = setTimeout(() => (typeBuffer = ''), 700);
    const start = Math.max(0, state.cursor);
    const order = [...state.shown.keys()];
    const rotated = order.slice(typeBuffer.length === 1 ? start + 1 : start).concat(order.slice(0, typeBuffer.length === 1 ? start + 1 : start));
    const hit = rotated.find((i) => state.shown[i].name.toLowerCase().startsWith(typeBuffer));
    if (hit !== undefined) moveTo(hit, false);
  }

  document.addEventListener('keydown', (ev) => {
    if (viewer.open) return viewerKey(ev);
    if (dlg.open || /^(INPUT|TEXTAREA|SELECT)$/.test(ev.target.tagName)) return;
    const ctrl = ev.ctrlKey || ev.metaKey;
    const key = ev.key;
    if (ev.altKey && key === 'ArrowUp') return (ev.preventDefault(), goUp());
    if (ev.altKey && !ctrl) {
      const k = key.toLowerCase();
      if (k === 't') return (ev.preventDefault(), newTab());
      if (k === 'w') return (ev.preventDefault(), closeTab(tabs.active));
      if (k === ']') return (ev.preventDefault(), switchTab((tabs.active + 1) % tabs.list.length));
      if (k === '[') return (ev.preventDefault(), switchTab((tabs.active - 1 + tabs.list.length) % tabs.list.length));
      if (/^[1-9]$/.test(k)) return (ev.preventDefault(), switchTab(+k - 1));
    }
    if (ctrl && key.toLowerCase() === 'a') return (ev.preventDefault(), selectAll());
    if (ctrl && key.toLowerCase() === 'c') return (ev.preventDefault(), setClip('copy'));
    if (ctrl && key.toLowerCase() === 'x') return (ev.preventDefault(), setClip('move'));
    if (ctrl && key.toLowerCase() === 'v') return (ev.preventDefault(), paste());
    if (ctrl && key.toLowerCase() === 'r') return (ev.preventDefault(), refresh());
    if (ctrl && key.toLowerCase() === 'h') return (ev.preventDefault(), toggleHidden());
    if (ctrl && key.toLowerCase() === 'l') return (ev.preventDefault(), editPath());
    if (ctrl && key.toLowerCase() === 'f') return (ev.preventDefault(), searchInput.focus());
    if (ctrl && ev.shiftKey && key.toLowerCase() === 'n') return (ev.preventDefault(), newFolder());
    if (ctrl || ev.altKey) return;
    switch (key) {
      case '/':
        ev.preventDefault();
        return searchInput.focus();
      case 'F3':
        ev.preventDefault();
        return toggleSplit();
      case 'F5':
        ev.preventDefault();
        return (split || embedded) && other.path ? sendToOther('copy') : refresh();
      case 'F6':
        ev.preventDefault();
        return (split || embedded) && other.path ? sendToOther('move') : undefined;
      case 'F2': {
        const sel = selectedEntries();
        if (sel.length === 1 && state.mode !== 'trash') renameEntry(sel[0]);
        return;
      }
      case 'Delete':
        ev.preventDefault();
        return ev.shiftKey ? deleteSelected() : trashSelected();
      case 'Backspace':
        ev.preventDefault();
        return goUp();
      case 'Escape':
        if (state.selection.size) clearSelection();
        else if (state.mode === 'search') {
          searchInput.value = '';
          state.query = '';
          refresh();
        }
        return;
      case 'Enter':
      case ' ': {
        ev.preventDefault();
        const sel = selectedEntries();
        if (sel.length) openEntry(sel[0]);
        return;
      }
      case 'Home':
        ev.preventDefault();
        return moveTo(0, ev.shiftKey);
      case 'End':
        ev.preventDefault();
        return moveTo(state.shown.length - 1, ev.shiftKey);
      case 'ArrowLeft':
      case 'ArrowRight':
      case 'ArrowUp':
      case 'ArrowDown':
        ev.preventDefault();
        return moveTo(neighbor(state.cursor, key), ev.shiftKey);
    }
    if (key.length === 1 && /\S/.test(key)) typeAhead(key);
  });

  const viewer = {
    el: $('#viewer'),
    stage: $('#v-stage'),
    open: false,
    list: [],
    idx: 0,
    scale: 1,
    fit: 1,
    x: 0,
    y: 0,
    nw: 0,
    nh: 0,
    img: null,
    fitMode: true,
    infoOpen: store.get('info', false),
    slideshow: 0,
    idleTimer: 0,
    drag: null,
  };

  const imageSrc = (e) => (e.native ? fileURL(e.path) : thumbURL(e.path, 2560));

  function openViewer(entry) {
    viewer.list = state.shown.filter(canView);
    viewer.idx = Math.max(0, viewer.list.findIndex((e) => e.path === entry.path));
    viewer.open = true;
    viewer.el.hidden = false;
    document.body.classList.add('noscroll');
    showCurrent();
    wakeViewer();
  }

  function closeViewer() {
    if (viewer.dirty && !confirm('Discard your unsaved changes?')) return;
    viewer.dirty = false;
    stopSlideshow();
    const current = viewer.list[viewer.idx];
    viewer.stage.replaceChildren();
    viewer.el.hidden = true;
    viewer.open = false;
    document.body.classList.remove('noscroll');
    if (document.fullscreenElement) document.exitFullscreen().catch(() => {});
    if (current) {
      const i = state.shown.findIndex((e) => e.path === current.path);
      if (i >= 0) moveTo(i, false);
    }
    content.focus({ preventScroll: true });
  }

  function messageEl(text, e) {
    const box = document.createElement('div');
    box.className = 'v-msg';
    box.innerHTML = `${icon(e.kind, 56)}<div>${esc(text)}</div><button class="btn" type="button">Open with default app</button>`;
    $('button', box).onclick = () => openExternal(e.path);
    return box;
  }

  function hint(text) {
    $$('.v-hint', viewer.stage).forEach((n) => n.remove());
    const n = document.createElement('div');
    n.className = 'v-hint';
    n.textContent = text;
    viewer.stage.append(n);
    return n;
  }

  function showCurrent() {
    const e = viewer.list[viewer.idx];
    if (!e) return closeViewer();
    const stage = viewer.stage;
    $('#v-title').textContent = e.name;
    $('#v-count').textContent = `${viewer.idx + 1} / ${viewer.list.length}`;
    stage.replaceChildren();
    stage.className = '';
    viewer.img = null;
    $('#v-bottom').hidden = e.kind !== 'image';
    if (e.kind === 'image') showImage(e);
    else if (e.kind === 'video') showVideo(e);
    else if (e.kind === 'audio') showAudio(e);
    else if (e.ext === 'pdf') {
      const frame = document.createElement('iframe');
      frame.className = 'v-frame';
      frame.src = fileURL(e.path);
      stage.append(frame);
    } else showText(e);
    if (viewer.infoOpen) loadViewerInfo(e);
    for (const d of [1, -1]) {
      const n = viewer.list[(viewer.idx + d + viewer.list.length) % viewer.list.length];
      if (n && n.kind === 'image' && n.path !== e.path) new Image().src = imageSrc(n);
    }
  }

  function showImage(e) {
    const img = new Image();
    img.className = 'v-img';
    img.draggable = false;
    img.alt = '';
    viewer.img = img;
    viewer.fitMode = true;
    viewer.stage.classList.add('loading');
    img.onload = () => {
      viewer.stage.classList.remove('loading');
      viewer.nw = img.naturalWidth || 800;
      viewer.nh = img.naturalHeight || 600;
      img.style.width = viewer.nw + 'px';
      img.style.height = viewer.nh + 'px';
      fitImage();
    };
    img.onerror = () => {
      viewer.stage.classList.remove('loading');
      viewer.img = null;
      viewer.stage.replaceChildren(messageEl('This image cannot be displayed here', e));
    };
    img.src = imageSrc(e);
    viewer.stage.append(img);
  }

  function applyTransform() {
    const img = viewer.img;
    if (!img) return;
    img.style.transform = `translate(-50%, -50%) translate(${viewer.x}px, ${viewer.y}px) scale(${viewer.scale})`;
    $('#v-zoom').textContent = Math.round(viewer.scale * 100) + '%';
    viewer.stage.classList.toggle('zoomed', viewer.scale > viewer.fit * 1.01);
  }

  function clampPan() {
    const r = viewer.stage.getBoundingClientRect();
    const mx = Math.max(0, (viewer.nw * viewer.scale - r.width) / 2);
    const my = Math.max(0, (viewer.nh * viewer.scale - r.height) / 2);
    viewer.x = clamp(viewer.x, -mx, mx);
    viewer.y = clamp(viewer.y, -my, my);
  }

  function fitImage() {
    if (!viewer.img) return;
    const r = viewer.stage.getBoundingClientRect();
    viewer.fit = Math.min(r.width / viewer.nw, r.height / viewer.nh, 1);
    viewer.scale = viewer.fit;
    viewer.x = viewer.y = 0;
    viewer.fitMode = true;
    applyTransform();
  }

  function zoomAt(cx, cy, target) {
    if (!viewer.img) return;
    const s = clamp(target, viewer.fit, 32);
    const ratio = s / viewer.scale;
    viewer.x = cx - (cx - viewer.x) * ratio;
    viewer.y = cy - (cy - viewer.y) * ratio;
    viewer.scale = s;
    viewer.fitMode = s <= viewer.fit * 1.001;
    if (viewer.fitMode) viewer.x = viewer.y = 0;
    clampPan();
    applyTransform();
  }

  viewer.stage.addEventListener(
    'wheel',
    (ev) => {
      if (!viewer.img) return;
      ev.preventDefault();
      const r = viewer.stage.getBoundingClientRect();
      const delta = ev.deltaMode === 1 ? ev.deltaY * 16 : ev.deltaY;
      zoomAt(ev.clientX - r.left - r.width / 2, ev.clientY - r.top - r.height / 2, viewer.scale * Math.exp(-delta * 0.0015));
    },
    { passive: false }
  );
  viewer.stage.addEventListener('pointerdown', (ev) => {
    if (!viewer.img || ev.button !== 0 || viewer.scale <= viewer.fit * 1.01) return;
    viewer.drag = { x: ev.clientX, y: ev.clientY, ox: viewer.x, oy: viewer.y };
    viewer.stage.setPointerCapture(ev.pointerId);
    viewer.stage.classList.add('dragging');
  });
  viewer.stage.addEventListener('pointermove', (ev) => {
    if (!viewer.drag) return;
    viewer.x = viewer.drag.ox + ev.clientX - viewer.drag.x;
    viewer.y = viewer.drag.oy + ev.clientY - viewer.drag.y;
    clampPan();
    applyTransform();
  });
  const endDrag = () => {
    viewer.drag = null;
    viewer.stage.classList.remove('dragging');
  };
  viewer.stage.addEventListener('pointerup', endDrag);
  viewer.stage.addEventListener('pointercancel', endDrag);
  viewer.stage.addEventListener('dblclick', (ev) => {
    if (!viewer.img) return;
    const r = viewer.stage.getBoundingClientRect();
    const cx = ev.clientX - r.left - r.width / 2;
    const cy = ev.clientY - r.top - r.height / 2;
    if (viewer.scale > viewer.fit * 1.01) fitImage();
    else zoomAt(cx, cy, Math.max(1, viewer.fit * 2));
  });
  window.addEventListener('resize', () => {
    if (viewer.open && viewer.img && viewer.fitMode) fitImage();
    else if (viewer.open && viewer.img) {
      clampPan();
      applyTransform();
    }
    if (!viewer.open && state.view === 'gallery') scheduleRender();
  });

  function showVideo(e) {
    const v = document.createElement('video');
    v.className = 'v-media';
    v.controls = true;
    v.autoplay = true;
    v.playsInline = true;
    v.src = fileURL(e.path);
    v.addEventListener('error', () => {
      if (v.dataset.fallback) {
        viewer.stage.replaceChildren(messageEl('This video cannot be played in the browser', e));
        return;
      }
      v.dataset.fallback = '1';
      const note = hint('Converting for playback…');
      v.addEventListener('playing', () => note.remove(), { once: true });
      v.src = '/api/stream?' + qs({ path: e.path });
      v.play().catch(() => {});
    });
    viewer.stage.append(v);
  }

  function showAudio(e) {
    const box = document.createElement('div');
    box.className = 'v-audio';
    box.innerHTML = `<div class="cover">${icon('audio', 72)}</div><div>${esc(e.name)}</div><audio controls autoplay src="${esc(fileURL(e.path))}"></audio>`;
    const img = new Image();
    img.onload = () => $('.cover', box).replaceChildren(img);
    img.src = thumbURL(e.path, 640);
    viewer.stage.append(box);
  }

  async function showText(e) {
    const box = document.createElement('div');
    box.className = 'codeview';
    box.innerHTML = '<div class="cv-bar"><span class="cv-lang">…</span><span class="cv-meta">Loading</span><span class="cv-dirty" hidden>● Unsaved</span><span class="cv-spacer"></span></div><div class="cv-body"></div>';
    viewer.stage.append(box);
    const bar = $('.cv-bar', box);
    const body = $('.cv-body', box);
    const setButtons = (items) => {
      bar.querySelectorAll('.cv-btn').forEach((b) => b.remove());
      items.forEach(([act, label, primary]) => bar.insertAdjacentHTML('beforeend', `<button type="button" class="cv-btn${primary ? ' primary' : ''}" data-act="${act}">${label}</button>`));
    };
    let res;
    let text;
    try {
      res = await request('/api/text?' + qs({ path: e.path, max: 2097152 }));
      text = await res.text();
    } catch (err) {
      if (box.isConnected) viewer.stage.replaceChildren(messageEl(err.status === 415 ? 'This file is not text' : err.message, e));
      return;
    }
    if (!box.isConnected) return;
    const truncated = !!res.headers.get('X-Truncated');
    let mtime = Number(res.headers.get('X-Mtime')) || 0;
    const crlf = text.includes('\r\n');
    if (crlf) text = text.replace(/\r\n/g, '\n');
    const nl = text.indexOf('\n');
    const key = text.length > 1200000 ? 'text' : LumenHL.detect(e.name, text.slice(0, nl < 0 ? 200 : nl));
    const canEdit = !state.meta.readOnly && !truncated && !text.includes('�');
    const indent = key === 'go' || key === 'make' ? '\t' : '    ';
    let editing = false;
    let ta = null;

    const viewButtons = () => setButtons([['wrap', 'Wrap'], ['copy', 'Copy'], ...(window.LumenExtras && LumenExtras.canRun(e) ? [['run', 'Run']] : []), ...(canEdit ? [['edit', 'Edit', true]] : [])]);
    const setDirty = (on) => {
      viewer.dirty = on;
      $('.cv-dirty', box).hidden = !on;
    };

    const renderView = () => {
      editing = false;
      setDirty(false);
      const lines = LumenHL.render(text, key).split('\n');
      if (lines.length > 1 && lines[lines.length - 1] === '') lines.pop();
      body.className = 'cv-body';
      body.style.setProperty('--lnw', String(lines.length).length + 'ch');
      body.innerHTML = lines.map((l, i) => `<div class="cl"><span class="ln">${i + 1}</span><span class="lc">${l || ' '}</span></div>`).join('') + (truncated ? '<div class="cv-note">… file truncated for preview</div>' : '');
      const wrap = store.get('wrap', null);
      body.classList.toggle('wrap', wrap === null ? ['md', 'txt', 'log'].includes(e.ext) : !!wrap);
      $('.cv-lang', box).textContent = LumenHL.label(key);
      $('.cv-meta', box).textContent = `${lines.length.toLocaleString()} lines · ${fmtSize(new Blob([text]).size)}${crlf ? ' · CRLF' : ''}`;
      viewButtons();
    };

    const renderEdit = () => {
      editing = true;
      body.className = 'cv-body editing';
      body.innerHTML = '<div class="ed-wrap"><pre class="ed-gutter"></pre><div class="ed-main"><pre class="ed-hl"></pre><textarea class="ed-ta" spellcheck="false" autocapitalize="off" autocomplete="off" wrap="off"></textarea></div></div>';
      ta = $('.ed-ta', body);
      const hl = $('.ed-hl', body);
      const gutter = $('.ed-gutter', body);
      ta.value = text;
      let timer = 0;
      let lastCount = 0;
      const paint = () => {
        const value = ta.value;
        hl.innerHTML = (value.length > 400000 ? esc(value) : LumenHL.render(value, key)) + '\n';
        const count = value.split('\n').length;
        if (count !== lastCount) {
          lastCount = count;
          gutter.textContent = Array.from({ length: count }, (_, i) => i + 1).join('\n');
          body.style.setProperty('--lnw', String(count).length + 'ch');
        }
        ta.style.height = hl.offsetHeight + 'px';
        $('.cv-meta', box).textContent = `${count.toLocaleString()} lines · editing`;
      };
      const schedule = () => {
        clearTimeout(timer);
        timer = setTimeout(paint, 90);
      };
      const insert = (value) => {
        if (!document.execCommand('insertText', false, value)) ta.setRangeText(value, ta.selectionStart, ta.selectionEnd, 'end');
      };
      ta.addEventListener('input', () => {
        setDirty(ta.value !== text);
        if (ta.value.length > 60000) schedule();
        else paint();
      });
      ta.addEventListener('keydown', (ev) => {
        ev.stopPropagation();
        if ((ev.ctrlKey || ev.metaKey) && ev.key.toLowerCase() === 's') {
          ev.preventDefault();
          return saveText();
        }
        if (ev.key === 'Escape') {
          ev.preventDefault();
          return leaveEdit();
        }
        if (ev.key === 'Enter' && !ev.ctrlKey && !ev.shiftKey) {
          ev.preventDefault();
          const before = ta.value.slice(0, ta.selectionStart);
          const lead = /^[ \t]*/.exec(before.slice(before.lastIndexOf('\n') + 1))[0];
          const opens = /[{[(:]\s*$/.test(before) && key !== 'text' && key !== 'md';
          return insert('\n' + lead + (opens ? indent : ''));
        }
        if (ev.key === 'Tab') {
          ev.preventDefault();
          const { selectionStart: s, selectionEnd: en, value } = ta;
          const lineStart = value.lastIndexOf('\n', s - 1) + 1;
          if (s === en && !ev.shiftKey) return insert(indent);
          const blockEnd = en > s && value[en - 1] === '\n' ? en - 1 : en;
          const lines = value.slice(lineStart, blockEnd).split('\n');
          const changed = lines.map((l) => (ev.shiftKey ? (l.startsWith(indent) ? l.slice(indent.length) : l.replace(/^ {1,4}/, '')) : indent + l));
          ta.setSelectionRange(lineStart, blockEnd);
          insert(changed.join('\n'));
          ta.setSelectionRange(lineStart, lineStart + changed.join('\n').length);
        }
      });
      ta.addEventListener('scroll', () => {
        ta.scrollTop = 0;
        ta.scrollLeft = 0;
      });
      setButtons([['save', 'Save', true], ['done', 'Done']]);
      paint();
      ta.focus();
      ta.setSelectionRange(0, 0);
      body.scrollTop = 0;
    };

    const saveText = async (force = false) => {
      if (!editing) return;
      const value = ta.value;
      try {
        const out = await sendJSON('/api/save', { path: e.path, content: crlf ? value.replace(/\n/g, '\r\n') : value, mtime, force });
        mtime = Number(out.mtime) || mtime;
        text = value;
        e.size = Number(out.size) || e.size;
        setDirty(false);
        toast('Saved');
      } catch (err) {
        if (err.status !== 409) return toast(err.message, 'error');
        const ok = await ask({ title: 'File changed on disk', message: 'Another program changed this file after you opened it. Overwrite it with your version?', confirm: 'Overwrite', danger: true });
        if (ok !== null) saveText(true);
      }
    };

    const leaveEdit = () => {
      if (viewer.dirty && !confirm('Discard your unsaved changes?')) return;
      renderView();
    };

    box.addEventListener('click', (ev) => {
      const btn = ev.target.closest('.cv-btn');
      if (!btn) return;
      const act = btn.dataset.act;
      if (act === 'copy') copyText(editing ? ta.value : text);
      else if (act === 'wrap') store.set('wrap', body.classList.toggle('wrap'));
      else if (act === 'edit') renderEdit();
      else if (act === 'run') LumenExtras.runDialog(e);
      else if (act === 'save') saveText();
      else if (act === 'done') leaveEdit();
    });

    renderView();
    if (state.editNext === e.path && canEdit) renderEdit();
    state.editNext = null;
  }

  async function loadViewerInfo(e) {
    const panel = $('#v-info');
    panel.hidden = false;
    viewer.el.classList.add('info-open');
    try {
      const info = await getJSON('/api/info?' + qs({ path: e.path }));
      if (viewer.list[viewer.idx]?.path === e.path) panel.innerHTML = `<dl>${rowsHTML(infoRows(info))}</dl>`;
    } catch {
      panel.innerHTML = '';
    }
  }

  function toggleInfo() {
    viewer.infoOpen = !viewer.infoOpen;
    store.set('info', viewer.infoOpen);
    $('#v-info').hidden = !viewer.infoOpen;
    viewer.el.classList.toggle('info-open', viewer.infoOpen);
    if (viewer.infoOpen) loadViewerInfo(viewer.list[viewer.idx]);
    if (viewer.img && viewer.fitMode) fitImage();
  }

  function step(d) {
    if (!viewer.list.length) return;
    if (viewer.dirty && !confirm('Discard your unsaved changes?')) return;
    viewer.dirty = false;
    viewer.idx = (viewer.idx + d + viewer.list.length) % viewer.list.length;
    showCurrent();
  }

  function startSlideshow() {
    stopSlideshow();
    viewer.slideshow = setInterval(() => step(1), 4000);
    toast('Slideshow started');
  }
  function stopSlideshow() {
    clearInterval(viewer.slideshow);
    viewer.slideshow = 0;
  }

  function wakeViewer() {
    viewer.el.classList.remove('idle');
    clearTimeout(viewer.idleTimer);
    viewer.idleTimer = setTimeout(() => viewer.el.classList.add('idle'), 2800);
  }
  viewer.el.addEventListener('mousemove', wakeViewer);

  async function trashCurrent() {
    const e = viewer.list[viewer.idx];
    if (!e) return;
    const res = await runOp({ action: 'trash', paths: [e.path] });
    if (!res || !res.count) return;
    viewer.list.splice(viewer.idx, 1);
    if (!viewer.list.length) {
      closeViewer();
    } else {
      viewer.idx = Math.min(viewer.idx, viewer.list.length - 1);
      showCurrent();
    }
    refresh();
  }

  function zoomCenter(f) {
    zoomAt(0, 0, viewer.scale * f);
  }

  function viewerKey(ev) {
    if (dlg.open) return;
    const media = $('video, audio', viewer.stage);
    switch (ev.key) {
      case 'Escape':
        closeViewer();
        break;
      case 'ArrowLeft':
        step(-1);
        break;
      case 'ArrowRight':
        step(1);
        break;
      case 'Home':
        viewer.idx = 0;
        showCurrent();
        break;
      case 'End':
        viewer.idx = viewer.list.length - 1;
        showCurrent();
        break;
      case '+':
      case '=':
        zoomCenter(1.25);
        break;
      case '-':
        zoomCenter(0.8);
        break;
      case '0':
        fitImage();
        break;
      case '1':
        zoomAt(0, 0, 1);
        break;
      case 'i':
        toggleInfo();
        break;
      case 'f':
        document.fullscreenElement ? document.exitFullscreen() : viewer.el.requestFullscreen().catch(() => {});
        break;
      case 's':
        viewer.slideshow ? (stopSlideshow(), toast('Slideshow stopped')) : startSlideshow();
        break;
      case 'Delete':
        if (!state.meta.readOnly) trashCurrent();
        break;
      case ' ':
      case 'k':
        if (!media) return;
        media.paused ? media.play() : media.pause();
        break;
      case 'j':
        if (!media) return;
        media.currentTime -= 10;
        break;
      case 'l':
        if (!media) return;
        media.currentTime += 10;
        break;
      default:
        return;
    }
    ev.preventDefault();
    wakeViewer();
  }

  viewer.el.addEventListener('click', (ev) => {
    const btn = ev.target.closest('[data-act]');
    if (!btn) return;
    const e = viewer.list[viewer.idx];
    switch (btn.dataset.act) {
      case 'close':
        return closeViewer();
      case 'prev':
        return step(-1);
      case 'next':
        return step(1);
      case 'info':
        return toggleInfo();
      case 'external':
        return e && openExternal(e.path);
      case 'download':
        return e && downloadFile(e);
      case 'fullscreen':
        return document.fullscreenElement ? document.exitFullscreen() : viewer.el.requestFullscreen().catch(() => {});
      case 'slideshow':
        return viewer.slideshow ? (stopSlideshow(), toast('Slideshow stopped')) : startSlideshow();
      case 'zoomin':
        return zoomCenter(1.25);
      case 'zoomout':
        return zoomCenter(0.8);
      case 'fit':
        return fitImage();
      case 'actual':
        return zoomAt(0, 0, 1);
    }
  });

  document.addEventListener('fullscreenchange', () => {
    if (viewer.open && viewer.img && viewer.fitMode) setTimeout(fitImage, 60);
  });

  const tabTitle = (p) => (p === 'trash:' ? 'Trash' : p.split('/').filter(Boolean).pop() || '/');
  const currentLocation = () => (state.mode === 'trash' ? 'trash:' : state.path);

  function renderTabs() {
    const bar = $('#tabs');
    bar.hidden = tabs.list.length < 2;
    if (bar.hidden) return;
    bar.innerHTML =
      tabs.list
        .map(
          (t, i) =>
            `<div class="tab${i === tabs.active ? ' on' : ''}" data-tab="${i}"${t.path !== 'trash:' ? ` data-drop="${esc(t.path)}"` : ''} title="${esc(t.path)}">${icon(t.path === 'trash:' ? 'trash' : 'folder', 15)}<span>${esc(tabTitle(t.path))}</span><button class="x" data-close-tab="${i}" tabindex="-1" type="button">${icon('close', 12)}</button></div>`
        )
        .join('') + `<button class="ib tab-add" data-newtab="1" type="button" title="New tab (Alt+T)">${icon('plus', 16)}</button>`;
  }

  function syncTab() {
    if (!tabs.list.length) tabs.list.push({ path: currentLocation() });
    tabs.list[tabs.active].path = currentLocation();
    renderTabs();
  }

  function newTab(path = currentLocation()) {
    scrolls.set(currentLocation() === 'trash:' ? 'trash' : state.path, content.scrollTop);
    tabs.list.push({ path });
    tabs.active = tabs.list.length - 1;
    renderTabs();
    navigate(path);
  }

  function switchTab(i) {
    if (i < 0 || i >= tabs.list.length || i === tabs.active) return;
    scrolls.set(currentLocation() === 'trash:' ? 'trash' : state.path, content.scrollTop);
    tabs.active = i;
    renderTabs();
    navigate(tabs.list[i].path);
  }

  function closeTab(i) {
    if (tabs.list.length < 2 || i < 0 || i >= tabs.list.length) return;
    const closedActive = i === tabs.active;
    tabs.list.splice(i, 1);
    if (i < tabs.active) tabs.active--;
    else if (closedActive) tabs.active = Math.min(tabs.active, tabs.list.length - 1);
    renderTabs();
    if (closedActive) navigate(tabs.list[tabs.active].path);
  }

  $('#tabs').addEventListener('click', (ev) => {
    const x = ev.target.closest('[data-close-tab]');
    if (x) return closeTab(+x.dataset.closeTab);
    if (ev.target.closest('[data-newtab]')) return newTab();
    const tab = ev.target.closest('[data-tab]');
    if (tab) switchTab(+tab.dataset.tab);
  });
  $('#tabs').addEventListener('auxclick', (ev) => {
    const tab = ev.target.closest('[data-tab]');
    if (tab && ev.button === 1) closeTab(+tab.dataset.tab);
  });
  content.addEventListener('mousedown', (ev) => {
    if (ev.button === 1 && ev.target.closest('.item')) ev.preventDefault();
  });
  content.addEventListener('auxclick', (ev) => {
    if (ev.button !== 1) return;
    const item = ev.target.closest('.item');
    const e = item && state.shown[+item.dataset.i];
    if (e && e.kind === 'folder') newTab(e.path);
  });

  function announce() {
    if (chan) chan.postMessage({ type: 'path', path: state.mode === 'dir' ? state.path : null });
  }

  function connect(sid) {
    if (chan) chan.close();
    chan = typeof BroadcastChannel === 'undefined' ? null : new BroadcastChannel('lumen-' + sid);
    if (!chan) return;
    chan.onmessage = (ev) => {
      const m = ev.data || {};
      if (m.type === 'path') {
        other.path = m.path;
        updateToolbar();
      } else if (m.type === 'hello') {
        announce();
      } else if (m.type === 'refresh') {
        refresh();
        loadPlaces().catch(() => {});
      } else if (m.type === 'unsplit' && !embedded && split) {
        toggleSplit();
      }
    };
    chan.postMessage({ type: 'hello' });
    announce();
  }

  function applySplit() {
    document.body.classList.toggle('split', split);
    let frame = $('#pane2');
    if (split && !frame) {
      const sid = Math.random().toString(36).slice(2, 10);
      frame = document.createElement('iframe');
      frame.id = 'pane2';
      frame.allow = 'fullscreen';
      frame.src = '/?' + qs({ pane: 'b', sid }) + pathToHash(state.mode === 'dir' ? state.path : state.home);
      $('#app').append(frame);
      connect(sid);
    } else if (!split && frame) {
      frame.remove();
      if (chan) chan.close();
      chan = null;
      other.path = null;
    }
    updateToolbar();
  }

  function toggleSplit() {
    if (embedded) {
      if (chan) chan.postMessage({ type: 'unsplit' });
      return;
    }
    split = !split;
    store.set('split', split);
    applySplit();
    scheduleRender();
  }

  async function sendToOther(mode) {
    const paths = selectedEntries().map((e) => e.path);
    if (!paths.length) return toast('Select something first');
    if (!other.path) return toast('The other pane is not showing a folder', 'error');
    const res = await runOp({ action: mode, paths, target: other.path });
    if (!res) return;
    if (res.count) toast(`${res.count} ${mode === 'copy' ? 'copied' : 'moved'} to ${tabTitle(other.path)}`);
    state.selection.clear();
    afterOp();
  }

  window.addEventListener('focus', () => document.body.classList.add('focused'));
  window.addEventListener('blur', () => document.body.classList.remove('focused'));
  if (document.hasFocus()) document.body.classList.add('focused');

  const SEARCH_TYPES = [
    ['image', 'Images'],
    ['video', 'Videos'],
    ['audio', 'Audio'],
    ['doc', 'Documents'],
    ['code', 'Code'],
    ['archive', 'Archives'],
    ['exe', 'Programs'],
    ['folder', 'Folders'],
  ];
  const SEARCH_AGES = [['', 'Any time'], ['1d', 'Today'], ['7d', 'Week'], ['30d', 'Month'], ['1y', 'Year']];
  const SEARCH_SIZES = [['', 'Any size'], ['1mb', '> 1 MB'], ['10mb', '> 10 MB'], ['100mb', '> 100 MB'], ['1gb', '> 1 GB']];
  const spanel = $('#spanel');

  function renderSearchPanel() {
    const text = searchInput.value;
    const typeMatch = text.match(/(?:^|\s)type:(\S+)/i);
    const types = typeMatch ? typeMatch[1].toLowerCase().split(',') : [];
    const ageMatch = text.match(/(?:^|\s)newer:(\S+)/i);
    const sizeMatch = text.match(/(?:^|\s)size>=?(\S+)/i);
    const age = ageMatch ? ageMatch[1].toLowerCase() : '';
    const size = sizeMatch ? sizeMatch[1].toLowerCase() : '';
    const chip = (group, value, label, on) => `<button type="button" class="chip${on ? ' on' : ''}" data-group="${group}" data-value="${esc(value)}">${esc(label)}</button>`;
    spanel.innerHTML = `
      <div class="sp-row"><span>Where</span><div class="chips">${[['folder', 'This folder'], ['everywhere', 'Everywhere'], ['system', 'Whole system']].map(([v, l]) => chip('scope', v, l, state.scope === v)).join('')}</div></div>
      <div class="sp-row"><span>Type</span><div class="chips">${SEARCH_TYPES.map(([v, l]) => chip('type', v, l, types.includes(v))).join('')}</div></div>
      <div class="sp-row"><span>Modified</span><div class="chips">${SEARCH_AGES.map(([v, l]) => chip('age', v, l, age === v)).join('')}</div></div>
      <div class="sp-row"><span>Size</span><div class="chips">${SEARCH_SIZES.map(([v, l]) => chip('size', v, l, size === v)).join('')}</div></div>
      <label class="sp-check"><input type="checkbox" id="sp-content" ${state.inFiles ? 'checked' : ''}> Search inside text files and code</label>
      <div class="sp-hint">Everywhere uses the instant index of your indexed folders. Whole system scans every local drive live and can take a while.</div>
      <div class="sp-hint">Examples: <code>type:image newer:7d</code> · <code>photo -edited size&gt;5mb</code> · <code>type:exe,msi</code> · <code>ext:png,jpg</code> · <code>re:^IMG_[0-9]+</code></div>`;
  }

  function applyChip(group, value) {
    if (group === 'scope') {
      state.scope = value;
      updateToolbar();
      if (state.query.trim()) runSearch();
      return renderSearchPanel();
    }
    let text = searchInput.value;
    const strip = (re) => text.replace(re, ' ').replace(/\s+/g, ' ').trim();
    if (group === 'type') {
      const m = text.match(/(?:^|\s)type:(\S+)/i);
      const current = m ? m[1].toLowerCase().split(',') : [];
      const next = current.includes(value) ? current.filter((x) => x !== value) : [...current, value];
      text = strip(/(?:^|\s)type:\S+/gi);
      if (next.length) text += (text ? ' ' : '') + 'type:' + next.join(',');
    } else if (group === 'age') {
      text = strip(/(?:^|\s)newer:\S+/gi);
      if (value) text += (text ? ' ' : '') + 'newer:' + value;
    } else if (group === 'size') {
      text = strip(/(?:^|\s)size>=?\S+/gi);
      if (value) text += (text ? ' ' : '') + 'size>' + value;
    }
    searchInput.value = text;
    state.query = text;
    if (text.trim()) runSearch();
    else if (state.mode === 'search') refresh();
    renderSearchPanel();
  }

  spanel.addEventListener('click', (ev) => {
    const btn = ev.target.closest('.chip');
    if (btn) applyChip(btn.dataset.group, btn.dataset.value);
  });
  spanel.addEventListener('change', (ev) => {
    if (ev.target.id !== 'sp-content') return;
    state.inFiles = ev.target.checked;
    if (state.query.trim()) runSearch();
  });
  searchInput.addEventListener('focus', () => {
    renderSearchPanel();
    spanel.hidden = false;
  });
  searchInput.addEventListener('input', () => {
    if (!spanel.hidden) renderSearchPanel();
  });
  searchInput.addEventListener('keydown', (ev) => {
    if (ev.key === 'Escape') spanel.hidden = true;
  });
  $('#sfilter').addEventListener('click', () => {
    spanel.hidden = !spanel.hidden;
    if (!spanel.hidden) renderSearchPanel();
  });
  document.addEventListener('mousedown', (ev) => {
    if (!spanel.hidden && !ev.target.closest('.searchbox')) spanel.hidden = true;
  });

  function paintStaticIcons() {
    $('#btn-back').innerHTML = icon('back');
    $('#btn-forward').innerHTML = icon('forward');
    $('#btn-up').innerHTML = icon('up');
    $('#btn-refresh').innerHTML = icon('refresh');
    $('.searchbox .si').innerHTML = icon('search', 16);
    $('#size-ic').innerHTML = icon('size', 16);
    $('#btn-hidden').innerHTML = icon('eye');
    $('#sfilter').innerHTML = icon('filter', 16);
    $('#btn-newfolder').innerHTML = icon('newfolder');
    $('#btn-newfile').innerHTML = icon('newfile');
    $('#btn-newtab').innerHTML = icon('plus');
    $('#btn-split').innerHTML = icon('split');
    $('#btn-home').innerHTML = icon('home');
    $('#btn-settings').innerHTML = icon('gear') + '<span>Settings</span>';
    $$('#views button').forEach((b) => (b.innerHTML = icon(b.dataset.view, 17)));
    const map = { slideshow: 'play', info: 'info', external: 'external', download: 'downloads', fullscreen: 'expand', close: 'close', prev: 'back', next: 'forward', zoomin: 'plus', zoomout: 'minus' };
    $$('#viewer [data-act]').forEach((b) => {
      const name = map[b.dataset.act];
      if (name) b.innerHTML = icon(name, b.classList.contains('v-nav') ? 34 : 18);
    });
  }

  async function boot() {
    document.body.classList.toggle('embedded', embedded);
    paintStaticIcons();
    updateToolbar();
    try {
      state.meta = await getJSON('/api/meta');
    } catch (err) {
      content.innerHTML = `<div class="empty">${icon('info', 44)}<p>${esc(err.message)}</p></div>`;
      return;
    }
    state.home = state.meta.home;
    if (embedded && launchQuery.get('sid')) connect(launchQuery.get('sid'));
    const params = new URLSearchParams(location.search);
    let start = hashPath();
    if (params.has('path')) {
      start = params.get('path');
      if (params.has('view')) state.pendingView = start.replace(/\/$/, '') + '/' + params.get('view');
    }
    history.replaceState(null, '', '/' + (start ? pathToHash(start) : ''));
    window.addEventListener('hashchange', route);
    await Promise.all([loadPlaces().catch(() => {}), route()]);
    updateToolbar();
    pollIndex();
    if (!embedded && store.get('split', false)) {
      split = true;
      applySplit();
    }
  }

  if (window.LumenExtras) {
    LumenExtras.bind({ icon, formDialog, toast, sendJSON, getJSON, state, afterOp, dlg });
  }
  boot();
})();
