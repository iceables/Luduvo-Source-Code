(function() {
    var timerPage = document.getElementById('timerPage');
    if (!timerPage) return;

    var targetEndMs = parseInt(timerPage.dataset.targetEndMs, 10) || 0;
    var totalDurationMs = parseInt(timerPage.dataset.totalDurationMs, 10) || 0;
    var initialPreload = timerPage.dataset.shouldPreload === 'true';

    var display = document.getElementById('livetimerDisplay');
    var ambientAudio = document.getElementById('ambientAudio');
    var eventVideo = document.getElementById('eventVideo');
    var videoSpinner = document.getElementById('videoSpinner');
    var lastSec = -1;
    var isInitialized = false;
    var isTransitioning = false;
    var videoDownloadStarted = false;
    var videoPlayed = false;
    var isAudioDownloading = false;
    var isVideoDownloading = false;

    function updateSpinnerVisibility() {
        if (videoSpinner) {
            if (isAudioDownloading || isVideoDownloading) {
                videoSpinner.style.display = 'block';
            } else {
                videoSpinner.style.display = 'none';
            }
        }
    }

    function setupAudioFallback() {
        var enableAudio = function() {
            if (ambientAudio) {
                ambientAudio.play().catch(function() {});
            }
            document.removeEventListener('click', enableAudio);
            document.removeEventListener('keydown', enableAudio);
            document.removeEventListener('touchstart', enableAudio);
            document.removeEventListener('mousemove', enableAudio);
        };
        document.addEventListener('click', enableAudio);
        document.addEventListener('keydown', enableAudio);
        document.addEventListener('touchstart', enableAudio);
        document.addEventListener('mousemove', enableAudio);
    }

    function preloadAndPlayAudio() {
        if (!ambientAudio) return;
        isAudioDownloading = true;
        updateSpinnerVisibility();

        fetch('/static/event/ambient.mp3')
            .then(function(res) {
                if (!res.body) {
                    return res.blob().then(function(b) { return [b]; });
                }
                var reader = res.body.getReader();
                var chunks = [];
                function read() {
                    return reader.read().then(function(result) {
                        if (result.done) {
                            return chunks;
                        }
                        chunks.push(result.value);
                        if (chunks.length === 1 && ambientAudio) {
                            var initialBlob = new Blob(chunks, { type: 'audio/mp3' });
                            ambientAudio.src = URL.createObjectURL(initialBlob);
                            ambientAudio.volume = 0.2;
                            ambientAudio.play().catch(function() {
                                setupAudioFallback();
                            });
                        }
                        return read();
                    });
                }
                return read();
            })
            .then(function(chunksOrBlob) {
                isAudioDownloading = false;
                updateSpinnerVisibility();
                if (!ambientAudio) return;
                var fullBlob;
                if (chunksOrBlob instanceof Blob) {
                    fullBlob = chunksOrBlob;
                } else {
                    fullBlob = new Blob(chunksOrBlob, { type: 'audio/mp3' });
                }
                var currentTime = ambientAudio.currentTime || 0;
                var isPaused = ambientAudio.paused;
                var fullUrl = URL.createObjectURL(fullBlob);
                ambientAudio.src = fullUrl;
                ambientAudio.currentTime = currentTime;
                ambientAudio.volume = 0.2;
                if (!isPaused) {
                    ambientAudio.play().catch(function() {
                        setupAudioFallback();
                    });
                }
            })
            .catch(function() {
                isAudioDownloading = false;
                updateSpinnerVisibility();
                if (!ambientAudio) return;
                ambientAudio.src = '/static/event/ambient.mp3';
                ambientAudio.volume = 0.2;
                ambientAudio.play().catch(function() {
                    setupAudioFallback();
                });
            });
    }

    preloadAndPlayAudio();

    function preloadVideo() {
        if (videoDownloadStarted) return;
        videoDownloadStarted = true;
        isVideoDownloading = true;
        updateSpinnerVisibility();

        fetch('/static/event/final.mp4')
            .then(function(res) {
                if (!res.body) {
                    return res.blob();
                }
                var reader = res.body.getReader();
                var chunks = [];
                function read() {
                    return reader.read().then(function(result) {
                        if (result.done) {
                            return new Blob(chunks, { type: 'video/mp4' });
                        }
                        chunks.push(result.value);
                        return read();
                    });
                }
                return read();
            })
            .then(function(blob) {
                var videoUrl = URL.createObjectURL(blob);
                eventVideo.src = videoUrl;
                eventVideo.load();
                isVideoDownloading = false;
                updateSpinnerVisibility();
            })
            .catch(function() {
                eventVideo.src = '/static/event/final.mp4';
                eventVideo.load();
                isVideoDownloading = false;
                updateSpinnerVisibility();
            });
    }

    if (initialPreload) {
        preloadVideo();
    }

    function stopAmbientAudio() {
        if (ambientAudio) {
            ambientAudio.pause();
            if (ambientAudio.parentNode) {
                ambientAudio.parentNode.removeChild(ambientAudio);
            }
            ambientAudio = null;
        }
    }

    function playFinalVideo() {
        if (videoPlayed) return;
        videoPlayed = true;

        stopAmbientAudio();

        isVideoDownloading = false;
        isAudioDownloading = false;
        updateSpinnerVisibility();

        if (!videoDownloadStarted) {
            eventVideo.src = '/static/event/final.mp4';
            eventVideo.load();
        }

        eventVideo.style.display = 'block';
        eventVideo.volume = 0.6;

        var preventMediaControl = function(e) {
            if (videoPlayed && !isTransitioning) {
                if (e.type === 'keydown' && (e.code === 'Space' || e.code === 'KeyK' || e.code === 'KeyF' || e.code === 'KeyM')) {
                    e.preventDefault();
                }
            }
        };
        window.addEventListener('keydown', preventMediaControl);

        eventVideo.addEventListener('ended', function() {
            eventVideo.style.display = 'none';
            finishAndTransition();
        });

        eventVideo.addEventListener('error', function() {
            eventVideo.style.display = 'none';
            finishAndTransition();
        });

        var p = eventVideo.play();
        if (p !== undefined) {
            p.catch(function() {
                eventVideo.style.display = 'none';
                finishAndTransition();
            });
        }
    }

    function formatTime(sec) {
        if (sec < 0) sec = 0;
        if (sec > 3600) sec = 3600;
        var m = Math.floor(sec / 60);
        var s = sec % 60;
        var mStr = (m < 10 ? '0' : '') + m;
        var sStr = (s < 10 ? '0' : '') + s;
        return mStr + ':' + sStr;
    }

    function initOdometerStructure(timeStr) {
        if (!display) return;
        display.innerHTML = '';
        for (var i = 0; i < timeStr.length; i++) {
            var char = timeStr[i];
            if (char === ':') {
                var colon = document.createElement('span');
                colon.className = 'digit-colon';
                colon.textContent = ':';
                display.appendChild(colon);
            } else {
                var cnt = document.createElement('span');
                cnt.className = 'dgtcnt';
                cnt.setAttribute('data-index', i);

                var stp = document.createElement('span');
                stp.className = 'dgtstp';

                for (var d = 0; d <= 9; d++) {
                    var num = document.createElement('span');
                    num.className = 'dgtnum';
                    num.textContent = d;
                    stp.appendChild(num);
                }

                cnt.appendChild(stp);
                display.appendChild(cnt);
            }
        }
        isInitialized = true;
    }

    function updateOdometer(timeStr, remainingSec) {
        if (!display) return;

        if (!isInitialized) {
            initOdometerStructure(timeStr);
        }

        display.classList.remove('m-anim-30m', 'm-anim-15m');
        if (remainingSec === 1800) {
            void display.offsetWidth;
            display.classList.add('m-anim-30m');
        } else if (remainingSec === 900) {
            void display.offsetWidth;
            display.classList.add('m-anim-15m');
        }

        for (var i = 0; i < timeStr.length; i++) {
            var char = timeStr[i];
            if (char !== ':') {
                var val = parseInt(char) || 0;
                var strip = display.querySelector('.dgtcnt[data-index="' + i + '"] .dgtstp');
                if (strip) {
                    strip.style.transform = 'translateY(-' + (val * 1.1) + 'em)';
                }
            }
        }
    }

    function finishAndTransition() {
        if (isTransitioning) return;
        isTransitioning = true;
        updateOdometer('00:00', 0);

        if (timerPage) {
            timerPage.classList.add('page-transitioning');
        }

        sessionStorage.removeItem('alreadyLoaded');

        setTimeout(function() {
            window.location.href = '/';
        }, 750);
    }

    function updateTimer() {
        if (isTransitioning || videoPlayed) return;
        var now = Date.now();
        var remainingMs = targetEndMs - now;
        var remainingSec = Math.ceil(remainingMs / 1000);

        if (ambientAudio) {
            if (remainingMs <= 2000 && remainingMs > 0) {
                var factor = remainingMs / 2000;
                ambientAudio.volume = Math.max(0, 0.2 * factor);
            } else if (remainingMs > 2000) {
                ambientAudio.volume = 0.2;
            }
        }

        if ((totalDurationMs > 0 && remainingMs <= (totalDurationMs * 0.5)) || (totalDurationMs <= 0 && remainingMs <= 1800000)) {
            preloadVideo();
        }

        if (remainingMs <= 0) {
            updateOdometer('00:00', 0);
            playFinalVideo();
            return;
        }

        if (remainingSec !== lastSec) {
            lastSec = remainingSec;
            var formatted = formatTime(remainingSec);
            updateOdometer(formatted, remainingSec);
        }
    }

    updateTimer();
    var timer = setInterval(function() {
        var now = Date.now();
        if (targetEndMs - now <= 0) {
            clearInterval(timer);
            updateTimer();
        } else {
            updateTimer();
        }
    }, 250);
})();