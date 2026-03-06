// 完整的管理后台 JavaScript
// 参考 Dify Marketplace 设计风格

const API_BASE = '/admin/api';

const utils = {
    formatDate(dateStr) {
        return new Date(dateStr).toLocaleString('zh-CN', {
            year: 'numeric', month: '2-digit', day: '2-digit',
            hour: '2-digit', minute: '2-digit'
        });
    },

    formatRelativeTime(dateStr) {
        const days = Math.floor((new Date() - new Date(dateStr)) / (1000 * 60 * 60 * 24));
        if (days === 0) return '今天';
        if (days === 1) return '昨天';
        if (days < 7) return `${days} 天前`;
        if (days < 30) return `${Math.floor(days / 7)} 周前`;
        if (days < 365) return `${Math.floor(days / 30)} 月前`;
        return `${Math.floor(days / 365)} 年前`;
    },

    getStatusBadge(status) {
        const badges = {
            pending: '<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">待审核</span>',
            approved: '<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">已通过</span>',
            rejected: '<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">已拒绝</span>',
            draft: '<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">草稿</span>',
            published: '<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">已发布</span>'
        };
        return badges[status] || status;
    },

    getPluginTypeBadge(pluginType) {
        if (!pluginType) return '';
        const badges = {
            provider: '<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800">provider</span>',
            transform: '<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800">transform</span>',
            interceptor: '<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-purple-100 text-purple-800">interceptor</span>'
        };
        return badges[pluginType] || `<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-800">${pluginType}</span>`;
    },

    getCapabilityBadges(capabilities) {
        if (!capabilities || !capabilities.length) return '';
        return capabilities.map(cap =>
            `<span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs bg-gray-100 text-gray-600">${cap}</span>`
        ).join(' ');
    },

    getPluginFromSub(sub) {
        return sub.edges?.plugin || {};
    },

    async request(url, options = {}) {
        const token = localStorage.getItem('admin_token');
        const headers = { 'Content-Type': 'application/json', ...options.headers };
        if (token) headers['Authorization'] = `Bearer ${token}`;

        const response = await fetch(`${API_BASE}${url}`, { ...options, headers });
        if (response.status === 401) {
            localStorage.clear();
            window.location.href = '/admin/login';
            return;
        }

        const data = await response.json();
        if (data.code !== 0) throw new Error(data.message || '请求失败');
        return data.data;
    }
};

const auth = {
    async checkAuth() {
        const token = localStorage.getItem('admin_token');
        if (!token) {
            window.location.href = '/admin/login';
            return false;
        }

        try {
            const user = await utils.request('/auth/me');
            this.setUserInfo(user);
            return true;
        } catch (error) {
            localStorage.clear();
            window.location.href = '/admin/login';
            return false;
        }
    },

    setUserInfo(user) {
        document.getElementById('userName').textContent = user.username;
        document.getElementById('userInitial').textContent = user.username.charAt(0).toUpperCase();
        const roleMap = { 'super_admin': '超级管理员', 'admin': '管理员', 'reviewer': '审核员' };
        document.getElementById('userRole').textContent = roleMap[user.role] || user.role;
    },

    async logout() {
        try {
            await utils.request('/auth/logout', { method: 'POST' });
        } finally {
            localStorage.clear();
            window.location.href = '/admin/login';
        }
    }
};

const views = {
    current: 'submissions',

    async render(viewName) {
        this.current = viewName;
        document.querySelectorAll('.nav-tab').forEach(tab => {
            tab.classList.remove('active', 'border-blue-500', 'text-blue-600');
            tab.classList.add('border-transparent', 'text-gray-500');
        });
        const activeTab = document.querySelector(`[data-view="${viewName}"]`);
        if (activeTab) {
            activeTab.classList.remove('border-transparent', 'text-gray-500');
            activeTab.classList.add('active', 'border-blue-500', 'text-blue-600');
        }

        const contentArea = document.getElementById('contentArea');
        contentArea.innerHTML = '<div class="flex justify-center items-center py-20"><div class="animate-spin h-10 w-10 border-4 border-blue-500 border-t-transparent rounded-full"></div></div>';

        try {
            if (viewName === 'submissions') await this.renderSubmissions();
            else if (viewName === 'stats') await this.renderStats();
            else await this.renderPlugins();
        } catch (error) {
            contentArea.innerHTML = `<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg">${error.message}</div>`;
        }
    },

    async renderSubmissions(status = '') {
        const params = new URLSearchParams();
        if (status) params.append('status', status);
        params.append('page', '1');
        params.append('page_size', '50');

        const result = await utils.request(`/submissions?${params}`);
        const submissions = result.submissions || [];

        document.getElementById('contentArea').innerHTML = `
            <div class="mb-8">
                <h2 class="text-3xl font-bold text-gray-900 mb-2">提交审核</h2>
                <p class="text-gray-600">管理社区提交的插件，审核通过后将在市场中展示</p>
            </div>

            <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-4 mb-6">
                <div class="flex items-center space-x-4">
                    <button onclick="views.renderSubmissions('')" class="px-4 py-2 text-sm font-medium rounded-lg ${status === '' ? 'bg-blue-50 text-blue-700 border border-blue-200' : 'text-gray-600 hover:bg-gray-50'}">
                        全部 (${result.pagination?.total || 0})
                    </button>
                    <button onclick="views.renderSubmissions('pending')" class="px-4 py-2 text-sm font-medium rounded-lg ${status === 'pending' ? 'bg-blue-50 text-blue-700 border border-blue-200' : 'text-gray-600 hover:bg-gray-50'}">
                        <span class="inline-block w-2 h-2 bg-yellow-400 rounded-full mr-2"></span>待审核
                    </button>
                    <button onclick="views.renderSubmissions('approved')" class="px-4 py-2 text-sm font-medium rounded-lg ${status === 'approved' ? 'bg-blue-50 text-blue-700 border border-blue-200' : 'text-gray-600 hover:bg-gray-50'}">
                        <span class="inline-block w-2 h-2 bg-green-400 rounded-full mr-2"></span>已通过
                    </button>
                    <button onclick="views.renderSubmissions('rejected')" class="px-4 py-2 text-sm font-medium rounded-lg ${status === 'rejected' ? 'bg-blue-50 text-blue-700 border border-blue-200' : 'text-gray-600 hover:bg-gray-50'}">
                        <span class="inline-block w-2 h-2 bg-red-400 rounded-full mr-2"></span>已拒绝
                    </button>
                </div>
            </div>

            ${submissions.length === 0 ? `
                <div class="text-center py-20">
                    <svg class="mx-auto h-16 w-16 text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"/>
                    </svg>
                    <h3 class="text-lg font-medium text-gray-900 mb-2">暂无提交</h3>
                    <p class="text-gray-500">当前没有符合条件的插件提交</p>
                </div>
            ` : `
                <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                    ${submissions.map(sub => this.renderSubmissionCard(sub)).join('')}
                </div>
            `}
        `;
    },

    renderSubmissionCard(sub) {
        const statusColors = {
            pending: 'border-yellow-200 bg-yellow-50',
            approved: 'border-green-200 bg-green-50',
            rejected: 'border-red-200 bg-red-50'
        };
        const p = utils.getPluginFromSub(sub);
        const pluginName = p.display_name || p.name || sub.plugin_name || '未命名插件';
        const pluginSlug = p.name || sub.plugin_id;
        const pluginType = p.plugin_type || '';
        const version = sub.edges?.version;
        const caps = version?.capabilities || [];

        return `
            <div class="plugin-card bg-white rounded-xl border-2 ${statusColors[sub.status] || 'border-gray-200'} p-6 cursor-pointer hover:shadow-lg transition-all"
                 onclick="views.showSubmissionDetail('${sub.id}')">
                <div class="flex items-start justify-between mb-4">
                    <div class="flex items-center space-x-3">
                        <div class="h-12 w-12 bg-gradient-to-br from-blue-400 to-blue-600 rounded-xl flex items-center justify-center flex-shrink-0">
                            <span class="text-white font-bold text-lg">${pluginName.charAt(0).toUpperCase()}</span>
                        </div>
                        <div class="flex-1 min-w-0">
                            <div class="flex items-center gap-2 mb-0.5">
                                <h3 class="text-lg font-semibold text-gray-900 truncate">${pluginName}</h3>
                                ${utils.getPluginTypeBadge(pluginType)}
                            </div>
                            <p class="text-sm text-gray-500 font-mono truncate">${pluginSlug}</p>
                        </div>
                    </div>
                    ${utils.getStatusBadge(sub.status)}
                </div>
                <p class="text-sm text-gray-600 mb-3 line-clamp-2">${p.description || sub.notes || '暂无描述'}</p>
                ${caps.length ? `<div class="flex flex-wrap gap-1 mb-3">${utils.getCapabilityBadges(caps)}</div>` : ''}
                <div class="flex items-center justify-between text-xs text-gray-500 pt-4 border-t border-gray-100">
                    <div class="flex items-center space-x-4">
                        <div class="flex items-center">
                            <svg class="h-4 w-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
                            </svg>
                            ${sub.submitter_name}
                        </div>
                        <div class="flex items-center">
                            <svg class="h-4 w-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
                            </svg>
                            ${utils.formatRelativeTime(sub.created_at)}
                        </div>
                    </div>
                </div>
            </div>
        `;
    },

    async showSubmissionDetail(id) {
        const submission = await utils.request(`/submissions/${id}`);
        const reviewNotes = submission.reviewer_notes || submission.review_comment || '';
        const p = utils.getPluginFromSub(submission);
        const pluginName = p.display_name || p.name || submission.plugin_name || '未命名插件';
        const pluginSlug = p.name || submission.plugin_id;
        const pluginType = p.plugin_type || '';
        const version = submission.edges?.version;

        const modal = document.createElement('div');
        modal.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4';
        modal.innerHTML = `
            <div class="bg-white rounded-2xl shadow-2xl max-w-3xl w-full max-h-[90vh] overflow-hidden flex flex-col">
                <div class="px-6 py-4 border-b border-gray-200 flex items-center justify-between bg-gradient-to-r from-blue-50 to-white">
                    <div class="flex items-center space-x-3">
                        <div class="h-12 w-12 bg-gradient-to-br from-blue-400 to-blue-600 rounded-xl flex items-center justify-center">
                            <span class="text-white font-bold text-lg">${pluginName.charAt(0).toUpperCase()}</span>
                        </div>
                        <div>
                            <div class="flex items-center gap-2">
                                <h3 class="text-xl font-bold text-gray-900">${pluginName}</h3>
                                ${utils.getPluginTypeBadge(pluginType)}
                            </div>
                            <p class="text-sm text-gray-500 font-mono">${pluginSlug}</p>
                        </div>
                    </div>
                    <button onclick="this.closest('.fixed').remove()" class="text-gray-400 hover:text-gray-600">
                        <svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                        </svg>
                    </button>
                </div>
                <div class="flex-1 overflow-y-auto px-6 py-6 space-y-6">
                    <div><label class="block text-sm font-medium text-gray-700 mb-2">状态</label>${utils.getStatusBadge(submission.status)}</div>
                    <div><label class="block text-sm font-medium text-gray-700 mb-2">描述</label><p class="text-sm text-gray-900 bg-gray-50 rounded-lg p-4">${p.description || submission.notes || '暂无描述'}</p></div>
                    <div class="grid grid-cols-2 gap-4">
                        <div><label class="block text-sm font-medium text-gray-700 mb-2">提交者</label><p class="text-sm text-gray-900">${submission.submitter_name}</p></div>
                        <div><label class="block text-sm font-medium text-gray-700 mb-2">邮箱</label><p class="text-sm text-gray-900">${submission.submitter_email}</p></div>
                    </div>
                    ${version ? `
                    <div class="bg-indigo-50 border border-indigo-200 rounded-lg p-4 space-y-3">
                        <h4 class="text-sm font-semibold text-indigo-900 flex items-center gap-2">
                            <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A2 2 0 013 12V7a4 4 0 014-4z"/></svg>
                            关联版本信息
                        </h4>
                        <div class="grid grid-cols-2 gap-3 text-sm">
                            <div>
                                <span class="text-indigo-700 font-medium">版本：</span>
                                <span class="text-indigo-900 font-mono">${version.version || '-'}</span>
                            </div>
                            <div>
                                <span class="text-indigo-700 font-medium">状态：</span>
                                ${utils.getStatusBadge(version.status)}
                            </div>
                            ${version.wasm_hash ? `
                            <div class="col-span-2">
                                <span class="text-indigo-700 font-medium">WASM Hash：</span>
                                <span class="text-indigo-900 font-mono text-xs break-all">${version.wasm_hash}</span>
                            </div>` : ''}
                        </div>
                        ${version.capabilities?.length ? `
                        <div>
                            <span class="text-indigo-700 font-medium text-sm">Capabilities：</span>
                            <div class="flex flex-wrap gap-1 mt-1">${utils.getCapabilityBadges(version.capabilities)}</div>
                        </div>` : ''}
                    </div>` : ''}
                    ${reviewNotes ? `<div><label class="block text-sm font-medium text-gray-700 mb-2">审核意见</label><p class="text-sm text-gray-900 bg-blue-50 rounded-lg p-4 border border-blue-200">${reviewNotes}</p></div>` : ''}
                </div>
                ${submission.status === 'pending' ? `
                <div class="px-6 py-4 border-t border-gray-200 bg-gray-50 space-y-3">
                    <textarea id="reviewComment" rows="3" class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" placeholder="请输入审核意见（拒绝时必填）"></textarea>
                    <div class="flex space-x-3">
                        <button onclick="views.reviewSubmission('${submission.id}', 'approve')" class="flex-1 px-4 py-2.5 bg-green-600 text-white rounded-lg hover:bg-green-700 font-medium">批准通过</button>
                        <button onclick="views.reviewSubmission('${submission.id}', 'reject')" class="flex-1 px-4 py-2.5 bg-red-600 text-white rounded-lg hover:bg-red-700 font-medium">拒绝</button>
                    </div>
                </div>
                ` : ''}
            </div>
        `;
        document.body.appendChild(modal);
    },

    async reviewSubmission(id, action) {
        const reviewerNotes = document.getElementById('reviewComment')?.value || '';
        if (action === 'reject' && !reviewerNotes.trim()) {
            alert('拒绝时必须填写审核意见');
            return;
        }

        try {
            await utils.request(`/submissions/${id}/review`, {
                method: 'PUT',
                body: JSON.stringify({ action, reviewer_notes: reviewerNotes })
            });
            document.querySelector('.fixed')?.remove();
            await this.render('submissions');
            alert(action === 'approve' ? '已批准通过' : '已拒绝');
        } catch (error) {
            alert('操作失败：' + error.message);
        }
    },

    async renderStats() {
        const stats = await utils.request('/submissions/stats');
        document.getElementById('contentArea').innerHTML = `
            <div class="mb-8">
                <h2 class="text-3xl font-bold text-gray-900 mb-2">统计数据</h2>
                <p class="text-gray-600">插件市场整体数据概览</p>
            </div>
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
                ${[
                    { label: '总提交数', value: stats.total, color: 'blue', icon: 'M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4' },
                    { label: '待审核', value: stats.pending, color: 'yellow', icon: 'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z' },
                    { label: '已通过', value: stats.approved, color: 'green', icon: 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z' },
                    { label: '已拒绝', value: stats.rejected, color: 'red', icon: 'M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z' }
                ].map(stat => `
                    <div class="bg-white rounded-xl shadow-sm border border-gray-200 p-6 hover:shadow-md transition-shadow">
                        <div class="h-12 w-12 bg-${stat.color}-100 rounded-xl flex items-center justify-center mb-4">
                            <svg class="h-6 w-6 text-${stat.color}-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="${stat.icon}"/>
                            </svg>
                        </div>
                        <p class="text-sm font-medium text-gray-600 mb-1">${stat.label}</p>
                        <p class="text-3xl font-bold text-${stat.color}-600">${stat.value}</p>
                    </div>
                `).join('')}
            </div>
            <div class="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
                <h3 class="text-lg font-semibold text-gray-900 mb-4">快速操作</h3>
                <button onclick="views.render('submissions')" class="flex items-center justify-between p-4 bg-gradient-to-r from-blue-50 to-blue-100 hover:from-blue-100 hover:to-blue-200 rounded-lg w-full">
                    <div class="flex items-center space-x-3">
                        <div class="h-10 w-10 bg-blue-500 rounded-lg flex items-center justify-center">
                            <svg class="h-5 w-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"/>
                            </svg>
                        </div>
                        <div class="text-left">
                            <p class="text-sm font-medium text-gray-900">查看待审核</p>
                            <p class="text-xs text-gray-600">${stats.pending} 个待处理</p>
                        </div>
                    </div>
                </button>
            </div>
        `;
    },

    _pluginTypeFilter: '',

    async renderPlugins(pluginType) {
        if (pluginType !== undefined) this._pluginTypeFilter = pluginType;
        const pt = this._pluginTypeFilter;

        const params = new URLSearchParams();
        params.append('page', '1');
        params.append('page_size', '50');
        if (pt) params.append('plugin_type', pt);

        const result = await utils.request(`/plugins?${params}`);
        const plugins = result.plugins || [];

        document.getElementById('contentArea').innerHTML = `
            <div class="mb-8">
                <h2 class="text-3xl font-bold text-gray-900 mb-2">插件管理</h2>
                <p class="text-gray-600">管理所有已注册的插件</p>
            </div>

            <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-4 mb-6">
                <div class="flex items-center space-x-3">
                    <span class="text-sm font-medium text-gray-700 mr-1">类型筛选：</span>
                    <button onclick="views.renderPlugins('')" class="px-4 py-2 text-sm font-medium rounded-lg ${pt === '' ? 'bg-blue-50 text-blue-700 border border-blue-200' : 'text-gray-600 hover:bg-gray-50'}">
                        全部 (${result.total || 0})
                    </button>
                    <button onclick="views.renderPlugins('interceptor')" class="px-4 py-2 text-sm font-medium rounded-lg flex items-center gap-1.5 ${pt === 'interceptor' ? 'bg-purple-50 text-purple-700 border border-purple-200' : 'text-gray-600 hover:bg-gray-50'}">
                        <span class="inline-block w-2 h-2 bg-purple-400 rounded-full"></span>Interceptor
                    </button>
                    <button onclick="views.renderPlugins('transform')" class="px-4 py-2 text-sm font-medium rounded-lg flex items-center gap-1.5 ${pt === 'transform' ? 'bg-green-50 text-green-700 border border-green-200' : 'text-gray-600 hover:bg-gray-50'}">
                        <span class="inline-block w-2 h-2 bg-green-400 rounded-full"></span>Transform
                    </button>
                    <button onclick="views.renderPlugins('provider')" class="px-4 py-2 text-sm font-medium rounded-lg flex items-center gap-1.5 ${pt === 'provider' ? 'bg-blue-50 text-blue-700 border border-blue-200' : 'text-gray-600 hover:bg-gray-50'}">
                        <span class="inline-block w-2 h-2 bg-blue-400 rounded-full"></span>Provider
                    </button>
                </div>
            </div>

            ${plugins.length === 0 ? `
                <div class="text-center py-20">
                    <svg class="mx-auto h-16 w-16 text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/>
                    </svg>
                    <h3 class="text-lg font-medium text-gray-900 mb-2">暂无插件</h3>
                    <p class="text-gray-500">当前没有符合条件的插件</p>
                </div>
            ` : `
                <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                    ${plugins.map(p => this.renderPluginCard(p)).join('')}
                </div>
            `}
        `;
    },

    renderPluginCard(p) {
        const typeGradients = {
            provider: 'from-blue-400 to-blue-600',
            transform: 'from-green-400 to-green-600',
            interceptor: 'from-purple-400 to-purple-600'
        };
        const gradient = typeGradients[p.plugin_type] || 'from-gray-400 to-gray-600';

        return `
            <div class="plugin-card bg-white rounded-xl border border-gray-200 p-6 hover:shadow-lg transition-all">
                <div class="flex items-start justify-between mb-4">
                    <div class="flex items-center space-x-3">
                        <div class="h-12 w-12 bg-gradient-to-br ${gradient} rounded-xl flex items-center justify-center flex-shrink-0">
                            <span class="text-white font-bold text-lg">${(p.display_name || p.name || 'P').charAt(0).toUpperCase()}</span>
                        </div>
                        <div class="flex-1 min-w-0">
                            <div class="flex items-center gap-2 mb-0.5">
                                <h3 class="text-lg font-semibold text-gray-900 truncate">${p.display_name || p.name}</h3>
                                ${utils.getPluginTypeBadge(p.plugin_type)}
                            </div>
                            <p class="text-sm text-gray-500 font-mono truncate">${p.name}</p>
                        </div>
                    </div>
                </div>
                <p class="text-sm text-gray-600 mb-4 line-clamp-2">${p.description || '暂无描述'}</p>
                <div class="flex items-center justify-between text-xs text-gray-500 pt-4 border-t border-gray-100">
                    <div class="flex items-center space-x-4">
                        <div class="flex items-center">
                            <svg class="h-4 w-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
                            </svg>
                            ${p.author || '-'}
                        </div>
                        <div class="flex items-center">
                            <svg class="h-4 w-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/>
                            </svg>
                            ${p.download_count || 0}
                        </div>
                    </div>
                    <div class="flex items-center gap-2">
                        ${p.is_official ? '<span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-amber-100 text-amber-800">官方</span>' : ''}
                        ${p.status === 'active' ? '<span class="inline-block w-2 h-2 bg-green-400 rounded-full" title="active"></span>' : '<span class="inline-block w-2 h-2 bg-gray-300 rounded-full" title="' + p.status + '"></span>'}
                    </div>
                </div>
            </div>
        `;
    }
};

document.addEventListener('DOMContentLoaded', async () => {
    if (!await auth.checkAuth()) return;
    document.querySelectorAll('.nav-tab').forEach(tab => {
        tab.addEventListener('click', (e) => {
            e.preventDefault();
            views.render(tab.dataset.view);
        });
    });
    document.getElementById('logoutBtn').addEventListener('click', () => {
        if (confirm('确定要退出登录吗？')) auth.logout();
    });
    await views.render('submissions');
});
