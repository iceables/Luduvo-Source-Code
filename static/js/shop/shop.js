let currentShopPage = 1;
let currentShopCat = 'all';
let currentShopCurrencies = new Set(['all']);
let shopSearchTimer = null;
let loadingShop = false;
let noMoreShopItems = false;

const shopEsc = function(str) {
    if (window.escapeHTML) return window.escapeHTML(String(str));
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
};

window.toggleShopDropdown = function(dropdownId) {
    const dropdown = document.getElementById(dropdownId);
    if (!dropdown) return;

    const isOpen = dropdown.classList.contains('open');

    document.querySelectorAll('.shp-dropdown.open').forEach(d => {
        if (d !== dropdown) d.classList.remove('open');
    });

    dropdown.classList.toggle('open', !isOpen);
};

window.selectShopDropdownOption = function(dropdownId, value, label) {
    const dropdown = document.getElementById(dropdownId);
    if (!dropdown) return;

    const labelEl = dropdown.querySelector('#shopSortLabel');
    const inputEl = dropdown.querySelector('#shopSortSelect');

    if (labelEl) labelEl.textContent = label;
    if (inputEl) inputEl.value = value;

    dropdown.querySelectorAll('.shp-dropdown-item').forEach(item => {
        item.classList.toggle('active', item.dataset.value === value);
    });

    dropdown.classList.remove('open');
    window.fetchShopItems(1, false);
};

window.toggleShopSidebar = function() {
    const sidebar = document.querySelector('.shp-sidebar');
    if (!sidebar) return;

    const isCollapsed = sidebar.classList.toggle('collapsed');

    try {
        localStorage.setItem('shopFilterCollapsed', isCollapsed ? 'true' : 'false');
    } catch (e) {}
};

window.toggleShopFilterSidebar = function() {
    window.toggleShopSidebar();
};

window.toggleShopCategoryCard = function() {
    window.toggleShopSidebar();
};

window.switchShopCategoryTab = function(cat, event) {
    const clickedTab = event ? event.currentTarget : null;
    if (!clickedTab || clickedTab.classList.contains('active')) return;

    document.querySelectorAll('.shp-cat-item').forEach(tab => tab.classList.remove('active'));
    clickedTab.classList.add('active');

    currentShopCat = cat;
    window.fetchShopItems(1, false);
};

window.selectShopCurrencyPill = function(currency, btn) {
    if (!btn) return;

    if (currency === 'all') {
        currentShopCurrencies.clear();
        currentShopCurrencies.add('all');
        document.querySelectorAll('.shp-currency-pills .crttypebtn').forEach(p => {
            p.classList.toggle('active', p.dataset.currency === 'all');
        });
    } else {
        currentShopCurrencies.delete('all');
        const allBtn = document.querySelector('.shp-currency-pills .crttypebtn[data-currency="all"]');
        if (allBtn) allBtn.classList.remove('active');

        if (currentShopCurrencies.has(currency)) {
            currentShopCurrencies.delete(currency);
            btn.classList.remove('active');
        } else {
            currentShopCurrencies.add(currency);
            btn.classList.add('active');
        }

        if (currentShopCurrencies.size === 0) {
            currentShopCurrencies.add('all');
            if (allBtn) allBtn.classList.add('active');
        }
    }

    window.fetchShopItems(1, false);
};

const shopOffRibbonHTML = `<div class="shp-ribbon shp-ribbon-off">
    <img src="/static/useful/icons/off.png" class="shp-ribicon" alt="Offsale">
    <span class="shp-ribtext">OFFSALE</span>
</div>`;

const shopSpecialRibbonHTML = `<div class="shp-ribbon shp-ribbon-special">
    <img src="/static/useful/icons/special.png" class="shp-ribicon" alt="Special">
    <span class="shp-ribtext">SPECIAL</span>
</div>`;

const shop3DaysRibbonHTML = `<div class="shp-ribbon shp-ribbon-3days">
    <img src="/static/useful/icons/limt.png" class="shp-ribicon" alt="Timed">
    <span class="shp-ribtext">3 DAYS</span>
</div>`;

window.shopItemCardHTML = function(item) {
    const safeName = shopEsc(item.name);
    const safeCreator = shopEsc(item.creator_name);
    const safeType = shopEsc(item.type);
    const isOffsale = item.onsale === 'false' || item.offsale === 'true';
    const isSpecial = item.special === 'true';
    const isTimer = item.timer === 'true' || item.limited_time === 'true' || Boolean(item.time_left || item.days_left);

    let badgeHtml = '';
    let ribbonHtml = '';
    if (isOffsale) {
        ribbonHtml = shopOffRibbonHTML;
    } else if (isSpecial) {
        ribbonHtml = shopSpecialRibbonHTML;
    } else if (isTimer) {
        const timerText = shopEsc(item.timer_text || (item.days_left ? `${item.days_left} DAYS` : '3 DAYS'));
        ribbonHtml = `<div class="shp-ribbon shp-ribbon-3days">
            <img src="/static/useful/icons/limt.png" class="shp-ribicon" alt="Timed">
            <span class="shp-ribtext">${timerText}</span>
        </div>`;
    } else {
        if (item.bucks === 0 && item.bits === 0) {
            badgeHtml = '<span class="avtitmbdg" style="background: rgba(40, 167, 69, 0.85);">Free</span>';
        }
    }

    let priceHtml = '';
    if (isOffsale) {
        priceHtml = '<span class="shp-price-off">Offsale</span>';
    } else if (item.bucks === 0 && item.bits === 0) {
        priceHtml = '<span class="shp-price-free">Free</span>';
    } else {
        if (item.bucks > 0) {
            priceHtml += `<span class="shp-price-val">
                <img src="/static/useful/icons/verticec.png" class="shp-price-ico" alt="Vertices">
                <span>${item.bucks}</span>
            </span>`;
        }
        if (item.bits > 0) {
            priceHtml += `<span class="shp-price-val">
                <img src="/static/useful/icons/ticketc.png" class="shp-price-ico" alt="Tickets">
                <span>${item.bits}</span>
            </span>`;
        }
    }

    return `
        <div class="shp-card" data-id="${item.id}">
            <div class="avtitmimg${isOffsale ? ' shp-offimg' : ''}">
                <img src="/api/v1/avatar/shop/${encodeURIComponent(safeType)}/${item.id}" alt="${safeName}" loading="lazy" class="avtldimg" onerror="this.src='/static/useful/temp/pfp.png'; this.classList.add('ld');" onload="this.classList.add('ld');">
                ${badgeHtml}
            </div>
            ${ribbonHtml}
            <div class="avtitmmeta">
                <span class="avtitmnm">${safeName}</span>
                <span class="avtitmcr">by ${safeCreator}</span>
            </div>
            <div class="shp-price-row">
                ${priceHtml}
            </div>
        </div>
    `;
};

window.checkViewportFill = function() {
    if (loadingShop || noMoreShopItems) return;

    const grid = document.getElementById('shopGrid');
    if (!grid) return;

    const windowHeight = window.innerHeight || document.documentElement.clientHeight;
    const fullHeight = Math.max(
        document.body.scrollHeight,
        document.body.offsetHeight,
        document.documentElement.clientHeight,
        document.documentElement.scrollHeight,
        document.documentElement.offsetHeight
    );

    if (fullHeight <= windowHeight + 150) {
        window.fetchShopItems(currentShopPage + 1, true);
    }
};

window.fetchShopItems = function(page = 1, append = false) {
    if (loadingShop) return;
    if (append && noMoreShopItems) return;

    const grid = document.getElementById('shopGrid');
    if (!grid) return;

    loadingShop = true;

    const spinnerHtml = '<div id="shopScrollSpinner" class="musld" style="grid-column: 1 / -1;"><svg class="dmspn" viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg">' +
        '<rect class="sq sq-1" x="43" y="7" width="14" height="14"/>' +
        '<rect class="sq sq-2" x="25" y="25" width="14" height="14"/>' +
        '<rect class="sq sq-3" x="7" y="43" width="14" height="14"/>' +
        '<rect class="sq sq-4" x="25" y="61" width="14" height="14"/>' +
        '<rect class="sq sq-5" x="43" y="79" width="14" height="14"/>' +
        '<rect class="sq sq-6" x="61" y="61" width="14" height="14"/>' +
        '<rect class="sq sq-7" x="79" y="43" width="14" height="14"/>' +
        '<rect class="sq sq-8" x="61" y="25" width="14" height="14"/>' +
        '</svg></div>';

    if (!append) {
        currentShopPage = 1;
        noMoreShopItems = false;
        grid.innerHTML = spinnerHtml;
    } else {
        const existingSpinner = document.getElementById('shopScrollSpinner');
        if (!existingSpinner) {
            grid.insertAdjacentHTML('beforeend', spinnerHtml);
        }
    }

    const barSearch = document.getElementById('shopBarSearch');
    const sideSearch = document.getElementById('shopSearchInput');
    const q = (sideSearch && sideSearch.value.trim()) || (barSearch && barSearch.value.trim()) || '';

    const sortEl = document.getElementById('shopSortSelect');
    const sort = sortEl ? sortEl.value : 'newest';

    const minPriceEl = document.getElementById('shopMinPrice');
    const minPrice = minPriceEl ? minPriceEl.value.trim() : '0';

    const maxPriceEl = document.getElementById('shopMaxPrice');
    const maxPrice = maxPriceEl ? maxPriceEl.value.trim() : '0';

    const creatorEl = document.getElementById('shopCreatorInput');
    const creator = creatorEl ? creatorEl.value.trim() : '';

    const currencyStr = Array.from(currentShopCurrencies).join(',');

    const params = new URLSearchParams({
        category: currentShopCat,
        q: q,
        currency: currencyStr,
        sort: sort,
        min_price: minPrice,
        max_price: maxPrice,
        creator: creator,
        page: page,
        limit: 20
    });

    fetch('/api/v1/shop/items?' + params.toString())
        .then(res => res.json())
        .then(data => {
            const spinner = document.getElementById('shopScrollSpinner');
            if (spinner) spinner.remove();

            if (!data || data.error) {
                if (!append) {
                    grid.innerHTML = '<div class="feedemt" style="grid-column: 1 / -1;">Failed to load shop items.</div>';
                }
                loadingShop = false;
                return;
            }

            const totalPages = Math.max(1, data.total_pages || 1);

            if (!data.items || data.items.length === 0) {
                if (!append) {
                    grid.innerHTML = '<div class="feedemt" style="grid-column: 1 / -1;">No items found matching your filters.</div>';
                }
                noMoreShopItems = true;
            } else {
                const cardsHtml = data.items.map(item => window.shopItemCardHTML(item)).join('');
                if (!append) {
                    grid.innerHTML = cardsHtml;
                } else {
                    grid.insertAdjacentHTML('beforeend', cardsHtml);
                }

                document.querySelectorAll('.avtldimg').forEach(img => {
                    if (img.complete) {
                        img.classList.add('ld');
                    } else {
                        img.addEventListener('load', () => {
                            img.classList.add('ld');
                            window.checkViewportFill();
                        });
                        img.addEventListener('error', () => {
                            img.classList.add('ld');
                        });
                    }
                });

                currentShopPage = page;
                if (page >= totalPages) {
                    noMoreShopItems = true;
                }
            }

            loadingShop = false;
            setTimeout(() => {
                window.checkViewportFill();
            }, 100);

            try {
                const url = new URL(window.location.href);
                if (q) url.searchParams.set('q', q); else url.searchParams.delete('q');
                if (currentShopCat !== 'all') url.searchParams.set('cat', currentShopCat); else url.searchParams.delete('cat');
                if (currencyStr && currencyStr !== 'all') url.searchParams.set('currency', currencyStr); else url.searchParams.delete('currency');
                if (sort !== 'newest') url.searchParams.set('sort', sort); else url.searchParams.delete('sort');
                if (page > 1) url.searchParams.set('page', page); else url.searchParams.delete('page');
                window.history.replaceState({}, document.title, url.search ? url.pathname + url.search : url.pathname);
            } catch (e) {}
        })
        .catch(() => {
            const spinner = document.getElementById('shopScrollSpinner');
            if (spinner) spinner.remove();
            if (!append) {
                grid.innerHTML = '<div class="feedemt" style="grid-column: 1 / -1;">Error loading shop items.</div>';
            }
            loadingShop = false;
        });
};

window.handleShopScroll = function() {
    if (loadingShop || noMoreShopItems) return;

    const grid = document.getElementById('shopGrid');
    if (!grid) return;

    const scrollTop = window.scrollY || window.pageYOffset || document.documentElement.scrollTop || 0;
    const windowHeight = window.innerHeight || document.documentElement.clientHeight;
    const fullHeight = Math.max(
        document.body.scrollHeight,
        document.body.offsetHeight,
        document.documentElement.clientHeight,
        document.documentElement.scrollHeight,
        document.documentElement.offsetHeight
    );

    if (scrollTop + windowHeight >= fullHeight - 350) {
        window.fetchShopItems(currentShopPage + 1, true);
    }
};

window.resetShopFilters = function() {
    const barSearch = document.getElementById('shopBarSearch');
    const sideSearch = document.getElementById('shopSearchInput');
    if (barSearch) barSearch.value = '';
    if (sideSearch) sideSearch.value = '';

    currentShopCurrencies.clear();
    currentShopCurrencies.add('all');
    document.querySelectorAll('.shp-currency-pills .crttypebtn').forEach(p => {
        p.classList.toggle('active', p.dataset.currency === 'all');
    });

    const sortInput = document.getElementById('shopSortSelect');
    if (sortInput) sortInput.value = 'newest';

    const sortLabel = document.getElementById('shopSortLabel');
    if (sortLabel) sortLabel.textContent = 'Newest';

    document.querySelectorAll('#shopSortMenu .shp-dropdown-item').forEach(item => {
        item.classList.toggle('active', item.dataset.value === 'newest');
    });

    const minPriceEl = document.getElementById('shopMinPrice');
    if (minPriceEl) minPriceEl.value = '';

    const maxPriceEl = document.getElementById('shopMaxPrice');
    if (maxPriceEl) maxPriceEl.value = '';

    const creatorEl = document.getElementById('shopCreatorInput');
    if (creatorEl) creatorEl.value = '';

    document.querySelectorAll('.shp-cat-item').forEach(tab => {
        tab.classList.toggle('active', tab.dataset.cat === 'all');
    });
    currentShopCat = 'all';

    window.fetchShopItems(1, false);
};

const initShopPage = () => {
    const activeTab = document.querySelector('.shp-cat-item.active');
    if (activeTab && activeTab.dataset.cat) {
        currentShopCat = activeTab.dataset.cat;
    }

    currentShopCurrencies.clear();
    const activePills = document.querySelectorAll('.shp-currency-pills .crttypebtn.active');
    if (activePills.length > 0) {
        activePills.forEach(p => {
            if (p.dataset.currency) currentShopCurrencies.add(p.dataset.currency);
        });
    } else {
        currentShopCurrencies.add('all');
    }

    if (localStorage.getItem('shopFilterCollapsed') === 'true') {
        const sidebar = document.querySelector('.shp-sidebar');
        if (sidebar) sidebar.classList.add('collapsed');
    }

    const barSearch = document.getElementById('shopBarSearch');
    const sideSearch = document.getElementById('shopSearchInput');

    const handleSearchInput = function() {
        if (shopSearchTimer) clearTimeout(shopSearchTimer);
        shopSearchTimer = setTimeout(() => window.fetchShopItems(1, false), 300);
    };

    if (barSearch) barSearch.oninput = handleSearchInput;
    if (sideSearch) sideSearch.oninput = handleSearchInput;

    const markLoaded = function(event) {
        const img = event.target;
        if (img && img.tagName === 'IMG' && img.classList.contains('avtldimg')) {
            img.classList.add('ld');
        }
    };
    document.addEventListener('load', markLoaded, true);
    document.addEventListener('error', markLoaded, true);

    document.querySelectorAll('.avtldimg').forEach(img => {
        if (img.complete) img.classList.add('ld');
    });

    window.removeEventListener('scroll', window.handleShopScroll);
    window.addEventListener('scroll', window.handleShopScroll);
    window.removeEventListener('resize', window.checkViewportFill);
    window.addEventListener('resize', window.checkViewportFill);
};

document.addEventListener('click', function(event) {
    if (!event.target.closest('.shp-dropdown')) {
        document.querySelectorAll('.shp-dropdown.open').forEach(d => d.classList.remove('open'));
    }
});

initShopPage();

document.addEventListener('htmx:afterSettle', initShopPage);
document.addEventListener('htmx:beforeTransition', function() {
    window.removeEventListener('scroll', window.handleShopScroll);
    window.removeEventListener('resize', window.checkViewportFill);
});