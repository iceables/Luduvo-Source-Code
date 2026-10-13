window.usernameChangesLeft = 2;
window.pendingUsernameChange = null;

window.updateHeaderVertices = function(bucks) {
    if (bucks === undefined || bucks === null) return;
    const elements = document.querySelectorAll('#userVerticesCount, .sbtnvrt .navtxt');
    elements.forEach(el => {
        el.textContent = bucks;
    });
};

window.openUsernameModal = function(newUsername) {
    window.pendingUsernameChange = newUsername;
    const changesLeft = window.usernameChangesLeft !== undefined ? window.usernameChangesLeft : 2;
    const safeUsername = window.escapeHTML ? window.escapeHTML(newUsername) : newUsername;

    window.showModal({
        title: 'Hold up!',
        bodyStyle: 'gap: 16px;',
        body: `
            <p style="font-family: 'Ubuntu', sans-serif; font-size: 14px; color: #3D3D3D; line-height: 1.5; margin: 0;">
                Are you sure you want to change your username to <strong>${safeUsername}</strong> for <span class="vrthl"><img src="/static/useful/icons/verticec.png" class="vrtico" alt="">100 Vertices</span>? You have <strong>${changesLeft}</strong> username changes left for the next 3 days!
            </p>
            <div style="display: flex; justify-content: flex-end; gap: 8px; margin-top: 4px;">
                <button type="button" class="primary hpyerr hpyinl hpysm" onclick="window.closeModal()">
                    <span>Cancel</span>
                </button>
                <button type="button" class="primary hpyprim hpyinl hpysm" onclick="window.confirmUsernameChange()">
                    <span>Confirm</span>
                </button>
            </div>
        `
    });
};

window.confirmUsernameChange = function() {
    const newUsername = window.pendingUsernameChange;
    window.closeModal();
    if (newUsername) {
        window.executeUsernameChange(newUsername);
    }
};

window.executeUsernameChange = function(newUsername) {
    const form = document.querySelector('form[action*="/settings/username"]');
    if (!form) return;

    const formData = new FormData(form);
    const submitBtn = form.querySelector('button[type="submit"]');
    if (submitBtn) submitBtn.disabled = true;

    clearFormMessages(form);

    fetch(form.action, {
        method: 'POST',
        body: formData,
        headers: {
            'Accept': 'application/json',
            'X-Requested-With': 'XMLHttpRequest'
        }
    })
    .then(res => res.json())
    .then(data => {
        if (submitBtn) submitBtn.disabled = false;

        if (data.error) {
            showFormError(form, data.error);
        } else if (data.success) {
            showFormSuccess(form, data.success);

            const usernameInput = form.querySelector('input[name="username"]');
            if (usernameInput) usernameInput.value = '';

            const curUsernameEl = document.querySelector('.setinf strong');
            if (curUsernameEl && data.username) {
                curUsernameEl.textContent = data.username;
            }
            const dropdownUserEl = document.querySelector('.dropusr a');
            if (dropdownUserEl && data.username) {
                dropdownUserEl.textContent = data.username;
            }
            if (data.bucks !== undefined) {
                window.updateHeaderVertices(data.bucks);
            }
            if (data.changes_left !== undefined) {
                window.usernameChangesLeft = data.changes_left;
            }
        }
    })
    .catch(() => {
        if (submitBtn) submitBtn.disabled = false;
        showFormError(form, 'An unexpected error occurred. Please try again.');
    });
};

window.selectSocialPlatform = function(iconEl) {
    if (!iconEl) return;
    const allIcons = document.querySelectorAll('.sclico');
    allIcons.forEach(i => i.classList.remove('active'));
    iconEl.classList.add('active');

    const platform = iconEl.dataset.platform;
    const name = iconEl.dataset.name;
    const placeholder = iconEl.dataset.placeholder;
    const val = iconEl.dataset.value || '';

    const platformInput = document.getElementById('sclPlatformInput');
    const valInput = document.getElementById('sclValInput');

    if (platformInput) platformInput.value = platform;
    if (valInput) {
        valInput.value = val;
        valInput.placeholder = placeholder || (name + ' handle or URL');
    }
};

window.updateViewportHeight = function() {
    const activeContent = document.querySelector('.settabcntnt.active');
    const viewport = document.querySelector('.settabvwp');
    if (activeContent && viewport) {
        viewport.style.height = activeContent.offsetHeight + 'px';
    }
};

window.updateSettingsTabIndicator = function() {
    const indicator = document.querySelector('.tabind');
    const activeTab = document.querySelector('.tabbtn.active');
    if (indicator && activeTab) {
        indicator.style.left = activeTab.offsetLeft + 'px';
        indicator.style.width = activeTab.offsetWidth + 'px';
        indicator.style.height = activeTab.offsetHeight + 'px';
        indicator.style.top = activeTab.offsetTop + 'px';
    }
};

window.switchSettingsTab = function(tabName, event) {
    const tabs = Array.from(document.querySelectorAll('.tabbtn'));
    const clickedTab = event.currentTarget;

    if (clickedTab.classList.contains('active')) return;

    const currentContent = document.querySelector('.settabcntnt.active');
    const targetContent = document.getElementById('tab-' + tabName);
    if (!targetContent) return;

    tabs.forEach(tab => tab.classList.remove('active'));
    clickedTab.classList.add('active');

    window.updateSettingsTabIndicator();

    const viewport = document.querySelector('.settabvwp');
    const currentHeight = currentContent ? currentContent.offsetHeight : 0;

    viewport.style.height = currentHeight + 'px';
    viewport.offsetHeight;

    if (currentContent) {
        currentContent.classList.remove('active');
    }
    targetContent.classList.add('active');

    const newHeight = targetContent.offsetHeight;
    viewport.style.height = newHeight + 'px';

    if (window.settingsHeightTimeout) {
        clearTimeout(window.settingsHeightTimeout);
    }
    window.settingsHeightTimeout = setTimeout(() => {
        viewport.style.height = '';
    }, 350);

    try {
        const url = new URL(window.location.href);
        url.searchParams.delete('tab');
        url.searchParams.delete('error');
        url.searchParams.delete('success');
        window.history.replaceState({}, document.title, url.pathname);
    } catch (e) {}
};

function clearFormMessages(form) {
    if (!form) return;
    form.querySelectorAll('.setinbox, .setareabox').forEach(el => el.classList.remove('inerr'));
    form.querySelectorAll('.frminerr, .frminsuc').forEach(el => el.remove());
}

function showFormError(form, message) {
    if (!form) return;
    let targetInput = null;
    const msgLower = message.toLowerCase();

    if (msgLower.includes('username')) {
        targetInput = form.querySelector('input[name="username"]');
    } else if (msgLower.includes('display name')) {
        targetInput = form.querySelector('input[name="displayname"]');
    } else if (msgLower.includes('current password')) {
        targetInput = form.querySelector('input[name="current_password"]');
    } else if (msgLower.includes('password')) {
        targetInput = form.querySelector('input[name="new_password"]');
    } else if (msgLower.includes('bio')) {
        targetInput = form.querySelector('textarea[name="bio"]');
    } else if (msgLower.includes('pronouns')) {
        targetInput = form.querySelector('input[name="pronouns"]');
    } else if (msgLower.includes('socials')) {
        targetInput = form.querySelector('input[name="value"]') || form.querySelector('input');
    }

    if (!targetInput) {
        targetInput = form.querySelector('input:not([type="hidden"]), textarea');
    }

    if (targetInput) {
        const box = targetInput.closest('.setinbox') || targetInput.closest('.setareabox');
        if (box) box.classList.add('inerr');

        const parent = targetInput.closest('.instkrw') || targetInput.closest('.ttpcnt') || targetInput.parentElement;
        let errDiv = parent.querySelector('.frminerr');
        if (!errDiv) {
            errDiv = document.createElement('span');
            errDiv.className = 'frminerr';
            parent.appendChild(errDiv);
        }
        errDiv.textContent = message;
        requestAnimationFrame(() => errDiv.classList.add('visible'));
    }
}

function showFormSuccess(form, message) {
    if (!form) return;
    let targetInput = form.querySelector('input:not([type="hidden"]), textarea');
    if (targetInput) {
        const parent = targetInput.closest('.instkrw') || targetInput.closest('.ttpcnt') || targetInput.parentElement;
        let succDiv = parent.querySelector('.frminsuc');
        if (!succDiv) {
            succDiv = document.createElement('span');
            succDiv.className = 'frminsuc';
            parent.appendChild(succDiv);
        }
        succDiv.textContent = message;
        requestAnimationFrame(() => succDiv.classList.add('visible'));
    }
}

const setupSettingsFormListeners = () => {
    document.addEventListener('submit', function(e) {
        const form = e.target;
        if (!form || !form.action) return;

        if (form.action.includes('/settings/username')) {
            e.preventDefault();
            const usernameInput = form.querySelector('input[name="username"]');
            const newUsername = usernameInput ? usernameInput.value.trim() : '';
            if (!newUsername) return;

            window.openUsernameModal(newUsername);
            return;
        }

        if (form.action.includes('/settings/displayname') ||
            form.action.includes('/settings/password') ||
            form.action.includes('/settings/bio') ||
            form.action.includes('/settings/pronouns') ||
            form.action.includes('/settings/socials')) {

            e.preventDefault();

            const formData = new FormData(form);
            const submitBtn = form.querySelector('button[type="submit"]');
            if (submitBtn) submitBtn.disabled = true;

            clearFormMessages(form);

            fetch(form.action, {
                method: 'POST',
                body: formData,
                headers: {
                    'Accept': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                }
            })
            .then(res => res.json())
            .then(data => {
                if (submitBtn) submitBtn.disabled = false;

                if (data.error) {
                    showFormError(form, data.error);
                } else if (data.success) {
                    showFormSuccess(form, data.success);

                    if (form.action.includes('/settings/displayname') && data.displayname) {
                        const curDisplayNameEl = document.getElementById('curDisplayNameText');
                        if (curDisplayNameEl) {
                            curDisplayNameEl.textContent = data.displayname;
                        }
                    }

                    if (form.action.includes('/settings/password')) {
                        form.querySelectorAll('input[type="password"]').forEach(input => input.value = '');
                    }

                    if (form.action.includes('/settings/socials')) {
                        const platformInput = form.querySelector('input[name="platform"]');
                        const valInput = form.querySelector('input[name="value"]');
                        if (platformInput && valInput) {
                            const plat = platformInput.value;
                            const activeIcon = document.querySelector(`.sclico[data-platform="${plat}"]`);
                            if (activeIcon) {
                                activeIcon.dataset.value = valInput.value;
                            }
                        }
                    }
                }
            })
            .catch(() => {
                if (submitBtn) submitBtn.disabled = false;
                showFormError(form, 'An unexpected error occurred. Please try again.');
            });
        }
    });
};

const initSettings = () => {
    const container = document.querySelector('.settabcnt');
    if (!container) return;

    const leftVal = parseInt(container.dataset.usernameChangesLeft);
    window.usernameChangesLeft = !isNaN(leftVal) ? leftVal : 2;

    setTimeout(() => {
        window.updateSettingsTabIndicator();
        window.updateViewportHeight();
    }, 50);

    setTimeout(() => {
        window.updateSettingsTabIndicator();
        window.updateViewportHeight();
    }, 150);

    setTimeout(() => {
        window.updateSettingsTabIndicator();
        window.updateViewportHeight();
    }, 350);

    const activeTabContent = document.querySelector('.settabcntnt.active');
    const activeForm = activeTabContent ? activeTabContent.querySelector('form') : document.querySelector('form');

    const initialErr = container.dataset.error;
    if (initialErr && activeForm) {
        showFormError(activeForm, initialErr);
    }

    const initialSuccess = container.dataset.success;
    if (initialSuccess && activeForm) {
        showFormSuccess(activeForm, initialSuccess);
    }

    const activeSclIco = document.querySelector('.sclico.active') || document.querySelector('.sclico');
    if (activeSclIco) {
        window.selectSocialPlatform(activeSclIco);
    }

    if (initialErr || initialSuccess) {
        try {
            const url = new URL(window.location.href);
            url.searchParams.delete('error');
            url.searchParams.delete('success');
            window.history.replaceState({}, document.title, url.pathname);
        } catch (e) {}
    }
};

setupSettingsFormListeners();
initSettings();

document.addEventListener('htmx:afterSettle', initSettings);

window.addEventListener('resize', () => {
    window.updateSettingsTabIndicator();
    window.updateViewportHeight();
});
