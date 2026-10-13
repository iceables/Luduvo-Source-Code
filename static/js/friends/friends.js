window.updateFriendsTabIndicator = function() {
    const indicator = document.querySelector('.tabcnt .tabind');
    const activeTab = document.querySelector('.tabcnt .tabbtn.active');
    if (indicator && activeTab) {
        indicator.style.left = activeTab.offsetLeft + 'px';
        indicator.style.width = activeTab.offsetWidth + 'px';
        indicator.style.height = activeTab.offsetHeight + 'px';
        indicator.style.top = activeTab.offsetTop + 'px';
    }
};

window.updateFriendsViewportHeight = function() {
    const activeContent = document.querySelector('.settabcntnt.active');
    const viewport = document.querySelector('.settabvwp');
    if (activeContent && viewport) {
        viewport.style.height = activeContent.offsetHeight + 'px';
    }
};

window.switchFriendsTab = function(tabName, event) {
    const tabs = Array.from(document.querySelectorAll('.tabcnt .tabbtn'));
    const clickedTab = event ? event.currentTarget : null;
    if (!clickedTab || clickedTab.classList.contains('active')) return;

    const currentContent = document.querySelector('.settabcntnt.active');
    const targetContent = document.getElementById('tab-' + tabName);
    if (!targetContent) return;

    tabs.forEach(tab => tab.classList.remove('active'));
    clickedTab.classList.add('active');

    window.updateFriendsTabIndicator();

    const viewport = document.querySelector('.settabvwp');
    const currentHeight = currentContent ? currentContent.offsetHeight : 0;

    if (viewport) {
        viewport.style.height = currentHeight + 'px';
        viewport.offsetHeight;

        if (currentContent) {
            currentContent.classList.remove('active');
        }
        targetContent.classList.add('active');

        const newHeight = targetContent.offsetHeight;
        viewport.style.height = newHeight + 'px';

        if (window.friendsHeightTimeout) {
            clearTimeout(window.friendsHeightTimeout);
        }
        window.friendsHeightTimeout = setTimeout(() => {
            viewport.style.height = '';
        }, 350);
    }

    try {
        const url = new URL(window.location.href);
        url.searchParams.delete('tab');
        window.history.replaceState({}, document.title, url.pathname);
    } catch (e) {}
};

const initFriendsPage = () => {
    setTimeout(() => {
        window.updateFriendsTabIndicator();
        window.updateFriendsViewportHeight();
    }, 50);
    setTimeout(() => {
        window.updateFriendsTabIndicator();
        window.updateFriendsViewportHeight();
    }, 150);
    setTimeout(() => {
        window.updateFriendsTabIndicator();
        window.updateFriendsViewportHeight();
    }, 350);
};

initFriendsPage();

document.addEventListener('htmx:afterSettle', initFriendsPage);

window.addEventListener('resize', () => {
    window.updateFriendsTabIndicator();
    window.updateFriendsViewportHeight();
});