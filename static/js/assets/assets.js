const initAssetsPage = () => {
    document.querySelectorAll('#assetGrid img.avtldimg').forEach(img => {
        if (img.complete && img.naturalWidth > 0) {
            img.classList.add('ld');
        } else {
            img.onload = () => img.classList.add('ld');
        }
    });

    const searchInput = document.getElementById('assetSearch');
    if (searchInput) {
        searchInput.oninput = function() {
            const q = this.value.trim().toLowerCase();
            document.querySelectorAll('#assetGrid .shp-card, #assetGrid .avtitm').forEach(item => {
                const nameEl = item.querySelector('.avtitmnm');
                const match = !q || (nameEl && nameEl.textContent.toLowerCase().indexOf(q) !== -1);
                item.style.display = match ? '' : 'none';
            });
        };
    }
};

initAssetsPage();
document.addEventListener('htmx:afterSettle', initAssetsPage);