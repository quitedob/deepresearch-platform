<template>
  <div class="welcome-page">
    <!-- Animated Background -->
    <div class="cyber-bg">
      <div class="grid-lines"></div>
      <div class="particle particle-1"></div>
      <div class="particle particle-2"></div>
      <div class="particle particle-3"></div>
      <div class="glow-orb orb-1"></div>
      <div class="glow-orb orb-2"></div>
      <div class="scan-line"></div>
    </div>

    <!-- Main Content -->
    <div class="welcome-main">
      <!-- Hero Section -->
      <section class="hero-section">
        <div class="logo-wrapper-3d">
          <div class="logo-ring-outer"></div>
          <div class="logo-ring-inner"></div>
          <img src="@/assets/images/hero-brain-3d.png" alt="AI Core" class="hero-image-3d" />
        </div>

        <h1 class="hero-title">
          <span class="title-text">Deep Research</span>
          <span class="pro-badge">PRO</span>
        </h1>

        <p class="hero-tagline">
          <span class="tagline-line">AI 驱动的智能研究平台</span>
          <span class="tagline-divider">|</span>
          <span class="tagline-tech">NEXT-GEN RESEARCH ENGINE</span>
        </p>

        <!-- Primary CTA - The Main Entry Point -->
        <div class="hero-cta">
          <button @click="startExploring" class="btn-launch">
            <span class="btn-glow"></span>
            <img src="@/assets/images/rocket-launch-3d.png" alt="" class="btn-icon-3d" />
            <span class="btn-text">免费注册，立即体验</span>
            <span class="btn-arrow">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none">
                <path d="M5 12h14M12 5l7 7-7 7" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </span>
          </button>
          
          <div class="cta-secondary">
            <a @click.prevent="skipToLogin" href="#" class="link-login">已有账号？立即登录</a>
            <span class="cta-divider">·</span>
            <a @click.prevent="skipToHome" href="#" class="link-skip">跳过引导</a>
          </div>
        </div>
      </section>

      <!-- Features Strip -->
      <section class="features-strip">
        <div class="feature-card glass-panel">
          <div class="feature-img-box">
            <img src="@/assets/images/feature-report-3d.png" alt="" class="feature-img-3d" />
          </div>
          <div class="feature-info">
            <h3>深度研报</h3>
            <p>自动生成专业级研究报告</p>
          </div>
        </div>

        <div class="feature-card glass-panel">
          <div class="feature-img-box">
            <img src="@/assets/images/feature-compute-3d.png" alt="" class="feature-img-3d" />
          </div>
          <div class="feature-info">
            <h3>实时计算</h3>
            <p>Python 沙盒即时数据分析</p>
          </div>
        </div>

        <div class="feature-card glass-panel">
          <div class="feature-img-box">
            <img src="@/assets/images/feature-global-3d.png" alt="" class="feature-img-3d" />
          </div>
          <div class="feature-info">
            <h3>全网检索</h3>
            <p>跨源智能知识引擎</p>
          </div>
        </div>
      </section>

      <!-- Bottom Actions -->
      <div class="bottom-bar">
        <div class="status-indicator">
          <span class="pulse-dot"></span>
          <span class="status-text">系统在线 · 响应正常</span>
        </div>
        
        <div class="quick-links">
          <router-link to="/help" class="quick-link">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none">
              <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="1.5"/>
              <path d="M9 9a3 3 0 115.83 1c0 2-3 3-3 3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
              <circle cx="12" cy="17" r="1" fill="currentColor"/>
            </svg>
            帮助中心
          </router-link>
          <router-link to="/policies" class="quick-link">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" stroke="currentColor" stroke-width="1.5"/>
            </svg>
            隐私政策
          </router-link>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'

const router = useRouter()

const startExploring = () => {
  // 设置欢迎流程完成标志，防止路由守卫死循环
  localStorage.setItem('welcome_completed', 'true')
  router.push('/register')
}

const skipToLogin = () => {
  localStorage.setItem('welcome_completed', 'true')
  router.push('/login')
}

const skipToHome = () => {
  localStorage.setItem('welcome_completed', 'true')
  const token = localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token')
  router.push(token ? '/home' : '/login')
}
</script>

<style scoped>
/* Import Fonts */
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800&family=JetBrains+Mono:wght@400;500&display=swap');

/* CSS Variables */
.welcome-page {
  --bg-primary: #050816;
  --bg-card: rgba(15, 23, 42, 0.6);
  --border-color: rgba(59, 130, 246, 0.2);
  --text-primary: #f1f5f9;
  --text-secondary: #94a3b8;
  --accent-blue: #3b82f6;
  --accent-purple: #8b5cf6;
  --accent-cyan: #06b6d4;
  --glow-color: rgba(59, 130, 246, 0.4);
}

/* Page Container */
.welcome-page {
  min-height: 100vh;
  background: var(--bg-primary);
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
  color: var(--text-primary);
  position: relative;
  overflow-x: hidden;
  display: flex;
  flex-direction: column;
}

/* Cyber Background */
.cyber-bg {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 0;
}

.grid-lines {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(59, 130, 246, 0.03) 1px, transparent 1px),
    linear-gradient(90deg, rgba(59, 130, 246, 0.03) 1px, transparent 1px);
  background-size: 60px 60px;
  mask-image: radial-gradient(ellipse at center, black 30%, transparent 70%);
}

.particle {
  position: absolute;
  width: 4px;
  height: 4px;
  background: var(--accent-blue);
  border-radius: 50%;
  opacity: 0.6;
  animation: floatParticle 20s infinite linear;
}

.particle-1 { top: 20%; left: 10%; animation-delay: 0s; }
.particle-2 { top: 60%; left: 80%; animation-delay: -7s; }
.particle-3 { top: 80%; left: 30%; animation-delay: -14s; }

@keyframes floatParticle {
  0%, 100% { transform: translate(0, 0) scale(1); opacity: 0.6; }
  25% { transform: translate(100px, -50px) scale(1.5); opacity: 0.8; }
  50% { transform: translate(50px, -100px) scale(1); opacity: 0.4; }
  75% { transform: translate(-50px, -50px) scale(1.2); opacity: 0.7; }
}

.glow-orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(100px);
  opacity: 0.3;
}

.orb-1 {
  top: -20%;
  left: -10%;
  width: 600px;
  height: 600px;
  background: linear-gradient(135deg, var(--accent-blue), var(--accent-purple));
  animation: orbMove 25s infinite alternate ease-in-out;
}

.orb-2 {
  bottom: -30%;
  right: -10%;
  width: 500px;
  height: 500px;
  background: linear-gradient(135deg, var(--accent-purple), var(--accent-cyan));
  animation: orbMove 30s infinite alternate-reverse ease-in-out;
}

@keyframes orbMove {
  0% { transform: translate(0, 0) scale(1); }
  100% { transform: translate(50px, 30px) scale(1.1); }
}

.scan-line {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 2px;
  background: linear-gradient(90deg, transparent, var(--accent-cyan), transparent);
  animation: scanMove 8s infinite linear;
  opacity: 0.3;
}

@keyframes scanMove {
  0% { top: 0; }
  100% { top: 100%; }
}

/* Main Content */
.welcome-main {
  position: relative;
  z-index: 10;
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  padding: 2rem;
  max-width: 1200px;
  margin: 0 auto;
  width: 100%;
}

/* Hero Section */
.hero-section {
  text-align: center;
  margin-bottom: 5rem;
}

.logo-wrapper-3d {
  position: relative;
  width: 220px;
  height: 220px;
  margin: 0 auto 3rem;
  display: flex;
  justify-content: center;
  align-items: center;
}

.logo-ring-outer {
  position: absolute;
  inset: -20px;
  border: 1px solid rgba(59, 130, 246, 0.3);
  border-radius: 50%;
  border-top-color: var(--accent-blue);
  border-bottom-color: var(--accent-purple);
  animation: spin 20s linear infinite;
  box-shadow: 0 0 50px rgba(59, 130, 246, 0.1);
}

.logo-ring-inner {
  position: absolute;
  inset: -5px;
  border: 1px solid rgba(139, 92, 246, 0.3);
  border-radius: 50%;
  border-left-color: var(--accent-cyan);
  border-right-color: var(--accent-purple);
  animation: spin 15s linear infinite reverse;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.hero-image-3d {
  width: 100%;
  height: 100%;
  object-fit: contain;
  filter: drop-shadow(0 0 30px rgba(59, 130, 246, 0.4));
  animation: floatOrb 6s ease-in-out infinite;
  z-index: 2;
  mix-blend-mode: screen; /* Removes black background */
}

@keyframes floatOrb {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-15px); }
}

.hero-title {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}

.title-text {
  font-size: 4.5rem;
  font-weight: 800;
  letter-spacing: -0.05em;
  background: linear-gradient(135deg, #fff 0%, #94a3b8 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  text-shadow: 0 0 50px rgba(59, 130, 246, 0.3);
}

.pro-badge {
  background: var(--accent-blue);
  color: #fff;
  font-size: 1rem;
  font-weight: 700;
  padding: 0.25rem 0.75rem;
  border-radius: 99px;
  letter-spacing: 0.05em;
  box-shadow: 0 0 20px rgba(59, 130, 246, 0.5);
}

.hero-tagline {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  font-size: 1.1rem;
  color: var(--text-secondary);
  margin-bottom: 4rem;
  flex-wrap: wrap;
}

.tagline-divider {
  opacity: 0.3;
}

.tagline-tech {
  color: var(--accent-cyan);
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.9rem;
  letter-spacing: 0.05em;
  opacity: 0.8;
}

.btn-launch {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 1rem;
  padding: 1.25rem 3rem;
  background: linear-gradient(135deg, var(--accent-blue), var(--accent-purple));
  border: none;
  border-radius: 100px;
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275);
  box-shadow: 0 0 30px rgba(59, 130, 246, 0.4);
  z-index: 10;
  overflow: hidden;
}

.btn-launch::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, var(--accent-blue), var(--accent-cyan));
  opacity: 0;
  transition: opacity 0.3s ease;
  z-index: 0;
}

.btn-launch:hover {
  transform: translateY(-5px) scale(1.02);
  box-shadow: 0 0 50px rgba(59, 130, 246, 0.6);
}

.btn-launch:hover::before {
  opacity: 1;
}

.btn-text {
  font-size: 1.25rem;
  font-weight: 700;
  color: #fff;
  z-index: 1;
}

.btn-arrow {
  color: #fff;
  z-index: 1;
  transition: transform 0.3s ease;
}

.btn-launch:hover .btn-arrow {
  transform: translateX(5px);
}

.cta-secondary {
  margin-top: 1.5rem;
}

.link-login {
  color: var(--text-secondary);
  text-decoration: none;
  font-size: 0.95rem;
  transition: color 0.2s;
  position: relative;
  cursor: pointer;
}

.link-login::after {
  content: '';
  position: absolute;
  bottom: -2px;
  left: 0;
  width: 100%;
  height: 1px;
  background: var(--text-secondary);
  transform: scaleX(0);
  transition: transform 0.2s;
}

.link-login:hover {
  color: #fff;
}

.link-login:hover::after {
  transform: scaleX(1);
}

.cta-divider {
  color: var(--text-secondary);
  opacity: 0.4;
  margin: 0 0.25rem;
}

.link-skip {
  color: var(--text-secondary);
  text-decoration: none;
  font-size: 0.95rem;
  transition: color 0.2s;
  cursor: pointer;
  opacity: 0.7;
}

.link-skip:hover {
  color: var(--accent-cyan);
  opacity: 1;
}

.btn-icon-3d {
  width: 36px;
  height: 36px;
  filter: drop-shadow(0 2px 5px rgba(0,0,0,0.3));
  mix-blend-mode: screen; /* Removes black background */
}

/* ... styles ... */

.features-strip {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 2rem;
  width: 100%;
  max-width: 1100px;
  margin: 0 auto 4rem;
}

.feature-card {
  position: relative;
  border-radius: 24px;
  padding: 2rem;
  overflow: hidden;
  transition: all 0.4s cubic-bezier(0.175, 0.885, 0.32, 1.275);
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.glass-panel {
  background: rgba(15, 23, 42, 0.4);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.05);
  box-shadow: 0 4px 30px rgba(0, 0, 0, 0.1);
}

.glass-panel::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(145deg, rgba(255, 255, 255, 0.05) 0%, transparent 100%);
  z-index: 0;
  pointer-events: none;
}

.feature-card:hover {
  transform: translateY(-10px);
  border-color: rgba(59, 130, 246, 0.3);
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3);
  background: rgba(15, 23, 42, 0.6);
}

.feature-img-box {
  position: relative;
  width: 100%;
  height: 180px;
  margin-bottom: 1.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1;
}

/* ... styles ... */

.feature-img-3d {
  width: 100%;
  height: 100%;
  object-fit: contain;
  filter: drop-shadow(0 10px 20px rgba(0, 0, 0, 0.3));
  mix-blend-mode: screen; /* Removes black background */
}

.feature-info h3 {
  margin: 0 0 0.5rem;
  font-size: 1.35rem;
  font-weight: 700;
  color: #fff;
}

.feature-info p {
  margin: 0;
  font-size: 1rem;
  color: var(--text-secondary);
  line-height: 1.6;
}

/* Bottom Bar */
.bottom-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  max-width: 1000px;
  padding-top: 2rem;
  border-top: 1px solid var(--border-color);
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.pulse-dot {
  width: 8px;
  height: 8px;
  background: #10b981;
  border-radius: 50%;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.5; transform: scale(1.2); }
}

.status-text {
  font-size: 0.85rem;
  color: var(--text-secondary);
  font-family: 'JetBrains Mono', monospace;
}

.quick-links {
  display: flex;
  gap: 1.5rem;
}

.quick-link {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--text-secondary);
  text-decoration: none;
  font-size: 0.9rem;
  transition: color 0.2s;
}

.quick-link:hover {
  color: var(--accent-blue);
}

/* Responsive */
@media (max-width: 768px) {
  .welcome-main {
    padding: 1.5rem;
  }

  .hero-tagline {
    flex-direction: column;
    gap: 0.5rem;
  }

  .tagline-divider {
    display: none;
  }

  .btn-launch {
    padding: 1rem 2rem;
    font-size: 1.1rem;
  }

  .features-strip {
    gap: 1rem;
  }

  .feature-card {
    padding: 1.25rem 1.5rem;
  }

  .bottom-bar {
    flex-direction: column;
    gap: 1.5rem;
    text-align: center;
  }

  .quick-links {
    flex-wrap: wrap;
    justify-content: center;
  }
}

@media (max-width: 480px) {
  .hero-title {
    flex-direction: column;
    gap: 0.5rem;
  }

  .logo-wrapper {
    width: 100px;
    height: 100px;
  }

  .logo-icon {
    width: 48px;
    height: 48px;
  }
}
</style>