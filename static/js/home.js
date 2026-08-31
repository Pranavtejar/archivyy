
(() => {
  // ---- Archive display ----

  const emptyState = document.getElementById('emptyState');
  const grid = document.getElementById('archiveGrid');
  let gridURLs = [];

  const EXT_MIME = {
    '.mp4': 'video/mp4',
    '.webm': 'video/webm',
    '.mov': 'video/quicktime',
    '.avi': 'video/x-msvideo',
    '.mkv': 'video/x-matroska',
    '.jpg': 'image/jpeg',
    '.jpeg': 'image/jpeg',
    '.png': 'image/png',
    '.gif': 'image/gif',
    '.webp': 'image/webp',
    '.svg': 'image/svg+xml',
    '.bmp': 'image/bmp',
    '.txt': 'text/plain',
    '.md': 'text/markdown',
    '.pdf': 'application/pdf',
    '.json': 'application/json',
    '.html': 'text/html',
    '.csv': 'text/csv',
    '.zip': 'application/zip',
    '.mp3': 'audio/mpeg',
    '.wav': 'audio/wav',
    '.ogg': 'audio/ogg',
  };

  function toBinaryString(bytes) {
    const chunk = 0x8000;
    const pieces = new Array(Math.ceil(bytes.length / chunk));
    for (let i = 0, j = 0; i < bytes.length; i += chunk, j++) {
      pieces[j] = String.fromCharCode.apply(null, bytes.subarray(i, i + chunk));
    }
    return pieces.join('');
  }

  function toBytes(str) {
    const bytes = new Uint8Array(str.length);
    for (let i = 0; i < str.length; i++) bytes[i] = str.charCodeAt(i) & 0xff;
    return bytes;
  }

  function boundaryFrom(contentType) {
    const m = /;\s*boundary=(?:"([^"]+)"|([^;]+))/i.exec(contentType);
    return m ? (m[1] || m[2] || '').trim() : '';
  }

  function parseHeaders(text) {
    const headers = {};
    for (const line of text.split('\r\n')) {
      const i = line.indexOf(':');
      if (i !== -1) {
        headers[line.slice(0, i).trim().toLowerCase()] = line.slice(i + 1).trim();
      }
    }
    return headers;
  }

  function filenameFromDisposition(disposition) {
    const star = /filename\*\s*=\s*(?:[^']*''|)([^;]+)/i.exec(disposition);
    if (star) {
      try {
        return decodeURIComponent(star[1].trim());
      } catch (_) {}
    }
    const plain = /filename\s*=\s*(?:"([^"]*)"|([^;]*))/i.exec(disposition);
    if (plain) return (plain[1] || plain[2] || '').trim();
    return '';
  }

  function mimeFor(filename, contentType) {
    const declared = (contentType || '').split(';')[0].trim().toLowerCase();
    if (declared && declared !== 'application/octet-stream') return declared;
    const ext = (filename.split('.').pop() || '').toLowerCase();
    return (ext && EXT_MIME['.' + ext]) || declared || 'application/octet-stream';
  }

  function parseMultipart(bytes, boundary) {
    const text = toBinaryString(bytes);
    const delimiter = '--' + boundary;
    const files = [];
    let pos = 0;

    while (pos < text.length) {
      const start = text.indexOf(delimiter, pos);
      if (start === -1) break;

      let p = start + delimiter.length;

      if (text.startsWith('--', p)) break;
      if (!text.startsWith('\r\n', p)) break;
      p += 2;

      let headerEnd = text.indexOf('\r\n\r\n', p);
      let headerStep = 4;
      if (headerEnd === -1) {
        headerEnd = text.indexOf('\n\n', p);
        headerStep = 2;
      }
      if (headerEnd === -1) break;

      const headers = parseHeaders(text.slice(p, headerEnd));
      const bodyStart = headerEnd + headerStep;

      const nextDelim = text.indexOf('\r\n' + delimiter, bodyStart);
      const bodyEnd = nextDelim === -1 ? text.length : nextDelim;

      const name = filenameFromDisposition(headers['content-disposition']);
      const type = headers['content-type'];
      const filename = name || `archive-item-${files.length}`;
      files.push(new File([toBytes(text.slice(bodyStart, bodyEnd))], filename, {
        type: mimeFor(filename, type),
      }));

      if (nextDelim === -1) break;
      pos = nextDelim;
    }

    return files;
  }

  function clearGrid() {
    if (!grid) return;
    gridURLs.forEach((url) => URL.revokeObjectURL(url));
    gridURLs = [];
    grid.innerHTML = '';
  }

  function renderFile(file) {
    const card = document.createElement('a');
    card.className = 'card';
    card.href = '/view/' + encodeURIComponent(file.name);

    const thumb = document.createElement('div');
    thumb.className = 'card-thumb';

    const thumbURL = URL.createObjectURL(file);
    gridURLs.push(thumbURL);

    let media = null;
    if (file.type.startsWith('image/')) {
      media = document.createElement('img');
      media.loading = 'lazy';
    } else if (file.type.startsWith('video/')) {
      media = document.createElement('video');
      media.muted = true;
      media.playsInline = true;
      media.preload = 'metadata';
      media.onloadeddata = () => { media.currentTime = 1; };
    }

    if (media) {
      media.src = thumbURL;
      thumb.appendChild(media);
    } else {
      const icon = document.createElement('span');
      icon.className = 'card-icon';
      const ext = (file.name.split('.').pop() || 'FILE').toUpperCase().slice(0, 5);
      icon.textContent = ext;
      thumb.appendChild(icon);
    }

    const info = document.createElement('div');
    info.className = 'card-info';

    const name = document.createElement('div');
    name.className = 'card-name';
    name.title = file.name;
    name.textContent = file.name;

    const size = document.createElement('div');
    size.className = 'card-size';
    size.textContent = formatSize(file.size);

    info.appendChild(name);
    info.appendChild(size);

    card.appendChild(thumb);
    card.appendChild(info);
    grid.appendChild(card);
  }

  async function loadArchive() {
    if (!grid || !emptyState) return;

    clearGrid();
    emptyState.textContent = 'The archive is empty';

    let files = [];
    try {
      const res = await fetch('/display');
      if (!res.ok) throw new Error(`/display responded with ${res.status}`);

      const boundary = boundaryFrom(res.headers.get('Content-Type') || '');
      if (!boundary) throw new Error('multipart boundary missing');

      const bytes = new Uint8Array(await res.arrayBuffer());
      files = parseMultipart(bytes, boundary);
    } catch (err) {
      console.error('Archive load failed:', err);
      emptyState.textContent = 'Could not reach the archive';
      emptyState.style.display = 'block';
      grid.style.display = 'none';
      return;
    }

    grid.style.display = '';
    emptyState.style.display = files.length ? 'none' : 'block';
    files.forEach(renderFile);
  }

  loadArchive();

  // ---- Upload UI ----

  const addBar = document.querySelector('.add-bar');
  if (!addBar) return;

  const form = document.getElementById('uploadForm');
  const chips = addBar.querySelectorAll('.type-chip');
  const typeInput = addBar.querySelector('input[name="data_type"]');
  const fileInput = document.getElementById('fileInput');
  const fileLabelText = document.getElementById('fileLabelText');
  const previewArea = document.getElementById('previewArea');
  const previewName = document.getElementById('previewName');
  const previewSize = document.getElementById('previewSize');
  const progressFill = document.getElementById('progressFill');
  const previewClear = document.getElementById('previewClear');
  const submitBtn = document.getElementById('submitBtn');
  const uploadMsg = document.getElementById('uploadMsg');

  let previewURL = null;

  chips.forEach(chip => {
    chip.addEventListener('click', () => {
      chips.forEach(c => c.classList.remove('active'));

      chip.classList.add('active');
      typeInput.value = chip.dataset.type;
      fileInput.accept = chip.dataset.accept;
    });
  });

  function formatSize(bytes) {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / 1048576).toFixed(1)} MB`;
  }

  function getThumb() {
    return document.getElementById('previewThumb');
  }

  function showPreview(file) {
    if (previewURL) {
      URL.revokeObjectURL(previewURL);
    }

    previewURL = URL.createObjectURL(file);

    previewName.textContent = file.name;
    previewSize.textContent = formatSize(file.size);
    progressFill.style.width = '0%';

    const old = getThumb();
    let el;

    if (file.type.startsWith('image/')) {
      el = document.createElement('img');
      el.src = previewURL;

      Object.assign(el.style, {
        width: '40px',
        height: '40px',
        borderRadius: '2px',
        objectFit: 'cover',
        border: '1px solid rgba(255,255,255,0.18)',
        flexShrink: '0'
      });
    } else if (file.type.startsWith('video/')) {
      el = document.createElement('video');
      el.src = previewURL;
      el.muted = true;
      el.playsInline = true;
      el.preload = 'metadata';

      Object.assign(el.style, {
        width: '40px',
        height: '40px',
        borderRadius: '2px',
        objectFit: 'cover',
        border: '1px solid rgba(255,255,255,0.18)',
        flexShrink: '0'
      });

      el.onloadeddata = () => {
        el.currentTime = 0.1;
      };
    } else {
      el = document.createElement('div');
      el.className = 'preview-icon';
      el.textContent = file.name.split('.').pop().toUpperCase();
    }

    el.id = 'previewThumb';
    old.replaceWith(el);

    fileLabelText.textContent = file.name;
    previewArea.style.display = 'flex';
  }

  function clearPreview() {
    if (previewURL) {
      URL.revokeObjectURL(previewURL);
      previewURL = null;
    }

    fileInput.value = '';
    fileLabelText.textContent = '+ File';
    previewArea.style.display = 'none';
    progressFill.style.width = '0%';

    uploadMsg.textContent = '';
    uploadMsg.className = 'upload-msg';
  }

  fileInput.addEventListener('change', () => {
    if (fileInput.files.length) {
      showPreview(fileInput.files[0]);
    } else {
      clearPreview();
    }
  });

  previewClear.addEventListener('click', clearPreview);

  form.addEventListener('submit', (e) => {
    e.preventDefault();

    if (!fileInput.files.length) {
      uploadMsg.textContent = 'Please select a file.';
      uploadMsg.className = 'upload-msg error';
      return;
    }

    const file = fileInput.files[0];

    console.log('FILE:', file);
    console.log('FILE NAME:', file.name);
    console.log('FILE TYPE:', file.type);
    console.log('FILE SIZE:', file.size);

    const fd = new FormData();

    fd.append(
      'title',
      form.querySelector('input[name="title"]').value
    );

    fd.append(
      'data_type',
      typeInput.value
    );

    fd.append(
      'file',
      file,
      file.name
    );

    const xhr = new XMLHttpRequest();

    submitBtn.disabled = true;
    submitBtn.textContent = '...';

    uploadMsg.textContent = 'Uploading...';
    uploadMsg.className = 'upload-msg';

    xhr.upload.addEventListener('progress', (ev) => {
      if (ev.lengthComputable) {
        const pct = Math.round((ev.loaded / ev.total) * 100);
        progressFill.style.width = `${pct}%`;
      }
    });

    xhr.addEventListener('load', () => {
      submitBtn.disabled = false;
      submitBtn.textContent = 'Add →';

      if (xhr.status >= 200 && xhr.status < 300) {
        uploadMsg.textContent = 'File uploaded';
        uploadMsg.className = 'upload-msg success';

        loadArchive();

        setTimeout(() => {
          clearPreview();
        }, 2000);
      } else {
        let msg = 'Upload failed';

        try {
          const response = JSON.parse(xhr.responseText);
          msg = response.error || msg;
        } catch (_) {}

        uploadMsg.textContent = msg;
        uploadMsg.className = 'upload-msg error';
        progressFill.style.width = '0%';
      }
    });

    xhr.addEventListener('error', () => {
      submitBtn.disabled = false;
      submitBtn.textContent = 'Add →';

      uploadMsg.textContent = 'Upload failed';
      uploadMsg.className = 'upload-msg error';
      progressFill.style.width = '0%';
    });

    xhr.open('POST', '/upload');
    xhr.send(fd);
  });
})();

