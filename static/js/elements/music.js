let musicNoticeTimer = null;
let musicFadeTimer = null;
window.currentAudio = new Audio();
window.currentAudio.volume = 0.5;
window.currentTrackData = null;
window.currentMusicVolume = 0.5;

const MUSIC_FADE_MS = 1200;
const PROFILE_MUSIC_VOLUME = 0.25;

function cancelMusicFade() {
    if (musicFadeTimer) {
        clearInterval(musicFadeTimer);
        musicFadeTimer = null;
    }
}

function fadeMusicVolume(target, onDone) {
    cancelMusicFade();
    const audio = window.currentAudio;
    if (!audio || audio.volume === target) {
        if (onDone) onDone();
        return;
    }
    const start = audio.volume;
    const steps = Math.max(1, Math.round(MUSIC_FADE_MS / 16));
    let step = 0;
    musicFadeTimer = setInterval(() => {
        step++;
        const t = Math.min(1, step / steps);
        audio.volume = start + (target - start) * t;
        if (t >= 1) {
            cancelMusicFade();
            if (onDone) onDone();
        }
    }, 16);
}

function playCurrentAudio() {
    window.currentAudio.play().catch(() => {});
    fadeMusicVolume(window.currentMusicVolume);
}

function pauseCurrentAudio() {
    if (window.currentAudio.paused) {
        cancelMusicFade();
        return;
    }
    fadeMusicVolume(0, () => window.currentAudio.pause());
}

function formatMusicTime(seconds) {
    if (isNaN(seconds) || seconds < 0) return '0:00';
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins}:${secs < 10 ? '0' : ''}${secs}`;
}

function setElementsText(ids, text) {
    ids.forEach(id => {
        const el = document.getElementById(id);
        if (el) el.textContent = text;
    });
}

function setElementsWidth(ids, width) {
    ids.forEach(id => {
        const el = document.getElementById(id);
        if (el) el.style.setProperty('width', width, 'important');
    });
}

function setElementsClass(ids, className) {
    ids.forEach(id => {
        const el = document.getElementById(id);
        if (el) el.className = className;
    });
}

function initMusicPlayerEvents() {
    if (window.musicPlayerEventsInited) return;
    window.musicPlayerEventsInited = true;

    window.currentAudio.addEventListener('timeupdate', () => {
        const cur = window.currentAudio.currentTime || 0;
        let dur = window.currentAudio.duration;
        if (isNaN(dur) || dur <= 0) dur = 30;
        const pct = `${Math.min(100, Math.max(0, (cur / dur) * 100))}%`;

        setElementsWidth(['musPlayerBarFill', 'setMusPlayerBarFill', 'prfMusBarFill'], pct);
        setElementsText(['musPlayerCurTime', 'setMusPlayerCurTime', 'prfMusCurTime'], formatMusicTime(cur));
        setElementsText(['musPlayerDurTime', 'setMusPlayerDurTime', 'prfMusDurTime'], formatMusicTime(dur));
    });

    window.currentAudio.addEventListener('ended', () => updateMusicPlayerUI(false));
    window.currentAudio.addEventListener('pause', () => updateMusicPlayerUI(false));
    window.currentAudio.addEventListener('play', () => updateMusicPlayerUI(true));
}

function updateMusicPlayerUI(isPlaying) {
    const iconClass = isPlaying ? 'fa-solid fa-pause' : 'fa-solid fa-play';
    setElementsClass(['musPlayerPlayIcon', 'setMusPlayerPlayIcon', 'prfMusPlayIcon'], iconClass);

    if (window.currentTrackData) {
        const activeTrackId = window.currentTrackData.id;
        document.querySelectorAll('.musitm').forEach(item => {
            const trackId = item.dataset.trackId;
            const btnIcon = item.querySelector('.musplybtn-sm i');
            const isTarget = String(trackId) === String(activeTrackId);
            item.classList.toggle('active', isTarget);
            if (btnIcon) {
                btnIcon.className = isTarget && isPlaying ? 'fa-solid fa-pause' : 'fa-solid fa-play';
            }
        });
    }
}

window.stopCurrentMusic = function() {
    if (window.currentAudio && !window.currentAudio.paused) {
        fadeMusicVolume(0, () => {
            window.currentAudio.pause();
            window.currentAudio.currentTime = 0;
        });
    } else {
        cancelMusicFade();
        if (window.currentAudio) window.currentAudio.currentTime = 0;
    }
    window.currentTrackData = null;
    updateMusicPlayerUI(false);
};

window.handleTrackPlayClick = function(btn) {
    if (!btn) return;
    const { trackId: id, title, artist, cover, preview } = btn.dataset;
    window.playTrackPreview(id, title, artist, cover, preview);
};

window.playTrackPreview = function(id, title, artist, cover, preview) {
    initMusicPlayerEvents();

    if (window.currentTrackData && String(window.currentTrackData.id) === String(id)) {
        if (window.currentAudio.paused) {
            window.currentMusicVolume = 0.5;
            playCurrentAudio();
        } else {
            pauseCurrentAudio();
        }
        return;
    }

    window.currentTrackData = { id, title, artist, cover, preview };
    window.currentMusicVolume = 0.5;
    window.currentAudio.src = preview;
    window.currentAudio.volume = 0;
    window.currentAudio.loop = false;
    playCurrentAudio();

    const playerCard = document.getElementById('musPlayerCard');
    if (playerCard) playerCard.style.display = 'flex';

    const coverEl = document.getElementById('musPlayerCover');
    if (coverEl) {
        coverEl.src = cover;
        coverEl.alt = title;
    }

    setElementsText(['musPlayerTitle'], title);
    setElementsText(['musPlayerArtist'], artist);

    updateMusicPlayerUI(true);
};

window.toggleCurrentTrack = function() {
    if (!window.currentTrackData || !window.currentAudio.src) return;
    if (window.currentAudio.paused) {
        playCurrentAudio();
    } else {
        pauseCurrentAudio();
    }
};

window.toggleSettingsTrack = window.toggleCurrentTrack;
window.toggleProfileTrack = window.toggleCurrentTrack;

window.seekCurrentTrack = function(event) {
    if (!window.currentAudio || !window.currentAudio.duration || isNaN(window.currentAudio.duration)) return;
    const rect = event.currentTarget.getBoundingClientRect();
    const pct = Math.max(0, Math.min(1, (event.clientX - rect.left) / rect.width));
    window.currentAudio.currentTime = pct * window.currentAudio.duration;
};

window.seekSettingsTrack = window.seekCurrentTrack;
window.seekProfileTrack = window.seekCurrentTrack;

window.toggleMuteCurrentTrack = function() {
    if (!window.currentAudio) return;
    window.currentAudio.muted = !window.currentAudio.muted;
    const volClass = window.currentAudio.muted ? 'fa-solid fa-volume-xmark' : 'fa-solid fa-volume-high';
    setElementsClass(['musPlayerVolIcon', 'setMusPlayerVolIcon', 'prfMusVolIcon'], volClass);
};

window.toggleMuteSettingsTrack = window.toggleMuteCurrentTrack;
window.toggleMuteProfileTrack = window.toggleMuteCurrentTrack;

window.updateProfileMusic = function() {
    if (!window.currentTrackData || !window.currentTrackData.id) return;
    const btn = document.getElementById('musPlayerUpdateBtn');
    if (btn) btn.disabled = true;

    const formData = new FormData();
    formData.append('track_id', window.currentTrackData.id);

    fetch('/settings/music', {
        method: 'POST',
        body: formData,
        headers: { 'Accept': 'application/json', 'X-Requested-With': 'XMLHttpRequest' }
    })
    .then(res => res.json())
    .then(data => {
        if (btn) btn.disabled = false;
        if (data.error) {
            alert(data.error);
        } else if (data.success) {
            const btnSpan = btn ? btn.querySelector('span') : null;
            if (btnSpan) {
                const origText = btnSpan.textContent;
                btnSpan.textContent = 'Updated!';
                setTimeout(() => { btnSpan.textContent = origText; }, 2000);
            }
            if (window.loadSettingsSavedMusic) {
                window.loadSettingsSavedMusic(window.currentTrackData.id);
            }
        }
    })
    .catch(() => {
        if (btn) btn.disabled = false;
    });
};

window.removeProfileMusic = function() {
    fetch('/settings/music/remove', {
        method: 'POST',
        headers: { 'Accept': 'application/json', 'X-Requested-With': 'XMLHttpRequest' }
    })
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            const card = document.getElementById('settingsMusPlayerCard');
            if (card) card.style.setProperty('display', 'none', 'important');

            const section = document.getElementById('settingsMusicSection');
            if (section) section.dataset.musicId = '0';

            window.stopCurrentMusic();
        }
    })
    .catch(() => {});
};

window.loadSettingsSavedMusic = function(musicId) {
    const card = document.getElementById('settingsMusPlayerCard');
    if (!musicId || musicId <= 0 || isNaN(musicId)) {
        if (card) card.style.setProperty('display', 'none', 'important');
        return;
    }

    fetch(`/api/v1/music/track/${musicId}`)
        .then(res => res.json())
        .then(track => {
            if (!track || !track.id || track.error) {
                if (card) card.style.setProperty('display', 'none', 'important');
                return;
            }

            const cover = (track.album && track.album.cover_small) ? track.album.cover_small : '/static/useful/temp/pfp.png';
            const artist = (track.artist && track.artist.name) ? track.artist.name : 'Unknown Artist';
            const preview = track.preview || '';

            const coverEl = document.getElementById('setMusPlayerCover');
            if (coverEl) {
                coverEl.src = cover;
                coverEl.alt = track.title;
            }

            setElementsText(['setMusPlayerTitle'], track.title);
            setElementsText(['setMusPlayerArtist'], artist);

            if (card) card.style.display = 'flex';

            initMusicPlayerEvents();
            window.currentTrackData = { id: track.id, title: track.title, artist, cover, preview };
            window.currentMusicVolume = 0.5;
            window.currentAudio.src = preview;
            window.currentAudio.volume = 0.5;
            window.currentAudio.loop = false;
        })
        .catch(() => {
            if (card) card.style.setProperty('display', 'none', 'important');
        });
};

window.initSettingsMusicPlayer = function() {
    const section = document.getElementById('settingsMusicSection');
    const card = document.getElementById('settingsMusPlayerCard');
    if (!section) return;

    const musicId = parseInt(section.dataset.musicId, 10);
    if (musicId && musicId > 0) {
        window.loadSettingsSavedMusic(musicId);
    } else {
        if (card) card.style.setProperty('display', 'none', 'important');
    }
};

window.initProfileMusicPlayer = function() {
    const widget = document.getElementById('profileMusicWidget');
    if (!widget) return;

    const musicId = parseInt(widget.dataset.musicId, 10);
    if (!musicId || musicId <= 0 || isNaN(musicId)) return;

    fetch(`/api/v1/music/track/${musicId}`)
        .then(res => res.json())
        .then(track => {
            if (!track || !track.id || track.error) return;

            const cover = (track.album && track.album.cover_small) ? track.album.cover_small : '/static/useful/temp/pfp.png';
            const artist = (track.artist && track.artist.name) ? track.artist.name : 'Unknown Artist';
            const preview = track.preview || '';

            const coverEl = document.getElementById('prfMusCover');
            if (coverEl) {
                coverEl.src = cover;
                coverEl.alt = track.title;
            }

            setElementsText(['prfMusTitle'], track.title);
            setElementsText(['prfMusArtist'], artist);

            initMusicPlayerEvents();
            window.currentTrackData = { id: track.id, title: track.title, artist, cover, preview };
            window.currentMusicVolume = PROFILE_MUSIC_VOLUME;
            window.currentAudio.src = preview;
            window.currentAudio.volume = 0;
            window.currentAudio.loop = true;

            const playPromise = window.currentAudio.play();
            if (playPromise !== undefined) {
                playPromise.then(() => {
                    fadeMusicVolume(PROFILE_MUSIC_VOLUME);
                    updateMusicPlayerUI(true);
                }).catch(() => {
                    updateMusicPlayerUI(false);
                    const enableAutoplay = () => {
                        if (window.currentAudio && window.currentAudio.src && window.location.pathname.startsWith('/user/')) {
                            window.currentMusicVolume = PROFILE_MUSIC_VOLUME;
                            window.currentAudio.volume = 0;
                            window.currentAudio.play().then(() => {
                                fadeMusicVolume(PROFILE_MUSIC_VOLUME);
                                updateMusicPlayerUI(true);
                            }).catch(() => {});
                        }
                    };
                    ['click', 'keydown', 'touchstart', 'pointerdown'].forEach(evt => document.addEventListener(evt, enableAutoplay, { once: true }));
                });
            }
        })
        .catch(() => {});
};

window.getSpinnerHTML = function() {
    return `
        <div class="musld">
            <svg class="dmspn" viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg">
                <rect class="sq sq-1" x="43" y="7" width="14" height="14"/>
                <rect class="sq sq-2" x="25" y="25" width="14" height="14"/>
                <rect class="sq sq-3" x="7" y="43" width="14" height="14"/>
                <rect class="sq sq-4" x="25" y="61" width="14" height="14"/>
                <rect class="sq sq-5" x="43" y="79" width="14" height="14"/>
                <rect class="sq sq-6" x="61" y="61" width="14" height="14"/>
                <rect class="sq sq-7" x="79" y="43" width="14" height="14"/>
                <rect class="sq sq-8" x="61" y="25" width="14" height="14"/>
            </svg>
        </div>
    `;
};

window.startNoticeTimer = function() {
    const btn = document.getElementById('musicNoticeContinueBtn');
    const btnText = document.getElementById('musicNoticeContinueText');
    if (!btn || !btnText) return;

    if (musicNoticeTimer) {
        clearInterval(musicNoticeTimer);
        musicNoticeTimer = null;
    }

    let timeLeft = 5;
    btn.disabled = true;
    btnText.textContent = `Continue (${timeLeft}s)`;

    musicNoticeTimer = setInterval(() => {
        timeLeft--;
        if (timeLeft > 0) {
            btnText.textContent = `Continue (${timeLeft}s)`;
        } else {
            clearInterval(musicNoticeTimer);
            musicNoticeTimer = null;
            btn.disabled = false;
            btnText.textContent = 'Continue';
        }
    }, 1000);
};

window.openMusicModal = function() {
    if (localStorage.getItem('musicNoticeSeen') === 'true') {
        const searchModal = document.getElementById('musicModal');
        if (searchModal) {
            searchModal.classList.add('active');
            const input = document.getElementById('musSearchInput');
            if (input) input.focus();
        }
        return;
    }

    const noticeModal = document.getElementById('musicNoticeModal');
    if (noticeModal) {
        noticeModal.classList.add('active');
        window.startNoticeTimer();
    }
};

window.closeMusicNoticeModal = function() {
    if (musicNoticeTimer) {
        clearInterval(musicNoticeTimer);
        musicNoticeTimer = null;
    }
    const noticeModal = document.getElementById('musicNoticeModal');
    if (noticeModal) noticeModal.classList.remove('active');
};

window.closeMusicNoticeModalOnOverlay = function(event) {
    if (event.target && event.target.id === 'musicNoticeModal') {
        window.closeMusicNoticeModal();
    }
};

window.proceedToMusicSearch = function() {
    const btn = document.getElementById('musicNoticeContinueBtn');
    if (btn && btn.disabled) return;

    localStorage.setItem('musicNoticeSeen', 'true');
    window.closeMusicNoticeModal();

    const searchModal = document.getElementById('musicModal');
    if (searchModal) {
        searchModal.classList.add('active');
        const input = document.getElementById('musSearchInput');
        if (input) input.focus();
    }
};

window.closeMusicModal = function() {
    const modal = document.getElementById('musicModal');
    if (modal) modal.classList.remove('active');
    pauseCurrentAudio();
};

window.closeMusicModalOnOverlay = function(event) {
    if (event.target && event.target.id === 'musicModal') {
        window.closeMusicModal();
    }
};

window.searchMusic = function(event) {
    if (event) {
        event.preventDefault();
        event.stopPropagation();
    }
    const input = document.getElementById('musSearchInput');
    const container = document.getElementById('musResults');
    if (!input || !container) return false;

    const query = input.value.trim();
    if (!query) return false;

    container.innerHTML = window.getSpinnerHTML();

    fetch('/api/v1/music/search?q=' + encodeURIComponent(query))
        .then(res => res.json())
        .then(data => {
            if (!data.data || data.data.length === 0) {
                container.innerHTML = '<div class="msempty">No tracks found</div>';
                return;
            }

            let html = '';
            data.data.forEach(track => {
                const cover = (track.album && track.album.cover_small) ? track.album.cover_small : '/static/useful/temp/pfp.png';
                const artist = (track.artist && track.artist.name) ? track.artist.name : 'Unknown Artist';
                const preview = track.preview || '';

                const safeTitle = window.escapeHTML ? window.escapeHTML(track.title) : track.title;
                const safeArtist = window.escapeHTML ? window.escapeHTML(artist) : artist;

                const isCurrentPlaying = window.currentTrackData &&
                    String(window.currentTrackData.id) === String(track.id) &&
                    !window.currentAudio.paused;

                const iconClass = isCurrentPlaying ? 'fa-solid fa-pause' : 'fa-solid fa-play';
                const itemActiveClass = (window.currentTrackData && String(window.currentTrackData.id) === String(track.id)) ? ' active' : '';

                const actionBtn = preview ? `
                    <button type="button" class="musplybtn-sm" title="Preview track" data-track-id="${track.id}" data-title="${safeTitle}" data-artist="${safeArtist}" data-cover="${cover}" data-preview="${preview}" onclick="window.handleTrackPlayClick(this)">
                        <i class="${iconClass}"></i>
                    </button>
                ` : '';

                html += `
                    <div class="musitm${itemActiveClass}" data-track-id="${track.id}">
                        <img src="${cover}" alt="${safeTitle}" class="muscvr">
                        <div class="musinf">
                            <div class="musttl">${safeTitle}</div>
                            <div class="musart">${safeArtist}</div>
                        </div>
                        ${actionBtn}
                    </div>
                `;
            });
            container.innerHTML = html;
        })
        .catch(() => {
            container.innerHTML = '<div class="msempty">Failed to load tracks. Please try again.</div>';
        });

    return false;
};

document.addEventListener('DOMContentLoaded', () => {
    window.initSettingsMusicPlayer();
    window.initProfileMusicPlayer();
});

document.addEventListener('htmx:afterSettle', () => {
    window.initSettingsMusicPlayer();
    window.initProfileMusicPlayer();
});

function stopMusicOnNavigation() {
    window.stopCurrentMusic();
}

window.addEventListener('popstate', stopMusicOnNavigation);
window.addEventListener('pagehide', stopMusicOnNavigation);
window.addEventListener('beforeunload', stopMusicOnNavigation);
window.addEventListener('pageshow', (event) => {
    if (event.persisted) stopMusicOnNavigation();
});

document.addEventListener('htmx:beforeTransition', stopMusicOnNavigation);
document.addEventListener('htmx:beforeSwap', stopMusicOnNavigation);