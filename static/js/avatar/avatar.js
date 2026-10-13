window.avatarPart = 'all';
window.avatarPartColors = {};
window.avatarCategory = 'all';
window.avatarSearchTimer = null;
window.avatarColorPicker = null;
window.avatarDirty = false;

const avtEsc = function(str) {
    if (window.escapeHTML) return window.escapeHTML(String(str));
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
};

const rgbToHex = function(col) {
    if (!col) return '#f3b700';
    col = col.trim();
    if (col.startsWith('#')) return col.toLowerCase();
    const match = col.match(/^rgb\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*\)$/i);
    if (match) {
        const r = parseInt(match[1], 10).toString(16).padStart(2, '0');
        const g = parseInt(match[2], 10).toString(16).padStart(2, '0');
        const b = parseInt(match[3], 10).toString(16).padStart(2, '0');
        return '#' + r + g + b;
    }
    return col;
};

window.updateAvatarTabIndicator = function() {
    const indicator = document.querySelector('.tabcnt .tabind');
    const activeTab = document.querySelector('.tabcnt .tabbtn.active');
    if (indicator && activeTab) {
        indicator.style.left = activeTab.offsetLeft + 'px';
        indicator.style.width = activeTab.offsetWidth + 'px';
        indicator.style.height = activeTab.offsetHeight + 'px';
        indicator.style.top = activeTab.offsetTop + 'px';
    }
};

window.switchAvatarTab = function(cat, event) {
    const clickedTab = event ? event.currentTarget : null;
    if (!clickedTab || clickedTab.classList.contains('active')) return;

    document.querySelectorAll('.tabcnt .tabbtn').forEach(tab => tab.classList.remove('active'));
    clickedTab.classList.add('active');

    window.updateAvatarTabIndicator();

    window.avatarCategory = cat;
    window.fetchAvatarInventory();
};

window.avatarItemCardHTML = function(item) {
    const worn = !!item.is_equipped;
    const type = avtEsc(item.type);
    return `
        <div class="avtitm${worn ? ' worn' : ''}" data-type="${type}" data-id="${item.id}">
            <div class="avtitmimg">
                <img src="/api/v1/avatar/shop/${encodeURIComponent(item.type)}/${item.id}" alt="${avtEsc(item.name)}" loading="lazy" class="avtldimg" onerror="this.src='/static/useful/temp/pfp.png';">
                ${worn ? '<span class="avtitmbdg">Worn</span>' : ''}
            </div>
            <div class="avtitmmeta">
                <span class="avtitmnm">${avtEsc(item.name)}</span>
                <span class="avtitmcr">by ${avtEsc(item.creator_name)}</span>
            </div>
            <button type="button" class="primary ${worn ? 'hpydng' : 'hpyprim'} hpyinl hpysm avtitembtn">
                <span>${worn ? 'Take Off' : 'Wear'}</span>
            </button>
        </div>
    `;
};

window.avatarSpinnerHTML = function() {
    return '<div class="musld"><svg class="dmspn" viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg">' +
        '<rect class="sq sq-1" x="43" y="7" width="14" height="14"/>' +
        '<rect class="sq sq-2" x="25" y="25" width="14" height="14"/>' +
        '<rect class="sq sq-3" x="7" y="43" width="14" height="14"/>' +
        '<rect class="sq sq-4" x="25" y="61" width="14" height="14"/>' +
        '<rect class="sq sq-5" x="43" y="79" width="14" height="14"/>' +
        '<rect class="sq sq-6" x="61" y="61" width="14" height="14"/>' +
        '<rect class="sq sq-7" x="79" y="43" width="14" height="14"/>' +
        '<rect class="sq sq-8" x="61" y="25" width="14" height="14"/>' +
        '</svg></div>';
};

window.renderAvatarGrid = function(items) {
    const grid = document.getElementById('avtGrid');
    if (!grid) return;
    if (!items.length) {
        grid.innerHTML = '<div class="feedemt">You don\'t own any items yet.</div>';
        return;
    }
    grid.innerHTML = items.map(window.avatarItemCardHTML).join('');
};

window.fetchAvatarInventory = async function() {
    const grid = document.getElementById('avtGrid');
    if (!grid) return;

    const searchInput = document.getElementById('avtSearch');
    const q = searchInput ? searchInput.value.trim() : '';

    grid.innerHTML = window.avatarSpinnerHTML();

    try {
        const res = await fetch('/api/v1/avatar/inventory?category=' + encodeURIComponent(window.avatarCategory) + '&q=' + encodeURIComponent(q), {
            headers: { 'Accept': 'application/json' }
        });
        if (!res.ok) throw new Error('request failed');
        const items = await res.json();
        window.renderAvatarGrid(Array.isArray(items) ? items : []);
    } catch {
        grid.innerHTML = '<div class="feedemt">Failed to load items. Please try again.</div>';
    }
};

window.avatarWearingChipHTML = function(item) {
    return `
        <div class="avtwchip" data-type="${avtEsc(item.type)}" data-id="${item.id}">
            <div class="avtwimg">
                <img src="/api/v1/avatar/shop/${encodeURIComponent(item.type)}/${item.id}" alt="${avtEsc(item.name)}" loading="lazy" class="avtldimg" onerror="this.src='/static/useful/temp/pfp.png';">
            </div>
            <div class="avtwmeta">
                <span class="avtwnm">${avtEsc(item.name)}</span>
                <span class="avtwt">${avtEsc(item.type_name)}</span>
            </div>
            <button type="button" class="avtwrmv" title="Take off" aria-label="Take off">
                <i class="fa-solid fa-xmark"></i>
            </button>
        </div>
    `;
};

window.refreshAvatarWearing = async function() {
    const list = document.getElementById('avtWearing');
    if (!list) return;

    try {
        const res = await fetch('/api/v1/avatar/wearing', {
            headers: { 'Accept': 'application/json' }
        });
        if (!res.ok) throw new Error('request failed');
        const items = await res.json();
        if (!Array.isArray(items) || !items.length) {
            list.innerHTML = '<div class="feedemt">You are not wearing any items.</div>';
            return;
        }
        list.innerHTML = items.map(window.avatarWearingChipHTML).join('');
    } catch {}
};

window.avatarOutfitChipHTML = function(outfit) {
    return `
        <div class="avtwchip" data-outfit-id="${outfit.id}">
            <div class="avtwimg">
                <img src="/avatar/outfit/${outfit.id}.png?t=${Date.now()}" alt="${avtEsc(outfit.name)}" loading="lazy" class="avtldimg" onerror="this.src='/static/useful/temp/pfp.png';">
            </div>
            <div class="avtwmeta">
                <span class="avtwnm">${avtEsc(outfit.name)}</span>
            </div>
            <div style="display: flex; gap: 4px; align-items: center;">
                <button type="button" class="primary hpyprim hpyinl hpysm avtoutfitwear" title="Wear Outfit">
                    <span>Wear</span>
                </button>
                <button type="button" class="avtwrmv avtoutfitdel" title="Delete Outfit" aria-label="Delete Outfit">
                    <i class="fa-solid fa-xmark"></i>
                </button>
            </div>
        </div>
    `;
};

window.refreshAvatarOutfits = async function() {
    const list = document.getElementById('avtOutfits');
    if (!list) return;

    try {
        const res = await fetch('/api/v1/avatar/outfits', {
            headers: { 'Accept': 'application/json' }
        });
        if (!res.ok) throw new Error('request failed');
        const outfits = await res.json();
        if (!Array.isArray(outfits) || !outfits.length) {
            list.innerHTML = '<div class="feedemt">No saved outfits yet.</div>';
            return;
        }
        list.innerHTML = outfits.map(window.avatarOutfitChipHTML).join('');
    } catch {}
};

window.refreshAvatarPreview = function() {
    const img = document.getElementById('avtPreview');
    const box = document.getElementById('avtChrBox');
    const lay = document.querySelector('.avtlay');
    if (!img || !lay || !lay.dataset.userid) return;
    if (box) box.classList.add('loading');
    img.src = '/avatar/' + lay.dataset.userid + '.png?t=' + Date.now();
};

window.avatarPost = async function(url, params) {
    const res = await fetch(url, {
        method: 'POST',
        headers: {
            'Accept': 'application/json',
            'Content-Type': 'application/x-www-form-urlencoded'
        },
        body: new URLSearchParams(params)
    });

    let data = null;
    try { data = await res.json(); } catch (e) {}

    if (!res.ok || (data && data.error)) {
        throw new Error((data && data.error) || 'Something went wrong. Please try again.');
    }
    return data;
};

window.avatarShowError = function(message) {
    if (window.showModal) {
        window.showModal({
            title: 'Oops!',
            body: `<p style="font-family: 'Ubuntu', sans-serif; font-size: 14px; color: #3D3D3D; line-height: 1.5; margin: 0;">${avtEsc(message)}</p>`
        });
    }
};

window.updateAvatarSaveBtn = function() {
    const btn = document.getElementById('avtSaveBtn');
    if (btn) btn.disabled = !window.avatarDirty;
};

window.markAvatarDirty = function() {
    window.avatarDirty = true;
    window.updateAvatarSaveBtn();
};

window.setAvatarItem = async function(type, id, wear, btn) {
    if (btn) btn.disabled = true;
    try {
        await window.avatarPost(wear ? '/avatar/wear' : '/avatar/unwear', { type: type, id: id });
        await Promise.all([window.fetchAvatarInventory(), window.refreshAvatarWearing()]);
        window.refreshAvatarPreview();
        window.markAvatarDirty();
    } catch (e) {
        window.avatarShowError(e.message);
        if (btn) btn.disabled = false;
    }
};

window.redrawAvatar = async function(event) {
    const btn = event ? event.currentTarget : null;
    if (btn) btn.disabled = true;
    try {
        await window.avatarPost('/avatar/rerender', {});
        window.refreshAvatarPreview();
    } catch (e) {
        window.avatarShowError(e.message);
    }
    if (btn) btn.disabled = false;
};

window.openSaveOutfitModal = function() {
    if (!window.showModal) return;
    window.showModal({
        title: 'Save Outfit',
        subtitle: 'Save your current avatar setup',
        body: `
            <form action="javascript:void(0);" onsubmit="window.saveCurrentOutfit(event)" style="display: flex; flex-direction: column; gap: 12px;">
                <div class="setinbox">
                    <input type="text" id="outfitNameInput" class="setin" placeholder="Outfit Name" maxlength="30" required autocomplete="off">
                </div>
                <div style="display: flex; justify-content: flex-end; gap: 8px; margin-top: 4px;">
                    <button type="button" class="primary hpyerr hpyinl hpysm" onclick="window.closeModal()">
                        <span>Cancel</span>
                    </button>
                    <button type="submit" class="primary hpysuc hpyinl hpysm" id="saveOutfitSubmitBtn">
                        <span>Save Outfit</span>
                    </button>
                </div>
            </form>
        `
    });
};

window.saveCurrentOutfit = async function(event) {
    if (event) event.preventDefault();
    const input = document.getElementById('outfitNameInput');
    const submitBtn = document.getElementById('saveOutfitSubmitBtn');
    if (!input) return;

    const name = input.value.trim();
    if (!name) return;

    if (submitBtn) submitBtn.disabled = true;

    try {
        await window.avatarPost('/avatar/outfit/save', { name: name });
        if (window.closeModal) window.closeModal();
        await window.refreshAvatarOutfits();
    } catch (e) {
        window.avatarShowError(e.message);
        if (submitBtn) submitBtn.disabled = false;
    }
};

window.confirmWearOutfit = function(outfitId, btn, outfitName) {
    if (!window.showModal) {
        window.wearOutfit(outfitId, btn);
        return;
    }
    const safeName = avtEsc(outfitName || 'this outfit');
    window.showModal({
        title: 'Wear Outfit',
        bodyStyle: 'gap: 16px;',
        body: `
            <p style="font-family: 'Ubuntu', sans-serif; font-size: 14px; color: #3D3D3D; line-height: 1.5; margin: 0;">
                Are you sure you want to wear <strong>${safeName}</strong>? This will replace your currently worn items and body colors.
            </p>
            <div style="display: flex; justify-content: flex-end; gap: 8px; margin-top: 4px;">
                <button type="button" class="primary hpyerr hpyinl hpysm" onclick="window.closeModal()">
                    <span>Cancel</span>
                </button>
                <button type="button" class="primary hpyprim hpyinl hpysm" onclick="window.confirmedWearOutfit(${outfitId})">
                    <span>Confirm</span>
                </button>
            </div>
        `
    });
};

window.confirmedWearOutfit = function(outfitId) {
    window.closeModal();
    window.wearOutfit(outfitId, null);
};

window.wearOutfit = async function(outfitId, btn) {
    if (btn) btn.disabled = true;
    try {
        await window.avatarPost('/avatar/outfit/wear', { id: outfitId });
        await Promise.all([window.fetchAvatarInventory(), window.refreshAvatarWearing()]);
        window.refreshAvatarPreview();
        window.markAvatarDirty();
    } catch (e) {
        window.avatarShowError(e.message);
        if (btn) btn.disabled = false;
    }
};

window.deleteOutfit = async function(outfitId, btn) {
    if (btn) btn.disabled = true;
    try {
        await window.avatarPost('/avatar/outfit/delete', { id: outfitId });
        await window.refreshAvatarOutfits();
    } catch (e) {
        window.avatarShowError(e.message);
        if (btn) btn.disabled = false;
    }
};

window.openAvatarColorPopover = function() {
    const popover = document.getElementById('avtColorPopover');
    if (!popover) return;
    popover.classList.add('active');
};

window.closeAvatarColorPopover = function() {
    const popover = document.getElementById('avtColorPopover');
    if (popover) {
        popover.classList.remove('active');
    }
};

window.avatarConfirmedColor = function(part) {
    if (part === 'all') {
        return window.avatarPartColors.head || '#f3b700';
    }
    if (window.avatarPartColors[part]) {
        return window.avatarPartColors[part];
    }
    const el = document.querySelector('.avtpart[data-part="' + part + '"]');
    if (el) {
        const bg = el.dataset.color || el.style.fill || el.getAttribute('fill');
        if (bg) {
            const hex = rgbToHex(bg);
            window.avatarPartColors[part] = hex;
            return hex;
        }
    }
    return '#f3b700';
};

window.paintAvatarFigure = function(part, hex) {
    const paint = function(el) {
        el.setAttribute('fill', hex);
        el.dataset.color = hex;
    };
    if (part === 'all') {
        document.querySelectorAll('.avtpart[data-part]').forEach(el => {
            if (el.dataset.part !== 'all') {
                paint(el);
            }
        });
    } else {
        const el = document.querySelector('.avtpart[data-part="' + part + '"]');
        if (el) {
            paint(el);
        }
    }
};

window.setAvatarPickerColor = function(hex) {
    if (!window.avatarColorPicker) return;
    window.avatarColorPicker.setColor(hex);
};

window.initAvatarColorPicker = function() {
    const el = document.getElementById('avtColorPicker');
    if (!el || el.dataset.iroInit) return;
    el.dataset.iroInit = '1';

    window.avatarColorPicker = new window.GlobalColorPicker(el, {
        width: 170,
        initialColor: window.avatarConfirmedColor(window.avatarPart),
        onChange: function(hex) {
            window.paintAvatarFigure(window.avatarPart, hex);
        },
        onInputEnd: function(hex) {
            if (hex.toLowerCase() === window.avatarConfirmedColor(window.avatarPart).toLowerCase()) return;
            window.applyAvatarColor(hex);
        }
    });
};

window.selectAvatarPart = function(part, name, userTriggered = false) {
    window.avatarPart = part;
    document.querySelectorAll('.avtpart[data-part]').forEach(el => {
        el.classList.toggle('sel', el.dataset.part === part);
    });
    const nameEl = document.getElementById('avtPartName');
    if (nameEl) nameEl.textContent = 'Paint ' + name;

    window.setAvatarPickerColor(window.avatarConfirmedColor(part));

    if (userTriggered) {
        window.openAvatarColorPopover();
    }
};

window.applyAvatarColor = async function(hex) {
    const part = window.avatarPart;
    const clean = hex.replace('#', '');

    try {
        await window.avatarPost('/avatar/color', { part: part, color: clean });

        if (part === 'all') {
            ['head', 'torso', 'larm', 'rarm', 'lleg', 'rleg'].forEach(p => {
                window.avatarPartColors[p] = hex;
            });
        } else {
            window.avatarPartColors[part] = hex;
        }

        window.refreshAvatarPreview();
        window.markAvatarDirty();
    } catch (e) {
        ['head', 'torso', 'larm', 'rarm', 'lleg', 'rleg'].forEach(p => {
            if (window.avatarPartColors[p]) window.paintAvatarFigure(p, window.avatarPartColors[p]);
        });
        window.setAvatarPickerColor(window.avatarConfirmedColor(window.avatarPart));
        window.avatarShowError(e.message);
    }
};

const initAvatarPage = () => {
    const lay = document.querySelector('.avtlay');
    if (!lay) return;

    window.avatarPartColors = {};
    document.querySelectorAll('.avtpart[data-part]').forEach(el => {
        const part = el.dataset.part;
        if (!part || part === 'all') return;
        const bg = el.dataset.color || el.style.backgroundColor || el.style.background;
        if (bg) {
            window.avatarPartColors[part] = rgbToHex(bg);
        }
    });

    window.avatarCategory = 'all';
    window.avatarDirty = false;
    window.updateAvatarSaveBtn();

    window.initAvatarColorPicker();
    window.selectAvatarPart('all', 'All Parts', false);

    const preview = document.getElementById('avtPreview');
    const chrBox = document.getElementById('avtChrBox');
    if (preview && chrBox) {
        preview.onload = function() {
            chrBox.classList.remove('loading');
        };
        preview.onerror = function() {
            if (!preview.dataset.fbk) {
                preview.dataset.fbk = '1';
                preview.src = '/static/useful/temp/pfp.png';
            }
            chrBox.classList.remove('loading');
        };
        if (preview.complete) {
            chrBox.classList.remove('loading');
            if (preview.naturalWidth === 0) {
                preview.onerror();
            }
        }
    }

    const searchInput = document.getElementById('avtSearch');
    if (searchInput) {
        searchInput.oninput = function() {
            if (window.avatarSearchTimer) clearTimeout(window.avatarSearchTimer);
            window.avatarSearchTimer = setTimeout(() => window.fetchAvatarInventory(), 300);
        };
    }

    if (!window.avatarClickBound) {
        document.addEventListener('click', function(event) {
            const part = event.target.closest('.avtpart[data-part]');
            if (part) {
                window.selectAvatarPart(part.dataset.part, part.dataset.name || part.dataset.part, true);
                return;
            }

            const itemBtn = event.target.closest('.avtitembtn');
            if (itemBtn) {
                const card = itemBtn.closest('.avtitm');
                if (card) {
                    window.setAvatarItem(card.dataset.type, card.dataset.id, !card.classList.contains('worn'), itemBtn);
                }
                return;
            }

            const removeBtn = event.target.closest('.avtwrmv:not(.avtoutfitdel)');
            if (removeBtn) {
                const chip = removeBtn.closest('.avtwchip');
                if (chip && chip.dataset.type) {
                    window.setAvatarItem(chip.dataset.type, chip.dataset.id, false, null);
                }
                return;
            }

            const outfitWearBtn = event.target.closest('.avtoutfitwear');
            if (outfitWearBtn) {
                const chip = outfitWearBtn.closest('.avtwchip');
                if (chip && chip.dataset.outfitId) {
                    const nameEl = chip.querySelector('.avtwnm');
                    window.confirmWearOutfit(chip.dataset.outfitId, outfitWearBtn, nameEl ? nameEl.textContent : '');
                }
                return;
            }

            const outfitDelBtn = event.target.closest('.avtoutfitdel');
            if (outfitDelBtn) {
                const chip = outfitDelBtn.closest('.avtwchip');
                if (chip && chip.dataset.outfitId) {
                    window.deleteOutfit(chip.dataset.outfitId, outfitDelBtn);
                }
                return;
            }

            if (!event.target.closest('#avtColorPopover') && !event.target.closest('.avtpart')) {
                window.closeAvatarColorPopover();
            }
        });

        const markLoaded = function(event) {
            const img = event.target;
            if (img && img.tagName === 'IMG' && img.classList.contains('avtldimg')) {
                img.classList.add('ld');
            }
        };
        document.addEventListener('load', markLoaded, true);
        document.addEventListener('error', markLoaded, true);

        window.avatarClickBound = true;
    }

    document.querySelectorAll('.avtldimg').forEach(img => {
        if (img.complete) img.classList.add('ld');
    });

    setTimeout(window.updateAvatarTabIndicator, 50);
    setTimeout(window.updateAvatarTabIndicator, 150);
    setTimeout(window.updateAvatarTabIndicator, 350);
};

initAvatarPage();

document.addEventListener('htmx:afterSettle', initAvatarPage);

window.addEventListener('resize', () => {
    window.updateAvatarTabIndicator();
});
