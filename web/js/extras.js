(() => {
  'use strict';

  const api = {};
  let bound = false;

  const THEMES = [
    ['rose', 'Rose pink'],
    ['red', 'Muted red'],
    ['blue', 'Slate blue'],
    ['green', 'Sage green'],
    ['gray', 'Soft gray'],
  ];

  function themeValue(name) {
    return THEMES.some(([id]) => id === name) ? name : 'rose';
  }

  function savedTheme() {
    try {
      const raw = localStorage.getItem('lumen.theme');
      return themeValue(raw === null ? 'rose' : JSON.parse(raw));
    } catch {
      return 'rose';
    }
  }

  function applyTheme(name) {
    const id = themeValue(name);
    document.documentElement.dataset.theme = id;
    return id;
  }

  function saveTheme(name) {
    const id = applyTheme(name);
    try {
      localStorage.setItem('lumen.theme', JSON.stringify(id));
    } catch {}
    return id;
  }

  function themeOptions() {
    const current = savedTheme();
    return THEMES.map(([id, label]) => `<option value="${id}"${id === current ? ' selected' : ''}>${label}</option>`).join('');
  }

  applyTheme(savedTheme());

  window.addEventListener('storage', (ev) => {
    if (ev.key === 'lumen.theme') applyTheme(savedTheme());
  });

  const lazy = 'IntersectionObserver' in window ? new IntersectionObserver((entries) => {
    for (const entry of entries) {
      if (!entry.isIntersecting) continue;
      const img = entry.target;
      lazy.unobserve(img);
      if (!img.dataset.src) continue;
      img.src = img.dataset.src;
      delete img.dataset.src;
    }
  }, { rootMargin: '300px' }) : null;

  function watchIcons(root) {
    const nodes = [];
    if (root.nodeType === 1 && root.matches('.xicon-img[data-src]')) nodes.push(root);
    if (root.querySelectorAll) nodes.push(...root.querySelectorAll('.xicon-img[data-src]'));
    for (const img of nodes) {
      if (lazy) {
        lazy.observe(img);
      } else {
        img.src = img.dataset.src;
        delete img.dataset.src;
      }
    }
  }

  const EXT_ICONS = {
    pdf: 'application-pdf',
    doc: 'x-office-document',
    docx: 'x-office-document',
    odt: 'x-office-document',
    rtf: 'x-office-document',
    epub: 'application-epub+zip',
    mobi: 'application-epub+zip',
    djvu: 'image-x-generic',
    xls: 'x-office-spreadsheet',
    xlsx: 'x-office-spreadsheet',
    ods: 'x-office-spreadsheet',
    csv: 'x-office-spreadsheet',
    tsv: 'x-office-spreadsheet',
    ppt: 'x-office-presentation',
    pptx: 'x-office-presentation',
    odp: 'x-office-presentation',
    txt: 'text-x-generic',
    md: 'text-markdown',
    markdown: 'text-markdown',
    rst: 'text-x-generic',
    log: 'text-x-generic',
    nfo: 'text-x-generic',
    srt: 'text-x-generic',
    vtt: 'text-x-generic',
    tex: 'text-x-tex',
    json: 'application-json',
    yaml: 'application-x-yaml',
    yml: 'application-x-yaml',
    toml: 'application-x-toml',
    ini: 'text-x-generic',
    conf: 'text-x-generic',
    cfg: 'text-x-generic',
    env: 'text-x-generic',
    xml: 'application-xml',
    desktop: 'application-x-desktop',
    service: 'text-x-generic',
    py: 'text-x-python',
    pyw: 'text-x-python',
    c: 'text-x-csrc',
    h: 'text-x-chdr',
    cpp: 'text-x-c++src',
    cc: 'text-x-c++src',
    cxx: 'text-x-c++src',
    hpp: 'text-x-c++src',
    cs: 'text-x-csharp',
    rs: 'text-x-rust',
    java: 'text-x-java',
    kt: 'text-x-kotlin',
    go: 'text-x-go',
    rb: 'text-x-ruby',
    php: 'text-x-php',
    js: 'application-javascript',
    mjs: 'application-javascript',
    cjs: 'application-javascript',
    jsx: 'text-x-javascript',
    ts: 'text-x-typescript',
    tsx: 'text-x-typescript',
    vue: 'text-x-generic',
    svelte: 'text-x-generic',
    html: 'text-html',
    htm: 'text-html',
    css: 'text-css',
    scss: 'text-css',
    sass: 'text-css',
    sh: 'text-x-script',
    bash: 'text-x-script',
    zsh: 'text-x-script',
    fish: 'text-x-script',
    lua: 'text-x-lua',
    pl: 'text-x-perl',
    r: 'text-x-r',
    sql: 'text-x-sql',
    swift: 'text-x-swift',
    dart: 'text-x-dart',
    asm: 'text-x-asm',
    s: 'text-x-asm',
    cmake: 'text-x-cmake',
    mk: 'text-x-makefile',
    gradle: 'text-x-generic',
    jpg: 'image-x-generic',
    jpeg: 'image-x-generic',
    jpe: 'image-x-generic',
    jfif: 'image-x-generic',
    png: 'image-x-generic',
    gif: 'image-x-generic',
    webp: 'image-x-generic',
    bmp: 'image-x-generic',
    tif: 'image-x-generic',
    tiff: 'image-x-generic',
    svg: 'image-x-generic',
    avif: 'image-x-generic',
    heic: 'image-x-generic',
    heif: 'image-x-generic',
    ico: 'image-x-generic',
    jxl: 'image-x-generic',
    apng: 'image-x-generic',
    mp4: 'video-x-generic',
    mkv: 'video-x-generic',
    webm: 'video-x-generic',
    mov: 'video-x-generic',
    avi: 'video-x-generic',
    m4v: 'video-x-generic',
    flv: 'video-x-generic',
    wmv: 'video-x-generic',
    mpg: 'video-x-generic',
    mpeg: 'video-x-generic',
    m2ts: 'video-x-generic',
    '3gp': 'video-x-generic',
    ogv: 'video-x-generic',
    vob: 'video-x-generic',
    mp3: 'audio-x-generic',
    flac: 'audio-x-generic',
    wav: 'audio-x-generic',
    ogg: 'audio-x-generic',
    oga: 'audio-x-generic',
    opus: 'audio-x-generic',
    m4a: 'audio-x-generic',
    aac: 'audio-x-generic',
    wma: 'audio-x-generic',
    aiff: 'audio-x-generic',
    mka: 'audio-x-generic',
    amr: 'audio-x-generic',
    zip: 'application-zip',
    tar: 'application-x-tar',
    gz: 'application-x-compressed-tar',
    tgz: 'application-x-compressed-tar',
    bz2: 'application-x-bzip-compressed-tar',
    tbz: 'application-x-bzip-compressed-tar',
    tbz2: 'application-x-bzip-compressed-tar',
    xz: 'application-x-xz-compressed-tar',
    txz: 'application-x-xz-compressed-tar',
    zst: 'application-zstd',
    '7z': 'application-x-7z-compressed',
    rar: 'application-x-rar',
    iso: 'application-x-cd-image',
    img: 'application-x-cd-image',
    deb: 'application-x-deb',
    rpm: 'application-x-rpm',
    appimage: 'application-x-executable',
    jar: 'application-java-archive',
    cab: 'application-x-cab',
    lz4: 'application-x-lz4',
    ttf: 'font-x-generic',
    otf: 'font-x-generic',
    woff: 'font-x-generic',
    woff2: 'font-x-generic',
    exe: 'application-x-executable',
    msi: 'application-x-msi',
    bin: 'application-x-executable',
    run: 'application-x-executable',
    lumen: 'security-high',
  };

  const KIND_ICONS = {
    folder: 'folder',
    image: 'image-x-generic',
    video: 'video-x-generic',
    audio: 'audio-x-generic',
    doc: 'x-office-document',
    text: 'text-x-generic',
    code: 'text-x-generic',
    archive: 'package-x-generic',
    other: 'application-x-executable',
  };

  const RUN_EXTS = new Set([
    'py', 'pyw', 'c', 'cpp', 'cc', 'cxx', 'go', 'rs', 'java', 'kt', 'js', 'mjs', 'cjs', 'ts',
    'sh', 'bash', 'zsh', 'fish', 'rb', 'php', 'pl', 'lua', 'r', 'swift', 'hs', 'ex', 'exs',
    'dart', 'jl', 's', 'asm', 'cs', 'coffee',
  ]);

  const folderIcon = (name) => {
    const lower = name.toLowerCase();
    if (/^(desktop|dolphin|xfce4-desktop)$/.test(lower)) return 'user-desktop';
    if (/^(downloads?|téléchargements)$/.test(lower)) return 'folder-download';
    if (/^(pictures?|photos?|images?|wallpapers?)$/.test(lower)) return 'folder-pictures';
    if (/^(music|audio|sounds?)$/.test(lower)) return 'folder-music';
    if (/^(videos?|movies?)$/.test(lower)) return 'folder-videos';
    if (/^(documents?|docs)$/.test(lower)) return 'folder-documents';
    if (/^(templates?)$/.test(lower)) return 'folder-templates';
    if (/^(public|shared|publicshare)$/.test(lower)) return 'folder-publicshare';
    return 'folder';
  };

  function iconName(e) {
    if (!e) return 'application-x-executable';
    if (e.kind === 'folder') return folderIcon(e.name || '');
    const ext = String(e.ext || '').toLowerCase();
    if (EXT_ICONS[ext]) return EXT_ICONS[ext];
    return KIND_ICONS[e.kind] || 'application-x-executable';
  }

  const fallbackName = (e) => (e && e.kind === 'folder' ? 'folder' : (KIND_ICONS[e && e.kind] || 'application-x-executable'));

  const esc = (s) => String(s).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]);

  function themeURL(name, size) {
    return '/api/icon?' + new URLSearchParams({ name, size: String(Math.round(size)) });
  }

  function themed(name, size, fallback, cls = '') {
    if (!name) return '';
    const fb = api.icon ? api.icon(fallback || 'application-x-executable', size) : '';
    return `<span class="xicon ${cls}" style="width:${size}px;height:${size}px" aria-hidden="true">${fb ? `<span class="xicon-fb">${fb}</span>` : ''}<img class="xicon-img" data-src="${esc(themeURL(name, size))}" alt="" decoding="async" draggable="false"></span>`;
  }

  function placeholder(e, size = 34) {
    if (!bound) return '';
    const ext = size > 30 && e.ext ? `<span class="xext">${esc(e.ext.toUpperCase())}</span>` : '';
    return `<div class="ph xph">${themed(iconName(e), size, fallbackName(e))}${ext}</div>`;
  }

  function appIcon(name, size = 20) {
    return themed(name || 'application-x-executable', size, 'application-x-executable', 'ow-icon');
  }

  function canRun(e) {
    return !!e && e.kind !== 'folder' && RUN_EXTS.has(String(e.ext || '').toLowerCase());
  }

  function bind(obj) {
    Object.assign(api, obj);
    bound = true;
  }

  async function encryptDialog(items) {
    if (!bound || !items.length) return;
    const data = await api.formDialog({
      title: items.length === 1 ? `Encrypt ${items[0].name}` : `Encrypt ${items.length} items`,
      confirm: 'Encrypt',
      body:
        `<label>Password<input type="password" name="pw" autocomplete="new-password" spellcheck="false"></label>` +
        `<label>Repeat the password<input type="password" name="pw2" autocomplete="new-password" spellcheck="false"></label>` +
        `<label class="check"><input type="checkbox" name="trash" value="1"> Move the originals to the trash after encrypting</label>` +
        `<div class="note">The encrypted files get the .lumen extension. Keep the password somewhere safe: it cannot be recovered.</div>`,
    });
    if (!data) return;
    if (String(data.pw || '').length < 4) return api.toast('Choose a password with at least 4 characters', 'error');
    if (data.pw !== data.pw2) return api.toast('The passwords do not match', 'error');
    const done = [];
    const failed = [];
    for (const item of items) {
      try {
        const res = await api.sendJSON('/api/crypt', { action: 'encrypt', path: item.path, password: data.pw, trash: !!data.trash });
        done.push(res.to);
        if (res.warning) api.toast(res.warning, 'error');
      } catch (err) {
        failed.push(`${item.name}: ${err.message}`);
      }
    }
    if (done.length) {
      api.state.pendingSelect = done[0];
      await api.afterOp();
      api.toast(done.length === 1 ? 'Encrypted ' + (items[0].name || 'the file') : `Encrypted ${done.length} items`);
    }
    if (failed.length) api.toast(failed.length === 1 ? failed[0] : `${failed.length} items could not be encrypted: ${failed[0]}`, 'error');
  }

  async function decryptDialog(items) {
    if (!bound || !items.length) return;
    const data = await api.formDialog({
      title: items.length === 1 ? `Decrypt ${items[0].name}` : `Decrypt ${items.length} files`,
      confirm: 'Decrypt',
      body:
        `<label>Password<input type="password" name="pw" autocomplete="current-password" spellcheck="false"></label>` +
        `<div class="note">Folders are restored from their encrypted archive in the same folder.</div>`,
    });
    if (!data) return;
    if (!data.pw) return api.toast('Enter the password', 'error');
    const done = [];
    const failed = [];
    for (const item of items) {
      try {
        const res = await api.sendJSON('/api/crypt', { action: 'decrypt', path: item.path, password: data.pw });
        done.push(res.to);
      } catch (err) {
        failed.push(`${item.name}: ${err.message}`);
      }
    }
    if (done.length) {
      api.state.pendingSelect = done[0];
      await api.afterOp();
      api.toast(done.length === 1 ? 'Decrypted' : `Decrypted ${done.length} files`);
    }
    if (failed.length) api.toast(failed.length === 1 ? failed[0] : `${failed.length} files could not be decrypted: ${failed[0]}`, 'error');
  }

  async function runDialog(e) {
    if (!bound) return;
    let info = {};
    try {
      info = await api.getJSON('/api/dev?' + new URLSearchParams({ path: e.path }));
    } catch {}
    api.dlg.innerHTML =
      `<form method="dialog" class="dlg wide dev-dlg"><h3>Compile &amp; run</h3>` +
      `<p>${esc(e.name)}${info.name ? ` · ${esc(info.name)}` : ''}</p>` +
      `<div class="note">Code runs on this computer with your user permissions. Only run files you trust.</div>` +
      `<label>Arguments (optional)<input type="text" id="dev-args" name="args" autocomplete="off" spellcheck="false" placeholder="--help or a file name"></label>` +
      `<pre class="dev-out" id="dev-out">Ready.</pre>` +
      `<div class="dev-status" id="dev-status"></div>` +
      `<div class="dlg-actions"><button type="button" class="btn" data-close>Close</button><button type="button" class="btn primary" id="dev-run">Run</button></div></form>`;
    api.dlg.returnValue = '';
    api.dlg.showModal();
    const out = api.dlg.querySelector('#dev-out');
    const status = api.dlg.querySelector('#dev-status');
    const args = api.dlg.querySelector('#dev-args');
    const button = api.dlg.querySelector('#dev-run');
    const go = async () => {
      button.disabled = true;
      button.textContent = 'Running…';
      out.classList.remove('err', 'ok');
      out.textContent = 'Running…';
      status.textContent = '';
      status.className = 'dev-status';
      try {
        const res = await api.sendJSON('/api/dev', { path: e.path, args: args.value });
        const lines = [];
        if (res.build) lines.push('Build: ' + res.build);
        if (res.command) lines.push('Run: ' + res.command);
        if (lines.length) lines.push('');
        lines.push(res.output || '(no output)');
        out.textContent = lines.join('\n');
        out.classList.add(res.ok ? 'ok' : 'err');
        status.textContent = res.ok
          ? `Finished in ${(res.durationMs / 1000).toFixed(2)} s`
          : res.timedOut
            ? 'Stopped after the time limit'
            : `Exited with code ${res.exit} after ${(res.durationMs / 1000).toFixed(2)} s`;
        status.className = 'dev-status ' + (res.ok ? 'ok' : 'fail');
      } catch (err) {
        out.textContent = err.message;
        out.classList.add('err');
        status.textContent = 'Could not run this file';
        status.className = 'dev-status fail';
      } finally {
        button.disabled = false;
        button.textContent = 'Run again';
      }
    };
    button.addEventListener('click', go);
    args.addEventListener('keydown', (ev) => {
      if (ev.key === 'Enter') {
        ev.preventDefault();
        go();
      }
    });
    args.focus();
  }

  function itemMenu(sel, readOnly) {
    const items = [];
    if (!bound || !sel.length) return items;
    const encrypted = sel.filter((e) => String(e.ext || '').toLowerCase() === 'lumen');
    if (encrypted.length === sel.length) {
      items.push({ label: sel.length > 1 ? `Decrypt ${sel.length} files…` : 'Decrypt…', disabled: readOnly, run: () => decryptDialog(sel) });
    } else if (!encrypted.length) {
      items.push({ label: sel.length > 1 ? `Encrypt ${sel.length} items…` : 'Encrypt…', disabled: readOnly, run: () => encryptDialog(sel) });
    }
    if (sel.length === 1 && canRun(sel[0])) {
      items.push({ label: 'Compile & run…', run: () => runDialog(sel[0]) });
    }
    return items;
  }

  function openEntry(e) {
    if (!bound || !e || String(e.ext || '').toLowerCase() !== 'lumen') return false;
    decryptDialog([e]);
    return true;
  }

  document.addEventListener(
    'load',
    (ev) => {
      const img = ev.target;
      if (img && img.classList && img.classList.contains('xicon-img')) img.parentElement?.classList.add('has-img');
    },
    true
  );
  document.addEventListener(
    'error',
    (ev) => {
      const img = ev.target;
      if (img && img.classList && img.classList.contains('xicon-img')) img.classList.add('bad');
    },
    true
  );

  if (document.body) {
    watchIcons(document.body);
    new MutationObserver((records) => {
      for (const record of records) {
        for (const node of record.addedNodes) {
          if (node.nodeType === 1) watchIcons(node);
        }
        for (const node of record.removedNodes) {
          if (node.nodeType !== 1) continue;
          const imgs = node.matches?.('.xicon-img[data-src]') ? [node] : [...(node.querySelectorAll?.('.xicon-img[data-src]') || [])];
          for (const img of imgs) if (lazy) lazy.unobserve(img);
        }
      }
    }).observe(document.body, { childList: true, subtree: true });
  }

  window.LumenExtras = { bind, placeholder, appIcon, iconName, canRun, itemMenu, openEntry, runDialog, encryptDialog, decryptDialog, theme: savedTheme, applyTheme, saveTheme, themeOptions };
})();
