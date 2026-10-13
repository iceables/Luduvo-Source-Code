window.createType = 'shop';
window.createCat = 'hat';
window.assetCategory = 'image';

var ASSET_TYPE_INFO = {
    image: {
        name: 'Image',
        accept: 'image/png,image/jpeg,image/webp,image/gif',
        exts: ['png', 'jpg', 'jpeg', 'webp', 'gif'],
        max: 8 * 1024 * 1024,
        formats: 'PNG, JPG, WEBP, GIF',
        limit: 'Up to 8 MB',
        msg: 'Choose a PNG, JPG, WEBP or GIF'
    },
    mesh: {
        name: 'Mesh',
        accept: '.glb,.obj',
        exts: ['glb', 'obj'],
        max: 25 * 1024 * 1024,
        formats: 'GLB, OBJ',
        limit: 'Up to 25 MB',
        msg: 'Choose a GLB or OBJ'
    },
    sound: {
        name: 'Sound',
        accept: '.mp3,.wav,.ogg',
        exts: ['mp3', 'wav', 'ogg'],
        max: 15 * 1024 * 1024,
        formats: 'MP3, WAV, OGG',
        limit: 'Up to 15 MB',
        msg: 'Choose an MP3, WAV or OGG'
    }
};

var CRT_CAT_NAMES = {
    hat: 'Hat',
    face: 'Face',
    shirt: 'Shirt',
    tshirt: 'T-Shirt',
    pants: 'Pants',
    tool: 'Tool'
};

function crtEsc(str) {
    if (window.escapeHTML) return window.escapeHTML(String(str));
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

window.updateCreateViewportHeight = function() {
    var viewport = document.querySelector('.crtvwp');
    var active = document.querySelector('.crtvwp > .settabcntnt.active');
    if (!viewport || !active) return;
    viewport.style.height = active.offsetHeight + 'px';
    if (window.createHeightTimeout) clearTimeout(window.createHeightTimeout);
    window.createHeightTimeout = setTimeout(function() {
        if (viewport) viewport.style.height = '';
    }, 350);
};

window.switchCreateType = function(type, event) {
    var tab = event ? event.currentTarget : null;
    if (!tab || tab.classList.contains('active')) return;

    var current = document.querySelector('.crtvwp > .settabcntnt.active');
    var target = document.getElementById('crt-' + type);
    if (!target) return;

    document.querySelectorAll('.crttypebtn').forEach(function(btn) {
        btn.classList.remove('active');
    });
    tab.classList.add('active');

    var viewport = document.querySelector('.crtvwp');
    var currentHeight = current ? current.offsetHeight : 0;
    if (viewport) {
        viewport.style.height = currentHeight + 'px';
        viewport.offsetHeight;
    }

    if (current) current.classList.remove('active');
    target.classList.add('active');

    window.createType = type;

    if (type === 'shop') {
        window.updateShopCategoryIndicator();
    } else if (type === 'asset') {
        window.updateAssetCategoryIndicator();
    }

    var newHeight = target.offsetHeight;
    if (viewport) {
        viewport.style.height = newHeight + 'px';
    }

    if (window.createHeightTimeout) clearTimeout(window.createHeightTimeout);
    window.createHeightTimeout = setTimeout(function() {
        if (viewport) viewport.style.height = '';
    }, 350);

    window.updateCreateUrl();
};

window.updateShopCategoryIndicator = function() {
    var tabs = document.querySelector('.crtshoptabs');
    if (!tabs) return;
    var indicator = tabs.querySelector('.tabind');
    var activeTab = tabs.querySelector('.tabbtn.active');
    if (indicator && activeTab) {
        indicator.style.left = activeTab.offsetLeft + 'px';
        indicator.style.width = activeTab.offsetWidth + 'px';
        indicator.style.height = activeTab.offsetHeight + 'px';
        indicator.style.top = activeTab.offsetTop + 'px';
    }
};

window.switchShopCategory = function(cat, event) {
    var tab = event ? event.currentTarget : null;
    if (!tab || tab.classList.contains('active')) return;

    var tabs = document.querySelector('.crtshoptabs');
    if (tabs) {
        tabs.querySelectorAll('.tabbtn').forEach(function(t) {
            t.classList.remove('active');
        });
        tab.classList.add('active');
    }

    window.createCat = cat;
    window.updateShopCategoryIndicator();

    var viewport = document.querySelector('.crtvwp');
    var shopPane = document.getElementById('crt-shop');
    var currentHeight = shopPane ? shopPane.offsetHeight : 0;
    if (viewport) {
        viewport.style.height = currentHeight + 'px';
        viewport.offsetHeight;
    }

    var display = CRT_CAT_NAMES[cat] || cat;
    var modelField = document.getElementById('crtShopModelFld');
    if (modelField) {
        modelField.classList.toggle('visible', cat === 'hat' || cat === 'face' || cat === 'tool');
    }

    var titleEl = document.getElementById('crtShopTitle');
    if (titleEl) titleEl.textContent = 'Create ' + display;
    var catNameEl = document.getElementById('crtShopCatName');
    if (catNameEl) catNameEl.textContent = display;
    var input = document.getElementById('crtShopCatInput');
    if (input) input.value = cat;

    if (window.setCreateCategory) window.setCreateCategory(cat);
    if (window.refreshCreatePreview) window.refreshCreatePreview();
    window.updateShopPreviewName();

    var newHeight = shopPane ? shopPane.offsetHeight : 0;
    if (viewport) {
        viewport.style.height = newHeight + 'px';
    }

    if (window.createHeightTimeout) clearTimeout(window.createHeightTimeout);
    window.createHeightTimeout = setTimeout(function() {
        if (viewport) viewport.style.height = '';
    }, 350);

    window.updateCreateUrl();
};

window.updateShopPreviewName = function() {
    var nameInput = document.getElementById('crtShopName');
    var catName = CRT_CAT_NAMES[window.createCat] || window.createCat;
    var nameEl = document.getElementById('crtShopPreviewName');
    if (!nameEl) return;
    var value = nameInput ? nameInput.value.trim() : '';
    nameEl.textContent = value || ('Untitled ' + catName);
};

window.handleShopFile = function(event) {
    var input = event.target;
    var file = input.files && input.files[0];
    if (!file) return;
    var msgEl = document.getElementById('crtShopFileMsg');
    var nameEl = document.getElementById('crtShopFileName');
    var previewImg = document.getElementById('crtShopPreviewImg');
    if (window.setCreateTextureFile) window.setCreateTextureFile(file);
    if (nameEl) nameEl.textContent = file.name;
    if (msgEl) msgEl.textContent = 'Choose a different file';
    if (previewImg) {
        previewImg.classList.remove('ld');
        var url = URL.createObjectURL(file);
        previewImg.onload = function() {
            previewImg.classList.add('ld');
            URL.revokeObjectURL(url);
        };
        previewImg.src = url;
    }
    if (window.refreshCreatePreview) window.refreshCreatePreview();
};

window.updateAssetCategoryIndicator = function() {
    var tabs = document.querySelector('.crtassettabs');
    if (!tabs) return;
    var indicator = tabs.querySelector('.tabind');
    var activeTab = tabs.querySelector('.tabbtn.active');
    if (indicator && activeTab) {
        indicator.style.left = activeTab.offsetLeft + 'px';
        indicator.style.width = activeTab.offsetWidth + 'px';
        indicator.style.height = activeTab.offsetHeight + 'px';
        indicator.style.top = activeTab.offsetTop + 'px';
    }
};

window.updateAssetTab = function() {
    var info = ASSET_TYPE_INFO[window.assetCategory] || ASSET_TYPE_INFO.image;

    var titleEl = document.getElementById('crtAssetTitle');
    if (titleEl) titleEl.textContent = 'Create ' + info.name;

    var typeInput = document.getElementById('crtAssetTypeInput');
    if (typeInput) typeInput.value = window.assetCategory;

    var fileInput = document.getElementById('crtAssetFile');
    if (fileInput) fileInput.accept = info.accept;

    var msgEl = document.getElementById('crtAssetFileMsg');
    if (msgEl) msgEl.textContent = info.msg;

    var tipEl = document.getElementById('crtAssetFileTip');
    if (tipEl) tipEl.textContent = info.formats;

    var fmtEl = document.getElementById('crtAssetFormats');
    if (fmtEl) fmtEl.textContent = info.formats;

    var limEl = document.getElementById('crtAssetLimit');
    if (limEl) limEl.textContent = info.limit;

    var nameEl = document.getElementById('crtAssetFileName');
    if (nameEl) nameEl.textContent = '';

    if (fileInput) fileInput.value = '';

    var texFld = document.getElementById('crtAssetTextureFld');
    if (texFld) {
        texFld.style.display = window.assetCategory === 'mesh' ? 'flex' : 'none';
        var texInput = document.getElementById('crtAssetTextureFile');
        if (texInput) texInput.value = '';
        var texNameEl = document.getElementById('crtAssetTextureName');
        if (texNameEl) texNameEl.textContent = '';
        var texMsgEl = document.getElementById('crtAssetTextureMsg');
        if (texMsgEl) texMsgEl.textContent = 'Choose a PNG or JPEG texture';
    }

    var badgeEl = document.getElementById('crtAssetPreviewBadge');
    if (badgeEl) {
        badgeEl.textContent = info.name;
        badgeEl.className = 'avtitmbdg adm-badge-' + window.assetCategory;
    }

    var previewTypeEl = document.getElementById('crtAssetPreviewType');
    if (previewTypeEl) {
        previewTypeEl.textContent = info.name + ' Asset';
    }

    var imgEl = document.getElementById('crtAssetPreviewImg');
    var audioIcon = document.getElementById('crtAssetPreviewAudioIcon');
    var meshIcon = document.getElementById('crtAssetPreviewMeshIcon');
    var imgWrap = document.getElementById('crtAssetImgWrap');

    if (imgEl && audioIcon && meshIcon && imgWrap) {
        if (window.assetCategory === 'sound') {
            imgEl.style.display = 'none';
            meshIcon.style.display = 'none';
            audioIcon.style.display = 'block';
            imgWrap.classList.add('avtitmico');
        } else if (window.assetCategory === 'mesh') {
            imgEl.style.display = 'none';
            audioIcon.style.display = 'none';
            meshIcon.style.display = 'block';
            imgWrap.classList.add('avtitmico');
        } else {
            audioIcon.style.display = 'none';
            meshIcon.style.display = 'none';
            imgEl.style.display = 'block';
            imgEl.src = '/static/useful/temp/pfp.png';
            imgWrap.classList.remove('avtitmico');
        }
    }

    window.updateAssetPreviewName();
};

window.updateAssetPreviewName = function() {
    var nameInput = document.getElementById('crtAssetName');
    var info = ASSET_TYPE_INFO[window.assetCategory] || ASSET_TYPE_INFO.image;
    var nameEl = document.getElementById('crtAssetPreviewName');
    if (!nameEl) return;
    var value = nameInput ? nameInput.value.trim() : '';
    nameEl.textContent = value || ('Untitled ' + info.name);
};

window.switchAssetCategory = function(cat, event) {
    var tab = event ? event.currentTarget : null;
    if (!tab || tab.classList.contains('active')) return;

    var tabs = document.querySelector('.crtassettabs');
    if (tabs) {
        tabs.querySelectorAll('.tabbtn').forEach(function(t) {
            t.classList.remove('active');
        });
        tab.classList.add('active');
    }

    window.assetCategory = cat;
    window.updateAssetCategoryIndicator();

    var viewport = document.querySelector('.crtvwp');
    var assetPane = document.getElementById('crt-asset');
    var currentHeight = assetPane ? assetPane.offsetHeight : 0;
    if (viewport) {
        viewport.style.height = currentHeight + 'px';
        viewport.offsetHeight;
    }

    window.updateAssetTab();

    var newHeight = assetPane ? assetPane.offsetHeight : 0;
    if (viewport) {
        viewport.style.height = newHeight + 'px';
    }

    if (window.createHeightTimeout) clearTimeout(window.createHeightTimeout);
    window.createHeightTimeout = setTimeout(function() {
        if (viewport) viewport.style.height = '';
    }, 350);

    window.updateCreateUrl();
};

window.handleAssetFile = function(event) {
    var input = event.target;
    var file = input.files && input.files[0];
    if (!file) return;

    var info = ASSET_TYPE_INFO[window.assetCategory] || ASSET_TYPE_INFO.image;
    var ext = (file.name.split('.').pop() || '').toLowerCase();

    if (info.exts.indexOf(ext) === -1) {
        input.value = '';
        window.createShowError('That file type is not supported for ' + info.name + ' assets. Supported formats: ' + info.formats + '.');
        return;
    }
    if (file.size > info.max) {
        input.value = '';
        window.createShowError(info.name + ' assets must be ' + info.limit.toLowerCase() + '.');
        return;
    }

    var nameEl = document.getElementById('crtAssetFileName');
    var msgEl = document.getElementById('crtAssetFileMsg');
    if (nameEl) nameEl.textContent = file.name;
    if (msgEl) msgEl.textContent = 'Choose a different file';

    if (window.assetCategory === 'image') {
        var previewImg = document.getElementById('crtAssetPreviewImg');
        var audioIcon = document.getElementById('crtAssetPreviewAudioIcon');
        var meshIcon = document.getElementById('crtAssetPreviewMeshIcon');
        var imgWrap = document.getElementById('crtAssetImgWrap');
        if (previewImg && imgWrap) {
            if (audioIcon) audioIcon.style.display = 'none';
            if (meshIcon) meshIcon.style.display = 'none';
            previewImg.style.display = 'block';
            imgWrap.classList.remove('avtitmico');
            previewImg.classList.remove('ld');
            var url = URL.createObjectURL(file);
            previewImg.onload = function() {
                previewImg.classList.add('ld');
                URL.revokeObjectURL(url);
            };
            previewImg.src = url;
        }
    }
};

window.handleAssetTextureFile = function(event) {
    var input = event.target;
    var file = input.files && input.files[0];
    if (!file) return;

    var ext = (file.name.split('.').pop() || '').toLowerCase();
    var validExts = ['png', 'jpg', 'jpeg', 'webp', 'gif'];
    if (validExts.indexOf(ext) === -1) {
        input.value = '';
        window.createShowError('The texture must be a PNG, JPG, WEBP, or GIF image.');
        return;
    }

    var nameEl = document.getElementById('crtAssetTextureName');
    var msgEl = document.getElementById('crtAssetTextureMsg');
    if (nameEl) nameEl.textContent = file.name;
    if (msgEl) msgEl.textContent = 'Choose a different texture';
};

window.handleGameFile = function(event) {
    var input = event.target;
    var file = input.files && input.files[0];
    if (!file) return;
    var nameEl = document.getElementById('crtGameFileName');
    var msgEl = document.getElementById('crtGameFileMsg');
    if (nameEl) nameEl.textContent = file.name;
    if (msgEl) msgEl.textContent = 'Choose a different file';
};

window.handleShopModelFile = function(event) {
    var input = event.target;
    var file = input.files && input.files[0];
    if (!file) return;
    var ext = (file.name.split('.').pop() || '').toLowerCase();
    if (ext !== 'glb' && ext !== 'obj') {
        input.value = '';
        window.createShowError('The 3D model must be a GLB or OBJ file.');
        return;
    }
    var nameEl = document.getElementById('crtShopModelName');
    var msgEl = document.getElementById('crtShopModelMsg');
    var viewBtn = document.getElementById('crtShopModelViewBtn');
    window.setCreateModelFile(file);
    if (window.refreshCreatePreview) window.refreshCreatePreview();
    var transformEl = document.getElementById('crtShopModelTransform');
    if (transformEl) transformEl.value = '';
    if (nameEl) nameEl.textContent = file.name;
    if (msgEl) msgEl.textContent = 'Choose a different file';
    if (viewBtn) viewBtn.disabled = false;
};

window.createShowError = function(message) {
    if (!window.showModal) return;
    window.showModal({
        title: 'Hold up!',
        body: '<p style="font-family: \'Ubuntu\', sans-serif; font-size: 14px; color: #3D3D3D; line-height: 1.5; margin: 0;">' + crtEsc(message) + '</p>'
    });
};

window.createShowPending = function(title, body) {
    if (!window.showModal) return;
    window.showModal({
        title: title,
        body: '<p style="font-family: \'Ubuntu\', sans-serif; font-size: 14px; color: #3D3D3D; line-height: 1.5; margin: 0;">' + body + '</p>'
    });
};

window.submitShopItem = function(event) {
    event.preventDefault();
    var form = event.target;
    var nameInput = form.querySelector('#crtShopName');
    var priceInput = form.querySelector('#crtShopPrice');
    var fileInput = form.querySelector('#crtShopFile');

    var pane = document.querySelector('.pane[data-can-clothing]');
    if (pane) {
        var isClothing = (window.createCat === 'shirt' || window.createCat === 'tshirt' || window.createCat === 'pants');
        var isAccessory = (window.createCat === 'hat' || window.createCat === 'tool' || window.createCat === 'face');
        if (isClothing && pane.dataset.canClothing === 'false') {
            window.createShowError('To upload shirts, pants and t-shirts, your account must be at least 1 day old.');
            return false;
        }
        if (isAccessory && pane.dataset.canAccessories === 'false') {
            window.createShowError('To upload hats, tools and faces, your account must be at least 3 days old.');
            return false;
        }
    }

    var name = nameInput ? nameInput.value.trim() : '';
    var price = priceInput ? priceInput.value.trim() : '';
    var file = fileInput && fileInput.files && fileInput.files[0];

    if (!name) {
        window.createShowError('Please give your item a name.');
        return false;
    }
    if (price === '' || isNaN(Number(price)) || Number(price) < 0) {
        window.createShowError('Please set a valid price in Vertices.');
        return false;
    }

    if (window.createCat === 'hat' || window.createCat === 'face' || window.createCat === 'tool') {
        var modelFile = window.getCreateModelFile ? window.getCreateModelFile() : null;
        if (!modelFile) {
            window.createShowError('Please choose a 3D model for your item.');
            return false;
        }
        var modelExt = (modelFile.name.split('.').pop() || '').toLowerCase();
        if (modelExt === 'obj' && !file) {
            window.createShowError('OBJ models require an asset texture file.');
            return false;
        }
    } else {
        if (!file) {
            window.createShowError('Please choose an asset file to publish.');
            return false;
        }
    }

    var catName = CRT_CAT_NAMES[window.createCat] || window.createCat;
    var safeName = crtEsc(name);
    var transformEl = document.getElementById('crtShopModelTransform');
    if (transformEl && window.getCreateModelTransformData) {
        transformEl.value = JSON.stringify(window.getCreateModelTransformData());
    }
    window.createShowPending('Publish ' + catName,
        'Publishing <strong>' + safeName + '</strong> to the shop will open once item creation is enabled on the server.');
    return false;
};

window.submitGame = function(event) {
    event.preventDefault();
    var form = event.target;
    var nameInput = form.querySelector('#crtGameName');
    var name = nameInput ? nameInput.value.trim() : '';
    if (!name) {
        window.createShowError('Please give your game a name.');
        return false;
    }
    window.createShowPending('Create Game',
        'Game creation will open here once game publishing is enabled on the server.');
    return false;
};

window.submitAsset = function(event) {
    event.preventDefault();
    var form = event.target;
    var nameInput = form.querySelector('#crtAssetName');
    var fileInput = form.querySelector('#crtAssetFile');
    var name = nameInput ? nameInput.value.trim() : '';
    var file = fileInput && fileInput.files && fileInput.files[0];

    var pane = document.querySelector('.pane[data-can-image]');
    if (pane) {
        if (window.assetCategory === 'image' && pane.dataset.canImage === 'false') {
            window.createShowError('To upload an image asset, your account must be at least 3 days old.');
            return false;
        }
        if ((window.assetCategory === 'mesh' || window.assetCategory === 'sound') && pane.dataset.canMeshSound === 'false') {
            window.createShowError('To upload a mesh or a sound, your account must be at least 7 days old.');
            return false;
        }
    }

    if (!name) {
        window.createShowError('Please give your asset a name.');
        return false;
    }
    if (!file) {
        window.createShowError('Please choose a file to upload.');
        return false;
    }

    var info = ASSET_TYPE_INFO[window.assetCategory] || ASSET_TYPE_INFO.image;
    var ext = (file.name.split('.').pop() || '').toLowerCase();

    if (info.exts.indexOf(ext) === -1) {
        window.createShowError('That file type is not supported for ' + info.name + ' assets. Supported formats: ' + info.formats + '.');
        return false;
    }
    if (file.size > info.max) {
        window.createShowError(info.name + ' assets must be ' + info.limit.toLowerCase() + '.');
        return false;
    }

    var fd = new FormData(form);
    fd.set('type', window.assetCategory);

    var btn = form.querySelector('button[type="submit"]');
    if (btn) btn.disabled = true;

    fetch('/create/assets', {
        method: 'POST',
        body: fd
    })
        .then(function(res) {
            if (res.redirected) {
                window.location.href = res.url;
                return;
            }
            if (!res.ok) {
                return res.json().then(function(data) {
                    throw new Error(data.error || 'Upload failed');
                });
            }
            window.location.href = '/create/assets';
        })
        .catch(function(err) {
            if (btn) btn.disabled = false;
            window.createShowError(err.message || 'Upload failed. Please try again.');
        });

    return false;
};

window.updateCreateUrl = function() {
    try {
        var url = new URL(window.location.href);
        url.searchParams.set('type', window.createType);
        if (window.createType === 'shop') {
            url.searchParams.set('cat', window.createCat);
        } else {
            url.searchParams.delete('cat');
        }
        if (window.createType === 'asset') {
            url.searchParams.set('asset', window.assetCategory);
        } else {
            url.searchParams.delete('asset');
        }
        var search = url.searchParams.toString();
        window.history.replaceState({}, document.title, search ? url.pathname + '?' + search : url.pathname);
    } catch (e) {}
};

(function() {
    var activeTypeTab = document.querySelector('.crttypebtn.active');
    if (activeTypeTab && activeTypeTab.dataset.type) {
        window.createType = activeTypeTab.dataset.type;
    }
    var activeCatTab = document.querySelector('.crtshoptabs .tabbtn.active');
    if (activeCatTab && activeCatTab.dataset.cat) {
        window.createCat = activeCatTab.dataset.cat;
    }
    var activeAssetTab = document.querySelector('.crtassettabs .tabbtn.active');
    if (activeAssetTab && activeAssetTab.dataset.assettype) {
        window.assetCategory = activeAssetTab.dataset.assettype;
    }
    if (window.setCreateCategory) window.setCreateCategory(window.createCat);

    var previewImg = document.getElementById('crtShopPreviewImg');
    if (previewImg && previewImg.complete && previewImg.naturalWidth > 0) {
        previewImg.classList.add('ld');
    } else if (previewImg) {
        previewImg.addEventListener('load', function() {
            previewImg.classList.add('ld');
        });
    }

    window.updateShopCategoryIndicator();
    window.updateAssetCategoryIndicator();
    window.updateAssetTab();
    window.updateCreateUrl();

    try {
        var errMsg = new URL(window.location.href).searchParams.get('error');
        if (errMsg) {
            window.createShowError(errMsg);
            var cleanUrl = new URL(window.location.href);
            cleanUrl.searchParams.delete('error');
            var search = cleanUrl.searchParams.toString();
            window.history.replaceState({}, document.title, search ? cleanUrl.pathname + '?' + search : cleanUrl.pathname);
        }
    } catch (e) {}

    setTimeout(function() {
        window.updateShopCategoryIndicator();
        window.updateAssetCategoryIndicator();
        window.updateCreateViewportHeight();
    }, 50);

    setTimeout(function() {
        window.updateShopCategoryIndicator();
        window.updateAssetCategoryIndicator();
        window.updateCreateViewportHeight();
    }, 150);

    setTimeout(function() {
        window.updateShopCategoryIndicator();
        window.updateAssetCategoryIndicator();
        window.updateCreateViewportHeight();
    }, 350);
})();

window.addEventListener('resize', function() {
    window.updateShopCategoryIndicator();
    window.updateAssetCategoryIndicator();
});