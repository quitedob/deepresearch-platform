<template>
  <div class="auth-page">
    <div class="auth-card">
      <div class="auth-header">
        <div class="logo-container">
          <img src="@/assets/images/hero-brain-3d.png" alt="Logo" class="logo-img" />
          <div class="brand-name">Deep Research</div>
        </div>
        <h2 class="title">登录您的账户</h2>
        <p class="subtitle">使用您的 Deep Research 帐户继续</p>
      </div>
      <form @submit.prevent="doLogin" class="form" novalidate>
        <div class="input-group">
          <input
            v-model.trim="username"
            type="text"
            id="username"
            autocomplete="username"
            required
            placeholder="请输入账号"
            @blur="touched.username = true"
            :class="{ invalid: usernameError && touched.username }"
          />
          <p v-if="usernameError && touched.username" class="error">{{ usernameError }}</p>
        </div>

        <div class="input-group">
          <input
            v-model.trim="password"
            :type="passwordFieldType"
            id="password"
            autocomplete="current-password"
            required
            placeholder="请输入密码"
            @blur="touched.password = true"
            :class="{ invalid: passwordError && touched.password }"
          />
          <button type="button" @click="togglePasswordVisibility" class="password-toggle">
            {{ passwordFieldType === 'password' ? '显示' : '隐藏' }}
          </button>
          <p v-if="passwordError && touched.password" class="error">{{ passwordError }}</p>
        </div>

        <label class="remember">
          <input type="checkbox" v-model="rememberMe" />
          <span>记住我</span>
        </label>

        <div class="footer-actions">
          <router-link class="link" to="/register">创建帐户</router-link>
          <button type="submit" class="primary-btn" :disabled="submitting">
            {{ submitting ? '登录中...' : '下一步' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive } from 'vue';
import { useRouter } from 'vue-router';
import { login } from '@/api/user.js';

const router = useRouter();
const username = ref('');
const password = ref('');
const rememberMe = ref(true);
const submitting = ref(false);
const passwordFieldType = ref('password');

const touched = reactive({
  username: false,
  password: false,
});

const usernameError = computed(() => {
  if (!username.value) return '请输入账号';
  return '';
});

const passwordError = computed(() => {
  if (!password.value) return '请输入密码';
  return '';
});

const togglePasswordVisibility = () => {
  passwordFieldType.value = passwordFieldType.value === 'password' ? 'text' : 'password';
};

const doLogin = async () => {
  touched.username = true;
  touched.password = true;

  if (usernameError.value || passwordError.value) return;

  try {
    submitting.value = true;

    const data = await login({
      username: username.value,
      password: password.value
    });

    const storage = rememberMe.value ? localStorage : sessionStorage;
    storage.setItem('auth_token', data.access_token);
    storage.setItem('user', JSON.stringify(data.user));
    // 登录成功即视为完成欢迎流程，防止路由守卫死循环
    localStorage.setItem('welcome_completed', 'true');

    const redirect = router.currentRoute.value.query?.redirect || '/';
    router.push(String(redirect));
  } catch (e) {
    alert(e.message || '登录失败');
  } finally {
    submitting.value = false;
  }
};

onMounted(() => {
  const token = localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token');
  if (token) {
    router.replace('/');
  }
});
</script>

<style scoped>
/* 引入字体 - 保持与Welcome一致 */
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;600;800&family=JetBrains+Mono:wght@400;700&display=swap');

.auth-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background-color: #030712; /* Dark tech background */
  padding: 24px;
  position: relative;
  overflow: hidden;
  font-family: 'Inter', sans-serif;
  
  /* Shared Tech Variables */
  --tech-card-bg: rgba(17, 24, 39, 0.7);
  --tech-border: rgba(59, 130, 246, 0.2);
  --tech-primary: #f8fafc;
  --tech-secondary: #94a3b8;
  --tech-accent: #3b82f6;
  --tech-glow: rgba(59, 130, 246, 0.5);
}

/* Background effects */
.auth-page::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-image: 
    linear-gradient(rgba(30, 41, 59, 0.5) 1px, transparent 1px),
    linear-gradient(90deg, rgba(30, 41, 59, 0.5) 1px, transparent 1px);
  background-size: 40px 40px;
  mask-image: radial-gradient(circle at center, black 40%, transparent 80%);
  opacity: 0.3;
  z-index: 0;
}

.auth-card {
  width: 100%;
  max-width: 420px;
  padding: 40px;
  background: var(--tech-card-bg);
  border: 1px solid var(--tech-border);
  border-radius: 24px;
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  box-shadow: 0 0 40px rgba(0, 0, 0, 0.5);
  transition: all 0.3s ease;
  z-index: 10;
  position: relative;
}

.auth-card:hover {
  box-shadow: 0 0 60px rgba(59, 130, 246, 0.2);
  border-color: rgba(59, 130, 246, 0.4);
}

.auth-header {
  text-align: center;
  margin-bottom: 32px;
}

.logo-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin-bottom: 24px;
}

.logo-img {
  width: 80px;
  height: 80px;
  object-fit: contain;
  mix-blend-mode: screen; /* Removes black background */
  filter: drop-shadow(0 0 20px rgba(59, 130, 246, 0.4));
  animation: floatLogo 6s ease-in-out infinite;
}

@keyframes floatLogo {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-5px); }
}

.brand-name {
  font-size: 24px;
  font-weight: 800;
  color: white;
  letter-spacing: -0.02em;
  background: linear-gradient(135deg, #fff 0%, #94a3b8 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.title {
  font-size: 20px;
  font-weight: 600;
  color: var(--tech-primary);
  margin: 0;
  margin-bottom: 8px;
}

.subtitle {
  font-size: 14px;
  color: var(--tech-secondary);
  line-height: 1.5;
  margin: 0;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.input-group {
  position: relative;
}

.input-group input {
  width: 100%;
  padding: 12px 16px;
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 12px;
  background-color: rgba(15, 23, 42, 0.6);
  color: white;
  font-size: 15px;
  transition: all 0.2s ease;
  height: 48px;
}

.input-group input::placeholder {
  color: rgba(255,255,255,0.3);
}

.input-group input:focus {
  outline: none;
  border-color: var(--tech-accent);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.2);
  background-color: rgba(15, 23, 42, 0.8);
}

.input-group input.invalid {
  border-color: #ef4444;
}

.input-group input.invalid:focus {
  box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.2);
}

.error {
  color: #ef4444;
  font-size: 13px;
  margin-top: 4px;
}

.remember {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--tech-secondary);
  cursor: pointer;
}

.remember input {
  width: 16px;
  height: 16px;
  border-radius: 4px;
  border: 1px solid rgba(255,255,255,0.2);
  background-color: rgba(15, 23, 42, 0.6);
  cursor: pointer;
  accent-color: var(--tech-accent);
}

.footer-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 12px;
}

.link {
  color: var(--tech-accent);
  text-decoration: none;
  font-weight: 500;
  font-size: 14px;
  transition: color 0.2s ease;
}

.link:hover {
  color: #60a5fa;
  text-decoration: underline;
}

.primary-btn {
  padding: 12px 32px;
  border-radius: 12px;
  border: none;
  background: linear-gradient(135deg, var(--tech-accent), #7c3aed);
  color: white;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  box-shadow: 0 0 20px rgba(59, 130, 246, 0.3);
}

.primary-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 0 30px rgba(59, 130, 246, 0.5);
}

.primary-btn:active {
  transform: translateY(0);
}

.primary-btn:disabled {
  background: rgba(255,255,255,0.1);
  color: rgba(255,255,255,0.3);
  cursor: not-allowed;
  box-shadow: none;
  transform: none;
}

.password-toggle {
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  color: var(--tech-secondary);
  cursor: pointer;
  font-size: 13px;
  padding: 4px 8px;
  border-radius: 4px;
}

.password-toggle:hover {
  color: white;
  background-color: rgba(255,255,255,0.1);
}

@media (max-width: 480px) {
  .auth-card {
    padding: 24px;
  }
}
</style>