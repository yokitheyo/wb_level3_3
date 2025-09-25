class CommentTree {
    constructor() {
        this.apiUrl = '/comments';
        this.currentPage = 1;
        this.limit = 10;
        this.currentSort = 'asc';
        this.isSearchMode = false;
        this.currentQuery = '';
        this.replyToId = null;
        this.hasMore = true;
        this.collapsedComments = new Set();

        this.initElements();
        this.attachEventListeners();
        this.loadComments();
    }

    initElements() {
        // Поиск
        this.searchInput = document.getElementById('searchInput');
        this.searchBtn = document.getElementById('searchBtn');
        this.clearBtn = document.getElementById('clearBtn');

        // Добавление
        this.authorInput = document.getElementById('authorInput');
        this.contentInput = document.getElementById('contentInput');
        this.addCommentBtn = document.getElementById('addCommentBtn');

        // Контейнер
        this.commentsContainer = document.getElementById('commentsContainer');
        this.sortSelect = document.getElementById('sortSelect');
        this.loading = document.getElementById('loading');
        this.noComments = document.getElementById('noComments');

        // Пагинация
        this.prevBtn = document.getElementById('prevBtn');
        this.nextBtn = document.getElementById('nextBtn');
        this.pageInfo = document.getElementById('pageInfo');

        // Модалка
        this.replyModal = document.getElementById('replyModal');
        this.replyTo = document.getElementById('replyTo');
        this.replyAuthorInput = document.getElementById('replyAuthorInput');
        this.replyContentInput = document.getElementById('replyContentInput');
        this.submitReplyBtn = document.getElementById('submitReplyBtn');
        this.cancelReplyBtn = document.getElementById('cancelReplyBtn');
        this.closeModal = document.getElementById('closeModal');
    }

    attachEventListeners() {
        this.searchBtn.addEventListener('click', () => this.search());
        this.clearBtn.addEventListener('click', () => this.clearSearch());
        this.searchInput.addEventListener('keypress', e => e.key === 'Enter' && this.search());

        this.addCommentBtn.addEventListener('click', () => this.createComment());
        this.contentInput.addEventListener('keypress', e => e.key === 'Enter' && e.ctrlKey && this.createComment());

        this.sortSelect.addEventListener('change', () => {
            this.currentSort = this.sortSelect.value;
            this.currentPage = 1;
            this.loadComments();
        });

        this.prevBtn.addEventListener('click', () => this.prevPage());
        this.nextBtn.addEventListener('click', () => this.nextPage());

        this.submitReplyBtn.addEventListener('click', () => this.submitReply());
        this.cancelReplyBtn.addEventListener('click', () => this.closeReplyModal());
        this.closeModal.addEventListener('click', () => this.closeReplyModal());
        this.replyModal.addEventListener('click', e => e.target === this.replyModal && this.closeReplyModal());
        document.addEventListener('keydown', e => e.key === 'Escape' && this.replyModal.style.display === 'block' && this.closeReplyModal());
    }

    async apiCall(url, options = {}) {
        try {
            const res = await fetch(url, {
                headers: { 'Content-Type': 'application/json' },
                ...options
            });
            if (!res.ok) {
                const err = await res.json();
                throw new Error(err.error || 'Ошибка сервера');
            }
            return res.status === 204 ? null : await res.json();
        } catch (e) {
            alert(e.message);
            throw e;
        }
    }

    showLoading(show = true) {
        this.loading.style.display = show ? 'block' : 'none';
        this.noComments.style.display = !show && this.commentsContainer.children.length === 0 ? 'block' : 'none';
    }

    async loadComments() {
        this.showLoading(true);
        const offset = (this.currentPage - 1) * this.limit;
        let url = this.isSearchMode
            ? `${this.apiUrl}/search?query=${encodeURIComponent(this.currentQuery)}&limit=${this.limit}&offset=${offset}`
            : `${this.apiUrl}?limit=${this.limit}&offset=${offset}&sort=${this.currentSort}`;
        try {
            const comments = await this.apiCall(url);
            this.renderComments(comments || []);
            this.hasMore = comments && comments.length === this.limit;
            this.updatePagination();
        } catch {
            this.renderComments([]);
            this.hasMore = false;
            this.updatePagination();
        } finally {
            this.showLoading(false);
        }
    }

    renderComments(comments) {
        // Очистка
        this.commentsContainer.innerHTML = '';
        if (!comments.length) return;

        comments.forEach(comment => this.renderComment(comment, 0, this.commentsContainer));
    }

    renderComment(comment, level, container) {
        const commentEl = document.createElement('div');
        commentEl.className = 'comment';
        commentEl.style.setProperty('--level', level);
        commentEl.dataset.id = comment.id;

        const totalChildren = this.countChildren(comment);
        const isCollapsed = this.collapsedComments.has(comment.id);
        const content = comment.deleted ? '[Комментарий удален]' : comment.content;

        commentEl.innerHTML = `
            <div class="comment-wrapper">
                <div class="comment-header">
                    <div class="comment-meta">
                        ${totalChildren ? `<button class="collapse-btn ${isCollapsed ? 'collapsed' : ''}" data-id="${comment.id}">${isCollapsed ? '▶' : '▼'}</button>` : '<span class="collapse-spacer">•</span>'}
                        <span class="comment-author">${comment.author}</span>
                        <span class="comment-date">${new Date(comment.created_at).toLocaleString()}</span>
                        ${totalChildren ? `<span class="children-count">(${totalChildren} ${this.getChildrenText(totalChildren)})</span>` : ''}
                    </div>
                    <div class="comment-actions">
                        ${!comment.deleted ? `<button class="reply-btn" data-id="${comment.id}" data-author="${comment.author}">Ответить</button>
                        <button class="delete-btn" data-id="${comment.id}">Удалить</button>` : ''}
                    </div>
                </div>
                <div class="comment-content">${content}</div>
            </div>
        `;

        // События
        if (!comment.deleted) {
            commentEl.querySelector('.reply-btn')?.addEventListener('click', () => this.openReplyModal(comment.id, comment.author, comment.content));
            commentEl.querySelector('.delete-btn')?.addEventListener('click', () => this.deleteComment(comment.id));
        }

        commentEl.querySelector('.collapse-btn')?.addEventListener('click', () => {
            this.toggleCollapse(comment.id);
        });

        // Контейнер для детей
        const childrenContainer = document.createElement('div');
        childrenContainer.className = 'children-container';
        commentEl.appendChild(childrenContainer);
        container.appendChild(commentEl);

        // Рекурсивно рендерим детей если не свернут
        if (comment.children && comment.children.length && !isCollapsed) {
            comment.children.forEach(child => this.renderComment(child, level + 1, childrenContainer));
        }
    }

    countChildren(comment) {
        if (!comment.children || !comment.children.length) return 0;
        let count = comment.children.length;
        comment.children.forEach(c => count += this.countChildren(c));
        return count;
    }

    getChildrenText(count) {
        if (count % 10 === 1 && count % 100 !== 11) return 'ответ';
        if ([2,3,4].includes(count % 10) && ![12,13,14].includes(count % 100)) return 'ответа';
        return 'ответов';
    }

    toggleCollapse(id) {
        this.collapsedComments.has(id) ? this.collapsedComments.delete(id) : this.collapsedComments.add(id);
        this.loadComments();
    }

    async createComment(parentId = null) {
        const author = parentId ? this.replyAuthorInput.value.trim() : this.authorInput.value.trim();
        const content = parentId ? this.replyContentInput.value.trim() : this.contentInput.value.trim();
        if (!author || !content) return alert('Заполните все поля');

        const payload = { author, content, ...(parentId && { parent_id: parentId }) };
        await this.apiCall(this.apiUrl, { method: 'POST', body: JSON.stringify(payload) });

        if (parentId) this.closeReplyModal();
        else { this.authorInput.value = ''; this.contentInput.value = ''; }

        this.loadComments();
    }

    async deleteComment(id) {
        if (!confirm('Удалить комментарий?')) return;
        await this.apiCall(`${this.apiUrl}/${id}`, { method: 'DELETE' });
        this.loadComments();
    }

    async search() {
        const query = this.searchInput.value.trim();
        if (!query) return alert('Введите запрос');
        this.isSearchMode = true;
        this.currentQuery = query;
        this.currentPage = 1;
        this.collapsedComments.clear();
        this.loadComments();
    }

    clearSearch() {
        this.searchInput.value = '';
        this.isSearchMode = false;
        this.currentQuery = '';
        this.currentPage = 1;
        this.collapsedComments.clear();
        this.loadComments();
    }

    openReplyModal(id, author, content) {
        this.replyToId = id;
        this.replyTo.textContent = `Ответ на ${author}: "${content.slice(0,100)}${content.length>100?'...':''}"`;
        this.replyAuthorInput.value = '';
        this.replyContentInput.value = '';
        this.replyModal.style.display = 'block';
        this.replyAuthorInput.focus();
    }

    closeReplyModal() { this.replyModal.style.display = 'none'; this.replyToId = null; }

    async submitReply() { if (this.replyToId) await this.createComment(this.replyToId); }

    updatePagination() {
        this.pageInfo.textContent = `Страница ${this.currentPage}`;
        this.prevBtn.disabled = this.currentPage <= 1;
        this.nextBtn.disabled = !this.hasMore;
    }

    prevPage() { if (this.currentPage>1){ this.currentPage--; this.loadComments(); } }
    nextPage() { if (this.hasMore){ this.currentPage++; this.loadComments(); } }
}

document.addEventListener('DOMContentLoaded', () => { new CommentTree(); });
