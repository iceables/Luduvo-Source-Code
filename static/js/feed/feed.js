let feedSocket = null;
let feedOffset = 10;
let loadingFeed = false;
let noMorePosts = false;
let lastSubmittedText = '';
let postObserver = null;
let listenersInitialized = false;
let currentTab = 'worldwide';
let activeFocusedPostId = null;
const reactionCooldowns = new Map();
const REPLY_ICON = '<i class="fa-solid fa-reply"></i>';
const FLAG_ICON = '<i class="fa-solid fa-flag"></i>';
const REACT_ICON = '<i class="fa-regular fa-face-laugh-squint"></i>';
const MAX_THREAD_DEPTH = 3;

const updateOdometer = (element, value) => {
    const valString = String(value);
    let strips = element.querySelectorAll('.dgtcnt');
    const isSmall = element.closest('.cmtrctbdg') !== null;
    const digitHeight = isSmall ? 20 : 28;
    
    if (strips.length !== valString.length) {
        let html = '';
        for (let i = 0; i < valString.length; i++) {
            html += '<span class="dgtcnt"><span class="dgtstp">';
            for (let j = 0; j <= 9; j++) {
                html += `<span class="dgtnum">${j}</span>`;
            }
            html += '</span></span>';
        }
        element.innerHTML = html;
        strips = element.querySelectorAll('.dgtcnt');
    }
    
    for (let i = 0; i < valString.length; i++) {
        const digit = parseInt(valString[i]) || 0;
        const strip = strips[i].querySelector('.dgtstp');
        if (strip) {
            strip.style.transform = `translateY(-${digit * digitHeight}px)`;
        }
    }
};

const animateReactionCounter = (span, targetValue) => {
    if (!span) return;
    updateOdometer(span, 0);
    setTimeout(() => updateOdometer(span, targetValue), 50);
};

const buildAvatar = (userId, username, wrapperClass) => {
    const avt = document.createElement('div');
    avt.className = wrapperClass;
    const img = document.createElement('img');
    img.src = `/api/v1/avatar/headshot/${userId}.png`;
    img.alt = username;
    img.onerror = () => { img.src = '/static/useful/temp/pfp.png'; };
    avt.appendChild(img);
    return avt;
};

const buildTimeTooltip = (timeAgo, fullDate, timeClass) => {
    const ttp = document.createElement('div');
    ttp.className = 'ttpcnt';
    const tme = document.createElement('span');
    tme.className = timeClass;
    tme.textContent = timeAgo;
    const txt = document.createElement('span');
    txt.className = 'ttptxt';
    txt.textContent = fullDate;
    ttp.appendChild(tme);
    ttp.appendChild(txt);
    return ttp;
};

const buildReplyButton = () => {
    const btn = document.createElement('div');
    btn.className = 'pstrpl';
    btn.title = 'Reply';
    btn.innerHTML = REPLY_ICON;
    return btn;
};

const buildFlagLink = href => {
    const a = document.createElement('a');
    a.className = 'pstflg';
    a.title = 'Report';
    a.href = href;
    a.setAttribute('hx-get', href);
    a.setAttribute('hx-target', 'body');
    a.setAttribute('hx-push-url', 'true');
    a.innerHTML = FLAG_ICON;
    return a;
};

const buildReactionBadge = (hasReacted, reactions, isComment) => {
    const bdg = document.createElement('div');
    bdg.className = isComment
        ? (hasReacted ? 'rctbdg cmtrctbdg reacted' : 'rctbdg cmtrctbdg')
        : (hasReacted ? 'rctbdg reacted' : 'rctbdg');
    const ico = document.createElement('div');
    ico.className = 'rctico';
    ico.innerHTML = REACT_ICON;
    const cnt = document.createElement('span');
    cnt.className = 'rctcnt';
    cnt.dataset.target = reactions || 0;
    bdg.appendChild(ico);
    bdg.appendChild(cnt);
    return { bdg, cnt };
};

const updateTabIndicator = () => {
    const indicator = document.querySelector('.tabind');
    const activeTab = document.querySelector('.tabbtn.active');
    if (indicator && activeTab) {
        indicator.style.left = `${activeTab.offsetLeft}px`;
        indicator.style.width = `${activeTab.offsetWidth}px`;
        indicator.style.height = `${activeTab.offsetHeight}px`;
        indicator.style.top = `${activeTab.offsetTop}px`;
    }
};

const toggleExpandFeedPost = element => {
    const postItem = element.closest('.pstitm');
    const postCard = element.closest('.pstcrd');
    if (postItem && postCard) {
        postItem.classList.toggle('expanded');
        postCard.classList.toggle('expanded');
        element.classList.toggle('expanded');
    }
};

const incrementReaction = element => {
    const wrapper = element.closest('.pstitm');
    if (!wrapper) return;
    const fid = parseInt(wrapper.dataset.id);
    const feedType = wrapper.dataset.feedType || currentTab;

    const key = `post:${fid}:${feedType}`;
    const now = Date.now();
    if (reactionCooldowns.has(key) && now - reactionCooldowns.get(key) < 2000) {
        showFeedError('Please wait a moment before reacting again.');
        return;
    }

    if (fid && feedSocket && feedSocket.readyState === WebSocket.OPEN) {
        reactionCooldowns.set(key, now);
        feedSocket.send(JSON.stringify({ type: 'react', fid, feed_type: feedType }));
    }
};

const incrementCommentReaction = element => {
    const cmtCard = element.closest('.cmtitm');
    if (!cmtCard) return;
    const cid = parseInt(cmtCard.dataset.commentId);

    const key = `cmt:${cid}`;
    const now = Date.now();
    if (reactionCooldowns.has(key) && now - reactionCooldowns.get(key) < 2000) {
        showFeedError('Please wait a moment before reacting again.');
        return;
    }

    if (cid && feedSocket && feedSocket.readyState === WebSocket.OPEN) {
        reactionCooldowns.set(key, now);
        feedSocket.send(JSON.stringify({ type: 'creact', cid, feed_type: currentTab }));
    }
};

const countTotalComments = commentsList => {
    if (!commentsList) return 0;
    let count = 0;
    commentsList.forEach(c => {
        count += 1;
        if (c.replies && c.replies.length > 0) {
            count += countTotalComments(c.replies);
        }
    });
    return count;
};

const createCommentElement = (comment, parentUser, depth = 0, includeReplies = true) => {
    const wrapper = document.createElement('div');
    wrapper.className = 'cmtitm';
    wrapper.dataset.commentId = comment.id;

    const card = document.createElement('div');
    card.className = 'cmtcrd';

    card.appendChild(buildAvatar(comment.user_id, comment.username, 'cmtavt'));

    const header = document.createElement('div');
    header.className = 'cmthdr';

    const author = document.createElement('a');
    author.className = 'cmtaut';
    author.href = `/user/${comment.user_id}`;
    author.setAttribute('hx-get', `/user/${comment.user_id}`);
    author.setAttribute('hx-target', 'body');
    author.setAttribute('hx-push-url', 'true');
    author.textContent = comment.username;

    header.appendChild(author);
    header.appendChild(buildTimeTooltip(comment.time_ago || 'Just now', comment.full_date || '', 'cmttme'));
    card.appendChild(header);

    if (parentUser) {
        const rep = document.createElement('div');
        rep.className = 'cmtrep';
        const arrow = document.createElement('i');
        arrow.className = 'fa-solid fa-reply';
        const lbl = document.createElement('span');
        lbl.textContent = 'replying to';
        const link = document.createElement('a');
        link.href = `/user/${parentUser.id}`;
        link.setAttribute('hx-get', `/user/${parentUser.id}`);
        link.setAttribute('hx-target', 'body');
        link.setAttribute('hx-push-url', 'true');
        link.textContent = `@${parentUser.username}`;
        rep.appendChild(arrow);
        rep.appendChild(lbl);
        rep.appendChild(link);
        card.appendChild(rep);
    }

    const actions = document.createElement('div');
    actions.className = 'cmtact';
    actions.appendChild(buildReplyButton());
    actions.appendChild(buildFlagLink(`/report/comment/${comment.id}`));
    card.appendChild(actions);

    const body = document.createElement('div');
    body.className = 'cmtbdy';
    body.textContent = comment.comment;
    card.appendChild(body);

    const { bdg: reactBdg, cnt: reactCnt } = buildReactionBadge(comment.has_reacted, comment.reactions, true);
    animateReactionCounter(reactCnt, comment.reactions || 0);

    wrapper.appendChild(card);
    wrapper.appendChild(reactBdg);

    if (includeReplies && comment.replies && comment.replies.length > 0) {
        const childDepth = depth + 1;
        const nested = document.createElement('div');
        nested.className = childDepth > MAX_THREAD_DEPTH ? 'cmtnest cmtnest-max' : 'cmtnest';
        const childParent = { id: comment.user_id, username: comment.username };
        comment.replies.forEach(r => {
            nested.appendChild(createCommentElement(r, childParent, childDepth, true));
        });
        wrapper.appendChild(nested);
    }

    return wrapper;
};

const renderCommentsForPost = (postWrapper, comments, isFocused) => {
    const cmtList = postWrapper.querySelector('.cmtlst');
    if (!cmtList) return;
    cmtList.innerHTML = '';

    if (!comments || comments.length === 0) return;

    if (isFocused) {
        comments.forEach(c => {
            cmtList.appendChild(createCommentElement(c, null, 0, true));
        });
        return;
    }

    const displayLimit = 2;
    const displayCount = Math.min(displayLimit, comments.length);
    for (let i = 0; i < displayCount; i++) {
        cmtList.appendChild(createCommentElement(comments[i], null, 0, false));
    }

    const totalCount = countTotalComments(comments);
    const extraCount = totalCount - displayCount;

    if (extraCount > 0) {
        const more = document.createElement('a');
        more.className = 'cmtmre';
        more.textContent = `View ${extraCount} more ${extraCount === 1 ? 'reply' : 'replies'}`;
        more.onclick = e => {
            e.preventDefault();
            focusSinglePost(postWrapper);
        };
        cmtList.appendChild(more);
    }
};

const loadPostComments = (postWrapper, isFocused = false) => {
    const feedId = postWrapper.dataset.id;
    const feedType = postWrapper.dataset.feedType || currentTab;

    fetch(`/api/v1/feed/comments?feed_id=${feedId}&feed_type=${feedType}`)
        .then(res => res.json())
        .then(comments => {
            renderCommentsForPost(postWrapper, comments, isFocused);
        })
        .catch(() => {});
};

const toggleInlineReplyInput = targetEl => {
    const postItem = targetEl.closest('.pstitm');
    const commentItem = targetEl.closest('.cmtitm');
    if (!postItem) return;

    const replyBtn = targetEl.closest('.pstrpl') || (commentItem ? commentItem.querySelector('.pstrpl') : postItem.querySelector('.pstrpl'));

    let existingInput = commentItem ? commentItem.querySelector('.cmtinbox') : postItem.querySelector('.cmtinbox');
    if (existingInput) {
        existingInput.classList.add('closing');
        if (replyBtn) replyBtn.classList.remove('active');
        setTimeout(() => existingInput.remove(), 220);
        return;
    }

    if (replyBtn) replyBtn.classList.add('active');

    const feedId = postItem.dataset.id;
    const parentId = commentItem ? commentItem.dataset.commentId : null;

    const form = document.createElement('form');
    form.className = 'cmtinbox';
    form.dataset.feedId = feedId;
    if (parentId) form.dataset.parentId = parentId;

    const input = document.createElement('input');
    input.type = 'text';
    input.className = 'feedin';
    input.placeholder = commentItem ? `Reply to @${commentItem.querySelector('.cmtaut').textContent}...` : 'Write a reply...';
    input.required = true;
    input.autocomplete = 'off';

    const btn = document.createElement('button');
    btn.type = 'submit';
    btn.className = 'feedsnd';
    btn.innerHTML = '<i class="fa-solid fa-paper-plane" style="font-size: 16px !important;"></i>';

    form.appendChild(input);
    form.appendChild(btn);

    form.onsubmit = e => {
        e.preventDefault();
        const content = input.value.trim();
        if (!content) return;

        if (feedSocket && feedSocket.readyState === WebSocket.OPEN) {
            const payload = {
                type: 'comment',
                content: content,
                feed_id: parseInt(feedId),
                feed_type: currentTab
            };
            if (parentId) payload.parent_id = parseInt(parentId);

            feedSocket.send(JSON.stringify(payload));
            if (replyBtn) replyBtn.classList.remove('active');
            form.classList.add('closing');
            setTimeout(() => form.remove(), 220);
        } else {
            showFeedError('WebSocket connection is closed.');
        }
    };

    if (commentItem) {
        const nestedList = commentItem.querySelector('.cmtnest');
        if (nestedList) {
            commentItem.insertBefore(form, nestedList);
        } else {
            commentItem.appendChild(form);
        }
    } else {
        postItem.insertBefore(form, postItem.querySelector('.cmtlst'));
    }
    input.focus();
};

const focusSinglePost = postItem => {
    const postsList = document.querySelector('.pstlst');
    if (!postsList) return;

    activeFocusedPostId = postItem.dataset.id;

    const allPosts = postsList.querySelectorAll('.pstitm');
    allPosts.forEach(p => {
        if (p !== postItem) {
            p.style.display = 'none';
        }
    });

    const postCard = postItem.querySelector('.pstcrd');
    if (postCard) {
        postCard.classList.remove('focused');
        void postCard.offsetWidth;
        postCard.classList.add('focused');
    }

    let backBtn = document.getElementById('backToFeedBtn');
    if (!backBtn) {
        backBtn = document.createElement('a');
        backBtn.id = 'backToFeedBtn';
        backBtn.className = 'bkcbtn';
        backBtn.href = '#';
        backBtn.textContent = 'Back to Feed';
        backBtn.onclick = e => {
            e.preventDefault();
            resetFeedView();
        };
        postsList.insertBefore(backBtn, postsList.firstChild);
    }

    loadPostComments(postItem, true);
};

const resetFeedView = () => {
    activeFocusedPostId = null;
    const postsList = document.querySelector('.pstlst');
    if (!postsList) return;

    const backBtn = document.getElementById('backToFeedBtn');
    if (backBtn) backBtn.remove();

    const allInputs = postsList.querySelectorAll('.cmtinbox');
    allInputs.forEach(input => input.remove());

    const allReplyBtns = postsList.querySelectorAll('.pstrpl.active');
    allReplyBtns.forEach(btn => btn.classList.remove('active'));

    const allCards = postsList.querySelectorAll('.pstcrd.focused');
    allCards.forEach(c => c.classList.remove('focused'));

    const allPosts = postsList.querySelectorAll('.pstitm');
    allPosts.forEach(p => {
        p.style.display = '';
        loadPostComments(p, false);
    });
};

const createPost = post => {
    const id = post.id ?? post.ID;
    const username = post.username ?? post.Username;
    const userId = post.user_id ?? post.UserID;
    const content = post.content ?? post.Content;
    const timeAgo = post.time_ago ?? post.TimeAgo;
    const fullDate = post.full_date ?? post.FullDate ?? new Date().toLocaleString();
    const reactions = post.reactions ?? post.Reactions ?? 0;
    const hasReacted = post.has_reacted ?? post.HasReacted ?? false;
    const feedType = post.feed_type ?? post.FeedType ?? currentTab;

    const wrapper = document.createElement('div');
    wrapper.className = 'pstitm';
    wrapper.dataset.id = id;
    wrapper.dataset.username = username;
    wrapper.dataset.feedType = feedType;

    const postCard = document.createElement('div');
    postCard.className = 'pstcrd';

    postCard.appendChild(buildAvatar(userId, username, 'pstavt'));

    const postHeader = document.createElement('div');
    postHeader.className = 'psthdr';

    const postAuthor = document.createElement('a');
    postAuthor.className = 'pstaut';
    postAuthor.href = `/user/${userId}`;
    postAuthor.setAttribute('hx-get', `/user/${userId}`);
    postAuthor.setAttribute('hx-target', 'body');
    postAuthor.setAttribute('hx-push-url', 'true');
    postAuthor.textContent = username;

    postHeader.appendChild(postAuthor);
    postHeader.appendChild(buildTimeTooltip(timeAgo, fullDate, 'psttme'));
    postCard.appendChild(postHeader);

    const postActions = document.createElement('div');
    postActions.className = 'pstact';
    postActions.appendChild(buildReplyButton());

    const postReactBtn = document.createElement('div');
    postReactBtn.className = 'pstrct';
    postReactBtn.title = 'Add Reaction';
    const reactImg = document.createElement('img');
    reactImg.src = '/static/useful/icons/fadd.png';
    reactImg.alt = 'Add Reaction';
    reactImg.className = 'pstico';
    postReactBtn.appendChild(reactImg);
    postActions.appendChild(postReactBtn);

    postActions.appendChild(buildFlagLink(`/report/feed/${id}`));
    postCard.appendChild(postActions);

    const postBody = document.createElement('div');
    postBody.className = 'pbdy';
    postBody.textContent = content;
    postCard.appendChild(postBody);

    const { bdg: reactBdg } = buildReactionBadge(hasReacted, reactions, false);

    const cmtList = document.createElement('div');
    cmtList.className = 'cmtlst';
    cmtList.dataset.feedId = id;

    wrapper.appendChild(postCard);
    wrapper.appendChild(reactBdg);
    wrapper.appendChild(cmtList);

    loadPostComments(wrapper, false);

    return wrapper;
};

const fetchFeedPosts = (tabName, offset = 0, limit = 10) => {
    loadingFeed = true;
    const postsList = document.querySelector('.pstlst');
    if (!postsList) return;

    fetch(`/api/v1/feed?type=${tabName}&offset=${offset}&limit=${limit}`)
        .then(res => {
            if (!res.ok) throw new Error('Failed to load feed');
            return res.json();
        })
        .then(data => {
            if (currentTab !== tabName) return;

            if (offset === 0) {
                postsList.innerHTML = '';
            }

            if (data && data.length > 0) {
                data.forEach(post => {
                    post.feed_type = tabName;
                    const wrapper = createPost(post);
                    postsList.appendChild(wrapper);
                    if (postObserver) {
                        postObserver.observe(wrapper);
                    }
                    const countSpan = wrapper.querySelector('.rctcnt');
                    const reactions = post.reactions ?? post.Reactions ?? 0;
                    animateReactionCounter(countSpan, reactions);
                });
                feedOffset = offset + data.length;
            } else {
                if (offset === 0 && tabName === 'friends') {
                    const placeholder = document.createElement('div');
                    placeholder.id = 'friends-feed-empty';
                    placeholder.className = 'feedemt';
                    placeholder.textContent = 'No active friends are posting right now.';
                    postsList.appendChild(placeholder);
                }
                noMorePosts = true;
            }
            loadingFeed = false;
        })
        .catch(() => {
            loadingFeed = false;
        });
};

const switchFeedTab = (clickedTab, tabName) => {
    if (currentTab === tabName) return;

    const tabs = document.querySelectorAll('.tabbtn');
    tabs.forEach(tab => tab.classList.remove('active'));
    clickedTab.classList.add('active');

    updateTabIndicator();

    currentTab = tabName;
    resetFeedView();

    const feedInput = document.querySelector('.feedin');
    if (feedInput) {
        feedInput.placeholder = tabName === 'friends' ? 'Chat in Friends Feed' : 'Chat in Worldwide Feed';
    }
    const hiddenTypeInput = document.getElementById('feedTypeInput');
    if (hiddenTypeInput) {
        hiddenTypeInput.value = tabName;
    }

    feedOffset = 0;
    noMorePosts = false;
    fetchFeedPosts(tabName, 0, 10);
};

const showFeedError = message => {
    const tabContainer = document.querySelector('.tabcnt');
    const feedForm = document.querySelector('.feedinbox');
    const postsList = document.querySelector('.pstlst');
    if (!tabContainer || !feedForm || !postsList) return;

    let errDiv = tabContainer.querySelector('.feederr');
    if (!errDiv) {
        errDiv = document.createElement('span');
        errDiv.className = 'feederr';
        tabContainer.insertBefore(errDiv, postsList);
    }
    errDiv.textContent = message;
    tabContainer.classList.add('haserr');
    feedForm.classList.add('inerr');

    requestAnimationFrame(() => errDiv.classList.add('visible'));

    setTimeout(() => {
        errDiv.classList.remove('visible');
        tabContainer.classList.remove('haserr');
        feedForm.classList.remove('inerr');
    }, 4000);
};

const setupGlobalDelegation = () => {
    if (listenersInitialized) return;
    listenersInitialized = true;

    document.addEventListener('click', e => {
        const tabBtn = e.target.closest('.tabbtn');
        if (tabBtn) {
            const tabName = tabBtn.dataset.tab;
            if (tabName) {
                switchFeedTab(tabBtn, tabName);
                return;
            }
        }

        const replyBtn = e.target.closest('.pstrpl');
        if (replyBtn && document.querySelector('.pstlst')?.contains(replyBtn)) {
            toggleInlineReplyInput(replyBtn);
            return;
        }

        const cmtReact = e.target.closest('.cmtrctbdg');
        if (cmtReact) {
            incrementCommentReaction(cmtReact);
            return;
        }

        const reactBdg = e.target.closest('.pstitm > .rctbdg:not(.cmtrctbdg)');
        if (reactBdg && document.querySelector('.pstlst')?.contains(reactBdg)) {
            incrementReaction(reactBdg);
            return;
        }

        const postBody = e.target.closest('.pbdy');
        if (postBody && document.querySelector('.pstlst')?.contains(postBody)) {
            const postWrapper = postBody.closest('.pstitm');
            if (postWrapper) {
                if (activeFocusedPostId) {
                    toggleExpandFeedPost(postBody);
                } else {
                    focusSinglePost(postWrapper);
                }
            }
            return;
        }
    });

    document.addEventListener('submit', e => {
        const feedForm = e.target.closest('.feedinbox');
        if (feedForm && !feedForm.classList.contains('cmtinbox')) {
            e.preventDefault();
            const input = feedForm.querySelector('.feedin');
            const content = input ? input.value.trim() : '';
            if (content) {
                if (feedSocket && feedSocket.readyState === WebSocket.OPEN) {
                    lastSubmittedText = content;
                    feedSocket.send(JSON.stringify({ type: 'post', content, feed_type: currentTab }));
                    input.value = '';
                } else {
                    showFeedError('WebSocket connection is closed. Please refresh the page.');
                }
            }
        }
    });

    window.addEventListener('resize', updateTabIndicator);
    window.addEventListener('scroll', onWindowScroll);
};

const onWindowScroll = () => {
    const postsList = document.querySelector('.pstlst');
    if (loadingFeed || noMorePosts || !postsList || activeFocusedPostId) return;

    const scrollTop = window.scrollY || document.documentElement.scrollTop || document.body.scrollTop || 0;
    const windowHeight = window.innerHeight || document.documentElement.clientHeight;
    const documentHeight = Math.max(
        document.body.scrollHeight,
        document.body.offsetHeight,
        document.documentElement.clientHeight,
        document.documentElement.scrollHeight,
        document.documentElement.offsetHeight
    );

    const triggerOffset = 200;
    if (scrollTop + windowHeight >= documentHeight - triggerOffset) {
        fetchFeedPosts(currentTab, feedOffset, 10);
    }
};

const initFeed = () => {
    const tabContainer = document.querySelector('.tabcnt');
    const postsList = document.querySelector('.pstlst');

    if (!tabContainer || !postsList) {
        if (feedSocket) {
            feedSocket.close();
            feedSocket = null;
            window.feedSocket = null;
        }
        return;
    }

    resetFeedView();

    currentTab = 'worldwide';
    feedOffset = document.querySelectorAll('.pstitm').length || 10;
    loadingFeed = false;
    noMorePosts = false;

    setupGlobalDelegation();

    const initialPosts = document.querySelectorAll('.pstitm');
    initialPosts.forEach(item => {
        loadPostComments(item, false);
    });

    const counts = document.querySelectorAll('.rctcnt');
    counts.forEach(span => {
        const target = parseInt(span.dataset.target) || 0;
        animateReactionCounter(span, target);
    });

    setTimeout(updateTabIndicator, 50);
    setTimeout(updateTabIndicator, 150);
    setTimeout(updateTabIndicator, 300);

    if (postObserver) {
        postObserver.disconnect();
    }

    postObserver = new IntersectionObserver(entries => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.classList.remove('offscreen');
            } else {
                entry.target.classList.add('offscreen');
            }
        });
    }, {
        rootMargin: '200px 0px 200px 0px'
    });

    document.querySelectorAll('.pstitm').forEach(item => {
        postObserver.observe(item);
    });

    if (feedSocket) {
        feedSocket.close();
    }

    const wsProtocol = location.protocol === 'https:' ? 'wss' : 'ws';
    feedSocket = new WebSocket(`${wsProtocol}://${location.host}/ws/feed`);
    window.feedSocket = feedSocket;

    feedSocket.onmessage = event => {
        let data;
        try {
            data = JSON.parse(event.data);
        } catch {
            return;
        }

        switch (data.type) {
        case 'new_post':
            if (data.feed_type === currentTab && !activeFocusedPostId) {
                feedOffset++;
                if (postsList) {
                    const emptyMsg = document.getElementById('friends-feed-empty');
                    if (emptyMsg) {
                        emptyMsg.remove();
                    }
                    const wrapper = createPost(data);
                    postsList.insertBefore(wrapper, postsList.firstChild);
                    if (postObserver) {
                        postObserver.observe(wrapper);
                    }

                    const countSpan = wrapper.querySelector('.rctcnt');
                    animateReactionCounter(countSpan, data.reactions ?? 0);
                }
            }
            break;

        case 'new_comment': {
            const comment = data.comment;
            if (!comment) break;
            const postWrapper = document.querySelector(`.pstitm[data-id="${comment.feed_id}"][data-feed-type="${comment.feed_type}"]`) ||
                            document.querySelector(`.pstitm[data-id="${comment.feed_id}"]`);
            if (postWrapper) {
                loadPostComments(postWrapper, activeFocusedPostId === String(comment.feed_id));
            }
            break;
        }

        case 'reaction_update': {
            const wrapper = document.querySelector(`.pstitm[data-id="${data.fid}"][data-feed-type="${data.feed_type}"]`) ||
                            document.querySelector(`.pstitm[data-id="${data.fid}"]`);
            if (wrapper) {
                const countSpan = wrapper.querySelector('.pstitm > .rctbdg:not(.cmtrctbdg) .rctcnt');
                if (countSpan) {
                    countSpan.dataset.target = data.reactions;
                    updateOdometer(countSpan, data.reactions);
                }
                const badge = wrapper.querySelector('.pstitm > .rctbdg:not(.cmtrctbdg)');
                if (badge) {
                    if (data.has_reacted) {
                        badge.classList.add('reacted');
                    } else {
                        badge.classList.remove('reacted');
                    }
                    badge.classList.add('rctpop');
                    setTimeout(() => badge.classList.remove('rctpop'), 300);
                }
            }
            break;
        }

        case 'comment_reaction_update': {
            const wrapper = document.querySelector(`.cmtitm[data-comment-id="${data.cid}"]`);
            if (wrapper) {
                const countSpan = wrapper.querySelector('.cmtrctbdg .rctcnt');
                if (countSpan) {
                    countSpan.dataset.target = data.reactions;
                    updateOdometer(countSpan, data.reactions);
                }
                const badge = wrapper.querySelector('.cmtrctbdg');
                if (badge) {
                    if (data.has_reacted) {
                        badge.classList.add('reacted');
                    } else {
                        badge.classList.remove('reacted');
                    }
                    badge.classList.add('rctpop');
                    setTimeout(() => badge.classList.remove('rctpop'), 300);
                }
            }
            break;
        }

        case 'error': {
            showFeedError(data.content);
            const input = document.querySelector('.feedin');
            if (input && lastSubmittedText) {
                input.value = lastSubmittedText;
            }
            break;
        }
        }
    };
};

initFeed();

document.addEventListener('htmx:afterSettle', initFeed);

document.addEventListener('htmx:beforeHistorySave', () => {
    resetFeedView();
});

document.addEventListener('htmx:beforeTransition', () => {
    resetFeedView();
    if (postObserver) {
        postObserver.disconnect();
    }
    if (feedSocket) {
        feedSocket.close();
        feedSocket = null;
        window.feedSocket = null;
    }
});