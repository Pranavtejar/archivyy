(() => {
  const addBar = document.querySelector('.add-bar');
  if (!addBar) return;

  const chips = addBar.querySelectorAll('.type-chip');
  const typeInput = addBar.querySelector('input[name="data_type"]');
  const fileInput = addBar.querySelector('input[type="file"]');
  const fileLabel = addBar.querySelector('#fileLabel');

  chips.forEach(chip => {
    chip.addEventListener('click', () => {
      chips.forEach(c => c.classList.remove('active'));
      chip.classList.add('active');
      typeInput.value = chip.dataset.type;
      fileInput.accept = chip.dataset.accept;
    });
  });

  fileInput.addEventListener('change', () => {
    if (fileInput.files.length) {
      fileLabel.textContent = fileInput.files[0].name;
    } else {
      fileLabel.textContent = '+ File';
    }
  });
})();
