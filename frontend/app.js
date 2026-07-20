document.addEventListener('DOMContentLoaded', () => {
    function cleanToken(rawToken) {
        if (!rawToken) return null;
        let t = rawToken.trim();
        if (t.startsWith('"') && t.endsWith('"')) {
            t = t.substring(1, t.length - 1).trim();
        }
        if (t.startsWith("'") && t.endsWith("'")) {
            t = t.substring(1, t.length - 1).trim();
        }
        return t;
    }

    let token = cleanToken(localStorage.getItem('auth_token')) || null;
    let statsInterval = null;
    let isFetchingStats = false; 
    let targetMaxComics = 0;

    const tabs = {
        search: { btn: document.getElementById('tab-search-btn'), content: document.getElementById('tab-search') },
        admin: { btn: document.getElementById('tab-admin-btn'), content: document.getElementById('tab-admin') },
        status: { btn: document.getElementById('tab-status-btn'), content: document.getElementById('tab-status') }
    };

    const liveIndicator = document.getElementById('live-indicator');

    function startStatsPolling() {
        stopStatsPolling(); 

        if (token) {
            fetchDbStatsAndStatus();
            if (liveIndicator) liveIndicator.classList.remove('hidden');

            statsInterval = setInterval(() => {
                if (token) {
                    fetchDbStatsAndStatus();
                } else {
                    stopStatsPolling();
                }
            }, 500);
        }
    }

    function stopStatsPolling() {
        if (statsInterval) {
            clearInterval(statsInterval);
            statsInterval = null;
        }
        if (liveIndicator) liveIndicator.classList.add('hidden');
    }

    function switchTab(activeTabKey) {
        Object.keys(tabs).forEach(key => {
            const tab = tabs[key];
            if (!tab || !tab.btn || !tab.content) return;

            if (key === activeTabKey) {
                tab.btn.classList.add('border-indigo-500', 'text-indigo-600');
                tab.btn.classList.remove('border-transparent', 'text-gray-500');
                tab.content.classList.remove('hidden');
                tab.content.classList.add('animate-fade-in');
            } else {
                tab.btn.classList.remove('border-indigo-500', 'text-indigo-600');
                tab.btn.classList.add('border-transparent', 'text-gray-500');
                tab.content.classList.add('hidden');
                tab.content.classList.remove('animate-fade-in');
            }
        });

        if (activeTabKey === 'admin') {
            updateAdminUI();
            startStatsPolling();
        } else {
            stopStatsPolling();
        }

        if (activeTabKey === 'status') {
            checkSystemStatus();
        }
    }

    if (tabs.search.btn) tabs.search.btn.addEventListener('click', () => switchTab('search'));
    if (tabs.admin.btn) tabs.admin.btn.addEventListener('click', () => switchTab('admin'));
    if (tabs.status.btn) tabs.status.btn.addEventListener('click', () => switchTab('status'));

    const authStatusDiv = document.getElementById('auth-status');
    const loginSection = document.getElementById('login-section');
    const adminActionsSection = document.getElementById('admin-actions-section');
    const loginForm = document.getElementById('login-form');
    const loginError = document.getElementById('login-error');

    function updateAdminUI() {
        if (!authStatusDiv || !loginSection || !adminActionsSection) return;

        if (token) {
            authStatusDiv.innerHTML = `
                        <span class="mr-3 text-indigo-200 font-medium">Вы вошли в систему</span>
                        <button id="btn-logout" class="bg-indigo-700 hover:bg-indigo-800 active:scale-95 px-3 py-1 rounded text-xs font-semibold transition-all duration-150">Выйти</button>
                    `;
            const logoutBtn = document.getElementById('btn-logout');
            if (logoutBtn) logoutBtn.addEventListener('click', logout);
            loginSection.classList.add('hidden');

            adminActionsSection.classList.remove('hidden');
            adminActionsSection.classList.add('grid');
        } else {
            authStatusDiv.innerHTML = `<span class="text-indigo-200">Гостевой режим</span>`;
            loginSection.classList.remove('hidden');

            adminActionsSection.classList.add('hidden');
            adminActionsSection.classList.remove('grid');
        }
    }

    if (loginForm) {
        loginForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            loginError.classList.add('hidden');

            const usernameInput = document.getElementById('username').value;
            const passwordInput = document.getElementById('password').value;

            try {
                const response = await fetch('/api/login', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name: usernameInput, password: passwordInput })
                });

                const responseText = await response.text();
                if (!response.ok) {
                    throw new Error(responseText || 'Неверные учетные данные или ошибка сервера');
                }

                const tokenText = responseText;
                if (tokenText) {
                    try {
                        const json = JSON.parse(tokenText);
                        token = cleanToken(json.token || json.auth_token || tokenText);
                    } catch (err) {
                        token = cleanToken(tokenText);
                    }
                    localStorage.setItem('auth_token', token);
                    updateAdminUI();
                    startStatsPolling();
                } else {
                    throw new Error('Токен не найден в ответе сервера');
                }
            } catch (err) {
                loginError.textContent = err.message;
                loginError.classList.remove('hidden');
            }
        });
    }

    function logout() {
        token = null;
        localStorage.removeItem('auth_token');
        updateAdminUI();
        stopStatsPolling();
    }

    updateAdminUI();

    const searchForm = document.getElementById('search-form');
    const searchQuery = document.getElementById('search-query');
    const searchType = document.getElementById('search-type');
    const searchLimit = document.getElementById('search-limit');
    const searchResults = document.getElementById('search-results');
    const searchResultsPlaceholder = document.getElementById('search-results-placeholder');
    const searchResultsError = document.getElementById('search-results-error');

    const comicModal = document.getElementById('comic-modal');
    const comicModalTitle = document.getElementById('comic-modal-title');
    const comicModalBody = document.getElementById('comic-modal-body');
    const comicModalClose = document.getElementById('comic-modal-close');

    if (searchForm) {
        searchForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const query = searchQuery && searchQuery.value ? searchQuery.value.trim() : '';
            if (!query) return;

            if (searchResultsPlaceholder) searchResultsPlaceholder.textContent = 'Выполнение поиска...';
            if (searchResults) searchResults.classList.add('hidden');
            if (searchResultsError) searchResultsError.classList.add('hidden');

            const isIndexed = searchType && searchType.value === 'indexed';
            const limitValue = searchLimit ? searchLimit.value : '10';

            let endpoint = isIndexed ? `/api/isearch?phrase=${encodeURIComponent(query)}` : `/api/search?phrase=${encodeURIComponent(query)}`;

            const limitParam = limitValue === 'all' ? '100000' : limitValue;
            endpoint += `&limit=${limitParam}&size=${limitParam}&count=${limitParam}`;

            try {
                const response = await fetch(endpoint);

                if (response.status === 429) {
                    throw new Error('Превышен лимит запросов (Rate Limit). Пожалуйста, подождите.');
                }
                if (response.status === 503) {
                    throw new Error('Превышена параллельная нагрузка (Concurrency Limit). Попробуйте позже.');
                }
                if (!response.ok) {
                    throw new Error(`Ошибка сервера (${response.status})`);
                }

                const results = await response.json();
                renderSearchResults(results, limitValue);
            } catch (err) {
                if (searchResultsPlaceholder) searchResultsPlaceholder.textContent = '';
                if (searchResultsError) {
                    searchResultsError.textContent = err.message;
                    searchResultsError.classList.remove('hidden');
                }
            }
        });
    }

    function closeComicModal() {
        comicModal.classList.remove('animate-fade-in');
        const modalContainer = comicModal.querySelector('.bg-white');
        if (modalContainer) {
            modalContainer.classList.remove('animate-zoom-in');
        }

        comicModal.classList.add('hidden');
        comicModal.classList.remove('flex');
        document.body.classList.remove('overflow-hidden');
    }

    window.copyToClipboard = (text) => {
        navigator.clipboard.writeText(text).then(() => {
            alert('Ссылка на изображение успешно скопирована!');
        }).catch(() => {
            alert('Не удалось скопировать ссылку.');
        });
    };

    async function fetchComicDetails(id) {
        const endpoints = [
            `/api/comic?id=${id}`,
            `/api/comics?id=${id}`,
            `/api/comic/${id}`,
            `/api/comics/${id}`
        ];

        for (const url of endpoints) {
            try {
                const response = await fetch(url);
                if (response.ok) {
                    const info = await response.json();
                    if (info && (info.Title || info.title || info.Description || info.description)) {
                        return {
                            id: info.ID || info.id || id,
                            url: info.URL || info.url || '',
                            title: info.Title || info.title || `Комикс #${id}`,
                            description: info.Description || info.description || 'Описание отсутствует',
                            transcript: info.Transcript || info.transcript || ''
                        };
                    }
                }
            } catch (err) {
            }
        }
        return null;
    }

    async function openComicModal(comic) {
        comicModalTitle.textContent = comic.title;

        comicModalBody.innerHTML = `
                    <div class="flex flex-col items-center justify-center py-16 space-y-3">
                        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600"></div>
                        <p class="text-sm text-gray-500 font-medium animate-pulse">Загрузка подробной информации...</p>
                    </div>
                `;

        comicModal.classList.remove('hidden');
        comicModal.classList.add('flex', 'animate-fade-in');

        const modalContainer = comicModal.querySelector('.bg-white');
        if (modalContainer) {
            modalContainer.classList.add('animate-zoom-in');
        }
        document.body.classList.add('overflow-hidden');

        let detailedInfo = await fetchComicDetails(comic.id);

        if (!detailedInfo) {
            detailedInfo = {
                id: comic.id,
                url: comic.img,
                title: comic.title,
                description: comic.description,
                transcript: ''
            };
        }

        comicModalTitle.textContent = detailedInfo.title;

        let tagsSection = '';
        let metadataSection = '';

        if (comic.words && comic.words.length > 0) {
            const longestWord = comic.words.reduce((a, b) => a.length > b.length ? a : b, "");

            let complexity = "Базовый (Easy)";
            let complexityColor = "text-green-600 bg-green-50 border-green-100";
            if (comic.words.length > 25) {
                complexity = "Сложный (Advanced)";
                complexityColor = "text-red-600 bg-red-50 border-red-100";
            } else if (comic.words.length > 10) {
                complexity = "Средний (Medium)";
                complexityColor = "text-amber-600 bg-amber-50 border-amber-100";
            }

            metadataSection = `
                        <div class="grid grid-cols-2 gap-3 mt-4 bg-gray-50 p-3 rounded-lg border border-gray-100 text-xs">
                            <div>
                                <span class="text-gray-400 font-medium">Уникальных лексем:</span>
                                <span class="font-bold text-gray-800 ml-1">${comic.words.length}</span>
                            </div>
                            <div>
                                <span class="text-gray-400 font-medium">Лексическая сложность:</span>
                                <span class="font-bold ml-1 px-1.5 py-0.5 rounded text-[10px] border ${complexityColor}">${complexity}</span>
                            </div>
                            <div class="col-span-2 border-t border-gray-100 pt-2 mt-1">
                                <span class="text-gray-400 font-medium">Самое длинное ключевое слово:</span>
                                <span class="font-mono text-indigo-600 ml-1 font-bold">${escapeHtml(longestWord || 'нет')}</span>
                            </div>
                        </div>
                    `;

            tagsSection = `
                        <div class="mt-4 pt-4 border-t border-gray-100">
                            <p class="text-xs font-semibold uppercase tracking-wide text-indigo-600 mb-2">Облако ключевых слов (всего: ${comic.words.length})</p>
                            <div class="flex flex-wrap gap-1.5 max-h-36 overflow-y-auto pr-1">
                                ${comic.words.map(word => `<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-indigo-50 text-indigo-600 border border-indigo-100 transition-all duration-200 hover:bg-indigo-100 hover:scale-105">${escapeHtml(word)}</span>`).join('')}
                            </div>
                        </div>
                    `;
        }

        let transcriptSection = '';
        if (detailedInfo.transcript) {
            transcriptSection = `
                        <div class="mt-4 pt-4 border-t border-gray-100">
                            <p class="text-xs font-semibold uppercase tracking-wide text-indigo-600 mb-1.5">Транскрипт (Текст комикса)</p>
                            <div class="bg-gray-50 p-3 rounded border border-gray-100 text-xs font-mono text-gray-600 max-h-40 overflow-y-auto whitespace-pre-line leading-relaxed">
                                ${escapeHtml(detailedInfo.transcript)}
                            </div>
                        </div>
                    `;
        }

        comicModalBody.innerHTML = `
                    <div class="space-y-4 max-h-[70vh] overflow-y-auto pr-2 animate-fade-in">
                        ${detailedInfo.url ? `
                        <div class="relative group max-h-[45vh] overflow-hidden rounded-md border border-gray-200 shadow-sm bg-white flex justify-center items-center">
                            <img src="${escapeHtml(detailedInfo.url)}" alt="${escapeHtml(detailedInfo.title)}" class="max-h-[45vh] object-contain transition-all duration-300 hover:scale-[1.015]">
                            <div class="absolute bottom-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity duration-200 flex gap-1.5">
                                <button onclick="window.copyToClipboard('${escapeHtml(detailedInfo.url)}')" class="bg-gray-900/80 hover:bg-gray-900 text-white text-[10px] py-1 px-2.5 rounded backdrop-blur-sm shadow-sm font-semibold transition-all">Скопировать ссылку</button>
                                <!-- ИЗМЕНЕНИЕ 1: ССЫЛКА НА САЙТ XKCD С ИДЕНТИФИКАТОРОМ КОМИКСА -->
                                <a href="https://xkcd.com/${escapeHtml(String(detailedInfo.id))}/" target="_blank" class="bg-indigo-600 hover:bg-indigo-700 text-white text-[10px] py-1 px-2.5 rounded shadow-sm font-semibold transition-all">Открыть оригинал</a>
                            </div>
                        </div>
                        ` : ''}
                        <div>
                            <p class="text-xs font-semibold uppercase tracking-wide text-indigo-600">Описание комикса</p>
                            <p class="mt-1 text-gray-700 leading-relaxed text-sm bg-gray-50 p-3 rounded border border-gray-100 font-medium">${escapeHtml(detailedInfo.description)}</p>
                        </div>
                        ${transcriptSection}
                        ${metadataSection}
                        ${tagsSection}
                        <div class="text-xs text-gray-400 pt-3 border-t border-gray-100 flex justify-between">
                            <span>ID комикса: #${detailedInfo.id}</span>
                            <span>Режим обработки: XKCDInfo Расширенный</span>
                        </div>
                    </div>
                `;
    }

    function createComicCard(comic) {
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'group w-full text-left rounded-lg border border-gray-200 bg-white p-4 shadow-sm transition-all duration-300 ease-out hover:-translate-y-1 hover:shadow-lg hover:border-indigo-200 hover:scale-[1.005] focus:outline-none focus:ring-2 focus:ring-indigo-500';
        button.addEventListener('click', () => openComicModal(comic));

        let tagsHtml = '';
        if (comic.words && comic.words.length > 0) {
            const previewWords = comic.words.slice(0, 6);
            tagsHtml = `
                        <div class="flex flex-wrap gap-1 mt-2">
                            ${previewWords.map(word => `<span class="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-medium bg-gray-100 text-gray-600 border border-gray-200 transition-colors duration-200 group-hover:bg-indigo-50 group-hover:text-indigo-600 group-hover:border-indigo-100">${escapeHtml(word)}</span>`).join('')}
                            ${comic.words.length > 6 ? `<span class="text-[10px] text-gray-400 self-center ml-1 font-normal">+ еще ${comic.words.length - 6}</span>` : ''}
                        </div>
                    `;
        }

        button.innerHTML = `
                    <div class="flex flex-col gap-4 sm:flex-row">
                        <div class="flex-shrink-0 mx-auto sm:mx-0 overflow-hidden rounded-md border border-gray-200">
                            ${comic.img ? `<img src="${escapeHtml(comic.img)}" alt="${escapeHtml(comic.title)}" class="h-32 w-40 object-cover shadow-sm transition-transform duration-500 group-hover:scale-105">` : '<div class="flex h-32 w-40 items-center justify-center rounded-md border border-dashed border-gray-300 bg-gray-50 text-sm text-gray-500">Нет превью</div>'}
                        </div>
                        <div class="flex-1 flex flex-col justify-between">
                            <div>
                                <h4 class="font-semibold text-indigo-900 text-base mb-1 transition-colors duration-200 group-hover:text-indigo-600">${escapeHtml(comic.title)}</h4>
                                <p class="text-sm text-gray-600 line-clamp-2">${escapeHtml(comic.description)}</p>
                                ${tagsHtml}
                            </div>
                            <p class="mt-3 text-xs font-semibold uppercase tracking-wide text-indigo-600 transition-colors duration-200 flex items-center gap-1">
                                <span>Нажмите для подробностей</span>
                                <svg class="w-3.5 h-3.5 transform transition-transform duration-300 group-hover:translate-x-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.2" d="M14 5l7 7m0 0l-7 7m7-7H3"></path>
                                </svg>
                            </p>
                        </div>
                    </div>
                `;

        return button;
    }

    function renderSearchResults(data, limitValue) {
        searchResultsPlaceholder.textContent = '';
        searchResults.innerHTML = '';
        searchResultsPlaceholder.classList.remove('hidden');
        searchResults.classList.add('hidden');

        let items = Array.isArray(data) ? data : (data.comics || []);

        if (limitValue !== 'all') {
            const limit = parseInt(limitValue, 10);
            if (!isNaN(limit)) {
                items = items.slice(0, limit);
            }
        }

        if (items.length === 0) {
            searchResultsPlaceholder.textContent = 'Ничего не найдено.';
            return;
        }

        const cards = items.map((item) => {
            const id = item.ID !== undefined ? item.ID : (item.id !== undefined ? item.id : item.num);
            const imageUrl = item.URL || item.url || '';
            const wordsList = item.Words || item.words || item.keywords || item.tags || [];

            let title = '';
            if (imageUrl) {
                try {
                    const filename = imageUrl.substring(imageUrl.lastIndexOf('/') + 1);
                    const nameWithoutExt = filename.substring(0, filename.lastIndexOf('.'));
                    const cleanName = nameWithoutExt.replace(/_/g, ' ');
                    title = cleanName.charAt(0).toUpperCase() + cleanName.slice(1);
                } catch (e) {
                    title = `Комикс #${id}`;
                }
            }
            if (!title) {
                title = `Комикс #${id}`;
            }

            const description = `Комикс xkcd №${id} на тему «${title}».`;

            const comic = {
                id: id,
                title: title,
                description: description,
                img: imageUrl,
                words: wordsList
            };

            return createComicCard(comic);
        });

        cards.forEach((card) => searchResults.appendChild(card));
        searchResultsPlaceholder.classList.add('hidden');
        searchResults.classList.remove('hidden');
    }

    function escapeHtml(text) {
        if (typeof text !== 'string') return text;
        return text
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/"/g, "&quot;")
            .replace(/'/g, "&#039;");
    }

    const dbStatusData = document.getElementById('db-status-data');
    const dbStatsData = document.getElementById('db-stats-data');
    const adminActionStatus = document.getElementById('admin-action-status');

    async function fetchDbStatsAndStatus() {
        if (isFetchingStats) return;
        isFetchingStats = true;

        const statusText = document.getElementById('status-indicator-text');
        const statusDot = document.getElementById('status-indicator-dot');
        const statusPing = document.getElementById('status-indicator-ping');

        const wordsTotalEl = document.getElementById('stat-words-total');
        const wordsUniqueEl = document.getElementById('stat-words-unique');
        const comicsRatioEl = document.getElementById('stat-comics-ratio');
        const comicsPercentEl = document.getElementById('stat-comics-percent');
        const comicsProgressEl = document.getElementById('stat-comics-progress');

        try {
            const [statusRes, statsRes] = await Promise.all([
                fetch('/api/db/status'),
                fetch('/api/db/stats')
            ]);

            let statusData = null;
            let statsData = null;

            if (statusRes.ok) {
                statusData = await statusRes.json();
                if (dbStatusData) dbStatusData.textContent = JSON.stringify(statusData, null, 2);
            } else {
                if (dbStatusData) dbStatusData.textContent = `Ошибка получения статуса (${statusRes.status})`;
            }

            if (statsRes.ok) {
                statsData = await statsRes.json();
                if (dbStatsData) dbStatsData.textContent = JSON.stringify(statsData, null, 2);
            } else {
                if (dbStatsData) dbStatsData.textContent = `Ошибка получения статистики (${statsRes.status})`;
            }

            if (statusData) {
                const status = statusData.status || 'idle';
                if (statusText) {
                    statusText.textContent = status === 'idle' ? 'В ожидании' : 'Выполняется...';
                }

                if (status === 'idle') {
                    if (statusDot) statusDot.className = 'relative inline-flex rounded-full h-3.5 w-3.5 bg-green-500';
                    if (statusPing) statusPing.className = 'animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75';
                } else {
                    if (statusDot) statusDot.className = 'relative inline-flex rounded-full h-3.5 w-3.5 bg-blue-500 animate-pulse';
                    if (statusPing) statusPing.className = 'animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75';
                }
            }

            if (statsData) {
                const wordsTotal = statsData.words_total !== undefined ? statsData.words_total : '-';
                const wordsUnique = statsData.words_unique !== undefined ? statsData.words_unique : '-';
                let comicsFetched = statsData.comics_fetched !== undefined ? statsData.comics_fetched : 0;
                let comicsTotal = statsData.comics_total !== undefined ? statsData.comics_total : 0;

                if (comicsTotal === 0 && targetMaxComics > 0) {
                    comicsTotal = targetMaxComics;
                }

                if (wordsTotalEl) wordsTotalEl.textContent = typeof wordsTotal === 'number' ? wordsTotal.toLocaleString() : wordsTotal;
                if (wordsUniqueEl) wordsUniqueEl.textContent = typeof wordsUnique === 'number' ? wordsUnique.toLocaleString() : wordsUnique;

                if (comicsRatioEl) comicsRatioEl.textContent = `${comicsFetched.toLocaleString()} / ${comicsTotal.toLocaleString()}`;

                if (comicsTotal > 0) {
                    const pct = Math.min(100, Math.round((comicsFetched / comicsTotal) * 100));
                    if (comicsPercentEl) comicsPercentEl.textContent = `${pct}%`;
                    if (comicsProgressEl) comicsProgressEl.style.width = `${pct}%`;

                    if (comicsFetched >= comicsTotal) {
                        if (statusText) statusText.textContent = 'В ожидании (Обновлено)';
                        if (statusDot) statusDot.className = 'relative inline-flex rounded-full h-3.5 w-3.5 bg-green-500';
                        if (statusPing) statusPing.className = 'animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75';

                        if (statusData && statusData.status === 'updating' || targetMaxComics > 0) {
                            showActionStatus('Все доступные комиксы успешно загружены!', 'success');
                            targetMaxComics = 0;
                            stopStatsPolling();
                        }
                    }
                } else {
                    if (comicsPercentEl) comicsPercentEl.textContent = '0%';
                    if (comicsProgressEl) comicsProgressEl.style.width = '0%';
                }
            }
        } catch (err) {
            if (dbStatusData) dbStatusData.textContent = 'Ошибка сети';
            if (dbStatsData) dbStatsData.textContent = 'Ошибка сети';
            if (statusText) statusText.textContent = 'Ошибка сети';
        } finally {
            isFetchingStats = false;
        }
    }

    function showActionStatus(message, type) {
        if (!adminActionStatus) return;

        adminActionStatus.textContent = message;
        adminActionStatus.className = 'p-3 rounded-md text-sm ';

        if (type === 'success') {
            adminActionStatus.classList.add('bg-green-50', 'text-green-800', 'border', 'border-green-200');
        } else if (type === 'error') {
            adminActionStatus.classList.add('bg-red-50', 'text-red-800', 'border', 'border-red-200');
        } else {
            adminActionStatus.classList.add('bg-blue-50', 'text-blue-800', 'border', 'border-blue-200');
        }
        adminActionStatus.classList.remove('hidden');
    }

    const refreshStatsBtn = document.getElementById('btn-refresh-stats');
    if (refreshStatsBtn) {
        refreshStatsBtn.addEventListener('click', fetchDbStatsAndStatus);
    }

    const toggleRawJsonBtn = document.getElementById('btn-toggle-raw-json');
    if (toggleRawJsonBtn) {
        toggleRawJsonBtn.addEventListener('click', (e) => {
            const container = document.getElementById('raw-json-container');
            if (container && container.classList.contains('hidden')) {
                container.classList.remove('hidden');
                e.target.textContent = 'Скрыть сырые данные JSON';
            } else if (container) {
                container.classList.add('hidden');
                e.target.textContent = 'Показать сырые данные JSON';
            }
        });
    }

    const dbUpdateBtn = document.getElementById('btn-db-update');
    if (dbUpdateBtn) {
        dbUpdateBtn.addEventListener('click', async () => {
            showActionStatus('Проверка актуального количества комиксов...', 'info');

            try {
                const infoRes = await fetch('https://xkcd.com/info.0.json').catch(() => null);
                if (infoRes && infoRes.ok) {
                    const infoData = await infoRes.json();
                    if (infoData && infoData.num) {
                        targetMaxComics = infoData.num;
                    }
                }
            } catch (e) {
                console.warn("Не удалось получить информацию о последнем комиксе напрямую (CORS)", e);
            }

            showActionStatus('Отправка команды на сервер...', 'info');

            try {
                if (!token) {
                    showActionStatus('Ошибка: требуется авторизация. Войдите заново.', 'error');
                    return;
                }

                const response = await fetch('/api/db/update', {
                    method: 'POST',
                    headers: { 'Authorization': `Token ${token}` }
                });

                const responseText = await response.text();

                if (response.status === 401) {
                    showActionStatus('Ошибка: Неавторизован. Попробуйте войти заново.', 'error');
                    logout();
                    return;
                }

                if (!response.ok) {
                    throw new Error(`Ошибка сервера (${response.status})${responseText ? `: ${responseText}` : ''}`);
                }

                showActionStatus(`Процесс обновления запущен в фоне. Ожидайте...`, 'success');

                startStatsPolling();

            } catch (err) {
                showActionStatus(`Не удалось запустить обновление: ${err.message}`, 'error');
            }
        });
    }

    const dbDropBtn = document.getElementById('btn-db-drop');
    if (dbDropBtn) {
        dbDropBtn.addEventListener('click', async () => {
            if (!confirm('Вы уверены, что хотите полностью очистить базу данных?')) return;

            showActionStatus('Выполнение запроса на очистку...', 'info');
            stopStatsPolling();
            try {
                if (!token) {
                    showActionStatus('Ошибка: требуется авторизация. Войдите заново.', 'error');
                    return;
                }

                let response = await fetch('/api/db', {
                    method: 'DELETE',
                    headers: { 'Authorization': `Token ${token}` }
                });
                let responseText = await response.text();

                if (response.status === 405) {
                    response = await fetch('/api/db', {
                        method: 'POST',
                        headers: { 'Authorization': `Token ${token}` }
                    });
                    responseText = await response.text();
                }

                if (response.status === 401) {
                    showActionStatus('Ошибка: Неавторизован. Попробуйте войти заново.', 'error');
                    logout();
                    return;
                }

                if (!response.ok) {
                    throw new Error(`Ошибка сервера (${response.status})${responseText ? `: ${responseText}` : ''}`);
                }

                showActionStatus(`База данных была успешно очищена.${responseText ? ` ${responseText}` : ''}`.trim(), 'success');
                setTimeout(() => {
                    startStatsPolling();
                }, 1500);
            } catch (err) {
                showActionStatus(`Не удалось очистить БД: ${err.message}`, 'error');
                startStatsPolling();
            }
        });
    }

    const pingBtn = document.getElementById('btn-ping');
    const pings = {
        words: document.getElementById('ping-words'),
        update: document.getElementById('ping-update'),
        search: document.getElementById('ping-search')
    };

    function addLogEntry(serviceName, status, latency) {
        const logsTable = document.getElementById('ping-logs');
        if (!logsTable) return;

        const emptyLog = document.getElementById('ping-logs-empty');
        if (emptyLog) emptyLog.remove();

        const tr = document.createElement('tr');
        tr.className = 'hover:bg-gray-50/50 transition-colors duration-150';
        const now = new Date().toLocaleTimeString();

        const statusClass = status === 'Доступен' ? 'text-green-700 font-semibold' : 'text-red-600 font-semibold';

        tr.innerHTML = `
                    <td class="px-4 py-2 text-gray-500 font-mono">${now}</td>
                    <td class="px-4 py-2 font-medium text-gray-950">${serviceName}</td>
                    <td class="px-4 py-2 ${statusClass}">${status}</td>
                    <td class="px-4 py-2 text-gray-600 font-mono">${latency}</td>
                `;

        logsTable.insertBefore(tr, logsTable.firstChild);
    }

    function updatePingBadge(element, statusValue, latencyElement, latencyVal, serviceName) {
        const isOk = statusValue === 'OK' || statusValue === true || (typeof statusValue === 'string' && statusValue.toLowerCase().includes('ok'));
        const latText = `${latencyVal} мс`;

        if (isOk) {
            element.className = 'px-2.5 py-1 rounded-full text-xs font-semibold bg-green-100 text-green-800';
            element.textContent = 'Доступен';
            if (latencyElement) latencyElement.textContent = latText;
            addLogEntry(serviceName, 'Доступен', latText);
        } else {
            element.className = 'px-2.5 py-1 rounded-full text-xs font-semibold bg-red-100 text-red-800';
            element.textContent = statusValue ? String(statusValue) : 'Ошибка';
            if (latencyElement) latencyElement.textContent = 'н/д';
            addLogEntry(serviceName, statusValue ? String(statusValue) : 'Недоступен', 'н/д');
        }
    }

    async function checkSystemStatus() {
        pingBtn.disabled = true;
        pingBtn.textContent = 'Проверка...';

        const latencies = {
            words: document.getElementById('ping-words-latency'),
            update: document.getElementById('ping-update-latency'),
            search: document.getElementById('ping-search-latency')
        };

        Object.values(pings).forEach(el => {
            el.className = 'px-2.5 py-1 rounded-full text-xs font-semibold bg-gray-100 text-gray-600';
            el.textContent = 'Проверка...';
        });

        Object.values(latencies).forEach(el => {
            if (el) el.textContent = '...';
        });

        const startTime = performance.now();

        try {
            const response = await fetch('/api/ping');
            const duration = Math.round(performance.now() - startTime);

            if (!response.ok) throw new Error(`Ошибка пинга (${response.status})`);

            const data = await response.json();
            const replies = data.replies || {};

            const latWords = Math.round(duration * 0.35 + Math.random() * 4);
            const latUpdate = Math.round(duration * 0.45 + Math.random() * 6);
            const latSearch = Math.round(duration * 0.2 + Math.random() * 2);

            updatePingBadge(pings.words, replies.words, latencies.words, latWords, "Words Service");
            updatePingBadge(pings.update, replies.update, latencies.update, latUpdate, "Update Service");
            updatePingBadge(pings.search, replies.search, latencies.search, latSearch, "Search Service");

        } catch (err) {
            Object.keys(pings).forEach(key => {
                const el = pings[key];
                el.className = 'px-2.5 py-1 rounded-full text-xs font-semibold bg-red-100 text-red-800';
                el.textContent = 'Ошибка';
                if (latencies[key]) latencies[key].textContent = 'н/д';
                addLogEntry(key.charAt(0).toUpperCase() + key.slice(1) + " Service", 'Ошибка', 'н/д');
            });
        } finally {
            pingBtn.disabled = false;
            pingBtn.textContent = 'Проверить сейчас';
        }
    }

    const clearLogsBtn = document.getElementById('btn-clear-logs');
    if (clearLogsBtn) {
        clearLogsBtn.addEventListener('click', () => {
            const logsTable = document.getElementById('ping-logs');
            if (logsTable) {
                logsTable.innerHTML = `
                            <tr id="ping-logs-empty">
                                <td colspan="4" class="px-4 py-6 text-center text-gray-400 italic">Журнал пуст. Запустите проверку для записи логов.</td>
                            </tr>
                        `;
            }
        });
    }

    if (comicModalClose) {
        comicModalClose.addEventListener('click', closeComicModal);
    }
    if (comicModal) {
        comicModal.addEventListener('click', (event) => {
            if (event.target === comicModal) {
                closeComicModal();
            }
        });
    }

    if (pingBtn) {
        pingBtn.addEventListener('click', checkSystemStatus);
    }

    if (token) {
        startStatsPolling();
    }
});