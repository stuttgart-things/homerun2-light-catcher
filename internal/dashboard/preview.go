package dashboard

// previewJS animates the preview strip of the dashboard: it follows the newest
// timeline event (live), or plays a clicked event from the history again.
//
// The effects are approximations drawn in the browser from the event's
// effect name, colors, speed and intensity -- what the strip itself shows is
// rendered by WLED. Effects without an own animation here fall back to a
// palette running along the strip.
const previewJS = `
var preview = (function() {
  var N = 30, HOLD_MS = 20000;
  var strip = document.getElementById('strip');
  var modeEl = document.getElementById('preview-mode');
  var labelEl = document.getElementById('preview-label');
  var infoEl = document.getElementById('preview-info');
  var liveBtn = document.getElementById('preview-live');
  var tl = document.getElementById('timeline');
  var leds = [];
  for (var i = 0; i < N; i++) {
    var d = document.createElement('div');
    d.className = 'led';
    strip.appendChild(d);
    leds.push(d);
  }

  // A few WLED device palettes, for effects that name one instead of colors.
  var devicePalettes = {
    'party': [[85,0,171],[132,0,124],[181,0,75],[229,0,27],[232,23,0],[184,71,0],[171,119,0],[171,171,0]],
    'cloud': [[0,0,255],[0,0,139],[135,206,235],[255,255,255]],
    'lava': [[0,0,0],[128,0,0],[255,0,0],[255,165,0],[255,255,255]],
    'ocean': [[25,25,112],[0,0,139],[0,128,128],[0,255,255],[127,255,212]],
    'forest': [[0,100,0],[85,107,47],[34,139,34],[107,142,35],[144,238,144]],
    'rainbow': [[255,0,0],[255,165,0],[255,255,0],[0,255,0],[0,0,255],[128,0,128]],
    'sunset': [[120,0,0],[179,22,0],[255,104,0],[167,22,18],[100,0,103],[16,0,130]],
    'red & blue': [[255,0,0],[0,0,255]],
    'pastel': [[61,135,184],[149,222,179],[255,252,214],[255,169,169]]
  };

  var cur = null, mode = 'live', holdUntil = 0, start = performance.now();
  var state = [];  // per LED: [r,g,b,level] for the sparkling effects

  function look(el) {
    if (!el || el.dataset.off) return null;
    var cols = [];
    try { cols = JSON.parse(el.dataset.rgb || '[]'); } catch (e) {}
    if (!cols.length) cols = devicePalettes[(el.dataset.pal || '').toLowerCase()] || devicePalettes.rainbow;
    return {
      id: el.dataset.id, fx: el.dataset.fx || 'Solid', cols: cols,
      sx: +el.dataset.sx || 128, ix: +el.dataset.ix || 255,
      label: el.dataset.label || '', pal: el.dataset.pal || ''
    };
  }

  function play(lk, how) {
    cur = lk;
    mode = how;
    start = performance.now();
    state = [];
    for (var i = 0; i < N; i++) state.push([0, 0, 0, 0]);
    holdUntil = how === 'replay' ? Date.now() + HOLD_MS : 0;
    modeEl.className = 'preview-mode ' + how;
    modeEl.textContent = how === 'replay' ? 'Replay' : 'Live';
    liveBtn.hidden = how !== 'replay';
    if (!lk) {
      labelEl.textContent = 'Light off';
      infoEl.textContent = 'Click an event in the timeline to see its effect again.';
    } else {
      labelEl.textContent = lk.label;
      infoEl.textContent = lk.fx + (lk.pal ? ' · palette ' + lk.pal : '') + ' · speed ' + lk.sx + ' · intensity ' + lk.ix +
        (how === 'replay' ? ' · returns to live after ' + HOLD_MS / 1000 + 's' : '');
    }
    markSelected();
  }

  function markSelected() {
    tl.querySelectorAll('details.event').forEach(function(d) {
      d.classList.toggle('selected', mode === 'replay' && cur && d.dataset.id === cur.id);
    });
  }

  // refresh follows the newest event while live; the page calls it after
  // every timeline poll.
  function refresh() {
    if (mode === 'replay' && Date.now() < holdUntil) { markSelected(); return; }
    var newest = look(tl.querySelector('[data-off], details.event'));
    if (mode === 'replay' || (newest ? newest.id : null) !== (cur ? cur.id : null)) play(newest, 'live');
  }

  tl.addEventListener('click', function(e) {
    if (e.target.closest('button.replay')) return;
    var sum = e.target.closest('details.event > summary');
    if (sum) play(look(sum.parentElement), 'replay');
  });
  liveBtn.addEventListener('click', function() { holdUntil = 0; mode = 'replay'; refresh(); });

  function mix(a, b, t) { return [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t, a[2] + (b[2] - a[2]) * t]; }
  function scale(c, k) { return [c[0] * k, c[1] * k, c[2] * k]; }
  // pal samples the colors as a cyclic gradient, x in [0,1).
  function pal(cols, x) {
    x = ((x % 1) + 1) % 1 * cols.length;
    var i = Math.floor(x);
    return mix(cols[i], cols[(i + 1) % cols.length], x - i);
  }
  function hue(h) {
    h = ((h % 1) + 1) % 1 * 6;
    var x = 1 - Math.abs(h % 2 - 1), c;
    c = h < 1 ? [1, x, 0] : h < 2 ? [x, 1, 0] : h < 3 ? [0, 1, x] : h < 4 ? [0, x, 1] : h < 5 ? [x, 0, 1] : [1, 0, x];
    return scale(c, 255);
  }
  function sparkle(cols, rate, decay) {
    for (var i = 0; i < N; i++) {
      var s = state[i];
      if (Math.random() < rate) { var c = cols[Math.floor(Math.random() * cols.length)]; state[i] = [c[0], c[1], c[2], 1]; }
      else s[3] = Math.max(0, s[3] - decay);
    }
  }

  function frame(now) {
    requestAnimationFrame(frame);
    var out = [];
    if (!cur) {
      for (var i = 0; i < N; i++) out.push(null);
    } else {
      var t = (now - start) / 1000 * (0.3 + cur.sx / 255 * 1.7);
      var cols = cur.cols, c0 = cols[0], c1 = cols[1] || [0, 0, 0];
      var fx = cur.fx.toLowerCase();
      var dens = 0.02 + cur.ix / 255 * 0.08;
      for (var i = 0; i < N; i++) {
        var p = i / N, c;
        switch (fx) {
          case 'solid': c = c0; break;
          case 'blink': c = (t * 2) % 1 < 0.5 ? c0 : c1; break;
          case 'strobe': case 'strobe mega': c = (t * 4) % 1 < 0.15 ? c0 : null; break;
          case 'strobe rainbow': c = (t * 4) % 1 < 0.15 ? hue(t / 4) : null; break;
          case 'blink rainbow': c = (t * 2) % 1 < 0.5 ? hue(t / 4) : null; break;
          case 'breathe': c = scale(c0, 0.15 + 0.85 * (0.5 - 0.5 * Math.cos(t * 2.5))); break;
          case 'fade': c = mix(c0, c1, 0.5 - 0.5 * Math.cos(t * 2)); break;
          case 'wipe': case 'sweep':
            var w = (t / 2) % 2; c = (w < 1 ? p < w : p >= w - 1) ? c0 : c1; break;
          case 'colorloop': c = hue(t / 6); break;
          case 'rainbow': case 'rainbow runner': case 'chase rainbow': c = hue(p - t / 4); break;
          case 'scan': case 'scanner': case 'scan dual': case 'lighthouse': case 'android':
            var pos = (0.5 - 0.5 * Math.cos(t * 1.5)) * (N - 1), dist = Math.abs(i - pos);
            if (fx === 'scan dual') dist = Math.min(dist, Math.abs(N - 1 - i - pos));
            c = dist < 4 ? mix(c0, c1, dist / 4) : c1; if (dist >= 4 && !cols[1]) c = null; break;
          case 'theater': case 'chase': case 'running': case 'chase 2': case 'chase flash':
            c = (i + Math.floor(t * 6)) % 3 === 0 ? c0 : (cols[1] ? scale(c1, 0.3) : null); break;
          case 'dynamic': case 'random colors': case 'colorful':
            c = pal(cols, Math.floor(t * 2) * 0.37 + ((i * 7919) % N) / N); break;
          case 'candle': c = scale(c0, 0.55 + 0.45 * Math.abs(Math.sin(t * 7 + Math.sin(t * 13 + i)))); break;
          case 'aurora': c = scale(pal(cols, p * 0.7 + t / 8), 0.35 + 0.65 * (0.5 + 0.5 * Math.sin(p * 9 - t * 1.3))); break;
          case 'dj light': case 'blurz':
            var beat = Math.pow(1 - (t * 1.5) % 1, 2), d = Math.abs(p - 0.5) * 2;
            c = scale(pal(cols, t / 3 + d / 2), Math.max(0, beat - d * 0.6)); break;
          case 'twinkle': case 'sparkle': case 'sparkle+': case 'dissolve': case 'fireworks': c = undefined; break;
          default: c = pal(cols, p - t / 6);
        }
        out.push(c);
      }
      if (out[0] === undefined) {
        sparkle(cols, fx === 'fireworks' ? dens / 3 : dens, fx === 'fireworks' ? 0.03 : 0.05);
        for (var j = 0; j < N; j++) {
          var s = state[j], base = fx === 'sparkle' || fx === 'sparkle+' ? scale(c0, 0.15) : [0, 0, 0];
          out[j] = mix(base, s, s[3]);
        }
      }
    }
    for (var k = 0; k < N; k++) {
      var col = out[k];
      if (!col || col[0] + col[1] + col[2] < 6) {
        leds[k].style.background = '#111827';
        leds[k].style.boxShadow = 'none';
      } else {
        var rgb = 'rgb(' + Math.round(col[0]) + ',' + Math.round(col[1]) + ',' + Math.round(col[2]) + ')';
        leds[k].style.background = rgb;
        leds[k].style.boxShadow = '0 0 8px ' + rgb;
      }
    }
  }

  refresh();
  requestAnimationFrame(frame);
  return { refresh: refresh };
})();
`
