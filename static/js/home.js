
(() => {
  // ---- Archive display ----

  const emptyState = document.getElementById('emptyState');
  const grid = document.getElementById('archiveGrid');

  function clearGrid() {
    if (grid) grid.innerHTML = '';
  }

  function showCardIcon(thumb, key) {
    const icon = document.createElement('span');
    icon.className = 'card-icon';
    const ext = (key.split('.').pop() || 'FILE').toUpperCase().slice(0, 5);
    icon.textContent = ext;
    thumb.appendChild(icon);
  }

  const IMAGE_TYPES = { 'image/jpeg': 1, 'image/png': 1, 'image/gif': 1, 'image/webp': 1, 'image/svg+xml': 1, 'image/bmp': 1, 'image/avif': 1 };
  const IMAGE_EXTS = { jpg: 1, jpeg: 1, png: 1, gif: 1, webp: 1, svg: 1, bmp: 1, avif: 1 };

  function isImage(item) {
    const ct = (item.ContentType || '').toLowerCase().split(';')[0].trim();
    if (IMAGE_TYPES[ct]) return true;
    const ext = (item.key.split('.').pop() || '').toLowerCase();
    return !!IMAGE_EXTS[ext];
  }

  function renderItem(item) {
    const card = document.createElement('a');
    card.className = 'card';
    card.href = '/view/' + encodeURIComponent(item.key);

    const thumb = document.createElement('div');
    thumb.className = 'card-thumb';

    const addImage = (src, alt) => {
      const img = document.createElement('img');
      img.loading = 'lazy';
      img.alt = alt;
      img.src = src;
      img.onerror = () => {
        img.remove();
        showCardIcon(thumb, item.key);
      };
      thumb.appendChild(img);
    };

    if (item.preview) {
      addImage(item.preview, item.key);
    } else if (isImage(item)) {
      addImage('/stream/' + encodeURIComponent(item.key), item.key);
    } else {
      showCardIcon(thumb, item.key);
    }

    const info = document.createElement('div');
    info.className = 'card-info';

    const title = item.title || item.key;

    const name = document.createElement('div');
    name.className = 'card-name';
    name.title = title;
    name.textContent = title;

    const meta = document.createElement('div');
    meta.className = 'card-size';
    meta.textContent = item.ContentType || 'application/octet-stream';

    info.appendChild(name);
    info.appendChild(meta);

    card.appendChild(thumb);
    card.appendChild(info);
    grid.appendChild(card);
  }

  async function loadArchive() {
    if (!grid || !emptyState) return;

    clearGrid();
    emptyState.textContent = 'The archive is empty';

    let items = [];
    try {
      const res = await fetch('/display', { cache: 'no-store' });
      if (!res.ok) throw new Error(`/display responded with ${res.status}`);
      const data = await res.json();
      items = Array.isArray(data) ? data : [];
    } catch (err) {
      console.error('Archive load failed:', err);
      emptyState.textContent = 'Could not reach the archive';
      emptyState.style.display = 'block';
      grid.style.display = 'none';
      return;
    }

    grid.style.display = '';
    emptyState.style.display = items.length ? 'none' : 'block';
    items.forEach(renderItem);
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
  const thumbLabel = document.getElementById('thumbLabel');
  const thumbInput = document.getElementById('thumbInput');
  const thumbLabelText = document.getElementById('thumbLabelText');

  let previewURL = null;

  const MEDIA_EXTS = { mp4: 1, webm: 1, mov: 1, m4v: 1, mkv: 1, avi: 1, ogv: 1, mp3: 1, wav: 1, ogg: 1, opus: 1, flac: 1, aac: 1, m4a: 1 };

  function isMediaFile(file) {
    const t = (file.type || '').toLowerCase();
    if (t.indexOf('video/') === 0 || t.indexOf('audio/') === 0) return true;
    const ext = (file.name.split('.').pop() || '').toLowerCase();
    return !!MEDIA_EXTS[ext];
  }

  function updateThumbField(file) {
    if (file && isMediaFile(file)) {
      thumbLabel.style.display = '';
    } else {
      thumbLabel.style.display = 'none';
      thumbInput.value = '';
      thumbLabelText.textContent = '+ Thumbnail';
    }
  }

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

    thumbLabel.style.display = 'none';
    thumbInput.value = '';
    thumbLabelText.textContent = '+ Thumbnail';

    uploadMsg.textContent = '';
    uploadMsg.className = 'upload-msg';
  }

  fileInput.addEventListener('change', () => {
    const f = fileInput.files[0];
    updateThumbField(f);

    if (f) {
      showPreview(f);
    } else {
      clearPreview();
    }
  });

  thumbInput.addEventListener('change', () => {
    thumbLabelText.textContent = thumbInput.files.length
      ? thumbInput.files[0].name
      : '+ Thumbnail';
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

    const thumbFile = thumbInput.files[0];
    if (thumbFile) {
      fd.append('thumbnail', thumbFile, thumbFile.name);
    }

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

