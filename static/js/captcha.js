(function () {
    async function sha256hex(text) {
        const data = new TextEncoder().encode(text);
        const buf = await crypto.subtle.digest('SHA-256', data);
        return Array.from(new Uint8Array(buf)).map(function (b) {
            return b.toString(16).padStart(2, '0');
        }).join('');
    }

    async function solve(form) {
        const status = form.querySelector('[data-captcha-status]');
        if (status) status.textContent = 'Checking you are human...';
        const res = await fetch('/api/v1/altcha', { credentials: 'same-origin' });
        if (!res.ok) throw new Error('captcha challenge failed');
        const ch = await res.json();
        const max = ch.maxnumber || 50000;
        let found = 0;
        for (let n = 1; n <= max; n++) {
            const hex = await sha256hex(ch.salt + String(n));
            if (hex === ch.challenge) {
                found = n;
                break;
            }
        }
        if (!found) throw new Error('captcha solve failed');
        const payload = btoa(JSON.stringify({
            algorithm: 'SHA-256',
            challenge: ch.challenge,
            number: found,
            salt: ch.salt,
            signature: ch.signature
        }));
        let input = form.querySelector('input[name="altcha"]');
        if (!input) {
            input = document.createElement('input');
            input.type = 'hidden';
            input.name = 'altcha';
            form.appendChild(input);
        }
        input.value = payload;
        if (status) status.textContent = 'Ready.';
    }

    document.querySelectorAll('form[data-captcha]').forEach(function (form) {
        form.addEventListener('submit', function (ev) {
            if (form.querySelector('input[name="altcha"]') && form.querySelector('input[name="altcha"]').value) return;
            ev.preventDefault();
            solve(form).then(function () { form.submit(); }).catch(function (err) {
                const status = form.querySelector('[data-captcha-status]');
                if (status) status.textContent = err.message;
            });
        });
    });
})();
