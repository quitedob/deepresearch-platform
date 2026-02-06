<template>
  <div :class="['question-card', { checked: question.isChecked, correct: question.isCorrect === true, incorrect: question.isCorrect === false }]">
    <div class="question-header">
      <span class="question-number">{{ index + 1 }}</span>
      <span :class="['question-type', question.type]">{{ typeLabel }}</span>
      <span v-if="question.difficulty" :class="['difficulty', question.difficulty]">
        {{ difficultyLabel }}
      </span>
      <span v-if="question.score" class="score">{{ question.score }}分</span>
    </div>

    <div class="question-content">
      <p class="question-text">{{ question.questionText || question.content }}</p>
    </div>

    <!-- 知识点标签 -->
    <div v-if="question.tags && question.tags.length > 0" class="tags-row">
      <span v-for="tag in question.tags" :key="tag" class="tag">{{ tag }}</span>
    </div>

    <!-- 单选题 -->
    <div v-if="question.type === 'single'" class="options-list">
      <label
        v-for="option in normalizedOptions"
        :key="option.value"
        :class="['option-item', { 
          selected: question.userAnswer === option.value,
          correct: question.isChecked && question.correctAnswer === option.value,
          incorrect: question.isChecked && question.userAnswer === option.value && question.userAnswer !== question.correctAnswer
        }]"
      >
        <input
          type="radio"
          :name="`question-${question.id}`"
          :value="option.value"
          :checked="question.userAnswer === option.value"
          @change="selectAnswer(option.value)"
          :disabled="question.isChecked"
        />
        <span class="option-label">{{ option.value }}</span>
        <span class="option-text">{{ option.text }}</span>
        <span v-if="question.isChecked && question.correctAnswer === option.value" class="correct-mark">✓</span>
      </label>
    </div>

    <!-- 多选题 -->
    <div v-else-if="question.type === 'multiple'" class="options-list">
      <label
        v-for="option in normalizedOptions"
        :key="option.value"
        :class="['option-item', { 
          selected: isMultipleSelected(option.value),
          correct: question.isChecked && isCorrectOption(option.value),
          incorrect: question.isChecked && isMultipleSelected(option.value) && !isCorrectOption(option.value)
        }]"
      >
        <input
          type="checkbox"
          :value="option.value"
          :checked="isMultipleSelected(option.value)"
          @change="toggleMultipleAnswer(option.value)"
          :disabled="question.isChecked"
        />
        <span class="option-label">{{ option.value }}</span>
        <span class="option-text">{{ option.text }}</span>
        <span v-if="question.isChecked && isCorrectOption(option.value)" class="correct-mark">✓</span>
      </label>
    </div>

    <!-- 判断题 -->
    <div v-else-if="question.type === 'judge'" class="judge-options">
      <button
        :class="['judge-btn', 'true-btn', { 
          selected: question.userAnswer === true,
          correct: question.isChecked && question.correctAnswer === true,
          incorrect: question.isChecked && question.userAnswer === true && question.correctAnswer !== true
        }]"
        @click="selectAnswer(true)"
        :disabled="question.isChecked"
      >
        ✓ 正确
      </button>
      <button
        :class="['judge-btn', 'false-btn', { 
          selected: question.userAnswer === false,
          correct: question.isChecked && question.correctAnswer === false,
          incorrect: question.isChecked && question.userAnswer === false && question.correctAnswer !== false
        }]"
        @click="selectAnswer(false)"
        :disabled="question.isChecked"
      >
        ✗ 错误
      </button>
    </div>

    <!-- 简答题 -->
    <div v-else-if="question.type === 'essay'" class="essay-answer">
      <textarea
        v-model="essayAnswer"
        placeholder="请输入你的答案..."
        :disabled="question.isChecked"
        rows="4"
      ></textarea>
      <div v-if="question.isChecked && question.correctAnswer" class="reference-answer">
        <strong>参考答案：</strong>
        <p>{{ question.correctAnswer }}</p>
      </div>
    </div>

    <!-- 解析区域 -->
    <div v-if="question.isChecked && question.explanation" class="explanation">
      <div class="explanation-header">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"></circle>
          <line x1="12" y1="16" x2="12" y2="12"></line>
          <line x1="12" y1="8" x2="12.01" y2="8"></line>
        </svg>
        解析
      </div>
      <p>{{ question.explanation }}</p>
    </div>

    <!-- 操作按钮 -->
    <div class="question-actions">
      <button
        v-if="!question.isChecked"
        @click="checkAnswer"
        :disabled="!hasAnswer"
        class="check-btn"
      >
        检查答案
      </button>
      <div v-else class="result-badge" :class="{ correct: question.isCorrect, incorrect: question.isCorrect === false }">
        {{ question.isCorrect === null ? '已提交' : (question.isCorrect ? '✓ 正确' : '✗ 错误') }}
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue';

const props = defineProps({
  question: {
    type: Object,
    required: true
  },
  index: {
    type: Number,
    required: true
  }
});

const emit = defineEmits(['answer', 'check']);

const essayAnswer = ref('');

const typeLabel = computed(() => {
  const labels = {
    single: '单选题',
    multiple: '多选题',
    judge: '判断题',
    essay: '简答题'
  };
  return labels[props.question.type] || '题目';
});

const difficultyLabel = computed(() => {
  const labels = {
    easy: '简单',
    medium: '中等',
    hard: '困难'
  };
  return labels[props.question.difficulty] || props.question.difficulty;
});

// 标准化选项格式 - 处理各种可能的选项格式
const normalizedOptions = computed(() => {
  const options = props.question.options;
  
  // 如果没有选项，返回空数组
  if (!options) return [];
  
  // 如果已经是正确格式的数组
  if (Array.isArray(options)) {
    return options.map((opt, idx) => {
      // 如果选项是对象且有value和text
      if (opt && typeof opt === 'object') {
        return {
          value: opt.value || opt.label || String.fromCharCode(65 + idx), // A, B, C, D
          text: opt.text || opt.content || opt.label || String(opt)
        };
      }
      // 如果选项是字符串
      if (typeof opt === 'string') {
        return {
          value: String.fromCharCode(65 + idx),
          text: opt
        };
      }
      return { value: String.fromCharCode(65 + idx), text: String(opt) };
    });
  }
  
  // 如果是对象格式 {A: "选项A", B: "选项B"}
  if (typeof options === 'object') {
    return Object.entries(options).map(([key, value]) => ({
      value: key,
      text: typeof value === 'string' ? value : (value?.text || String(value))
    }));
  }
  
  return [];
});

const hasAnswer = computed(() => {
  if (props.question.type === 'essay') {
    return essayAnswer.value.trim().length > 0;
  }
  if (props.question.type === 'multiple') {
    return Array.isArray(props.question.userAnswer) && props.question.userAnswer.length > 0;
  }
  return props.question.userAnswer !== null && props.question.userAnswer !== undefined;
});

const isMultipleSelected = (value) => {
  return Array.isArray(props.question.userAnswer) && props.question.userAnswer.includes(value);
};

const isCorrectOption = (value) => {
  const correctAnswer = props.question.correctAnswer;
  if (Array.isArray(correctAnswer)) {
    return correctAnswer.includes(value);
  }
  // 处理字符串格式的多选答案，如 "A,B,C"
  if (typeof correctAnswer === 'string' && correctAnswer.includes(',')) {
    return correctAnswer.split(',').map(s => s.trim()).includes(value);
  }
  return correctAnswer === value;
};

const selectAnswer = (answer) => {
  if (props.question.isChecked) return;
  emit('answer', { questionId: props.question.id, answer });
};

const toggleMultipleAnswer = (value) => {
  if (props.question.isChecked) return;
  let currentAnswers = Array.isArray(props.question.userAnswer) ? [...props.question.userAnswer] : [];
  
  if (currentAnswers.includes(value)) {
    currentAnswers = currentAnswers.filter(a => a !== value);
  } else {
    currentAnswers.push(value);
  }
  
  emit('answer', { questionId: props.question.id, answer: currentAnswers });
};

const checkAnswer = () => {
  if (props.question.type === 'essay') {
    emit('answer', { questionId: props.question.id, answer: essayAnswer.value });
  }
  emit('check', props.question.id);
};

// 同步简答题答案
watch(() => props.question.userAnswer, (newVal) => {
  if (props.question.type === 'essay' && typeof newVal === 'string') {
    essayAnswer.value = newVal;
  }
});
</script>

<style scoped>
.question-card {
  background: var(--secondary-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 16px;
  margin-bottom: 16px;
  transition: all 0.3s ease;
}

.question-card.checked.correct {
  border-color: #22c55e;
  background: rgba(34, 197, 94, 0.05);
}

.question-card.checked.incorrect {
  border-color: #ef4444;
  background: rgba(239, 68, 68, 0.05);
}

.question-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.question-number {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--accent-color);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
}

.question-type {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.question-type.single { background: #dbeafe; color: #1d4ed8; }
.question-type.multiple { background: #fce7f3; color: #be185d; }
.question-type.judge { background: #d1fae5; color: #047857; }
.question-type.essay { background: #fef3c7; color: #b45309; }

.difficulty {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  margin-left: auto;
}

.difficulty.easy { background: #d1fae5; color: #047857; }
.difficulty.medium { background: #fef3c7; color: #b45309; }
.difficulty.hard { background: #fee2e2; color: #dc2626; }

.score {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  background: #e0e7ff;
  color: #4338ca;
}

.tags-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 12px;
}

.tag {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  background: var(--hover-bg);
  color: var(--text-secondary);
}

.question-content {
  margin-bottom: 16px;
}

.question-text {
  margin: 0;
  color: var(--text-primary);
  font-size: 15px;
  line-height: 1.6;
}

/* 选项样式 */
.options-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.option-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.option-item:hover:not(.correct):not(.incorrect) {
  background: var(--hover-bg);
}

.option-item.selected {
  border-color: var(--accent-color);
  background: rgba(0, 122, 255, 0.05);
}

.option-item.correct {
  border-color: #22c55e;
  background: rgba(34, 197, 94, 0.1);
}

.option-item.incorrect {
  border-color: #ef4444;
  background: rgba(239, 68, 68, 0.1);
}

.option-item input {
  margin: 0;
}

.option-label {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--border-color);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
}

.option-item.selected .option-label {
  background: var(--accent-color);
  color: white;
}

.option-text {
  flex: 1;
  color: var(--text-primary);
  font-size: 14px;
}

.correct-mark {
  color: #22c55e;
  font-weight: bold;
}

/* 判断题样式 */
.judge-options {
  display: flex;
  gap: 12px;
}

.judge-btn {
  flex: 1;
  padding: 12px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--primary-bg);
  color: var(--text-primary);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.judge-btn:hover:not(:disabled) {
  background: var(--hover-bg);
}

.judge-btn.selected {
  border-color: var(--accent-color);
  background: rgba(0, 122, 255, 0.1);
}

.judge-btn.correct {
  border-color: #22c55e;
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.judge-btn.incorrect {
  border-color: #ef4444;
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.judge-btn:disabled {
  cursor: not-allowed;
  opacity: 0.7;
}

/* 简答题样式 */
.essay-answer textarea {
  width: 100%;
  padding: 12px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--primary-bg);
  color: var(--text-primary);
  font-size: 14px;
  resize: vertical;
  font-family: inherit;
}

.essay-answer textarea:focus {
  outline: none;
  border-color: var(--accent-color);
}

.reference-answer {
  margin-top: 12px;
  padding: 12px;
  background: rgba(34, 197, 94, 0.1);
  border-radius: 8px;
  border-left: 3px solid #22c55e;
}

.reference-answer strong {
  color: #22c55e;
  font-size: 13px;
}

.reference-answer p {
  margin: 8px 0 0 0;
  color: var(--text-primary);
  font-size: 14px;
  line-height: 1.6;
}

/* 解析区域 */
.explanation {
  margin-top: 16px;
  padding: 12px;
  background: rgba(0, 122, 255, 0.05);
  border-radius: 8px;
  border-left: 3px solid var(--accent-color);
}

.explanation-header {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--accent-color);
  font-weight: 600;
  font-size: 13px;
  margin-bottom: 8px;
}

.explanation p {
  margin: 0;
  color: var(--text-primary);
  font-size: 14px;
  line-height: 1.6;
}

/* 操作按钮 */
.question-actions {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.check-btn {
  padding: 8px 20px;
  border: none;
  border-radius: 6px;
  background: var(--accent-color);
  color: white;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.check-btn:hover:not(:disabled) {
  background: #0056b3;
}

.check-btn:disabled {
  background: var(--border-color);
  cursor: not-allowed;
}

.result-badge {
  padding: 6px 16px;
  border-radius: 20px;
  font-size: 13px;
  font-weight: 500;
}

.result-badge.correct {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.result-badge.incorrect {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}
</style>
