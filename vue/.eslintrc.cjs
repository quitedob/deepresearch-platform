module.exports = {
  root: true,
  env: {
    node: true,
    browser: true,
    es2022: true
  },
  extends: [
    'eslint:recommended',
    'plugin:vue/vue3-essential'
  ],
  parserOptions: {
    ecmaVersion: 'latest',
    sourceType: 'module'
  },
  plugins: ['vue'],
  rules: {
    'no-console': process.env.NODE_ENV === 'production' ? 'warn' : 'off',
    'no-debugger': process.env.NODE_ENV === 'production' ? 'warn' : 'off',
    'no-constant-condition': ['error', { checkLoops: false }], // 允许 while(true) 用于流式读取
    'no-unused-vars': ['error', { argsIgnorePattern: '^_', varsIgnorePattern: '^_' }], // 允许以_开头的未使用变量
    'vue/multi-word-component-names': 'off',
    'vue/no-unused-vars': 'warn'
  },
  globals: {
    process: 'readonly'
  }
};
