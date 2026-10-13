window.activeModalOnClose = null;

window.escapeHTML = function(str) {
    if (!str) return '';
    return str
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
};

window.showModal = function(options) {
    const modal = document.getElementById(options.id || 'appModal');
    if (!modal) return;

    const titleEl = modal.querySelector('.mtitle');
    const subtitleEl = modal.querySelector('.msub');
    const bodyEl = modal.querySelector('.mbody');

    if (titleEl && options.title !== undefined) {
        titleEl.textContent = options.title;
    }

    if (subtitleEl) {
        if (options.subtitle) {
            subtitleEl.textContent = options.subtitle;
            subtitleEl.style.display = 'block';
        } else {
            subtitleEl.style.display = 'none';
            subtitleEl.textContent = '';
        }
    }

    if (bodyEl && options.body !== undefined) {
        if (typeof options.body === 'string') {
            bodyEl.innerHTML = options.body;
        } else if (options.body instanceof HTMLElement) {
            bodyEl.innerHTML = '';
            bodyEl.appendChild(options.body);
        }
    }

    if (bodyEl) {
        if (options.bodyStyle) {
            bodyEl.setAttribute('style', options.bodyStyle);
        } else {
            bodyEl.removeAttribute('style');
        }
    }

    window.activeModalOnClose = options.onClose || null;
    modal.classList.add('active');
};

window.closeModal = function(modalId) {
    const modal = document.getElementById(modalId || 'appModal') || document.querySelector('.movl.active');
    if (modal) {
        modal.classList.remove('active');
    }
    if (typeof window.activeModalOnClose === 'function') {
        const cb = window.activeModalOnClose;
        window.activeModalOnClose = null;
        cb();
    }
};

window.closeModalOnOverlay = function(event) {
    if (event.target && event.target.classList.contains('movl')) {
        window.closeModal(event.target.id);
    }
};