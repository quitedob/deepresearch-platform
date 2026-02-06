// src/router/index.js
import { createRouter, createWebHistory } from 'vue-router';
import Home from '@/views/Home.vue'; // 引入主页视图组件
import Homepage from '@/views/Homepage.vue'; // 引入酷炫主页组件
import Welcome from '@/views/Welcome.vue'; // 引入欢迎引导页面
import HelpCenter from '@/views/HelpCenter.vue'; // 1. 引入帮助中心组件
import TermsAndPolicies from '@/views/TermsAndPolicies.vue'; // 2. 引入条款政策组件
import Login from '@/views/Login.vue'; // 3. 引入登录页面
import Register from '@/views/Register.vue'; // 4. 引入注册页面
import Admin from '@/views/Admin.vue'; // 5. 引入管理员页面
import AISpace from '@/views/AISpace.vue'; // 6. 引入AI空间页面

const routes = [
    {
        path: '/',
        redirect: '/home'
    },
    {
        path: '/landing',
        name: 'Homepage',
        component: Homepage
    },
    {
        path: '/welcome',
        name: 'Welcome',
        component: Welcome
    },
    {
        path: '/home',
        name: 'Home',
        component: Home
    },
    // 登录页面
    {
        path: '/login',
        name: 'Login',
        component: Login
    },
    // 注册页面
    {
        path: '/register',
        name: 'Register',
        component: Register
    },
    // 已删除无用的路由：代码沙盒、文档管理、管理员、研究项目、Agent管理等
    // 3. 添加帮助中心页面的路由规则
    {
        path: '/help',
        name: 'HelpCenter',
        component: HelpCenter
    },
    // 4. 添加条款与政策页面的路由规则
    {
        path: '/policies',
        name: 'TermsAndPolicies',
        component: TermsAndPolicies
    },
    // 5. 管理员页面
    {
        path: '/admin',
        name: 'Admin',
        component: Admin,
        meta: { requiresAdmin: true }
    },
    // 6. AI空间页面
    {
        path: '/ai-space',
        name: 'AISpace',
        component: AISpace,
        meta: { requiresAuth: true }
    }
];

const router = createRouter({
    history: createWebHistory(),
    routes
});

// 路由守卫：配置JWT认证和公开页面
router.beforeEach((to) => {
    const token = localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token');
    const hasCompletedWelcome = localStorage.getItem('welcome_completed') === 'true';

    // 根地址重定向处理
    if (to.path === '/') {
        // 如果用户未登录，重定向到着陆页
        if (!token) {
            return { path: '/landing' };
        }

        // 如果用户已登录但未完成欢迎流程，重定向到欢迎页面
        if (!hasCompletedWelcome) {
            return { path: '/welcome' };
        }

        // 已登录且完成欢迎流程，重定向到AI聊天页面
        return { path: '/home' };
    }

    // 公开页面 - 无需认证即可访问
    const publicPaths = ['/landing', '/welcome', '/login', '/register', '/help', '/policies'];
    if (publicPaths.includes(to.path)) {
        // 如果用户已登录访问登录/注册页面，重定向到主页
        if (token && (to.path === '/login' || to.path === '/register')) {
            return { path: '/home' };
        }

        // 如果已登录用户访问欢迎页面且已完成欢迎流程，重定向到主页
        if (token && to.path === '/welcome' && hasCompletedWelcome) {
            return { path: '/home' };
        }

        return true;
    }

    // 需要认证的页面
    if (!token) {
        return { path: '/login', query: { redirect: to.fullPath } };
    }

    // 欢迎流程检查：新用户首次登录后应先完成欢迎流程
    if (token && !hasCompletedWelcome && to.path !== '/welcome') {
        return { path: '/welcome' };
    }

    // 主页特殊处理：确保欢迎流程逻辑正确执行
    if (to.path === '/home' && token && !hasCompletedWelcome) {
        return { path: '/welcome' };
    }

    // 管理员专属页面
    const adminPaths = ['/admin'];
    if (adminPaths.some(path => to.path.startsWith(path))) {
        try {
            const userStr = localStorage.getItem('user') || sessionStorage.getItem('user');
            if (userStr) {
                const user = JSON.parse(userStr);
                if (user.role !== 'admin') {
                    return { path: '/home', query: { error: 'access_denied' } };
                }
            } else {
                return { path: '/login', query: { redirect: to.fullPath } };
            }
        } catch (error) {
            return { path: '/login' };
        }
    }

    return true;
});

export default router;