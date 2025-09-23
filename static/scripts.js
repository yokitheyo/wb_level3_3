class CommentTree {
    constructor() {
        this.apiUrl = '/comments';
        this.currentPage = 1;
        this.limit = 10;
        this.currentSort = 'asc';
        this.isSearchMode = false;
        this.currentQuery = '';
        this.replyToId = null;

        this.initElements();
        this.attachEventListeners();
        this.loadComments();
    }

    initElements() {
        // Поиск
        this.searchInput = document.getElementById('searchInput');
        this.searchBtn = document.getElementById('searchBtn');
        this.clearBtn = document.getElementById('clearBtn');

        // Форма добавления
        this.authorInput = document.getElementById('authorInput');
        this.contentInput = document.getElementById('contentInput');
        this.addCommentBtn = document.getElementById('addCommentBtn');

        // Комментарии
        this.commentsContainer = document.getElementById('commentsContainer');
        this.sortSelect = document.getElementById('sortSelect');
        this.loading = document.getElementById('loading');
        this.noComments = document.getElementById('noComments');

        // Пагинация
        this.prevBtn = document.getElementById('prevBtn');
        this.nextBtn = document.getElementById('nextBtn');
        this.pageInfo = document.getElementById('pageInfo');

        // Модальное окно
        this.replyModal = document.getElementById('replyModal');
        this.replyTo = document.getElementById('replyTo');
        this.replyAuthorInput = document.getElementById('replyAuthorInput');
        this.replyContentInput = document.getElementById('replyContentInput');
        this.submitReplyBtn = document.getElementById('submitReplyBtn');
        this.cancelReplyBtn = document.getElementById('cancelReplyBtn');
        this.closeModal = document.getElementById('closeModal');
    }

    attachEventListeners() {
        // Поиск
        this.searchBtn.addEventListener('click', () => this.search());
        this.clearBtn.addEventListener('click', () => this.clearSearch());
        this.searchInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.search();
        });

        // Добавление комментария
        this.addCommentBtn.addEventListener('click', () => this.createComment());
        this.contentInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter' && e.ctrlKey) this.createComment();
        });

        // Сортировка
        this.sortSelect.addEventListener('change', () => {
            this.currentSort = this.sortSelect.value;
            this.loadComments();
        });

        // Пагинация
        this.prevBtn.addEventListener('click', () => this.prevPage());
        this.nextBtn.addEventListener('click', () => this.nextPage());

        // Модальное окно
        this.submitReplyBtn.addEventListener('click', () => this.submitReply());
        this.cancelReplyBtn.addEventListener('click', () => this.closeReplyModal());
        this.closeModal.addEventListener('click', () => this.closeReplyModal());
        this.replyModal.addEventListener('click', (e) => {
            if (e.target === this.replyModal) this.closeReplyModal();
        });

        // Закрытие модального окна по ESC
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && this.replyModal.style.display === 'block') {
                this.closeReplyModal();
            }
        });
    }

    async apiCall(url, options = {}) {
        try {
            const response = await fetch(url, {
                headers: {
                    'Content-Type': 'application/json',
                },
                ...options
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Ошибка сервера');
            }

            return response.status === 204 ? null : await response.json();
        } catch (error) {
            console.error('API Error:', error);
            this.showError(error.message);
            throw error;
        }
    }

    showError(message) {
        alert(message); // В продакшене лучше использовать toast-уведомления
    }

    showLoading(show = true) {
        this.loading.style.display = show ? 'block' : 'none';
        if (!show && this.commentsContainer.children.length === 1) {
            this.noComments.style.display = 'block';
        } else {
            this.noComments.style.display = 'none';
        }
    }

    async loadComments() {
        this.showLoading(true);

        try {
            const offset = (this.currentPage - 1) * this.limit;
            let url;

            if (this.isSearchMode) {
                url = `${this.apiUrl}/search?query=${encodeURIComponent(this.currentQuery)}&limit=${this.limit}&offset=${offset}`;
            } else {
                url = `${this.apiUrl}?limit=${this.limit}&offset=${offset}&sort=${this.currentSort}`;
            }

            const comments = await this.apiCall(url);
            this.renderComments(comments || []);
            this.updatePagination(comments?.length || 0);
        } catch (error) {
            this.renderComments([]);
        } finally {
            this.showLoading(false);
        }
    }

    renderComments(comments) {
        // Очищаем контейнер, оставляя служебные элементы
        const serviceElements = [this.loading, this.noComments];
        Array.from(this.commentsContainer.children).forEach(child => {
            if (!serviceElements.includes(child)) {
                child.remove();
            }
        });

        if (comments.length === 0) {
            this.noComments.style.display = 'block';
            return;
        }

        comments.forEach(comment => {
            this.renderComment(comment, 0);
        });
    }

    renderComment(comment, level = 0) {
        const commentEl = document.createElement('div');
        commentEl.className = `comment ${level > 0 ? 'reply' : ''} ${level > 1 ? `level-${Math.min(level, 3)}` : ''}`;
        commentEl.dataset.id = comment.id;

        if (comment.deleted) {
            commentEl.classList.add('deleted-comment');
        }

        const date = new Date(comment.created_at).toLocaleString('ru-RU');
        const content = comment.deleted ? '[Комментарий удален]' : comment.content;

        commentEl.innerHTML = `
            <div class="comment-header">
                <div class="comment-meta">
                    <span class="comment-author">${comment.author}</span>
                    <span class="comment-date">${date}</span>
                </div>
                <div class="comment-actions">
                    ${!comment.deleted ? `
                        <button class="reply-btn" data-id="${comment.id}" data-author="${comment.author}">
                            Ответить
                        </button>
                        <button class="delete-btn" data-id="${comment.id}">
                            Удалить
                        </button>
                    ` : ''}
                </div>
            </div>
            <div class="comment-content">${content}</div>
        `;

        // Добавляем обработчики событий
        if (!comment.deleted) {
            const replyBtn = commentEl.querySelector('.reply-btn');
            const deleteBtn = commentEl.querySelector('.delete-btn');

            replyBtn.addEventListener('click', () => {
                this.openReplyModal(comment.id, comment.author, comment.content);
            });

            deleteBtn.addEventListener('click', () => {
                this.deleteComment(comment.id);
            });
        }

        this.commentsContainer.appendChild(commentEl);

        // Рекурсивно отображаем дочерние комментарии
        if (comment.children && comment.children.length > 0) {
            comment.children.forEach(child => {
                this.renderComment(child, level + 1);
            });
        }
    }

    async createComment(parentId = null) {
        const author = parentId ? this.replyAuthorInput.value.trim() : this.authorInput.value.trim();
        const content = parentId ? this.replyContentInput.value.trim() : this.contentInput.value.trim();

        if (!author || !content) {
            this.showError('Пожалуйста, заполните все поля');
            return;
        }

        try {
            const payload = {
                author,
                content,
                ...(parentId && { parent_id: parentId })
            };

            await this.apiCall(this.apiUrl, {
                method: 'POST',
                body: JSON.stringify(payload)
            });

            // Очищаем форму
            if (parentId) {
                this.closeReplyModal();
            } else {
                this.authorInput.value = '';
                this.contentInput.value = '';
            }

            // Перезагружаем комментарии
            this.loadComments();

        } catch (error) {
            // Ошибка уже обработана в apiCall
        }
    }

    async deleteComment(id) {
        if (!confirm('Вы уверены, что хотите удалить этот комментарий?')) {
            return;
        }

        try {
            await this.apiCall(`${this.apiUrl}/${id}`, {
                method: 'DELETE'
            });

            this.loadComments();
        } catch (error) {
            // Ошибка уже обработана в apiCall
        }
    }

    async search() {
        const query = this.searchInput.value.trim();
        if (!query) {
            this.showError('Введите поисковый запрос');
            return;
        }

        this.isSearchMode = true;
        this.currentQuery = query;
        this.currentPage = 1;
        this.loadComments();
    }

    clearSearch() {
        this.searchInput.value = '';
        this.isSearchMode = false;
        this.currentQuery = '';
        this.currentPage = 1;
        this.loadComments();
    }

    openReplyModal(commentId, authorName, content) {
        this.replyToId = commentId;
        this.replyTo.textContent = `Ответ на комментарий от ${authorName}: "${content.substring(0, 100)}${content.length > 100 ? '...' : ''}"`;
        this.replyAuthorInput.value = '';
        this.replyContentInput.value = '';
        this.replyModal.style.display = 'block';
        this.replyAuthorInput.focus();
    }

    closeReplyModal() {
        this.replyModal.style.display = 'none';
        this.replyToId = null;
    }

    async submitReply() {
        if (this.replyToId) {
            await this.createComment(this.replyToId);
        }
    }

    updatePagination(commentsCount) {
        this.pageInfo.textContent = `Страница ${this.currentPage}`;
        this.prevBtn.disabled = this.currentPage <= 1;
        this.nextBtn.disabled = commentsCount < this.limit;
    }

    prevPage() {
        if (this.currentPage > 1) {
            this.currentPage--;
            this.loadComments();
        }
    }

    nextPage() {
        this.currentPage++;
        this.loadComments();
    }
}

// Инициализация приложения после загрузки DOM
document.addEventListener('DOMContentLoaded', () => {
    new CommentTree();
});