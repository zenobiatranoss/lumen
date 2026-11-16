(function () {
  const set = (s) => new Set(s.split(/\s+/).filter(Boolean));

  const JS_KW = 'break case catch class const continue debugger default delete do else export extends finally for function if import in instanceof new return switch throw try typeof var void while with yield let static async await of get set from as';
  const TS_KW = JS_KW + ' interface type enum implements namespace declare abstract readonly private public protected keyof infer is satisfies override';
  const C_KW = 'auto break case const continue default do else enum extern for goto if inline register restrict return sizeof static struct switch typedef union volatile while';
  const CPP_KW = C_KW + ' alignas alignof and bool catch char class constexpr consteval constinit decltype delete double explicit export false float friend int long mutable namespace new noexcept not nullptr operator or override private protected public short signed template this throw true try typeid typename unsigned using virtual void wchar_t concept requires co_await co_return co_yield';
  const JAVA_KW = 'abstract assert boolean break byte case catch char class const continue default do double else enum extends final finally float for goto if implements import instanceof int interface long native new package private protected public return short static strictfp super switch synchronized this throw throws transient try void volatile while var record sealed permits yield';
  const CS_KW = 'abstract as base bool break byte case catch char checked class const continue decimal default delegate do double else enum event explicit extern finally fixed float for foreach goto if implicit in int interface internal is lock long namespace new object operator out override params private protected public readonly ref return sbyte sealed short sizeof stackalloc static string struct switch this throw try typeof uint ulong unchecked unsafe ushort using virtual void volatile while var async await record init get set yield where partial';
  const KOTLIN_KW = 'as break class continue do else for fun if in interface is object package return super this throw try typealias val var when while by catch constructor delegate dynamic field file finally get import init param property receiver set setparam where abstract annotation companion const crossinline data enum expect external final infix inline inner internal lateinit noinline open operator out override private protected public reified sealed suspend tailrec vararg';
  const SWIFT_KW = 'associatedtype class deinit enum extension fileprivate func import init inout internal let open operator private precedencegroup protocol public rethrows static struct subscript typealias var break case catch continue default defer do else fallthrough for guard if in repeat return throw switch where while as await async is try throws some any actor override final lazy weak mutating';
  const DART_KW = 'abstract as assert async await break case catch class const continue covariant default deferred do dynamic else enum export extends extension external factory final finally for Function get hide if implements import in interface is late library mixin new on operator part required rethrow return set show static super switch sync this throw try typedef var void while with yield';
  const PY_KW = 'and as assert async await break class continue def del elif else except finally for from global if import in is lambda nonlocal not or pass raise return try while with yield match case';
  const GO_KW = 'break case chan const continue default defer else fallthrough for func go goto if import interface map package range return select struct switch type var';
  const RUST_KW = 'as async await break const continue crate dyn else enum extern fn for if impl in let loop match mod move mut pub ref return self Self static struct super trait type unsafe use where while union macro_rules';
  const PHP_KW = 'abstract and array as break callable case catch class clone const continue declare default do echo else elseif empty enddeclare endfor endforeach endif endswitch endwhile enum extends final finally fn for foreach function global goto if implements include include_once instanceof insteadof interface isset list match namespace new or print private protected public readonly require require_once return static switch throw trait try unset use var while xor yield';
  const RUBY_KW = 'alias and begin break case class def defined? do else elsif end ensure for if in module next not or redo rescue retry return self super then undef unless until when while yield require require_relative attr_accessor attr_reader attr_writer puts print lambda proc';
  const LUA_KW = 'and break do else elseif end for function goto if in local not or repeat return then until while';
  const SH_KW = 'if then else elif fi case esac for select while until do done in function time coproc local export readonly declare typeset unset set shift return exit break continue source alias eval exec trap cd echo printf read test';
  const PS_KW = 'begin break catch class continue data do dynamicparam else elseif end exit filter finally for foreach from function if in param process return switch throw trap try until using var while workflow';
  const SQL_KW = 'select from where and or not in is null like between join inner left right full outer cross on as group by order having limit offset union all distinct insert into values update set delete create alter drop table index view database schema primary key foreign references unique default check constraint add column if exists case when then else end asc desc with recursive returning begin commit rollback transaction truncate grant revoke int integer bigint smallint varchar char text boolean date timestamp datetime float double decimal serial';
  const PERL_KW = 'my our local use no package sub return if elsif else unless while until for foreach last next redo goto do eval die warn print say require and or not xor eq ne lt gt le ge cmp x';
  const R_KW = 'if else repeat while function for next break in library require return';
  const ZIG_KW = 'align allowzero and anyframe anytype asm async await break callconv catch comptime const continue defer else enum errdefer error export extern fn for if inline linksection noalias noinline nosuspend opaque or orelse packed pub resume return struct suspend switch test threadlocal try union unreachable usingnamespace var volatile while';
  const LIT = 'true false null nil None True False undefined NaN Infinity NULL nullptr self this super void';

  const defs = {
    js: { name: 'JavaScript', line: ['//'], block: [['/*', '*/']], str: ['"', "'", '`'], kw: JS_KW, lit: LIT, deco: true },
    ts: { name: 'TypeScript', line: ['//'], block: [['/*', '*/']], str: ['"', "'", '`'], kw: TS_KW, lit: LIT, types: 'string number boolean any unknown never object symbol bigint void', deco: true },
    py: { name: 'Python', line: ['#'], str: ['"""', "'''", '"', "'"], kw: PY_KW, lit: LIT, deco: true, types: 'int str float bool list dict set tuple bytes object type' },
    go: { name: 'Go', line: ['//'], block: [['/*', '*/']], str: ['"', '`'], char: true, kw: GO_KW, lit: 'true false nil iota', types: 'string int int8 int16 int32 int64 uint uint8 uint16 uint32 uint64 uintptr float32 float64 complex64 complex128 bool byte rune error any' },
    rust: { name: 'Rust', line: ['//'], block: [['/*', '*/']], str: ['"'], char: true, kw: RUST_KW, lit: 'true false None Some Ok Err', deco: true, types: 'i8 i16 i32 i64 i128 isize u8 u16 u32 u64 u128 usize f32 f64 bool char str String Vec Option Result Box' },
    c: { name: 'C', line: ['//'], block: [['/*', '*/']], str: ['"'], char: true, kw: C_KW, lit: 'true false NULL', pre: true, types: 'int char short long float double void unsigned signed size_t bool uint8_t uint16_t uint32_t uint64_t int8_t int16_t int32_t int64_t' },
    cpp: { name: 'C++', line: ['//'], block: [['/*', '*/']], str: ['"'], char: true, kw: CPP_KW, lit: 'true false NULL nullptr', pre: true, types: 'int size_t string vector map set unique_ptr shared_ptr uint8_t uint16_t uint32_t uint64_t int8_t int16_t int32_t int64_t' },
    cs: { name: 'C#', line: ['//'], block: [['/*', '*/']], str: ['"'], char: true, kw: CS_KW, lit: 'true false null', pre: true, deco: true },
    java: { name: 'Java', line: ['//'], block: [['/*', '*/']], str: ['"'], char: true, kw: JAVA_KW, lit: 'true false null', deco: true, types: 'String Integer Long Double Boolean Object List Map Set' },
    kotlin: { name: 'Kotlin', line: ['//'], block: [['/*', '*/']], str: ['"'], char: true, kw: KOTLIN_KW, lit: 'true false null', deco: true },
    swift: { name: 'Swift', line: ['//'], block: [['/*', '*/']], str: ['"'], kw: SWIFT_KW, lit: 'true false nil', deco: true },
    dart: { name: 'Dart', line: ['//'], block: [['/*', '*/']], str: ['"', "'"], kw: DART_KW, lit: 'true false null', deco: true },
    php: { name: 'PHP', line: ['//', '#'], block: [['/*', '*/']], str: ['"', "'"], kw: PHP_KW, lit: 'true false null TRUE FALSE NULL', dollar: true, ci: true },
    ruby: { name: 'Ruby', line: ['#'], str: ['"', "'"], kw: RUBY_KW, lit: 'true false nil', dollar: true, sym: true },
    lua: { name: 'Lua', line: ['--'], block: [['--[[', ']]']], str: ['"', "'"], kw: LUA_KW, lit: 'true false nil' },
    sh: { name: 'Shell', line: ['#'], str: ['"', "'"], kw: SH_KW, lit: 'true false', dollar: true },
    ps: { name: 'PowerShell', line: ['#'], block: [['<#', '#>']], str: ['"', "'"], kw: PS_KW, lit: '$true $false $null', dollar: true, ci: true },
    sql: { name: 'SQL', line: ['--'], block: [['/*', '*/']], str: ["'"], kw: SQL_KW, lit: 'true false null', ci: true },
    perl: { name: 'Perl', line: ['#'], str: ['"', "'"], kw: PERL_KW, lit: 'undef', dollar: true },
    r: { name: 'R', line: ['#'], str: ['"', "'"], kw: R_KW, lit: 'TRUE FALSE NULL NA Inf NaN' },
    zig: { name: 'Zig', line: ['//'], str: ['"'], char: true, kw: ZIG_KW, lit: 'true false null undefined', deco: true },
    make: { name: 'Makefile', line: ['#'], str: ['"', "'"], kw: 'ifeq ifneq ifdef ifndef else endif include define endef export override', lit: '', dollar: true, targets: true },
    docker: { name: 'Dockerfile', line: ['#'], str: ['"', "'"], kw: 'FROM RUN CMD LABEL EXPOSE ENV ADD COPY ENTRYPOINT VOLUME USER WORKDIR ARG ONBUILD STOPSIGNAL HEALTHCHECK SHELL AS', lit: '', dollar: true },
    cmake: { name: 'CMake', line: ['#'], str: ['"'], kw: 'if else elseif endif foreach endforeach while endwhile function endfunction macro endmacro set project add_executable add_library target_link_libraries target_include_directories find_package include cmake_minimum_required option message install', lit: 'ON OFF TRUE FALSE', dollar: true, ci: true },
    json: { name: 'JSON', line: ['//'], block: [['/*', '*/']], str: ['"'], kw: '', lit: 'true false null', keys: true },
    conf: { name: 'Config', line: ['#', ';'], str: ['"', "'"], kw: '', lit: 'true false yes no on off null' },
    css: { name: 'CSS', block: [['/*', '*/']], str: ['"', "'"], kw: '', lit: '', css: true },
    scss: { name: 'SCSS', line: ['//'], block: [['/*', '*/']], str: ['"', "'"], kw: '', lit: '', css: true },
    html: { name: 'HTML', html: true },
    xml: { name: 'XML', html: true },
    yaml: { name: 'YAML', kv: true },
    toml: { name: 'TOML', kv: true },
    ini: { name: 'INI', kv: true },
    md: { name: 'Markdown', md: true },
    diff: { name: 'Diff', diff: true },
    text: { name: 'Plain text' },
  };

  const compiled = {};
  function lang(key) {
    if (compiled[key]) return compiled[key];
    const d = defs[key] || defs.text;
    const out = Object.assign({ key }, d);
    out.kwSet = set(d.kw || '');
    out.litSet = set(d.lit || '');
    out.typeSet = set(d.types || '');
    compiled[key] = out;
    return out;
  }

  const extMap = {
    js: 'js', mjs: 'js', cjs: 'js', jsx: 'js', ts: 'ts', tsx: 'ts', mts: 'ts', cts: 'ts', py: 'py', pyw: 'py', pyi: 'py', go: 'go', rs: 'rust',
    c: 'c', h: 'c', cc: 'cpp', cpp: 'cpp', cxx: 'cpp', hpp: 'cpp', hh: 'cpp', hxx: 'cpp', ino: 'cpp', cs: 'cs', java: 'java', kt: 'kotlin', kts: 'kotlin',
    swift: 'swift', dart: 'dart', php: 'php', rb: 'ruby', lua: 'lua', sh: 'sh', bash: 'sh', zsh: 'sh', fish: 'sh', ksh: 'sh', ps1: 'ps', psm1: 'ps',
    sql: 'sql', pl: 'perl', pm: 'perl', r: 'r', zig: 'zig', mk: 'make', cmake: 'cmake', json: 'json', jsonc: 'json', json5: 'json', map: 'json',
    css: 'css', scss: 'scss', less: 'scss', sass: 'scss', html: 'html', htm: 'html', xhtml: 'html', vue: 'html', svelte: 'html', xml: 'xml', svg: 'xml',
    xsl: 'xml', xaml: 'xml', plist: 'xml', csproj: 'xml', yaml: 'yaml', yml: 'yaml', toml: 'toml', ini: 'ini', cfg: 'ini', conf: 'conf', desktop: 'ini',
    service: 'ini', md: 'md', markdown: 'md', diff: 'diff', patch: 'diff', env: 'conf', properties: 'conf', gradle: 'java', kt_: 'kotlin',
  };

  const nameMap = {
    makefile: 'make', gnumakefile: 'make', dockerfile: 'docker', 'cmakelists.txt': 'cmake', '.bashrc': 'sh', '.zshrc': 'sh', '.profile': 'sh',
    '.bash_profile': 'sh', '.bash_aliases': 'sh', pkgbuild: 'sh', gemfile: 'ruby', rakefile: 'ruby', '.gitignore': 'conf', '.gitattributes': 'conf',
    '.env': 'conf', '.editorconfig': 'ini', '.gitconfig': 'ini', jenkinsfile: 'js', 'package.json': 'json', '.babelrc': 'json', '.eslintrc': 'json',
  };

  function detect(name, firstLine) {
    const lower = name.toLowerCase();
    if (nameMap[lower]) return nameMap[lower];
    const dot = lower.lastIndexOf('.');
    if (dot >= 0 && extMap[lower.slice(dot + 1)]) return extMap[lower.slice(dot + 1)];
    const first = firstLine || '';
    if (first.startsWith('#!')) {
      if (/python/.test(first)) return 'py';
      if (/node|deno|bun/.test(first)) return 'js';
      if (/ruby/.test(first)) return 'ruby';
      if (/perl/.test(first)) return 'perl';
      if (/php/.test(first)) return 'php';
      if (/lua/.test(first)) return 'lua';
      return 'sh';
    }
    if (/^\s*<\?xml/.test(first) || /^\s*<!doctype html/i.test(first) || /^\s*<html/i.test(first)) return /xml/.test(first) ? 'xml' : 'html';
    if (/^\s*[{[]\s*$/.test(first) || /^\s*\{\s*"/.test(first)) return 'json';
    return 'text';
  }

  const esc = (s) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

  function emitter() {
    let out = '';
    return {
      add(cls, text) {
        if (!text) return;
        const safe = esc(text);
        if (!cls) {
          out += safe;
          return;
        }
        out += '<span class="t-' + cls + '">' + safe.split('\n').join('</span>\n<span class="t-' + cls + '">') + '</span>';
      },
      result: () => out,
    };
  }

  const reIdent = /[A-Za-z_][\w$]*/y;
  const reDollar = /\$(?:\{[^}\n]*\}|\([^)\n]*\)|[A-Za-z_][\w]*|[0-9@#?*!$-])/y;
  const reNumber = /0[xX][0-9a-fA-F_]+[uUlL]*|0[bB][01_]+|\d[\d_]*(?:\.\d+)?(?:[eE][+-]?\d+)?[a-zA-Z]*|\.\d+(?:[eE][+-]?\d+)?/y;
  const reChar = /'(?:\\.[^'\n]*|[^'\\\n])'/y;

  function sticky(re, text, i) {
    re.lastIndex = i;
    const m = re.exec(text);
    return m ? m[0] : null;
  }

  function readString(text, i, quote) {
    const triple = quote.length === 3;
    let j = i + quote.length;
    while (j < text.length) {
      const c = text[j];
      if (c === '\\') {
        j += 2;
        continue;
      }
      if (text.startsWith(quote, j)) return j + quote.length;
      if (c === '\n' && !triple && quote !== '`') return j;
      j++;
    }
    return text.length;
  }

  function scanCode(text, L, em) {
    const n = text.length;
    let i = 0;
    let plain = '';
    let prevSig = '';
    let lineStart = true;
    const flush = () => {
      if (plain) {
        em.add('', plain);
        plain = '';
      }
    };
    const tok = (cls, s) => {
      flush();
      em.add(cls, s);
      prevSig = s[s.length - 1];
      lineStart = false;
    };
    const stringQuotes = (L.str || []).slice().sort((a, b) => b.length - a.length);
    const lowerLookup = L.ci;
    while (i < n) {
      const c = text[i];
      if (c === '\n') {
        plain += c;
        i++;
        lineStart = true;
        continue;
      }
      if (c === ' ' || c === '\t' || c === '\r') {
        plain += c;
        i++;
        continue;
      }
      if (L.targets && lineStart) {
        const m = /^[^\s#:=$]+(?=[^\n:]*:(?!=))[^\n:]*/.exec(text.slice(i, i + 200));
        if (m) {
          tok('fn', m[0]);
          i += m[0].length;
          continue;
        }
      }
      let handled = false;
      if (L.block) {
        for (const [open, close] of L.block) {
          if (text.startsWith(open, i)) {
            const end = text.indexOf(close, i + open.length);
            const stop = end < 0 ? n : end + close.length;
            tok('com', text.slice(i, stop));
            i = stop;
            handled = true;
            break;
          }
        }
        if (handled) continue;
      }
      if (L.line) {
        for (const lc of L.line) {
          if (text.startsWith(lc, i)) {
            if (/^[#;]$/.test(lc) && i > 0 && !/\s/.test(text[i - 1])) continue;
            let end = text.indexOf('\n', i);
            if (end < 0) end = n;
            tok('com', text.slice(i, end));
            i = end;
            handled = true;
            break;
          }
        }
        if (handled) continue;
      }
      if (L.pre && lineStart && c === '#') {
        let end = text.indexOf('\n', i);
        if (end < 0) end = n;
        tok('pre', text.slice(i, end));
        i = end;
        continue;
      }
      if (L.char && c === "'") {
        const ch = sticky(reChar, text, i);
        if (ch) {
          tok('str', ch);
          i += ch.length;
          continue;
        }
        if (L.key === 'rust') {
          const lt = sticky(/'[A-Za-z_]\w*/y, text, i);
          if (lt) {
            tok('type', lt);
            i += lt.length;
            continue;
          }
        }
      }
      let quote = null;
      for (const q of stringQuotes) {
        if (text.startsWith(q, i) && !(L.char && q === "'")) {
          quote = q;
          break;
        }
      }
      if (quote) {
        const end = readString(text, i, quote);
        let cls = 'str';
        if (L.keys) {
          const after = sticky(/\s*:/y, text, end);
          if (after) cls = 'prop';
        }
        tok(cls, text.slice(i, end));
        i = end;
        continue;
      }
      if (L.dollar && c === '$') {
        const v = sticky(reDollar, text, i);
        if (v) {
          tok('var', v);
          i += v.length;
          continue;
        }
      }
      if (L.deco && c === '@') {
        const d = sticky(/@[A-Za-z_][\w.]*/y, text, i);
        if (d) {
          tok('deco', d);
          i += d.length;
          continue;
        }
      }
      if (L.sym && c === ':') {
        const s = sticky(/:[A-Za-z_]\w*[?!]?/y, text, i);
        if (s && prevSig !== ':') {
          tok('lit', s);
          i += s.length;
          continue;
        }
      }
      if (/\d/.test(c) || (c === '.' && /\d/.test(text[i + 1] || ''))) {
        const num = sticky(reNumber, text, i);
        if (num) {
          tok('num', num);
          i += num.length;
          continue;
        }
      }
      if (/[A-Za-z_]/.test(c)) {
        const word = sticky(reIdent, text, i);
        const key = lowerLookup ? word.toLowerCase() : word;
        const next = text.slice(i + word.length, i + word.length + 40).match(/^\s*(\(|:(?!:))/);
        let cls = '';
        if (L.kwSet.has(key)) cls = 'kw';
        else if (L.litSet.has(key)) cls = 'lit';
        else if (L.typeSet.has(key)) cls = 'type';
        else if (prevSig === '.' && !(next && next[1] === '(')) cls = 'prop';
        else if (next && next[1] === '(') cls = 'fn';
        else if (/^[A-Z]/.test(word) && /[a-z]/.test(word)) cls = 'type';
        else if (L.keys && next && next[1] === ':') cls = 'prop';
        tok(cls, word);
        i += word.length;
        continue;
      }
      plain += c;
      prevSig = c;
      lineStart = false;
      i++;
    }
    flush();
  }

  function scanCss(text, L, em) {
    const n = text.length;
    let i = 0;
    let depth = 0;
    let plain = '';
    const flush = () => {
      if (plain) {
        em.add('', plain);
        plain = '';
      }
    };
    const tok = (cls, s) => {
      flush();
      em.add(cls, s);
    };
    while (i < n) {
      const c = text[i];
      if (text.startsWith('/*', i)) {
        const end = text.indexOf('*/', i + 2);
        const stop = end < 0 ? n : end + 2;
        tok('com', text.slice(i, stop));
        i = stop;
        continue;
      }
      if (L.line && text.startsWith('//', i)) {
        let end = text.indexOf('\n', i);
        if (end < 0) end = n;
        tok('com', text.slice(i, end));
        i = end;
        continue;
      }
      if (c === '"' || c === "'") {
        const end = readString(text, i, c);
        tok('str', text.slice(i, end));
        i = end;
        continue;
      }
      if (c === '{') depth++;
      if (c === '}') depth = Math.max(0, depth - 1);
      if (c === '@') {
        const m = sticky(/@[\w-]+/y, text, i);
        if (m) {
          tok('kw', m);
          i += m.length;
          continue;
        }
      }
      if (c === '#' && /[0-9a-fA-F]/.test(text[i + 1] || '') && depth > 0) {
        const m = sticky(/#[0-9a-fA-F]{3,8}\b/y, text, i);
        if (m) {
          tok('num', m);
          i += m.length;
          continue;
        }
      }
      if (/\d/.test(c) || (c === '.' && /\d/.test(text[i + 1] || '') && depth > 0)) {
        const m = sticky(/\d*\.?\d+(?:[a-zA-Z%]+)?/y, text, i);
        if (m) {
          tok('num', m);
          i += m.length;
          continue;
        }
      }
      if (c === '$' || (c === '-' && text[i + 1] === '-')) {
        const m = sticky(/[$-][-\w]+/y, text, i);
        if (m) {
          tok('var', m);
          i += m.length;
          continue;
        }
      }
      if ((c === '.' || c === '#') && depth === 0) {
        const m = sticky(/[.#][\w-]+/y, text, i);
        if (m) {
          tok(c === '.' ? 'type' : 'fn', m);
          i += m.length;
          continue;
        }
      }
      if (/[A-Za-z_-]/.test(c)) {
        const word = sticky(/[A-Za-z_-][\w-]*/y, text, i);
        const rest = text.slice(i + word.length, i + word.length + 60);
        let cls = '';
        if (depth === 0) cls = 'kw';
        else if (/^\s*:/.test(rest) && !/^\s*::/.test(rest)) cls = 'prop';
        else if (/^\(/.test(rest)) cls = 'fn';
        else cls = 'lit';
        tok(cls, word);
        i += word.length;
        continue;
      }
      plain += c;
      i++;
    }
    flush();
  }

  function scanHtml(text, L, em) {
    const n = text.length;
    let i = 0;
    let plain = '';
    const flush = () => {
      if (plain) {
        em.add('', plain);
        plain = '';
      }
    };
    while (i < n) {
      if (text.startsWith('<!--', i)) {
        flush();
        const end = text.indexOf('-->', i + 4);
        const stop = end < 0 ? n : end + 3;
        em.add('com', text.slice(i, stop));
        i = stop;
        continue;
      }
      if (text[i] === '<' && /[A-Za-z/!?]/.test(text[i + 1] || '')) {
        flush();
        const head = sticky(/<[/!?]?[\w:.-]*/y, text, i);
        const tagName = head.replace(/^<[/!?]?/, '').toLowerCase();
        em.add('kw', head);
        i += head.length;
        while (i < n && text[i] !== '>') {
          const c = text[i];
          if (c === '"' || c === "'") {
            const end = text.indexOf(c, i + 1);
            const stop = end < 0 ? n : end + 1;
            em.add('str', text.slice(i, stop));
            i = stop;
            continue;
          }
          const attr = sticky(/[\w:@.-]+/y, text, i);
          if (attr) {
            em.add('prop', attr);
            i += attr.length;
            continue;
          }
          em.add('', c);
          i++;
        }
        let selfClosed = false;
        if (i < n) {
          selfClosed = text[i - 1] === '/';
          em.add('kw', '>');
          i++;
        }
        if (!selfClosed && !head.startsWith('</') && (tagName === 'script' || tagName === 'style')) {
          const closeAt = text.toLowerCase().indexOf('</' + tagName, i);
          const stop = closeAt < 0 ? n : closeAt;
          const inner = text.slice(i, stop);
          const innerLang = lang(tagName === 'script' ? 'js' : 'css');
          if (innerLang.css) scanCss(inner, innerLang, em);
          else scanCode(inner, innerLang, em);
          i = stop;
        }
        continue;
      }
      if (text[i] === '&') {
        const ent = sticky(/&#?\w+;/y, text, i);
        if (ent) {
          flush();
          em.add('num', ent);
          i += ent.length;
          continue;
        }
      }
      plain += text[i];
      i++;
    }
    flush();
  }

  function scanKeyValue(text, L, em) {
    const conf = lang('conf');
    const lines = text.split('\n');
    lines.forEach((line, idx) => {
      if (idx) em.add('', '\n');
      const trimmed = line.trim();
      if (!trimmed) return em.add('', line);
      if (trimmed[0] === '#' || (L.key === 'ini' && trimmed[0] === ';')) return em.add('com', line);
      const section = /^(\s*)(\[\[?[^\]]*\]\]?)(.*)$/.exec(line);
      if (section && L.key !== 'yaml') {
        em.add('', section[1]);
        em.add('type', section[2]);
        scanCode(section[3], conf, em);
        return;
      }
      const kv = L.key === 'yaml' ? /^(\s*(?:-\s+)*)([^\s#:][^:#]*?)(\s*:)(\s|$)(.*)$/.exec(line) : /^(\s*)([^=:#;\s][^=]*?)(\s*[=:])(\s*)(.*)$/.exec(line);
      if (kv) {
        em.add('', kv[1]);
        em.add('prop', kv[2]);
        em.add('', kv[3] + kv[4]);
        scanCode(kv[5], conf, em);
        return;
      }
      const item = /^(\s*-\s+)(.*)$/.exec(line);
      if (item) {
        em.add('', item[1]);
        scanCode(item[2], conf, em);
        return;
      }
      scanCode(line, conf, em);
    });
  }

  function scanMarkdown(text, L, em) {
    let fenced = false;
    text.split('\n').forEach((line, idx) => {
      if (idx) em.add('', '\n');
      if (/^\s*(```|~~~)/.test(line)) {
        fenced = !fenced;
        return em.add('str', line);
      }
      if (fenced) return em.add('str', line);
      if (/^\s{0,3}#{1,6}\s/.test(line)) return em.add('head', line);
      if (/^\s*>/.test(line)) return em.add('com', line);
      if (/^\s*([-*_])(\s*\1){2,}\s*$/.test(line)) return em.add('com', line);
      const bullet = /^(\s*(?:[-*+]|\d+[.)])\s+)(.*)$/.exec(line);
      let rest = line;
      if (bullet) {
        em.add('kw', bullet[1]);
        rest = bullet[2];
      }
      const re = /(`[^`\n]+`)|(\*\*[^*\n]+\*\*|__[^_\n]+__)|(\[[^\]\n]*\]\([^)\n]*\))|(<https?:[^>\n]+>|https?:\/\/[^\s)]+)/g;
      let last = 0;
      let m;
      while ((m = re.exec(rest))) {
        em.add('', rest.slice(last, m.index));
        em.add(m[1] ? 'str' : m[2] ? 'type' : m[3] ? 'fn' : 'num', m[0]);
        last = m.index + m[0].length;
      }
      em.add('', rest.slice(last));
    });
  }

  function scanDiff(text, L, em) {
    text.split('\n').forEach((line, idx) => {
      if (idx) em.add('', '\n');
      let cls = '';
      if (/^(\+\+\+|---)\s/.test(line) || /^(diff |index )/.test(line)) cls = 'kw';
      else if (line.startsWith('@@')) cls = 'fn';
      else if (line.startsWith('+')) cls = 'add';
      else if (line.startsWith('-')) cls = 'del';
      em.add(cls, line);
    });
  }

  function render(text, key) {
    const L = lang(key);
    const em = emitter();
    if (L.html) scanHtml(text, L, em);
    else if (L.kv) scanKeyValue(text, L, em);
    else if (L.md) scanMarkdown(text, L, em);
    else if (L.diff) scanDiff(text, L, em);
    else if (L.css) scanCss(text, L, em);
    else if (L.kwSet && (L.line || L.block || L.str || L.kwSet.size)) scanCode(text, L, em);
    else em.add('', text);
    return em.result();
  }

  window.LumenHL = { detect, render, label: (key) => lang(key).name };
})();
