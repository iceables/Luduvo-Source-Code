window.toggleGlobalDropdown = function(dropdown) {
    const target = typeof dropdown === 'string' ? document.getElementById(dropdown) : dropdown;
    if (!target) return;

    const isOpen = target.classList.contains('open') || target.classList.contains('active');

    document.querySelectorAll('.glb-dropdown.open, .glb-dropdown.active, .shp-dropdown.open, .shp-dropdown.active').forEach(d => {
        if (d !== target) {
            d.classList.remove('open', 'active');
        }
    });

    target.classList.toggle('open', !isOpen);
    target.classList.toggle('active', !isOpen);
};

window.selectGlobalDropdownOption = function(dropdown, value, label, callback) {
    const target = typeof dropdown === 'string' ? document.getElementById(dropdown) : dropdown;
    if (!target) return;

    const labelEl = target.querySelector('.glb-dropdown-label, .shp-dropdown-label, #shopSortLabel, #admRoleLabel');
    const inputEl = target.querySelector('input[type="hidden"]');

    if (labelEl) labelEl.textContent = label;
    if (inputEl) {
        inputEl.value = value;
        inputEl.dispatchEvent(new Event('change', { bubbles: true }));
    }

    target.querySelectorAll('.glb-dropdown-item, .shp-dropdown-item').forEach(item => {
        item.classList.toggle('active', item.dataset.value === String(value));
    });

    target.classList.remove('open', 'active');

    if (typeof callback === 'function') {
        callback(value, label);
    } else if (target.id === 'shopSortDropdown' && typeof window.fetchShopItems === 'function') {
        window.fetchShopItems(1, false);
    } else if (target.id === 'admRoleDropdown' && typeof window.fetchAdminUsers === 'function') {
        window.fetchAdminUsers(1);
    }
};

document.addEventListener('click', function(event) {
    if (!event.target.closest('.glb-dropdown') && !event.target.closest('.shp-dropdown')) {
        document.querySelectorAll('.glb-dropdown.open, .glb-dropdown.active, .shp-dropdown.open, .shp-dropdown.active').forEach(d => {
            d.classList.remove('open', 'active');
        });
    }
});