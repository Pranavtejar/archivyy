
// No treasure is too obscure.
(() => {
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

