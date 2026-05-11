<template>
  <div class="not-found">
    <div class="content">
      <div class="code">404</div>
      <p class="message">页面不存在</p>
      <router-link to="/home" class="back-btn">返回主页</router-link>
    </div>
  </div>
</template>

<style scoped>
.not-found {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100dvh;
  background: var(--primary-bg);
  color: var(--text-primary);
}

.content {
  text-align: center;
}

.code {
  font-size: 80px;
  font-weight: 700;
  color: var(--text-secondary);
  line-height: 1;
}

.message {
  font-size: 18px;
  color: var(--text-secondary);
  margin: 16px 0 32px;
}

.back-btn {
  padding: 10px 24px;
  background: var(--button-bg);
  color: var(--button-text);
  border-radius: 8px;
  text-decoration: none;
  font-size: 14px;
  font-weight: 500;
  transition: opacity 0.2s;
}

.back-btn:hover {
  opacity: 0.85;
}
</style>
