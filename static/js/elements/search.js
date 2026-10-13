let globalSearchTimer = null;
let globalSearchCache = new Map();
let currentSelectedIndex = -1;

const escapeSearchHTML = str => {
    if (!str) return '';
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
};

const isUserContextPage = () => {
    const p = window.location.pathname.toLowerCase();
    return p.startsWith('/user') || p.startsWith('/u/') || p.startsWith('/friends') || p.startsWith('/users');
};

const getSearchDropdown = pillEl => {
    let dropdown = pillEl.querySelector('.srch-dropdown');
    if (!dropdown) {
        dropdown = document.createElement('div');
        dropdown.className = 'srch-dropdown';
        pillEl.appendChild(dropdown);
    }
    return dropdown;
};

const renderSearchDropdown = (dropdown, data, query, isShopPrimary) => {
    const shopItems = data.shop_items || [];
    const users = data.users || [];

    let html = '';

    if (isShopPrimary) {
        html += `
            <div class="srch-sec">
                <div class="srch-hdr srch-action-item" data-action="shop-search" data-query="${escapeSearchHTML(query)}">
                    <i class="fa-solid fa-cart-shopping"></i>
                    <span>Shop search<span class="srch-cat-cur">(current category)</span></span>
                </div>
                ${shopItems.map(item => `
                    <div class="srch-item" data-type="shop" data-id="${item.id}" data-name="${escapeSearchHTML(item.name)}">
                        <span>${escapeSearchHTML(item.name)} - ${escapeSearchHTML(item.type_name || item.type)}</span>
                    </div>
                `).join('')}
            </div>
        `;

        if (users.length > 0) {
            html += `
                <div class="srch-sec">
                    <div class="srch-oth-hdr">Other results</div>
                    ${users.map(user => `
                        <div class="srch-item" data-type="user" data-id="${user.id}" data-username="${escapeSearchHTML(user.username)}">
                            <span>${escapeSearchHTML(user.username)} - User</span>
                        </div>
                    `).join('')}
                </div>
            `;
        }
    } else {
        html += `
            <div class="srch-sec">
                <div class="srch-hdr srch-action-item" data-action="user-search" data-query="${escapeSearchHTML(query)}">
                    <i class="fa-solid fa-user"></i>
                    <span>User search<span class="srch-cat-cur">(current category)</span></span>
                </div>
                ${users.map(user => `
                    <div class="srch-item" data-type="user" data-id="${user.id}" data-username="${escapeSearchHTML(user.username)}">
                        <span>${escapeSearchHTML(user.username)} - User</span>
                    </div>
                `).join('')}
            </div>
        `;

        if (shopItems.length > 0) {
            html += `
                <div class="srch-sec">
                    <div class="srch-oth-hdr">Other results</div>
                    ${shopItems.map(item => `
                        <div class="srch-item" data-type="shop" data-id="${item.id}" data-name="${escapeSearchHTML(item.name)}">
                            <span>${escapeSearchHTML(item.name)} - ${escapeSearchHTML(item.type_name || item.type)}</span>
                        </div>
                    `).join('')}
                </div>
            `;
        }
    }

    dropdown.innerHTML = html;
    dropdown.classList.add('active');
    currentSelectedIndex = -1;
};

const performSearch = (input, query) => {
    const pill = input.closest('.pill');
    if (!pill) return;
    const dropdown = getSearchDropdown(pill);

    if (!query) {
        dropdown.classList.remove('active');
        dropdown.innerHTML = '';
        currentSelectedIndex = -1;
        return;
    }

    if (globalSearchCache.has(query)) {
        renderSearchDropdown(dropdown, globalSearchCache.get(query), query, !isUserContextPage());
        return;
    }

    fetch(`/api/v1/search?q=${encodeURIComponent(query)}&limit=5`)
        .then(res => res.json())
        .then(data => {
            if (input.value.trim() !== query) return;
            globalSearchCache.set(query, data);
            renderSearchDropdown(dropdown, data, query, !isUserContextPage());
        })
        .catch(() => {
            renderSearchDropdown(dropdown, { shop_items: [], users: [] }, query, !isUserContextPage());
        });
};

const navigateToSearchResult = itemEl => {
    if (!itemEl) return;
    const type = itemEl.dataset.type;
    const id = itemEl.dataset.id;
    const name = itemEl.dataset.name;
    const action = itemEl.dataset.action;
    const query = itemEl.dataset.query;

    let targetUrl = '/';
    if (action === 'shop-search') {
        targetUrl = `/shop?q=${encodeURIComponent(query || '')}`;
    } else if (action === 'user-search') {
        targetUrl = `/users?q=${encodeURIComponent(query || '')}`;
    } else if (type === 'shop') {
        targetUrl = `/shop?q=${encodeURIComponent(name || '')}`;
    } else if (type === 'user') {
        targetUrl = `/user/${id}`;
    }

    const dropdown = itemEl.closest('.srch-dropdown');
    if (dropdown) dropdown.classList.remove('active');

    if (window.htmx) {
        window.htmx.ajax('GET', targetUrl, { target: 'body', pushUrl: true });
    } else {
        window.location.href = targetUrl;
    }
};

const handleSearchKeyDown = (e, input) => {
    const pill = input.closest('.pill');
    if (!pill) return;
    const dropdown = getSearchDropdown(pill);
    if (!dropdown.classList.contains('active')) return;

    const items = Array.from(dropdown.querySelectorAll('.srch-item, .srch-action-item'));
    if (items.length === 0) return;

    if (e.key === 'ArrowDown') {
        e.preventDefault();
        currentSelectedIndex = (currentSelectedIndex + 1) % items.length;
        items.forEach((item, i) => item.classList.toggle('selected', i === currentSelectedIndex));
        items[currentSelectedIndex].scrollIntoView({ block: 'nearest' });
    } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        currentSelectedIndex = (currentSelectedIndex - 1 + items.length) % items.length;
        items.forEach((item, i) => item.classList.toggle('selected', i === currentSelectedIndex));
        items[currentSelectedIndex].scrollIntoView({ block: 'nearest' });
    } else if (e.key === 'Enter') {
        e.preventDefault();
        if (currentSelectedIndex >= 0 && items[currentSelectedIndex]) {
            navigateToSearchResult(items[currentSelectedIndex]);
        } else {
            const q = input.value.trim();
            if (q) {
                dropdown.classList.remove('active');
                const targetUrl = isUserContextPage() ? `/users?q=${encodeURIComponent(q)}` : `/shop?q=${encodeURIComponent(q)}`;
                if (window.htmx) {
                    window.htmx.ajax('GET', targetUrl, { target: 'body', pushUrl: true });
                } else {
                    window.location.href = targetUrl;
                }
            }
        }
    } else if (e.key === 'Escape') {
        dropdown.classList.remove('active');
        currentSelectedIndex = -1;
    }
};

document.addEventListener('input', e => {
    const input = e.target.closest('.pill .input');
    if (!input) return;
    const query = input.value.trim();
    if (globalSearchTimer) clearTimeout(globalSearchTimer);
    if (!query) {
        const pill = input.closest('.pill');
        if (pill) {
            const dropdown = pill.querySelector('.srch-dropdown');
            if (dropdown) dropdown.classList.remove('active');
        }
        return;
    }
    globalSearchTimer = setTimeout(() => performSearch(input, query), 120);
});

document.addEventListener('focusin', e => {
    const input = e.target.closest('.pill .input');
    if (!input) return;
    input.setAttribute('autocomplete', 'off');
    const query = input.value.trim();
    if (query) {
        performSearch(input, query);
    }
});

document.addEventListener('keydown', e => {
    const input = e.target.closest('.pill .input');
    if (!input) return;
    handleSearchKeyDown(e, input);
});

document.addEventListener('click', e => {
    const item = e.target.closest('.srch-item, .srch-action-item');
    if (item) {
        navigateToSearchResult(item);
        return;
    }

    if (!e.target.closest('.pill')) {
        document.querySelectorAll('.srch-dropdown.active').forEach(d => d.classList.remove('active'));
        currentSelectedIndex = -1;
    }
});

document.addEventListener('htmx:beforeTransition', () => {
    document.querySelectorAll('.srch-dropdown.active').forEach(d => d.classList.remove('active'));
    currentSelectedIndex = -1;
});